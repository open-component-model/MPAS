// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
)

const (
	externalSecret               = "external-secrets"
	externalSecretWebhook        = "external-secrets-webhook"
	externalSecretCertController = "external-secrets-cert-controller"
)

type externalSecretOptions struct {
	gitRepository         gitprovider.UserRepository
	dir                   string
	branch                string
	targetPath            string
	namespace             string
	provider              string
	timeout               time.Duration
	commitMessageAppendix string
}

// externalSecretInstall is used to install external-secrets
type externalSecretInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
	kustomizer    Kustomizer

	*externalSecretOptions
}

// newExternalSecretInstall returns a new component install
func newExternalSecretInstall(name, version string, repository ocm.Repository, opts *externalSecretOptions) (*externalSecretInstall, error) {
	c := &externalSecretInstall{
		componentName:         name,
		version:               version,
		repository:            repository,
		externalSecretOptions: opts,
		kustomizer: NewKustomizer(&kustomizerOptions{
			componentName: name,
			version:       version,
			repository:    repository,
			dir:           opts.dir,
			host:          env.DefaultExternalSecretsHost,
		}),
	}

	return c, nil
}

func (c *externalSecretInstall) Install(ctx context.Context, component string) (string, error) {
	res, err := c.kustomizer.GenerateKustomizedResourceData(component)
	if err != nil {
		return "", fmt.Errorf("failed to generate component yaml: %w", err)
	}

	sha, err := c.createCommit(ctx, res)
	if err != nil {
		return "", fmt.Errorf("failed to reconcile external secret: %w", err)
	}

	return sha, nil
}

func (c *externalSecretInstall) createCommit(ctx context.Context, content []byte) (string, error) {
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
