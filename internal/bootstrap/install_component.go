// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/kubeutils"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
)

type componentOptions struct {
	gitRepository gitprovider.UserRepository
	branch        string
	targetPath    string
	namespace     string
	dir           string
	provider      string
	// we bookkeep the installed components so we can cleanup unnecessary namespaces
	installedNS           map[string][]string
	commitMessageAppendix string
	timeout               time.Duration
}

// componentInstall is used to install a component
type componentInstall struct {
	componentName string
	version       string
	kustomizer    *Kustomizer

	*componentOptions
}

// newComponentInstall returns a new component install
func newComponentInstall(name, version string, repository ocm.Repository, opts *componentOptions) (*componentInstall, error) {
	c := &componentInstall{
		componentName:    name,
		version:          version,
		componentOptions: opts,
		kustomizer: NewKustomizer(&kustomizerOptions{
			componentName: name,
			version:       version,
			repository:    repository,
			dir:           opts.dir,
		}),
	}

	return c, nil
}

func (c *componentInstall) install(ctx context.Context, component string) (string, error) {
	res, err := c.kustomizer.generateKustomizedResourceData(component)
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
