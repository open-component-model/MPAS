//go:build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	defaultTimeoutSeconds = 600
)

// RunLocalTilt takes the local Tiltfile in the MPAS repository and executes it instead of
// building one up dynamically.
func RunLocalTilt() env.Func {
	return func(ctx context.Context, config *envconf.Config) (_ context.Context, err error) {
		tctx, cancel := context.WithTimeout(ctx, defaultTimeoutSeconds*time.Second)

		defer cancel()

		dir, err := os.Getwd()
		if err != nil {
			return ctx, fmt.Errorf("failed to get working folder: %w", err)
		}

		path, err := locateLocalTiltfile(dir)
		if err != nil {
			return ctx, fmt.Errorf("failed to locate local Tiltfile: %w", err)
		}

		cmd := exec.CommandContext(tctx, "tilt", "ci", "-f", path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("output from tilt: ", string(output))

			return ctx, err
		}

		return ctx, nil
	}
}

// starts from dir and tries finding the controller by stepping outside
// until root is reached.
func locateLocalTiltfile(dir string) (string, error) {
	separatorIndex := strings.LastIndex(dir, "/")
	for separatorIndex > 0 {
		if _, err := os.Stat(filepath.Join(dir, "Tiltfile")); err == nil {
			return filepath.Join(dir, "Tiltfile"), nil
		}

		separatorIndex = strings.LastIndex(dir, string(os.PathSeparator))
		dir = dir[0:separatorIndex]
	}

	return "", fmt.Errorf("failed to find controller %s", "Tiltfile")
}
