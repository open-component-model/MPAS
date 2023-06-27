// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/printer"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	defaultsNamespace = "mpas-system"
)

var (
	defaultOutput = os.Stdout
)

// New returns a new cobra.Command for mpas
func New(ctx context.Context, args []string) *cobra.Command {
	cfg := &config.MpasConfig{
		Printer: printer.Newprinter("", defaultOutput),
	}
	cmd := &cobra.Command{
		Use:           "mpas",
		Long:          `mpas is a CLI tool for managing (MPAS) multi platform automation system.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
	cmd.Print()

	cfg.SetContext(ctx)
	cfg.KubeConfigArgs = genericclioptions.NewConfigFlags(false)
	cfg.AddFlags(cmd.PersistentFlags())
	setDefaultNamespace(cfg.KubeConfigArgs)
	cfg.KubeConfigArgs.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewBootstrap(cfg))

	cmd.InitDefaultHelpCmd()
	return cmd
}

func setDefaultNamespace(kubeConfigArgs *genericclioptions.ConfigFlags) error {
	*kubeConfigArgs.Namespace = defaultsNamespace
	fromEnv := os.Getenv("MPAS_SYSTEM_NAMESPACE")
	if fromEnv != "" {
		if e := validation.IsDNS1123Label(fromEnv); len(e) > 0 {
			return fmt.Errorf(" ignoring invalid MPAS_SYSTEM_NAMESPACE: %q", fromEnv)
		}

		*kubeConfigArgs.Namespace = fromEnv
	}
	return nil
}
