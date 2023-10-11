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
	certManagerRepoURL       = "https://github.com/cert-manager/cert-manager/releases"
	certManagerReleaseAPIURL = "https://api.github.com/repos/cert-manager/cert-manager/releases"
)

// CertManager generates CertManager manifests based on the given version.
type CertManager struct {
	// Version is the version of CertManager.
	Version string
	// Registry is the registry to get the controller images from.
	Registry string
	// Components are the components of CertManager.
	Components []string
	// Path is the path to the manifests.
	Path string
	// Content is the content of the install.yaml file.
	Content string
}

// GenerateManifests generates CertManager manifests for the given version.
// If the version is invalid, an error is returned.
func (c *CertManager) GenerateManifests(ctx context.Context, tmpDir string) error {
	if c.Version == "latest" {
		latest, err := getLatestVersion(ctx, certManagerReleaseAPIURL)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest version for %s: %s", "cert-manager", err)
		}

		c.Version = latest
	}

	if err := validateVersion(ctx, c.Version, certManagerReleaseAPIURL, "cert-manager"); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	tmpDir = filepath.Join(tmpDir, "cert-manager")
	content, err := fetch(ctx, certManagerRepoURL, c.Version, tmpDir, "cert-manager.yaml")
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	c.Registry = env.DefaultCertManagerHost
	c.Components = []string{"cert-manager-controller", "cert-manager-webhook", "cert-manager-cainjector"}
	c.Path = filepath.Join("cert-manager", "cert-manager.yaml")
	c.Content = string(content)

	return nil
}

// GenerateLocalizationFromTemplate generates localization files from a template.
func (c *CertManager) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	for _, c := range c.Components {
		// add localization
		tmpl += fmt.Sprintf(loc, c, c)
	}

	return tmpl, nil
}

// GenerateImages returns a map of images from the components.
func (c *CertManager) GenerateImages() (map[string][]string, error) {
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

func (c *CertManager) GetPath() string {
	return c.Path
}
