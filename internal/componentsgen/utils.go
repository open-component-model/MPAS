// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

func fetch(ctx context.Context, url, version, dir, filename string) ([]byte, error) {
	ghURL := fmt.Sprintf("%s/latest/download/%s", url, filename)
	if strings.HasPrefix(version, "v") {
		ghURL = fmt.Sprintf("%s/download/%s/%s", url, version, filename)
	}

	resp, err := getFrom(ctx, ghURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from url: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s from %s, status: %s", filename, ghURL, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	if err := writeFile(dir, filename, string(content)); err != nil {
		return nil, fmt.Errorf("failed to write out file: %w", err)
	}

	return content, nil
}

func getFrom(ctx context.Context, ghURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, ghURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for %s, error: %w", ghURL, err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to download manifests.tar.gz from %s, error: %w", ghURL, err)
	}
	return resp, nil
}

func validateVersion(ctx context.Context, version, url, name string) error {
	ver := version
	if ver == "" {
		return fmt.Errorf("version is empty")
	}

	if !strings.HasPrefix(ver, "v") && ver != "latest" {
		ver = "v" + ver
	}

	ghURL := fmt.Sprintf(url+"/tags/%s", ver)
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
		return fmt.Errorf("target version %s does not exist for %s", ver, name)
	default:
		return fmt.Errorf("error while validating version %s for %s: %s", ver, name, resp.Status)
	}
}

func getLatestVersion(ctx context.Context, releaseAPIURL string) (string, error) {
	ghURL := fmt.Sprintf("%s/latest", releaseAPIURL)
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

func computeHash(payload []byte) (string, error) {
	h := sha256.New()
	_, err := h.Write(payload)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func writeFile(rootDir string, path string, content string) error {
	output, err := securejoin.SecureJoin(rootDir, path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(output), os.ModePerm); err != nil {
		return fmt.Errorf("unable to create dir, error: %w", err)
	}

	if err := os.WriteFile(output, []byte(content), os.ModePerm); err != nil {
		return fmt.Errorf("unable to write file, error: %w", err)
	}
	return nil
}
