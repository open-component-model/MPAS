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
	"path/filepath"
	"strings"
)

const (
	defaultRegistry = "ghcr.io/open-component-model"
)

type Controller struct {
	Name          string
	Version       string
	Registry      string
	Path          string
	ReleaseURL    string
	ReleaseAPIURL string
	Content       *string
}

func (o *Controller) GenerateManifests(ctx context.Context, tmpDir string) error {
	if err := o.validateVersion(ctx); err != nil {
		return fmt.Errorf("version %s does not exist for %s: %s", o.Name, o.Version, err)
	}

	if err := o.fetch(ctx); err != nil {
		return fmt.Errorf("failed to download install.yaml file: %w", err)
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

func (o *Controller) getLatestVersion(ctx context.Context) (string, error) {
	ghURL := fmt.Sprintf("%s/latest", o.ReleaseAPIURL)
	resp, err := getFrom(ctx, ghURL)
	if err != nil {
		return "", err
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

func (o *Controller) validateVersion(ctx context.Context) error {
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
			return fmt.Errorf("failed to retrieve latest version for %s: %s", o.Name, err)
		}
		o.Version = latest
	}

	ghURL := fmt.Sprintf(o.ReleaseAPIURL+"/tags/%s", ver)
	resp, err := getFrom(ctx, ghURL)
	if err != nil {
		return err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("target version %s does not exist for %s", ver, o.Name)
	default:
		return fmt.Errorf("target version %s does not exist for %s, (%d)", ver, o.Name, resp.StatusCode)
	}
}

func (o *Controller) fetch(ctx context.Context) error {
	ghURL := fmt.Sprintf("%s/download/%s/install.yaml", o.ReleaseURL, o.Version)
	resp, err := getFrom(ctx, ghURL)
	if err != nil {
		return err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, status: %s", ghURL, resp.Status)
	}

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return fmt.Errorf("failed to write to buffer: %s", err)
	}

	data := buf.String()
	o.Content = &data
	return nil
}

func (o *Controller) writeFile(rootDir string) (string, error) {
	path := filepath.Join(o.Name, "install.yaml")
	err := writeFile(rootDir, path, *o.Content)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (o *Controller) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	// add localization
	tmpl += fmt.Sprintf(loc, o.Name, o.Name)
	return tmpl, nil
}

func (o *Controller) GenerateImages() (map[string][]string, error) {
	var images = make(map[string][]string)
	index := strings.Index(*o.Content, fmt.Sprintf("%s/%s", o.Registry, o.Name))
	var image string
	for i := index; i < len(*o.Content); i++ {
		v := string((*o.Content)[i])
		if v == "\n" {
			break
		}
		image += v
	}

	if im := strings.Split(image, ":"); len(im) != 2 {
		image += ":" + o.Version
	} else {
		image = im[0] + ":" + o.Version
	}
	images[image] = []string{
		o.Name,
		o.Version,
	}

	return images, nil
}

func (o *Controller) GetPath() string {
	return o.Path
}
