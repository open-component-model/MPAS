// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// TODO: make this an option.
	certManagerRepoURL = "https://github.com/cert-manager/cert-manager/releases"
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
	if err := c.validateCertManager(c.Version); err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	content, err := fetch(ctx, certManagerRepoURL, c.Version, tmpDir)
	if err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	c.Registry = "quay.io/jetstack"
	c.Components = []string{"cert-manager-controller", "cert-manager-webhook", "cert-manager-cainjector"}
	c.Path = "cert-manager.yaml"
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

func (c *CertManager) validateCertManager(version string) error {
	ver := version
	if ver == "" {
		return fmt.Errorf("version is empty")
	}
	if ok, err := existingVersion(ver); err != nil || !ok {
		if err == nil {
			return fmt.Errorf("targeted version '%s' does not exist", ver)
		}
	}
	return nil
}

// existingVersion calls the GitHub API to confirm the given version does exist.
func existingVersion(version string) (bool, error) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	ghURL := fmt.Sprintf("https://api.github.com/repos/cert-manager/cert-manager/releases/tags/%s", version)
	c := http.DefaultClient
	c.Timeout = 15 * time.Second

	res, err := c.Get(ghURL)
	if err != nil {
		return false, fmt.Errorf("GitHub API call failed: %w", err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	switch res.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("GitHub API returned an unexpected status code (%d)", res.StatusCode)
	}
}

func fetch(ctx context.Context, url, version, dir string) ([]byte, error) {
	ghURL := fmt.Sprintf("%s/latest/download/cert-manager.yaml", url)
	if strings.HasPrefix(version, "v") {
		ghURL = fmt.Sprintf("%s/download/%s/cert-manager.yaml", url, version)
	}

	req, err := http.NewRequest("GET", ghURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to download cert-manager.yaml from %s, error: %w", ghURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download mcert-manager.yaml from %s, status: %s", ghURL, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "cert-manager.yaml"), content, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write out file: %w", err)
	}

	return content, nil
}
