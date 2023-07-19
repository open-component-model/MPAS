// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	Version = "0.0.0-dev.0"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cmd, err := New(ctx, os.Args[1:])
	if err != nil {
		return err
	}
	if err := cmd.ExecuteContext(ctx); err != nil {
		return err
	}
	return nil
}
