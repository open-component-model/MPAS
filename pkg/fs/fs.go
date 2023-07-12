// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package fs

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CreateArchive creates a tar gzip archive from the given source directory.
// It returns the path to the archive and an error if the archive cannot be created.
func CreateArchive(src, archiveName string) (string, error) {
	tf, err := os.Create(path.Join(os.TempDir(), archiveName))
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer func() {
		if err != nil {
			os.Remove(tf.Name())
		}
	}()

	if !strings.HasSuffix(src, string(filepath.Separator)) {
		src += string(filepath.Separator)
	}

	if err := tarGzip(src, tf); err != nil {
		return "", fmt.Errorf("failed to create tar gzip archive: %w", err)
	}

	if err := tf.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Chmod(tf.Name(), 0o600); err != nil {
		return "", fmt.Errorf("failed to change file mode of %q: %w", tf.Name(), err)
	}

	return tf.Name(), nil
}

func tarGzip(src string, writers ...io.Writer) error {
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("failed to stat source directory %q: %w", src, err)
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk path %q: %w", file, err)
		}

		// Ignore anything that is not a file or directories e.g. symlinks
		if m := fi.Mode(); !(m.IsRegular() || m.IsDir()) {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header for file %q: %w", file, err)
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to open file %q: %w", file, err)
		}

		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return fmt.Errorf("failed to copy file %q to tar archive: %w", file, err)
		}

		return f.Close()
	})
}
