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

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/pkg/git"
	"github.com/open-component-model/mpas/pkg/env"
	"github.com/open-component-model/mpas/pkg/kubeutils"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kustypes "sigs.k8s.io/kustomize/api/types"
)

type componentOptions struct {
	// gitClient  repository.Client
	kubeClient client.Client

	restClientGetter genericclioptions.RESTClientGetter

	gitRepository gitprovider.UserRepository

	branch    string
	target    string
	namespace string
	dir       string

	timeout time.Duration

	signature             git.Signature
	commitMessageAppendix string
	gpgKeyRing            openpgp.EntityList
	gpgPassphrase         string
	gpgKeyID              string
}

// componentOption is a function that configures a componentInstall.
type componentOption func(*componentOptions)

type componentInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
	components    []string
	installedNS   map[string][]string
	componentOptions

	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

func withComponentTimeout(timeout time.Duration) componentOption {
	return func(o *componentOptions) {
		o.timeout = timeout
	}
}

func withComponentKubeConfig(kubeconfig genericclioptions.RESTClientGetter) componentOption {
	return func(o *componentOptions) {
		o.restClientGetter = kubeconfig
	}
}

func withComponentKubeClient(kubeClient client.Client) componentOption {
	return func(o *componentOptions) {
		o.kubeClient = kubeClient
	}
}

func withComponentGitRepository(gitRepository gitprovider.UserRepository) componentOption {
	return func(o *componentOptions) {
		o.gitRepository = gitRepository
	}
}

func withComponentBranch(branch string) componentOption {
	return func(o *componentOptions) {
		o.branch = branch
	}
}

func withComponentTarget(target string) componentOption {
	return func(o *componentOptions) {
		o.target = target
	}
}

func withComponentNamespace(namespace string) componentOption {
	return func(o *componentOptions) {
		o.namespace = namespace
	}
}

func withComponentDir(dir string) componentOption {
	return func(o *componentOptions) {
		o.dir = dir
	}
}

func NewComponentInstall(name, version string, repository ocm.Repository, installedNS map[string][]string, opts ...componentOption) (*componentInstall, error) {
	c := &componentInstall{
		componentName: name,
		version:       version,
		repository:    repository,
		installedNS:   installedNS,
	}
	for _, o := range opts {
		o(&c.componentOptions)
	}

	return c, nil
}

func (c *componentInstall) Install(ctx context.Context, component string) error {
	cv, err := GetComponentVersion(c.repository, c.componentName, c.version)
	if err != nil {
		return fmt.Errorf("failed to get component version: %w", err)
	}

	componentResource, ocmConfig, imagesResources, comps, err := getResources(cv, component)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	c.components = comps

	if componentResource == nil || ocmConfig == nil {
		return fmt.Errorf("failed to get component resource or ocm config")
	}

	kfile, kus, err := c.generateKustomization(componentResource, ocmConfig)
	if err != nil {
		return fmt.Errorf("failed to generate kustomization: %w", err)
	}

	kconfig, err := unMarshallConfig(ocmConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshall config: %w", err)
	}

	res, err := c.generateComponentYaml(kconfig, imagesResources, kus, kfile)
	if err != nil {
		return fmt.Errorf("failed to generate component yaml: %w", err)
	}

	err = c.reconcileComponents(ctx, res)
	if err != nil {
		return fmt.Errorf("failed to reconcile components: %w", err)
	}

	return nil
}

func (c *componentInstall) reconcileComponents(ctx context.Context, content []byte) error {
	if _, ok := c.installedNS[c.namespace]; ok {
		// remove ns from content
		objects, err := kubeutils.YamlToUnstructructured(content)
		if err != nil {
			return fmt.Errorf("failed to convert yaml to unstructured: %w", err)
		}

		content, err = kubeutils.UnstructuredToYaml(kubeutils.FilterUnstructured(objects, kubeutils.NSFilter(c.namespace)))
		if err != nil {
			return fmt.Errorf("failed to convert unstructured to yaml: %w", err)
		}
	}

	data := string(content)
	path := filepath.Join(c.target, c.namespace, fmt.Sprintf("%s.yaml", strings.Split(c.componentName, "/")[2]))
	if _, err := c.gitRepository.Commits().Create(ctx,
		c.branch,
		c.commitMessageAppendix,
		[]gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &data,
			},
		}); err != nil {
		return fmt.Errorf("failed to create component: %w", err)
	}

	return nil
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
