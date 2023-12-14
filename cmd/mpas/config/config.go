// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"

	"github.com/open-component-model/mpas/internal/env"
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
	// PollInterval is used for Api call where we wait on a resource to be ready.
	PollInterval time.Duration
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
	// Components is the list of components to install.
	Components []string
	// Owner is the owner of the management repository.
	Owner string
	// Repository is the name of the management repository.
	Repository string
	// FromFile is the path to a file containing the bootstrap component in archive format.
	FromFile string
	// Registry is the registry to use for the bootstrap components
	Registry string
	// Hostname is the hostname of the Git provider.
	Hostname string
	// Path is the target path to use in the management repository to store the bootstrap component.
	Path string
	// Interval is the interval to use to sync the bootstrap component.
	Interval string
	// CommitMessageAppendix is the appendix to add to the commit message
	// for example to skip CI
	CommitMessageAppendix string
	// Private indicates whether the management repository should be private.
	Private bool
	// CaFile defines and optional root certificate for the git repository used by flux.
	CaFile string
}

// AddFlags adds the bootstrap flags to the given flag set.
func (m *BootstrapConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringSliceVar(&m.Components, "components", []string{env.ExternalSecretsName}, "The components to include in the management repository")
	flags.StringVar(&m.Owner, "owner", "", "The owner of the management repository")
	flags.StringVar(&m.Repository, "repository", "", "The name of the management repository")
	flags.StringVar(&m.FromFile, "from-file", "", "The path to a file containing the bootstrap component in archive format")
	flags.StringVar(&m.Registry, "registry", env.DefaultBootstrapComponentLocation, "The registry to use to retrieve the bootstrap component. Defaults to ghcr.io/open-component-model/mpas-bootstrap-component")
	flags.StringVar(&m.Hostname, "hostname", "", "The hostname of the Git provider")
	flags.StringVar(&m.Path, "path", ".", "The target path to use in the management repository to store the bootstrap component")
	flags.StringVar(&m.Interval, "interval", "5m", "The interval to use to sync the bootstrap component")
	flags.StringVar(&m.CommitMessageAppendix, "commit-message-appendix", "", "The appendix to add to the commit message, e.g. [ci skip]")
	flags.BoolVar(&m.Private, "private", false, "Whether the management repository should be private")
	flags.StringVar(&m.CaFile, "ca-file", "", "Root certificate for the remote git server.")
}

// GithubConfig is the configuration for the GitHub bootstrap command.
type GithubConfig struct {
	BootstrapConfig
	Personal bool
}

// AddFlags adds the GitHub bootstrap flags to the given flag set.
func (g *GithubConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&g.Personal, "personal", false, "The personal access token to use to access the Github API")
	g.BootstrapConfig.AddFlags(flags)
}

// GiteaConfig is the configuration for the GitHub bootstrap command.
type GiteaConfig struct {
	BootstrapConfig
	Personal bool
}

// AddFlags adds the Gitea bootstrap flags to the given flag set.
func (g *GiteaConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&g.Personal, "personal", false, "The personal access token to use to access the Gitea API")
	g.BootstrapConfig.AddFlags(flags)
}

// GitlabConfig is the configuration for the Gitlab bootstrap command.
type GitlabConfig struct {
	BootstrapConfig
	Personal  bool
	TokenType string
}

// AddFlags adds the Gitea bootstrap flags to the given flag set.
func (g *GitlabConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&g.Personal, "personal", false, "The personal access token to use to access the Gitlab API")
	flags.StringVar(&g.TokenType, "token-type", "oauth2", "The token type of the Gitlab token. By default it's set to oauth2.")
	g.BootstrapConfig.AddFlags(flags)
}

// CreateConfig is the configuration shared by the create commands.
type CreateConfig struct {
	Prune    bool
	Interval string
}

// AddFlags adds the create flags to the given flag set.
func (c *CreateConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&c.Prune, "prune", false, "Whether to prune the resource under deletion")
	flags.StringVar(&c.Interval, "interval", "5m", "The interval to use to sync the resource")
}

// ProjectConfig is the configuration for the create project command.
type ProjectConfig struct {
	CreateConfig
	Provider            string
	Owner               string
	Branch              string
	Visibility          string
	Personal            bool
	Domain              string
	Maintainers         []string
	Email               string
	Message             string
	Author              string
	AlreadyExistsPolicy string
	SecretRef           string
}

// AddFlags adds the project flags to the given flag set.
func (p *ProjectConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&p.Provider, "provider", "", "The provider to use to create the project, e.g. github")
	flags.StringVar(&p.Owner, "owner", "", "The owner of the project")
	flags.StringVar(&p.Branch, "branch", "", "The branch to use for the project")
	flags.StringVar(&p.Visibility, "visibility", "", "The visibility of the project")
	flags.BoolVar(&p.Personal, "personal", false, "The personal access token to use to access the Gitea API")
	flags.StringVar(&p.Domain, "domain", "", "The domain to use for the project")
	flags.StringSliceVar(&p.Maintainers, "maintainers", []string{}, "The maintainers of the project")
	flags.StringVar(&p.Email, "email", "", "The email to use for templating commit messages")
	flags.StringVar(&p.Message, "message", "", "The message to use for templating commit messages")
	flags.StringVar(&p.Author, "author", "", "The author to use for templating commit messages")
	flags.StringVar(&p.AlreadyExistsPolicy, "already-exists-policy", "", "The policy to use when the project already exists")
	flags.StringVar(&p.SecretRef, "secret-ref", "", "The name of an existing secret to use for authentication to the provider")
	p.CreateConfig.AddFlags(flags)
}

// ComponentSubscriptionConfig is the configuration for the create component subscription command.
type ComponentSubscriptionConfig struct {
	CreateConfig
	Component            string
	Semver               string
	SourceUrl            string
	SourceSecretRef      string
	DestinationUrl       string
	DestinationSecretRef string
	ServiceAccount       string
	Verify               []string
}

// AddFlags adds the component subscription flags to the given flag set.
func (c *ComponentSubscriptionConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Component, "component", "", "The component to subscribe to")
	flags.StringVar(&c.Semver, "semver", "", "The semver constraint to use for the component")
	flags.StringVar(&c.SourceUrl, "source-url", "", "The source URL to use for the component")
	flags.StringVar(&c.SourceSecretRef, "source-secret-ref", "", "The name of an existing secret to use for authentication to the source")
	flags.StringVar(&c.DestinationUrl, "target-url", "", "The target URL to use for the component")
	flags.StringVar(&c.DestinationSecretRef, "target-secret-ref", "", "The name of an existing secret to use for authentication to the target")
	flags.StringVar(&c.ServiceAccount, "service-account", "", "The service account to use for the component")
	flags.StringSliceVar(&c.Verify, "verify", []string{}, "The public keys to use to verify the component, e.g. key1:pubkey1,key2:pubkey2")
	c.CreateConfig.AddFlags(flags)
}

// ProductDeploymentGeneratorConfig is the configuration for the create product generator command.
type ProductDeploymentGeneratorConfig struct {
	CreateConfig
	SubscriptionName      string
	SubscriptionNamespace string
	RepositoryName        string
	RepositoryNamespace   string
	ServiceAccount        string
}

// AddFlags adds the product generator flags to the given flag set.
func (p *ProductDeploymentGeneratorConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&p.SubscriptionName, "subscription-name", "", "The name of the component subscription")
	flags.StringVar(&p.SubscriptionNamespace, "subscription-namespace", "", "The namespace of the component subscription")
	flags.StringVar(&p.RepositoryName, "repository-name", "", "The name of the repository")
	flags.StringVar(&p.RepositoryNamespace, "repository-namespace", "", "The namespace of the repository")
	flags.StringVar(&p.ServiceAccount, "service-account", "", "The service account to use for the component")
	p.CreateConfig.AddFlags(flags)
}
