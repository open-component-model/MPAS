// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	certManager           = "cert-manager"
	certManagerCAInjector = "cert-manager-cainjector"
	certManagerWebhook    = "cert-manager-webhook"
)

var (
	//go:embed certmanager/bootstrap.yaml
	certManagerBootstrapManifest []byte
)

type certManagerOptions struct {
	kubeClient            client.Client
	restClientGetter      genericclioptions.RESTClientGetter
	gitRepository         gitprovider.UserRepository
	dir                   string
	branch                string
	targetPath            string
	namespace             string
	provider              string
	timeout               time.Duration
	commitMessageAppendix string
}

// certManagerInstall is used to install cert-manager
type certManagerInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
	*certManagerOptions
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

func (c *certManagerInstall) Install(ctx context.Context, component string) (string, error) {
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

	content := resources.componentResource

	sha, err := c.createCommit(ctx, content)
	if err != nil {
		return "", fmt.Errorf("failed to reconcile components: %w", err)
	}

	return sha, nil
}

func (c *certManagerInstall) createCommit(ctx context.Context, content []byte) (string, error) {
	data := string(content)
	if c.provider == env.ProviderGitea {
		data = base64.StdEncoding.EncodeToString(content)
	}
	path := filepath.Join(c.targetPath, c.namespace, fmt.Sprintf("%s.yaml", strings.Split(c.componentName, "/")[2]))
	commitMsg := fmt.Sprintf("Add %s %s manifests", c.componentName, c.version)
	if c.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + c.commitMessageAppendix
	}
	bootstrapYaml := filepath.Join(c.targetPath, c.namespace, "bootstrap.yaml")
	commit, err := c.gitRepository.Commits().Create(ctx,
		c.branch,
		commitMsg,
		[]gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &data,
			},
			{
				Path:    &bootstrapYaml,
				Content: pointer.String(string(certManagerBootstrapManifest)),
			},
		})
	if err != nil {
		return "", fmt.Errorf("failed to create component: %w", err)
	}

	return commit.Get().Sha, nil
}
