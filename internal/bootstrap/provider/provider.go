// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	"github.com/fluxcd/go-git-providers/gitea"
	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
)

// rewrite of https://github.com/fluxcd/flux2/tree/main/pkg/bootstrap/provider

var (
	// providers is a map of provider names to factory functions.
	// It is populated by calls to register.
	providers providerMap
)

func init() {
	// Register the default providers
	providers = make(providerMap)
	providers.register(env.ProviderGithub, githubProviderFunc)
	providers.register(env.ProviderGitea, giteaProviderFunc)
	providers.register(env.ProviderGitlab, gitlabProviderFunc)
}

// ProviderOptions contains the options for the provider
type ProviderOptions struct {
	Provider           string
	Hostname           string
	Token              string
	Username           string
	DestructiveActions bool
}

// GitProvider is a provider for git repositories
type GitProvider struct{}

// New returns a new GitProvider
func New() *GitProvider {
	return &GitProvider{}
}

// Build returns a new gitprovider.Client
func (g *GitProvider) Build(opts ProviderOptions) (gitprovider.Client, error) {
	if factory, ok := providers[opts.Provider]; ok {
		return factory(opts)
	}
	return nil, fmt.Errorf("provider %s not supported", opts.Provider)
}

// providerMap is a map of provider names to factory functions
type providerMap map[string]factoryFunc

// factoryFunc is a factory function that creates a new gitprovider.Client
type factoryFunc func(opts ProviderOptions) (gitprovider.Client, error)

// register registers a new provider
func (m providerMap) register(name string, provider factoryFunc) {
	m[name] = provider
}

// githubProviderFunc returns a new gitprovider.Client for github
func githubProviderFunc(opts ProviderOptions) (gitprovider.Client, error) {
	o := makeProviderOpts(opts)
	client, err := github.NewClient(o...)
	if err != nil {
		return nil, err
	}
	return client, err
}

func giteaProviderFunc(opts ProviderOptions) (gitprovider.Client, error) {
	o := makeProviderOpts(opts)
	client, err := gitea.NewClient(opts.Token, o...)
	if err != nil {
		return nil, err
	}
	return client, err
}

func gitlabProviderFunc(opts ProviderOptions) (gitprovider.Client, error) {
	o := makeProviderOpts(opts)
	// TODO: Put that into an option somewhere.
	client, err := gitlab.NewClient(opts.Token, "oauth2", o...)
	if err != nil {
		return nil, err
	}
	return client, err
}

func makeProviderOpts(opts ProviderOptions) []gitprovider.ClientOption {
	o := []gitprovider.ClientOption{
		gitprovider.WithOAuth2Token(opts.Token),
		gitprovider.WithDestructiveAPICalls(opts.DestructiveActions),
	}
	if opts.Hostname != "" {
		o = append(o, gitprovider.WithDomain(opts.Hostname))
	}
	return o
}
