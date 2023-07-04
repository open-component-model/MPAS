// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_OCM(t *testing.T) {
	tmpdir := t.TempDir()
	name := "github.com/ocm/test"
	octx := om.New(datacontext.MODE_SHARED)
	comp, err := NewComponent(octx, name, "v0.8.3",
		WithProvider("ocm"),
		WithRepositoryURL("ghcr.io/ocm/test"),
		WithUsername("my-user"),
		WithToken("my-token"))
	require.NoError(t, err)
	assert.Equal(t, name, comp.Name)
	assert.Equal(t, "v0.8.3", comp.Version)

	// create transfert archive
	ctf, err := CreateCTF(octx, fmt.Sprintf("%s/%s", tmpdir, "ctf"), accessio.FormatDirectory)
	require.NoError(t, err)
	defer ctf.Close()

	// add component to transfert archive
	err = comp.AddToCTF(ctf)
	require.NoError(t, err)
	defer comp.Close()

	text := []byte("hello world")
	fPath, err := writeFile(tmpdir, text)
	require.NoError(t, err)
	err = comp.AddResource(WithResourceName("my-file"),
		WithResourceType("file"),
		WithResourcePath(fPath),
		WithResourceVersion("v0.1.0"),
	)
	require.NoError(t, err)
	err = comp.AddResource(WithResourceType("ociImage"),
		WithResourceName("my-image"),
		WithResourceVersion("v0.1.0"),
		WithResourceImage("ghcr.io/my-registry/my-image:v0.1.0"))
	require.NoError(t, err)
}

func Test_ParseURL(t *testing.T) {
	testCases := []struct {
		name         string
		url          string
		expectedHost string
		expectedPath string
	}{
		{
			name:         "ghcr.io",
			url:          "ghcr.io/ocm/test",
			expectedHost: "ghcr.io",
			expectedPath: "/ocm/test",
		},
		{
			name:         "docker.io",
			url:          "docker.io/ocm/test",
			expectedHost: "docker.io",
			expectedPath: "/ocm/test",
		},
		{
			name:         "https://ghcr.io",
			url:          "https://ghcr.io/ocm/test",
			expectedHost: "ghcr.io",
			expectedPath: "/ocm/test",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := parseURL(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedHost, u.Host)
			assert.Equal(t, tc.expectedPath, u.Path)
		})
	}
}

func writeFile(tmpdir string, data []byte) (string, error) {
	file, err := os.Create(filepath.Join(tmpdir, "my-file.txt"))
	if err != nil {
		return "", nil
	}
	defer file.Close()
	err = os.WriteFile(file.Name(), data, 0644)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}
