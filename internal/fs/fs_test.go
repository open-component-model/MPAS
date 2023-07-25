// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package fs

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CreateArchive(t *testing.T) {
	tmpDir := t.TempDir()
	dir1 := "dir1"
	nestedDirs := []string{"dir2/dir3/dir4", "dir2/dir3/dir5", "dir2/dir3/dir6"}
	err := createDir(tmpDir, dir1)
	assert.NoError(t, err)
	files := []string{"file1", "file2", "file3"}
	err = createFiles(path.Join(tmpDir, dir1), files)
	assert.NoError(t, err)
	dir2 := "dir2"
	err = createDir(tmpDir, dir2)
	assert.NoError(t, err)
	err = createNestedDirs(path.Join(tmpDir, dir2), nestedDirs)
	assert.NoError(t, err)

	testsCases := []struct {
		name     string
		src      string
		expected string
	}{
		{
			name:     "empty src",
			src:      "",
			expected: "",
		},
		{
			name:     "src is a file",
			src:      path.Join(tmpDir, dir1, files[0]),
			expected: "",
		},
		{
			name:     "src is a directory",
			src:      path.Join(tmpDir, dir1),
			expected: "test.tar.gz",
		},
		{
			name:     "src is a directory with subdirectories",
			src:      path.Join(tmpDir, dir2),
			expected: "test.tar.gz",
		},
	}

	for _, tc := range testsCases {
		t.Run(tc.name, func(t *testing.T) {
			archiveName := "test.tar.gz"
			archivePath, err := CreateArchive(tc.src, archiveName)
			if tc.expected == "" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, path.Join(os.TempDir(), tc.expected), archivePath)
			}
		})
	}
}

func createDir(tmpDir, dir string) error {
	err := os.Mkdir(path.Join(tmpDir, dir), 0o755)
	if err != nil {
		return err
	}
	return nil
}

func createNestedDirs(tmpDir string, nestedDirs []string) error {
	for _, dir := range nestedDirs {
		err := os.MkdirAll(path.Join(tmpDir, dir), 0o755)
		if err != nil {
			return err
		}
	}
	return nil
}

func createFiles(tmpDir string, files []string) error {
	for _, file := range files {
		err := createFile(path.Join(tmpDir, file))
		if err != nil {
			return err
		}
	}
	return nil
}

func createFile(file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}
