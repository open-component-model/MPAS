// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/bootstrap"
	"github.com/open-component-model/mpas/pkg/bootstrap/provider"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	defaultTokenVar       = "GITHUB_TOKEN"
	githubDefaultHostname = "github.com"
)

// BootstrapGithubCmd is a command for bootstrapping a GitHub repository
type BootstrapGithubCmd struct {
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

func NewBootstrapGithubCmd(cfg *config.BootstrapConfig) *cobra.Command {
	c := &config.GithubConfig{}
	cmd := &cobra.Command{
		Use:     "bootstrap-github [flags]",
		Short:   "Bootstrap an mpas management repository",
		Long:    `Bootstrap an mpas management repository.`,
		Example: `  # bootstrapCmd an mpas management repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			b := BootstrapGithubCmd{
				Owner:      cfg.Owner,
				Personal:   c.Personal,
				Repository: cfg.Repository,
				FromFile:   cfg.FromFile,
				Registry:   cfg.Registry,
				Components: append(config.DefaultComponents, cfg.Components...),
			}

			token := os.Getenv(defaultTokenVar)
			if token != "" {
				var err error
				token, err = passwdFromStdin("Github token: ")
				if err != nil {
					return fmt.Errorf("failed to read token from stdin: %w", err)
				}
			}
			b.Token = token

			return b.Execute()

		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
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
		Provider:           provider.ProviderGithub,
		Hostname:           hostname,
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
func (b *BootstrapGithubCmd) Cleanup(ctx context.Context) error {
	if b.bootstrapper != nil {
		return b.bootstrapper.DeleteManagementRepository(ctx)
	}
	return nil
}

func passwdFromStdin(prompt string) (string, error) {
	// Get the initial state of the terminal.
	initialTermState, err := terminal.GetState(syscall.Stdin)
	if err != nil {
		return "", err
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		fmt.Println("\n^C received, exiting")
		// Restore the terminal to its initial state.
		terminal.Restore(syscall.Stdin, initialTermState)
	}()

	fmt.Print(prompt)
	passwd, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Println()

	signal.Stop(signalChan)

	return string(passwd), nil
}
