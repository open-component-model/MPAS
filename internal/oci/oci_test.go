// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tag struct {
	Tags []string `json:"tags"`
}

func Test_GetLatesVersion(t *testing.T) {
	versions := []string{"v0.1.0", "v0.2.0", "v0.3.0", "v1.0.0-alpha.1", "v1.0.0-beta.1", "v1.0.0-rc.1", "v1.0.0-rc.2", "v1.0.0"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/my-artifact/tags/list":
			payload := tag{
				Tags: versions,
			}
			data, err := json.Marshal(payload)
			require.NoError(t, err)
			_, err = w.Write(data)
			require.NoError(t, err)
		default:
			fmt.Println(r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	repoURL := fmt.Sprintf("%s/%s", srv.URL, "my-artifact")
	repo := Repository{
		RepositoryURL: repoURL,
		PlainHTTP:     true,
	}
	ver, err := repo.GetLatestVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, versions[len(versions)-1], ver)
}

func Test_PullArtifact(t *testing.T) {
	var (
		blobs [][]byte
		descs []ocispec.Descriptor
	)
	appendBlob := func(mediaType string, blob []byte) {
		blobs = append(blobs, blob)
		descs = append(descs, ocispec.Descriptor{
			MediaType: mediaType,
			Digest:    digest.FromBytes(blob),
			Size:      int64(len(blob)),
		})
	}
	generateManifest := func(config ocispec.Descriptor, layers ...ocispec.Descriptor) {
		manifest := ocispec.Manifest{
			Config: config,
			Layers: layers,
		}
		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		appendBlob(ocispec.MediaTypeImageManifest, manifestJSON)
	}
	appendBlob(ocispec.MediaTypeImageConfig, []byte("config"))
	appendBlob(ocispec.MediaTypeImageLayer, []byte("foo"))
	appendBlob(ocispec.MediaTypeImageLayer, []byte("bar"))
	generateManifest(descs[0], descs[1:3]...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/my-artifact/manifests/v1.0.0":
			manifest := descs[len(descs)-1]
			w.Header().Set("Content-Type", manifest.MediaType)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", manifest.Size))
			w.Write(blobs[len(blobs)-1])
		case "/v2/my-artifact/blobs/" + descs[0].Digest.String():
			w.Header().Set("Content-Type", descs[0].MediaType)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", descs[0].Size))
			w.Write(blobs[0])
		case "/v2/my-artifact/blobs/" + descs[1].Digest.String():
			w.Header().Set("Content-Type", descs[1].MediaType)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", descs[1].Size))
			w.Write(blobs[1])
		case "/v2/my-artifact/blobs/" + descs[2].Digest.String():
			w.Header().Set("Content-Type", descs[2].MediaType)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", descs[2].Size))
			w.Write(blobs[2])
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	repoURL := fmt.Sprintf("%s/%s", srv.URL, "my-artifact")
	repo := Repository{
		RepositoryURL: repoURL,
		PlainHTTP:     true,
	}
	_, err := repo.PullArtifact(ctx, "v1.0.0")
	require.NoError(t, err)
}
