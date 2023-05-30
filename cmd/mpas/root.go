// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"
)

// New returns a new cobra.Command for mpas
func New(args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "mpas",
		Long: `mpas is a CLI tool for managing (MPAS) multi platform automation system.`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
	cmd.Print()

	cfg.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewBootstrap())

	cmd.InitDefaultHelpCmd()
	return cmd
}
