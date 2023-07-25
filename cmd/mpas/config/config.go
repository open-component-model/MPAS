// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/open-component-model/mpas/internal/printer"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// MpasConfig is the global configuration for the mpas CLI.
type MpasConfig struct {
	// Printer is the printer to use for output.
	Printer *printer.Printer
	// Timeout is the timeout to use for operations.
	Timeout string
	// DockerconfigPath is the path to the docker config file.
	DockerconfigPath string
	// KubeConfigArgs are the kubeconfig arguments.
	KubeConfigArgs *genericclioptions.ConfigFlags
	// PlainHTTP indicates whether to use plain HTTP instead of HTTPS.
	PlainHTTP bool
	// Export indicates whether to export to a file.
	Export bool
	// ExportPath is the path to export to.
	ExportPath string
}

// AddFlags adds the global flags to the given flag set.
func (m *MpasConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&m.Timeout, "timeout", "5m", "The timeout to use for operations")
	flags.StringVar(&m.DockerconfigPath, "dockerconfigpath", "~/.docker/config.json", "The path to the docker config file")
	flags.BoolVar(&m.PlainHTTP, "plain-http", false, "Whether to use plain HTTP instead of HTTPS")
	flags.BoolVar(&m.Export, "export", false, "Whether to export to a file")
	flags.StringVar(&m.ExportPath, "export-path", "", "The path to export to. Defaults to the temporary directory")
}

// BootstrapConfig is the configuration shared by the bootstrap commands.
type BootstrapConfig struct {
	Components            []string
	Owner                 string
	Repository            string
	FromFile              string
	Registry              string
	Hostname              string
	Path                  string
	Interval              string
	CommitMessageAppendix string
	Private               bool
}

// AddFlags adds the bootstrap flags to the given flag set.
func (m *BootstrapConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringSliceVar(&m.Components, "components", []string{}, "The components to include in the management repository")
	flags.StringVar(&m.Owner, "owner", "", "The owner of the management repository")
	flags.StringVar(&m.Repository, "repository", "", "The name of the management repository")
	flags.StringVar(&m.FromFile, "from-file", "", "The path to a file containing the bootstrap component in archive format")
	flags.StringVar(&m.Registry, "registry", "", "The registry to use to retrieve the bootstrap component. Defaults to ghcr.io/open-component-model/mpas-bootstrap-component")
	flags.StringVar(&m.Hostname, "hostname", "", "The hostname of the Git provider")
	flags.StringVar(&m.Path, "path", ".", "The target path to use in the management repository to store the bootstrap component")
	flags.StringVar(&m.Interval, "interval", "5m", "The interval to use to sync the bootstrap component")
	flags.StringVar(&m.CommitMessageAppendix, "commit-message-appendix", "", "The appendix to add to the commit message, e.g. [ci skip]")
	flags.BoolVar(&m.Private, "private", false, "Whether the management repository should be private")
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
