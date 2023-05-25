// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-component-model/mpas/cmd/mpas/bootstrap"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	defaultghTokenVar = "GITHUB_TOKEN"
)

// New returns a new cobra.Command for github bootstrap
func NewBoostrapGithub() *cobra.Command {
	c := &config.GithubConfig{}
	cmd := &cobra.Command{
		Use:     "bootstrap-github [flags]",
		Short:   "Bootstrap an mpas management repository on Github",
		Example: `  # bootstrapCmd an mpas management repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			b := bootstrap.BootstrapGithubCmd{
				Owner:      c.Owner,
				Personal:   c.Personal,
				Repository: c.Repository,
				FromFile:   c.FromFile,
				Registry:   c.Registry,
				Hostname:   c.Hostname,
				Components: append(defaultComponents, c.Components...),
			}

			token := os.Getenv(defaultghTokenVar)
			if token != "" {
				var err error
				token, err = passwdFromStdin("Github token: ")
				if err != nil {
					return fmt.Errorf("failed to read token from stdin: %w", err)
				}
			}
			b.Token = token

			if b.Personal && b.Owner == "" {
				return fmt.Errorf("owner must be set when using personal")
			}

			return b.Execute()

		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
}

// passwdFromStdin reads a password from stdin.
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
