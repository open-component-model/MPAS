// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifestsgen

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

// Flux is a version of Flux.
// It is used to generate Flux manifests.
type Flux struct {
	Version    string
	Registry   string
	Components []string
	Path       string
	Content    *string
}

// GenerateFluxManifests generates Flux manifests for the given version.
// It returns the generated manifests as a Manifest object.
// If the version is invalid, an error is returned.
func (f *Flux) GenerateManifests(ctx context.Context, tmpDir string) error {
	if err := validateFluxVersion(f.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	o := install.MakeDefaultOptions()
	o.Version = f.Version

	manifest, err := install.Generate(o, "")
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	if tmpDir != "" {
		if _, err := manifest.WriteFile(tmpDir); err != nil {
			return fmt.Errorf("failed to write manifests to temporary directory: %w", err)
		}
	}

	f.Registry = o.Registry
	f.Components = o.Components
	f.Path = manifest.Path
	f.Content = &manifest.Content

	return nil
}

func (f *Flux) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	for _, c := range f.Components {
		// add localization
		tmpl += fmt.Sprintf(loc, c, c)
	}

	return tmpl, nil
}

func (f *Flux) GenerateImages() (map[string][]string, error) {
	var images = make(map[string][]string)
	for _, c := range f.Components {
		index := strings.Index(*f.Content, fmt.Sprintf("%s/%s", f.Registry, c))
		var image string
		for i := index; i < len(*f.Content); i++ {
			v := string((*f.Content)[i])
			if v == "\n" {
				break
			}
			image += v
		}
		images[image] = []string{
			c,
			strings.Split(string(image), ":")[1],
		}
	}

	return images, nil
}
