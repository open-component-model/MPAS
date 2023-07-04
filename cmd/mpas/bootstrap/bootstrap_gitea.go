// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/bootstrap"
	"github.com/open-component-model/mpas/pkg/bootstrap/provider"
	"github.com/open-component-model/mpas/pkg/kubeutils"
)

// BootstrapGiteaCmd is a command for bootstrapping a Gitea repository
type BootstrapGiteaCmd struct {
	Owner                 string
	Token                 string
	Personal              bool
	Hostname              string
	Repository            string
	FromFile              string
	Registry              string
	DockerconfigPath      string
	Target                string
	CommitMessageAppendix string
	Interval              time.Duration
	Timeout               time.Duration
	Components            []string
	DestructiveActions    bool
	bootstrapper          *bootstrap.Bootstrap
}

// Execute executes the command and returns an error if one occurred.
func (b *BootstrapGiteaCmd) Execute(cfg *config.MpasConfig) error {
	t, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cfg.Context(), t)
	defer cancel()

	if b.Hostname == "" {
		return fmt.Errorf("hostname must be specified")
	}

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

	kubeClient, err := kubeutils.KubeClient(cfg.KubeConfigArgs)
	if err != nil {
		return err
	}

	b.bootstrapper, err = bootstrap.New(providerClient,
		bootstrap.WithOwner(b.Owner),
		bootstrap.WithRepositoryName(b.Repository),
		bootstrap.WithPersonal(b.Personal),
		bootstrap.WithFromFile(b.FromFile),
		bootstrap.WithRegistry(b.Registry),
		bootstrap.WithPrinter(cfg.Printer),
		bootstrap.WithComponents(b.Components),
		bootstrap.WithToken(b.Token),
		bootstrap.WithTransportType("https"),
		bootstrap.WithDockerConfigPath(b.DockerconfigPath),
		bootstrap.WithTarget(b.Target),
		bootstrap.WithKubeClient(kubeClient),
		bootstrap.WithRESTClientGetter(cfg.KubeConfigArgs),
		bootstrap.WithInterval(b.Interval),
		bootstrap.WithTimeout(b.Timeout),
		bootstrap.WithCommitMessageAppendix(b.CommitMessageAppendix),
	)

	if err != nil {
		return err
	}

	return b.bootstrapper.Run(ctx)
}

// Cleanup cleans up the resources created by the command.
func (b *BootstrapGiteaCmd) Cleanup(ctx context.Context) error {
	if b.bootstrapper != nil {
		return b.bootstrapper.DeleteManagementRepository(ctx)
	}
	return nil
}
