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

// BootstrapGiteaCmd is a command for bootstrapping a Gitea repository
type BootstrapGiteaCmd struct {
	Owner              string
	Token              string
	Personal           bool
	Hostname           string
	Repository         string
	FromFile           string
	Registry           string
	Components         []string
	DestructiveActions bool
	bootstrapper       *bootstrap.Bootstrap
}

// Execute executes the command and returns an error if one occurred.
func (b *BootstrapGiteaCmd) Execute() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	providerOpts := provider.ProviderOptions{
		Provider:           provider.ProviderGitea,
		Hostname:           b.Hostname,
		Token:              b.Token,
		DestructiveActions: b.DestructiveActions,
	}

	providerClient, err := provider.New().Build(providerOpts)
	if err != nil {
		return err
	}

	b.bootstrapper = bootstrap.New(ctx, providerClient,
		bootstrap.WithOwner(b.Owner),
		bootstrap.WithRepositoryName(b.Repository),
		bootstrap.WithPersonal(b.Personal),
		bootstrap.WithFromFile(b.FromFile),
		bootstrap.WithRegistry(b.Registry),
	)

	return b.bootstrapper.Run()
}

// Cleanup cleans up the resources created by the command.
func (b *BootstrapGiteaCmd) Cleanup(ctx context.Context) error {
	if b.bootstrapper != nil {
		return b.bootstrapper.DeleteManagementRepository(ctx)
	}
	return nil
}
