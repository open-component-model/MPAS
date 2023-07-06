// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/bootstrap"
	"github.com/open-component-model/mpas/pkg/bootstrap/provider"
	"github.com/open-component-model/mpas/pkg/env"
	"github.com/open-component-model/mpas/pkg/kubeutils"
)

const (
	githubDefaultHostname = "github.com"
)

// BootstrapGithubCmd is a command for bootstrapping a GitHub repository
type BootstrapGithubCmd struct {
	// Owner is the owner of the repository
	Owner string
	// Token is the token to use for authentication
	Token string
	// Personal indicates whether the repository is a personal repository
	Personal bool
	// Hostname is the hostname of the Github instance
	Hostname string
	// Repository is the name of the repository
	Repository string
	// FromFile is the path to a file archive to use for bootstrapping
	FromFile string
	// Registry is the registry to use for the bootstrap components
	Registry string
	// DockerconfigPath is the path to the docker config file
	DockerconfigPath string
	// Path is the path in the repository to use to host the bootstraped components yamls
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
	bootstrapper       *bootstrap.Bootstrap
}

// Execute executes the command and returns an error if one occurred.
func (b *BootstrapGithubCmd) Execute(cfg *config.MpasConfig) error {
	t, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cfg.Context(), t)
	defer cancel()

	hostname := githubDefaultHostname
	if b.Hostname != "" {
		hostname = b.Hostname
	}

	providerOpts := provider.ProviderOptions{
		Provider:           env.ProviderGithub,
		Hostname:           hostname,
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
	)

	if err != nil {
		return err
	}

	return b.bootstrapper.Run(ctx)
}

// Cleanup cleans up the resources created by the command.
func (b *BootstrapGithubCmd) Cleanup(ctx context.Context) error {
	if b.bootstrapper != nil {
		return b.bootstrapper.DeleteManagementRepository(ctx)
	}
	return nil
}
