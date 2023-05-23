// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"
)

func New(c MpasConfig, args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "mpas",
		Long: `mpas is a CLI tool for managing Open Component Model (OCM) projects.`,
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&c.Kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization and master location information.")
	flags.Parse(args)

	cmd.AddCommand(NewBoostrap(MpasConfig{}))

	cmd.InitDefaultHelpCmd()
	return cmd
}

// MpasConfig is the configuration for the mpas CLI.
type MpasConfig struct {
	Kubeconfig string
}
