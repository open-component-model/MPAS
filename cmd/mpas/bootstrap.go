// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/bootstrap"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/env"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewBootstrap returns a new cobra.Command for bootstrap
func NewBootstrap(cfg *config.MpasConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bootstrap [provider] [flags]",
		Short:   "Bootstrap the MPAS system into a Kubernetes cluster.",
		Long:    "Bootstrap the MPAS system into a Kubernetes cluster.",
		Example: "mpas bootstrap [flags]",
	}

	cmd.AddCommand(NewBootstrapGithub(cfg))
	cmd.AddCommand(NewBootstrapGitea(cfg))

	return cmd
}

// NewBootstrapGithub returns a new cobra.Command for github bootstrap
func NewBootstrapGithub(cfg *config.MpasConfig) *cobra.Command {
	c := &config.GithubConfig{}
	cmd := &cobra.Command{
		Use:   "github [flags]",
		Short: "Bootstrap an mpas management repository on Github",
		Example: `  - Bootstrap with a private organization repository
    mpas bootstrap github --owner ocmOrg --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --path clusters/my-cluster

    - Bootstrap with a private user repository
    mpas bootstrap github --owner myUser --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --personal --path clusters/my-cluster

    - Bootstrap with a public user repository
    mpas bootstrap github --owner myUser --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --personal --private=false --path clusters/my-cluster

    - Bootstrap with a public organization repository
    mpas bootstrap github --owner ocmOrg --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --private=false --path clusters/my-cluster
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			b := bootstrap.GithubCmd{
				Owner:                 c.Owner,
				Personal:              c.Personal,
				Repository:            c.Repository,
				FromFile:              c.FromFile,
				Registry:              c.Registry,
				DockerconfigPath:      cfg.DockerconfigPath,
				Path:                  c.Path,
				CommitMessageAppendix: c.CommitMessageAppendix,
				Hostname:              c.Hostname,
				Components:            append(env.Components, c.Components...),
			}

			if len(c.Components) != 0 {
				return fmt.Errorf("additional  components are not yet supported for github")
			}

			token := os.Getenv(env.GithubTokenVar)
			if token == "" {
				token, err = passwdFromStdin("Github token: ")
				if err != nil {
					return fmt.Errorf("failed to read token from stdin: %w", err)
				}
			}
			b.Token = token

			if b.Owner == "" {
				return fmt.Errorf("owner must be set")
			}

			if b.Repository == "" {
				return fmt.Errorf("repository must be set")
			}

			if b.Registry == "" && b.FromFile == "" {
				return fmt.Errorf("either registry or from-file must be set")
			}

			b.Timeout, err = time.ParseDuration(cfg.Timeout)
			if err != nil {
				return err
			}

			b.Interval, err = time.ParseDuration(c.Interval)
			if err != nil {
				return err
			}

			return b.Execute(cmd.Context(), cfg)

		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
}

// NewBootstrapGitea returns a new cobra.Command for gitea bootstrap
func NewBootstrapGitea(cfg *config.MpasConfig) *cobra.Command {
	c := &config.GiteaConfig{}
	cmd := &cobra.Command{
		Use:   "gitea [flags]",
		Short: "Bootstrap an mpas management repository on Gitea",
		Example: `  - Bootstrap with a private organization repository
    mpas bootstrap gitea --owner ocmOrg --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --path clusters/my-cluster --hostname gitea.example.com

    - Bootstrap with a private user repository
    mpas bootstrap gitea --owner myUser --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --personal --path clusters/my-cluster --hostname gitea.example.com

    - Bootstrap with a public user repository
    mpas bootstrap gitea --owner myUser --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --personal --private=false --path clusters/my-cluster --hostname gitea.example.com

    - Bootstrap with a public organization repository
    mpas bootstrap gitea --owner ocmOrg --repository mpas --registry ghcr.io/open-component-model/mpas-bootstrap-component --private=false --path clusters/my-cluster --hostname gitea.example.com
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			b := bootstrap.GiteaCmd{
				Owner:                 c.Owner,
				Personal:              c.Personal,
				Repository:            c.Repository,
				FromFile:              c.FromFile,
				Registry:              c.Registry,
				DockerconfigPath:      cfg.DockerconfigPath,
				Path:                  c.Path,
				CommitMessageAppendix: c.CommitMessageAppendix,
				Hostname:              c.Hostname,
				Components:            append(env.Components, c.Components...),
			}

			if len(c.Components) != 0 {
				return fmt.Errorf("additional  components are not yet supported for gitea")
			}

			token := os.Getenv(env.GiteaTokenVar)
			if token == "" {
				token, err = passwdFromStdin("Gitea token: ")
				if err != nil {
					return fmt.Errorf("failed to read token from stdin: %w", err)
				}
			}
			b.Token = token

			if b.Owner == "" {
				return fmt.Errorf("owner must be set")
			}

			if b.Hostname == "" {
				return fmt.Errorf("hostname must be set")
			}

			if b.Repository == "" {
				return fmt.Errorf("repository must be set")
			}

			if b.Registry == "" && b.FromFile == "" {
				return fmt.Errorf("either registry or from-file must be set")
			}

			b.Timeout, err = time.ParseDuration(cfg.Timeout)
			if err != nil {
				return err
			}

			b.Interval, err = time.ParseDuration(c.Interval)
			if err != nil {
				return err
			}

			return b.Execute(cmd.Context(), cfg)

		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
}

// passwdFromStdin reads a password from stdin.
func passwdFromStdin(prompt string) (string, error) {
	// Get the initial state of the terminal.
	initialTermState, err := term.GetState(syscall.Stdin)
	if err != nil {
		return "", err
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		fmt.Println("\n^C received, exiting")
		// Restore the terminal to its initial state.
		err := term.Restore(syscall.Stdin, initialTermState)
		if err != nil {
			fmt.Printf("failed to restore terminal state: %v\n", err)
		}
		os.Exit(1)
	}()

	fmt.Print(prompt)
	passwd, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Println()

	signal.Stop(signalChan)

	return string(passwd), nil
}
