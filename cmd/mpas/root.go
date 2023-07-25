// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/mpas/internal/printer"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/utils/pointer"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	defaultOutput = os.Stdout
)

// New returns a new cobra.Command for mpas
func New(ctx context.Context, args []string) (*cobra.Command, error) {
	p, err := printer.Newprinter(defaultOutput)
	if err != nil {
		return nil, err
	}
	cfg := &config.MpasConfig{
		Printer: p,
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

	cfg.KubeConfigArgs = genericclioptions.NewConfigFlags(false)
	cfg.AddFlags(cmd.PersistentFlags())
	err = setDefaultNamespace(cfg.KubeConfigArgs)
	if err != nil {
		return nil, err
	}
	cfg.KubeConfigArgs.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewBootstrap(cfg))

	cmd.InitDefaultHelpCmd()

	// This is required because controller-runtime expects its consumers to
	// set a logger through log.SetLogger within 30 seconds of the program's
	// initalization. If not set, the entire debug stack is printed as an
	// error, see: https://github.com/kubernetes-sigs/controller-runtime/blob/ed8be90/pkg/log/log.go#L59
	// Since we have our own logging and don't care about controller-runtime's
	// logger, we configure it's logger to do nothing.
	ctrllog.SetLogger(logr.New(ctrllog.NullLogSink{}))
	return cmd, nil
}

func setDefaultNamespace(kubeConfigArgs *genericclioptions.ConfigFlags) error {
	*kubeConfigArgs.Namespace = env.DefaultsNamespace
	kubeConfigArgs.Namespace = pointer.String(env.DefaultsNamespace)
	fromEnv := os.Getenv("MPAS_SYSTEM_NAMESPACE")
	if fromEnv != "" {
		if e := validation.IsDNS1123Label(fromEnv); len(e) > 0 {
			return fmt.Errorf("invalid namespace %s: %v", fromEnv, e)
		}

		kubeConfigArgs.Namespace = pointer.String(fromEnv)
	}
	return nil
}
