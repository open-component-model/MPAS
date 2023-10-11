// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/mpas/internal/kubeutils"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kustypes "sigs.k8s.io/kustomize/api/types"
)

type componentOptions struct {
	kubeClient       client.Client
	restClientGetter genericclioptions.RESTClientGetter
	gitRepository    gitprovider.UserRepository
	branch           string
	targetPath       string
	namespace        string
	dir              string
	provider         string
	// we bookkeep the installed components so we can cleanup unnecessary namespaces
	installedNS           map[string][]string
	commitMessageAppendix string
	timeout               time.Duration
}

// componentInstall is used to install a component
type componentInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
	*componentOptions
	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

// newComponentInstall returns a new component install
func newComponentInstall(name, version string, repository ocm.Repository, opts *componentOptions) (*componentInstall, error) {
	c := &componentInstall{
		componentName:    name,
		version:          version,
		repository:       repository,
		componentOptions: opts,
	}

	return c, nil
}

func (c *componentInstall) install(ctx context.Context, component string) (string, error) {
	cv, err := getComponentVersion(c.repository, c.componentName, c.version)
	if err != nil {
		return "", fmt.Errorf("failed to get component version: %w", err)
	}

	resources, err := getResources(cv, component)
	if err != nil {
		return "", fmt.Errorf("failed to get resources: %w", err)
	}

	if resources.componentResource == nil || resources.ocmConfig == nil {
		return "", fmt.Errorf("failed to get component resource or ocm config")
	}

	kfile, kus, err := c.generateKustomization(resources.componentResource, resources.ocmConfig)
	if err != nil {
		return "", fmt.Errorf("failed to generate kustomization: %w", err)
	}

	kconfig, err := unMarshallConfig(resources.ocmConfig)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshall config: %w", err)
	}

	res, err := c.generateComponentYaml(kconfig, resources.imagesResources, kus, kfile)
	if err != nil {
		return "", fmt.Errorf("failed to generate component yaml: %w", err)
	}

	sha, err := c.reconcileComponents(ctx, res)
	if err != nil {
		return "", fmt.Errorf("failed to reconcile components: %w", err)
	}

	return sha, nil
}

func (c *componentInstall) reconcileComponents(ctx context.Context, content []byte) (string, error) {
	if _, ok := c.installedNS[c.namespace]; ok {
		// remove ns from content
		objects, err := kubeutils.YamlToUnstructructured(content)
		if err != nil {
			return "", fmt.Errorf("failed to convert yaml to unstructured: %w", err)
		}

		content, err = kubeutils.UnstructuredToYaml(kubeutils.FilterUnstructured(objects, kubeutils.NSFilter(c.namespace)))
		if err != nil {
			return "", fmt.Errorf("failed to convert unstructured to yaml: %w", err)
		}
	}

	data := SetProviderDataFormat(c.provider, content)
	path := filepath.Join(c.targetPath, c.namespace, fmt.Sprintf("%s.yaml", strings.Split(c.componentName, "/")[2]))
	commitMsg := fmt.Sprintf("Add %s %s manifests", c.componentName, c.version)
	if c.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + c.commitMessageAppendix
	}
	commit, err := c.gitRepository.Commits().Create(ctx,
		c.branch,
		commitMsg,
		[]gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &data,
			},
		})
	if err != nil {
		return "", fmt.Errorf("failed to create component: %w", err)
	}

	return commit.Get().Sha, nil
}

func (c *componentInstall) generateKustomization(componentResource []byte, ocmConfig []byte) (string, kustypes.Kustomization, error) {
	if err := os.WriteFile(filepath.Join(c.dir, fmt.Sprintf("%s.yaml", strings.Split(c.componentName, "/")[2])), componentResource, os.ModePerm); err != nil {
		return "", kustypes.Kustomization{}, err
	}

	return genKus(c.dir, ocmConfig, fmt.Sprintf("./%s.yaml", strings.Split(c.componentName, "/")[2]))
}

func (c *componentInstall) generateComponentYaml(kconfig *cfd.ConfigData, imagesResources map[string]nameTag, kus kustypes.Kustomization, kfile string) ([]byte, error) {
	for _, loc := range kconfig.Localization {
		image := imagesResources[loc.Resource.Name]
		kus.Images = append(kus.Images, kustypes.Image{
			Name:    fmt.Sprintf("%s/%s", env.DefaultOCMHost, loc.Resource.Name),
			NewName: image.Name,
			NewTag:  image.Tag,
		})
	}

	return buildKustomization(kus, kfile, c.dir, &c.mu)
}
