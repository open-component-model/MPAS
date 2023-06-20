// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"

	"github.com/open-component-model/mpas/pkg/printer"
	"github.com/spf13/pflag"
)

// DefaultComponents is the list of components to include in the management repository by default.
var DefaultComponents = []string{
	"ocm-controller",
	"flux",
}

// MpasConfig is the global configuration for the mpas CLI.
type MpasConfig struct {
	Kubeconfig string
	Printer    *printer.Printer
	Timeout    string
	ctx        context.Context
}

// SetContext sets the context to use for operations.
func (m *MpasConfig) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// Context returns the context to use for operations.
func (m *MpasConfig) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

// AddFlags adds the global flags to the given flag set.
func (m *MpasConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&m.Kubeconfig, "kubeconfig", "", "Path to a kubeconfig file")
	flags.StringVar(&m.Timeout, "timeout", "5m", "The timeout to use for operations")
}

// BootstrapConfig is the configuration shared by the bootstrap commands.
type BootstrapConfig struct {
	Components []string
	Owner      string
	Repository string
	FromFile   string
	Registry   string
	Hostname   string
}

// AddFlags adds the bootstrap flags to the given flag set.
func (m *BootstrapConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringSliceVar(&m.Components, "components", []string{}, "The components to include in the management repository")
	flags.StringVar(&m.Owner, "owner", "", "The owner of the management repository")
	flags.StringVar(&m.Repository, "repository", "", "The name of the management repository")
	flags.StringVar(&m.FromFile, "from-file", "", "The path to a file containing the bootstrap component in archive format")
	flags.StringVar(&m.Registry, "registry", "", "The registry to use to retrieve the bootstrap component")
	flags.StringVar(&m.Hostname, "hostname", "", "The hostname of the Git provider")
}

// GithubConfig is the configuration for the Github bootstrap command.
type GithubConfig struct {
	BootstrapConfig
	Personal bool
}

// AddFlags adds the Github bootstrap flags to the given flag set.
func (g *GithubConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&g.Personal, "personal", false, "The personal access token to use to access the Github API")
	g.BootstrapConfig.AddFlags(flags)
}

// GiteaConfig is the configuration for the Github bootstrap command.
type GiteaConfig struct {
	BootstrapConfig
	Personal bool
}

// AddFlags adds the Gitea bootstrap flags to the given flag set.
func (g *GiteaConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&g.Personal, "personal", false, "The personal access token to use to access the Gitea API")
	g.BootstrapConfig.AddFlags(flags)
}
