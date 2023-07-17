// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nameTag struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func Test_FetchLatestComponentVersion(t *testing.T) {
	versions := []string{"v0.1.0", "v0.2.0", "v0.3.0", "v1.0.0-alpha.1", "v1.0.0-beta.1", "v1.0.0-rc.1", "v1.0.0-rc.2", "v1.0.0"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/ocm/test/component-descriptors/test/tags/list":
			payload := nameTag{
				Name: "ocm/test",
				Tags: versions,
			}
			data, err := json.Marshal(payload)
			require.NoError(t, err)
			_, err = w.Write(data)
			require.NoError(t, err)
		case "/v2/ocm/test/component-descriptors/test/manifests/v1.0.0":
			manifest := `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
  	"mediaType": "application/vnd.oci.image.config.v1+json",
  	"digest": "sha256:1234567890",
  	"size": 702
  },
  "layers": [
  	{
  		"mediaType": "application/vnd.oci.image.layer.v1.tar",
  		"digest": "sha256:1234567890",
  		"size": "1234567890"
  	}
  ]
}`
			fmt.Println(r.URL.Path)
			w.Write([]byte(manifest))
		default:
			fmt.Println(r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	octx := ocm.DefaultContext()
	repo, err := makeOCIRepository(octx, srv.URL, "ocm/test")
	require.NoError(t, err)
	c, err := repo.LookupComponent("test")
	require.NoError(t, err)
	cv, err := fetchLatestComponentVersion(c, "test")
	require.NoError(t, err)
	assert.Equal(t, versions[len(versions)-1], cv.Original())
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
