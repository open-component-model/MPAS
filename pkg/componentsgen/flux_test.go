// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fluxlocalizationTemplate = `- name: %s
file: gotk-components.yaml
image: spec.template.spec.containers[0].image
resource:
  name: %s
`

func Test_FluxLastVersion(t *testing.T) {
	tmpDir := t.TempDir()
	apiURL := "https://api.github.com/repos"
	baseURL, err := url.Parse(install.MakeDefaultOptions().BaseURL)
	require.NoError(t, err)
	apiURL += baseURL.Path
	latest, err := getLatestVersion(context.Background(), apiURL)
	require.NoError(t, err)
	fmt.Println(latest)
	f := &Flux{
		Version: latest,
	}

	err = f.GenerateManifests(context.Background(), tmpDir)
	require.NoError(t, err)

	assert.NotEmpty(t, f.Content)
	assert.NotEmpty(t, f.Path)
	assert.NotEmpty(t, f.Registry)
	assert.NotEmpty(t, f.Components)

	loc, err := f.GenerateLocalizationFromTemplate(localizationTemplateHeader, fluxlocalizationTemplate)
	require.NoError(t, err)
	assert.NotEmpty(t, loc)
	assert.Contains(t, loc, "source-controller")
	assert.Contains(t, loc, "kustomize-controller")
	assert.Contains(t, loc, "helm-controller")
	assert.Contains(t, loc, "notification-controller")

	images, err := f.GenerateImages()
	require.NoError(t, err)
	assert.NotEmpty(t, images)
}
