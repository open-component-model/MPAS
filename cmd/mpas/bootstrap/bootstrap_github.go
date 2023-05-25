// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"time"

	"github.com/open-component-model/mpas/pkg/bootstrap"
	"github.com/open-component-model/mpas/pkg/bootstrap/provider"
)

const (
	githubDefaultHostname = "github.com"
)

// BootstrapGithubCmd is a command for bootstrapping a GitHub repository
type BootstrapGithubCmd struct {
	Owner      string
	Token      string
	Personal   bool
	Hostname   string
	Repository string
	FromFile   string
	Registry   string
	Components []string
}

// Execute executes the command and returns an error if one occurred.
func (b *BootstrapGithubCmd) Execute() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hostname := githubDefaultHostname
	if b.Hostname != "" {
		hostname = b.Hostname
	}

	providerOpts := provider.ProviderOptions{
		Provider: provider.ProviderGithub,
		Hostname: hostname,
		Token:    b.Token,
	}

	providerClient, err := provider.New().Build(providerOpts)
	if err != nil {
		return err
	}

	bootstrapper := bootstrap.New(ctx, providerClient,
		bootstrap.WithOwner(b.Owner),
		bootstrap.WithRepositoryName(b.Repository),
		bootstrap.WithPersonal(b.Personal),
		bootstrap.WithFromFile(b.FromFile),
		bootstrap.WithRegistry(b.Registry),
	)

	return bootstrapper.Run()
}
