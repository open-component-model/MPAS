// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

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

func validateVersion(version string) error {
	if version == "" || version == "latest" {
		return fmt.Errorf("version must not be empty or latest")
	}

	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version must start with v")
	}
	return nil
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
