// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"github.com/open-component-model/mpas/cmd/mpas/bootstrap/provider"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/spf13/cobra"
)

var cfg config.BootstrapConfig

func NewBootstrapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bootstrap [provider] [flags]",
		Short:   "Bootstrap the MPAS system into a Kubernetes cluster.",
		Long:    "Bootstrap the MPAS system into a Kubernetes cluster.",
		Example: "mpas bootstrap github [flags]",
	}

	cfg.AddFlags(cmd.PersistentFlags())
	cmd.AddCommand(provider.NewBootstrapGithubCmd(&cfg))

	return cmd
}
