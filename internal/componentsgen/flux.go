// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
)

// Flux generates Flux manifests based on the given version.
type Flux struct {
	// Version is the version of Flux.
	Version string
	// Registry is the registry to get the controller images from.
	Registry string
	// Components are the components of Flux.
	Components []string
	// Path is the path to the manifests.
	Path string
	// Content is the content of the install.yaml file.
	Content *string
}

// GenerateFluxManifests generates Flux manifests for the given version.
// If the version is invalid, an error is returned.
func (f *Flux) GenerateManifests(ctx context.Context, tmpDir string) error {
	if err := f.validateFluxVersion(f.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	o := install.MakeDefaultOptions()
	o.Version = f.Version
	o.Components = append(o.Components, o.ComponentsExtra...)

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

// GenerateLocalizationFromTemplate generates localization files from a template.
func (f *Flux) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	for _, c := range f.Components {
		// add localization
		tmpl += fmt.Sprintf(loc, c, c)
	}

	return tmpl, nil
}

// GenerateImages returns a map of images from the components.
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

func (f *Flux) GetPath() string {
	return f.Path
}

func (f *Flux) validateFluxVersion(version string) error {
	ver := version
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
