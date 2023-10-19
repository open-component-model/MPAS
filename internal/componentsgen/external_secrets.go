// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/open-component-model/mpas/internal/env"
)

const (
	externalSecretsRepoURL       = "https://github.com/external-secrets/external-secrets/releases"
	externalSecretsReleaseAPIURL = "https://api.github.com/repos/external-secrets/external-secrets/releases"
)

// ExternalSecrets generates ExternalSecrets manifests based on the given version.
type ExternalSecrets struct {
	// Version is the version of ExternalSecrets.
	Version string
	// Registry is the registry to get the controller images from.
	Registry string
	// Components are the components of ExternalSecrets.
	Components []string
	// Path is the path to the manifests.
	Path string
	// Content is the content of the external-secrets.yaml file.
	Content string
}

// GenerateManifests generates ExternalSecrets manifests for the given version.
// If the version is invalid, an error is returned.
func (c *ExternalSecrets) GenerateManifests(ctx context.Context, tmpDir string) error {
	if c.Version == "latest" {
		latest, err := getLatestVersion(ctx, externalSecretsReleaseAPIURL)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest version for %s: %s", "external-secrets", err)
		}

		c.Version = latest
	}

	if err := validateVersion(ctx, c.Version, externalSecretsReleaseAPIURL, "external-secrets"); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	tmpDir = filepath.Join(tmpDir, "external-secrets")
	content, err := fetch(ctx, externalSecretsRepoURL, c.Version, tmpDir, "external-secrets.yaml")
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	c.Registry = env.DefaultExternalSecretsHost
	c.Components = []string{"external-secrets"}
	c.Path = filepath.Join("external-secrets", "external-secrets.yaml")
	c.Content = string(content)

	return nil
}

// GenerateLocalizationFromTemplate generates localization files from a template.
func (c *ExternalSecrets) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	for _, c := range c.Components {
		// add localization
		tmpl += fmt.Sprintf(loc, c, c)
	}

	return tmpl, nil
}

// GenerateImages returns a map of images from the components.
func (c *ExternalSecrets) GenerateImages() (map[string][]string, error) {
	var images = make(map[string][]string)
	for _, component := range c.Components {
		index := strings.Index(c.Content, fmt.Sprintf("%s/%s", c.Registry, component))
		var image string
		for i := index; i < len(c.Content); i++ {
			v := string((c.Content)[i])
			if v == "\n" {
				break
			}
			image += v
		}
		// cert-manager manifest wraps strings into ""
		image = strings.Trim(image, "\"")
		version := strings.Trim(strings.Split(image, ":")[1], "\"")
		images[image] = []string{
			component,
			version,
		}
	}

	return images, nil
}

func (c *ExternalSecrets) GetPath() string {
	return c.Path
}
