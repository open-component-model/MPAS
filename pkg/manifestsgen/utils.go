// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package manifestsgen

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
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

func validateFluxVersion(version string) error {
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
