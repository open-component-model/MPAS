// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/internal/bootstrap"
	"github.com/open-component-model/mpas/internal/bootstrap/provider"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/mpas/internal/kubeutils"
)

// GiteaCmd is a command for bootstrapping a Gitea repository
type GiteaCmd struct {
	// Owner is the owner of the repository
	Owner string
	// Token is the token to use for authentication
	Token string
	// Personal indicates whether the repository is a personal repository
	Personal bool
	// Hostname is the hostname of the Gitea instance
	Hostname string
	// Repository is the name of the repository
	Repository string
	// FromFile is the path to a file archive to use for bootstrapping
	FromFile string
	// Registry is the registry to use for the bootstrap components
	Registry string
	// DockerconfigPath is the path to the docker config file
	DockerconfigPath string
	// Path is the path in the repository to use to host the bootstrapped components yamls
	Path string
	// CommitMessageAppendix is the appendix to add to the commit message
	// for example to skip CI
	CommitMessageAppendix string
	// Private indicates whether the repository is private
	Private bool
	// Interval is the interval to use for reconciling
	Interval time.Duration
	// Timeout is the timeout to use for operations
	Timeout time.Duration
	// Components is the list of components to install
	Components []string
	// DestructiveActions indicates whether destructive actions are allowed
	DestructiveActions bool
	// TestURL is the URL to use for testing the management repository
	TestURL string
	// DeveloperCertificateName defines if a developer certificate has to be provided or not
	DeveloperCertificateName string
	bootstrapper             *bootstrap.Bootstrap
}

// Execute executes the command and returns an error if one occurred.
func (b *GiteaCmd) Execute(ctx context.Context, cfg *config.MpasConfig) error {
	t, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	if b.Hostname == "" {
		return fmt.Errorf("hostname must be specified")
	}

	providerOpts := provider.ProviderOptions{
		Provider:           env.ProviderGitea,
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

	visibility := "public"
	if b.Private {
		visibility = "private"
	}

	transport := "https"
	if cfg.PlainHTTP {
		transport = "http"
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
		bootstrap.WithTransportType(transport),
		bootstrap.WithDockerConfigPath(b.DockerconfigPath),
		bootstrap.WithTarget(b.Path),
		bootstrap.WithKubeClient(kubeClient),
		bootstrap.WithRESTClientGetter(cfg.KubeConfigArgs),
		bootstrap.WithInterval(b.Interval),
		bootstrap.WithTimeout(b.Timeout),
		bootstrap.WithCommitMessageAppendix(b.CommitMessageAppendix),
		bootstrap.WithVisibility(visibility),
		bootstrap.WithTestURL(b.TestURL),
		bootstrap.WithDevCertificate(b.DeveloperCertificateName),
	)

	if err != nil {
		return err
	}

	return b.bootstrapper.Run(ctx)
}

// Cleanup cleans up the resources created by the command.
func (b *GiteaCmd) Cleanup(ctx context.Context) error {
	if b.bootstrapper != nil {
		return b.bootstrapper.DeleteManagementRepository(ctx)
	}
	return nil
}
