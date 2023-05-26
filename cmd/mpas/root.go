// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/open-component-model/mpas/cmd/mpas/bootstrap"
	"github.com/spf13/cobra"
)

func New(args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "mpas",
		Long: `mpas is a CLI tool for managing MPAS projects and bootstrapping the MPAS system into a cluster.`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
	cmd.Print()

	cfg.AddFlags(cmd.PersistentFlags())
	cmd.AddCommand(bootstrap.NewBootstrapCmd())

	cmd.InitDefaultHelpCmd()
	return cmd
}
