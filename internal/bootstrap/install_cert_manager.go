// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-component-model/mpas/internal/kubeutils"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type certManagerOptions struct {
	kubeClient       client.Client
	restClientGetter genericclioptions.RESTClientGetter
	dir              string
	timeout          time.Duration
}

// certManagerInstall is used to install cert-manager
type certManagerInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
	*certManagerOptions
	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

// newCertManagerInstall returns a new component install
func newCertManagerInstall(name, version string, repository ocm.Repository, opts *certManagerOptions) (*certManagerInstall, error) {
	c := &certManagerInstall{
		componentName:      name,
		version:            version,
		repository:         repository,
		certManagerOptions: opts,
	}

	return c, nil
}

func (c *certManagerInstall) Install(ctx context.Context, component string) error {
	cv, err := getComponentVersion(c.repository, c.componentName, c.version)
	if err != nil {
		return fmt.Errorf("failed to get component version: %w", err)
	}

	resources, err := getResources(cv, component)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	if resources.componentResource == nil || resources.ocmConfig == nil {
		return fmt.Errorf("failed to get component resource or ocm config")
	}

	content := resources.componentResource

	tmp, err := os.MkdirTemp("", "cert-manager")
	if err != nil {
		return fmt.Errorf("failed to create temp folder: %w", err)
	}
	defer os.RemoveAll(tmp)

	if err := os.WriteFile(filepath.Join(tmp, "cert-manager.yaml"), content, 0o600); err != nil {
		return fmt.Errorf("failed to write out cert manager manifest: %w", err)
	}

	if _, err := kubeutils.Apply(ctx, c.restClientGetter, tmp, filepath.Join(tmp, "cert-manager.yaml")); err != nil {
		return fmt.Errorf("failed to apply manifest to cluster: %w", err)
	}

	return nil
}
