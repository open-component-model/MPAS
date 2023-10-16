// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import "context"

// Generator is a component that generates manifests.
// It can also generate localization files from a template, and images.
type Generator interface {
	GenerateManifests(ctx context.Context, tmpDir string) error
	GenerateLocalizationFromTemplate(tmpl, loc string) (string, error)
	GenerateImages() (map[string][]string, error)
	GetPath() string
}

var (
	_ Generator = &Flux{}
	_ Generator = &Controller{}
	_ Generator = &CertManager{}
)
