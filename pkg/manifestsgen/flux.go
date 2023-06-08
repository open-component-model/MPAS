// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifestsgen

import (
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

// FluxVersion is a version of Flux.
// It is used to generate Flux manifests.
type FluxVersion string

// GenerateFluxManifests generates Flux manifests for the given version.
// It returns the generated manifests as a Manifest object.
// If the version is invalid, an error is returned.
func (v FluxVersion) GenerateFluxManifests(tmpDir string) (string, error) {
	if err := v.validateVersion(); err != nil {
		return "", fmt.Errorf("invalid version: %w", err)
	}

	o := install.MakeDefaultOptions()
	o.Version = string(v)

	manifest, err := install.Generate(o, "")
	if err != nil {
		return "", fmt.Errorf("install failed: %w", err)
	}

	if tmpDir != "" {
		if _, err := manifest.WriteFile(tmpDir); err != nil {
			return "", fmt.Errorf("failed to write manifests to temporary directory: %w", err)
		}
	}

	return manifest.Path, nil
}

func (v FluxVersion) validateVersion() error {
	ver := string(v)
	if ver == "" {
		return fmt.Errorf("version is empty")
	}

	if ver != install.MakeDefaultOptions().Version && !strings.HasPrefix(ver, "v") {
		return fmt.Errorf("targeted version '%s' must be prefixed with 'v'", ver)
	}

	if ok, err := install.ExistingVersion(ver); err != nil || !ok {
		if err == nil {
			return fmt.Errorf("targeted version '%s' does not exist", ver)
		}
	}
	return nil
}
