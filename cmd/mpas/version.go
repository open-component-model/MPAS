// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/spf13/cobra"
)

// NewVersion returns a new cobra.Command to provide version information.
func NewVersion(cfg *config.MpasConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "returns the version of the current binary",
		Long:  "returns the version of the current binary",
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Printer.Println(Version)
		},
	}

	return cmd
}
