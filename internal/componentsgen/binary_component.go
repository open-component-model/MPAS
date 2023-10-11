// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Binary is a component that contains a binary file.
type Binary struct {
	// Version is the version of the binary.
	Version string
	// Path is the path to the binary file.
	Path string
	// Content is the content of the binary file as bytes.
	Content []byte
	// BinURL is the URL to the binary file.
	BinURL string
	// HashURL is the URL to the checksum file.
	HashURL string
}

// Get downloads the binary file and the checksum file and validates the checksum.
func (b *Binary) Get(ctx context.Context, tmpDir string) error {
	if err := b.validateVersion(); err != nil {
		return err
	}

	var (
		hashDL string
		bin    []byte
		err    error
	)
	if bin, err = b.fetchBinary(ctx); err != nil {
		return fmt.Errorf("failed to download binary file: %w", err)
	}

	if hashDL, err = b.fetchHash(ctx); err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}

	hash, err := computeHash(bin)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	if hash != hashDL {
		return fmt.Errorf("hash mismatch: %s != %s", hash, hashDL)
	}

	b.Content = bin

	if tmpDir != "" {
		path, err := b.writeFile(tmpDir)
		if err != nil {
			return fmt.Errorf("failed to write manifests to temporary directory: %w", err)
		}
		b.Path = path
	}
	return nil
}

func (b *Binary) fetchBinary(ctx context.Context) ([]byte, error) {
	resp, err := getFrom(ctx, b.BinURL)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download binary from %s, status: %s", b.BinURL, resp.Status)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	data := buf.Bytes()
	return data, nil
}

func (b *Binary) fetchHash(ctx context.Context) (string, error) {
	resp, err := getFrom(ctx, b.HashURL)
	if err != nil {
		return "", err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download binary from %s, status: %s", b.HashURL, resp.Status)
	}

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", fmt.Errorf("failed to  copy to buffer: %w", err)
	}

	lines := strings.Split(buf.String(), "\n")
	substrs := strings.Split(b.BinURL, "/")
	for _, line := range lines {
		if strings.Contains(line, substrs[len(substrs)-1]) {
			hash := strings.Split(line, " ")[0]
			if hash == "" {
				return "", fmt.Errorf("failed to get hash for %s", b.BinURL)
			}
			return hash, nil
		}
	}
	return "", fmt.Errorf("failed to get hash for %s", b.BinURL)
}

func (b *Binary) writeFile(rootDir string) (string, error) {
	substrs := strings.Split(b.BinURL, "/")
	path := substrs[len(substrs)-1]
	err := writeFile(rootDir, path, string(b.Content))
	if err != nil {
		return "", err
	}
	return path, nil
}

func (b *Binary) validateVersion() error {
	if b.Version == "" || b.Version == "latest" {
		return fmt.Errorf("version must not be empty or latest")
	}

	if !strings.HasPrefix(b.Version, "v") {
		return fmt.Errorf("version must start with v")
	}
	return nil
}
