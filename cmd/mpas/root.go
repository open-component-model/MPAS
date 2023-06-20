// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/printer"
	"github.com/spf13/cobra"
)

const (
	defaultKubeconfig = "~/.kube/config"
)

var defaultOutput = os.Stdout

// New returns a new cobra.Command for mpas
func New(args []string) *cobra.Command {
	cfg := &config.MpasConfig{
		Printer: printer.Newprinter("", defaultOutput),
	}
	cmd := &cobra.Command{
		Use:  "mpas",
		Long: `mpas is a CLI tool for managing (MPAS) multi platform automation system.`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
	cmd.Print()

	cfg.AddFlags(cmd.PersistentFlags())

	if cfg.Kubeconfig != "" {
		cfg.Kubeconfig = defaultKubeconfig
	}

	cmd.AddCommand(NewBootstrap(cfg))

	cmd.InitDefaultHelpCmd()
	return cmd
}
