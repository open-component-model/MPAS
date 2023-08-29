// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-component-model/mpas/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// _NOTE_: It's important to format this deployment with spaces. Otherwise, kustomization will fail
// with invalid Token for the install.yaml.
const (
	deployment = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: git-controller
  namespace: ocm-system
spec:
  selector:
    matchLabels:
      app: git-controller
  replicas: 1
  template:
    metadata:
      labels:
        app: git-controller
    spec:
      containers:
      - name: manager
        image: open-component-model/git-controller
`
	ocmlocalizationTemplate = `- name: %s
file: install.yaml
image: spec.template.spec.containers[0].image
resource:
  name: %s
`
	localizationTemplateHeader = `apiVersion: config.ocm.software/v1alpha1
kind: ConfigData
metadata:
  name: ocm-config
localization:
`
)

func Test_Controller(t *testing.T) {
	tmpDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/v0.1.0/install.yaml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(deployment))
		case "/tags/v0.1.0":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "v0.1.0"}`))
		case "/latest":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tag_name": "v0.1.0"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	testCases := []struct {
		name        string
		version     string
		expectedErr bool
	}{
		{
			name:    "valid version",
			version: "v0.1.0",
		},
		{
			name:        "invalid version",
			version:     "1.0.0.0",
			expectedErr: true,
		},
		{
			name:        "latest version",
			version:     "latest",
			expectedErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Controller{
				Name:          "git-controller",
				Version:       tc.version,
				ReleaseAPIURL: server.URL,
				ReleaseURL:    server.URL,
				Registry:      env.DefaultOCMHost,
			}
			err := c.GenerateManifests(context.Background(), tmpDir)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Contains(t, *c.Content, "path: registry-root.pem")

			locs, err := c.GenerateLocalizationFromTemplate(localizationTemplateHeader, ocmlocalizationTemplate)
			require.NoError(t, err)
			assert.Contains(t, locs, "git-controller")
		})
	}
}
