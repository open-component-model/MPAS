// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/open-component-model/mpas/pkg/bootstrap"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewBoostrap(c MpasConfig) *cobra.Command {
	b := &bootstrapCmd{cfg: c}
	cmd := &cobra.Command{
		Use:     "bootstrapCmd [flags]",
		Short:   "Bootstrap an mpas management repository",
		Long:    `Bootstrap an mpas management repository.`,
		Example: `  # bootstrapCmd an mpas management repository`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    b.run,
	}

	b.addBootstrapFlags(cmd, cmd.Flags())

	return cmd
}

type bootstrapCmd struct {
	owner      string
	repository string
	publicKey  string
	fromFile   string
	registry   string
	cfg        MpasConfig
}

func (b *bootstrapCmd) addBootstrapFlags(cmd *cobra.Command, flags *pflag.FlagSet) {
	flags.StringVar(&b.owner, "owner", "", "The owner of the management repository")
	flags.StringVar(&b.repository, "repository", "", "The name of the management repository")
	flags.StringVar(&b.publicKey, "public-key", "", "The public key to use for the management repository")
	flags.StringVar(&b.fromFile, "from-file", "", "The path to a file containing the public key to use for the management repository")
	flags.StringVar(&b.registry, "registry", "", "The registry to use for the management repository")
}

func (b *bootstrapCmd) run(cmd *cobra.Command, args []string) error {
	// create kube client
	k := b.cfg.Kubeconfig
	if k == "" {
		if k = os.Getenv("KUBECONFIG"); k == "" {
			return fmt.Errorf("no kubeconfig provided")
		}
	}

	_ = bootstrap.New()
	return nil
}
