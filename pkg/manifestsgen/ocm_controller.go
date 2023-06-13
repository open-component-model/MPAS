// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifestsgen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

const (
	releaseAPIURL        = "https://api.github.com/repos/open-component-model/ocm-controller/releases"
	releaseURL           = "https://github.com/open-component-model/ocm-controller/releases"
	defaultRegistry      = "ghcr.io/open-component-model"
	defaultComponentName = "ocm-controller"
)

type OcmController struct {
	Version  string
	Registry string
	Path     string
	Content  *string
}

func (o *OcmController) GenerateManifests(ctx context.Context, tmpDir string) error {
	if err := o.validateVersion(ctx); err != nil {
		return fmt.Errorf("version %s does not exist for ocm-controller: %s", o.Version, err)
	}

	if err := o.fetch(ctx); err != nil {
		return fmt.Errorf("✗ failed to download install.yaml file: %w", err)
	}

	if tmpDir != "" {
		path, err := o.writeFile(tmpDir)
		if err != nil {
			return fmt.Errorf("failed to write manifests to temporary directory: %w", err)
		}
		o.Path = path
	}

	o.Registry = defaultRegistry
	return nil
}

func (o *OcmController) getLatestVersion(ctx context.Context) (string, error) {
	ghURL := fmt.Sprintf("%s/latest", releaseAPIURL)
	req, err := http.NewRequest(http.MethodGet, ghURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to download manifests.tar.gz from %s, error: %w", ghURL, err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	type meta struct {
		Tag string `json:"tag_name"`
	}

	var m meta
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", fmt.Errorf("failed to decode response body: %s", err)
	}

	return m.Tag, err
}

func (o *OcmController) validateVersion(ctx context.Context) error {
	ver := o.Version
	if ver == "" {
		return fmt.Errorf("version is empty")
	}

	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}

	if ver == "latest" {
		latest, err := o.getLatestVersion(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest version for ocm-controller: %s", err)
		}
		o.Version = latest
	}

	ghURL := fmt.Sprintf(releaseAPIURL+"/tags/%s", ver)
	req, err := http.NewRequest(http.MethodGet, ghURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, error: %w", ghURL, err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("target version %s does not exist for ocm-controller", ver)
	default:
		return fmt.Errorf("target version %s does not exist for ocm-controller, (%d)", ver, resp.StatusCode)
	}
}

func (o *OcmController) fetch(ctx context.Context) error {
	ghURL := fmt.Sprintf("%s/download/%s/install.yaml", releaseURL, o.Version)
	req, err := http.NewRequest(http.MethodGet, ghURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, error: %w", ghURL, err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, status: %s", ghURL, resp.Status)
	}

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	data := buf.String()
	o.Content = &data
	return nil
}

func (o *OcmController) writeFile(rootDir string) (string, error) {
	path := filepath.Join("ocm-controller", "install.yaml")
	output, err := securejoin.SecureJoin(rootDir, path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(output), os.ModePerm); err != nil {
		return "", fmt.Errorf("unable to create dir, error: %w", err)
	}

	if err := os.WriteFile(output, []byte(*o.Content), os.ModePerm); err != nil {
		return "", fmt.Errorf("unable to write file, error: %w", err)
	}
	return path, nil
}

func (o *OcmController) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	// add localization
	tmpl += fmt.Sprintf(loc, defaultComponentName, defaultComponentName)
	return tmpl, nil
}

func (o *OcmController) GenerateImages() (map[string][]string, error) {
	var images = make(map[string][]string)
	index := strings.Index(*o.Content, fmt.Sprintf("%s/%s", o.Registry, defaultComponentName))
	var image string
	for i := index; i < len(*o.Content); i++ {
		v := string((*o.Content)[i])
		if v == "\n" {
			break
		}
		image += v
	}
	ver := ""
	if im := strings.Split(image, ":"); len(im) != 2 {
		ver = o.Version
		image += ":" + ver
	}
	images[image] = []string{
		defaultComponentName,
		ver,
	}

	return images, nil
}
