//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-component-model/ocm-e2e-framework/shared"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

func createTestComponentVersion(t *testing.T) *features.FeatureBuilder {
	t.Helper()

	return features.New("Setup OCM component for testing").
		Setup(setup.AddComponentVersions(podinfoBackend(t))).
		Setup(setup.AddComponentVersions(podinfoFrontend(t))).
		Setup(setup.AddComponentVersions(podinfoRedis(t))).
		Setup(setup.AddComponentVersions(podinfo(t)))
}

func podinfo(t *testing.T) setup.Component {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", "product_description.yaml"))
	if err != nil {
		t.Fatal("failed to read setup file: %w", err)
	}

	return setup.Component{
		Component: shared.Component{
			Name:    "mpas.ocm.software/podinfo",
			Version: "1.0.0",
		},
		ComponentVersionModifications: []shared.ComponentModification{
			shared.BlobResource(shared.Resource{
				Name:    "product-description",
				Data:    string(content),
				Type:    "productdescription.mpas.ocm.software",
				Version: "1.0.0",
			}),
			shared.ComponentVersionRef(shared.ComponentRef{
				Name:          "backend",
				Version:       "1.0.0",
				ComponentName: "mpas.ocm.software/podinfo/backend",
			}),
			shared.ComponentVersionRef(shared.ComponentRef{
				Name:          "frontend",
				Version:       "1.0.0",
				ComponentName: "mpas.ocm.software/podinfo/frontend",
			}),
			shared.ComponentVersionRef(shared.ComponentRef{
				Name:          "redis",
				Version:       "1.0.0",
				ComponentName: "mpas.ocm.software/redis",
			}),
		},
	}
}

func podinfoBackend(t *testing.T) setup.Component {
	t.Helper()

	configContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "backend", "config.yaml"))
	if err != nil {
		t.Fatal("failed to read config file: %w", err)
	}

	manifestContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "backend", "manifests.tar"))
	if err != nil {
		t.Fatal("failed to read manifest file: %w", err)
	}

	schemaContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "backend", "schema.cue"))
	if err != nil {
		t.Fatal("failed to read schema file: %w", err)
	}

	return setup.Component{
		Component: shared.Component{
			Name:    "mpas.ocm.software/podinfo/backend",
			Version: "1.0.0",
		},
		ComponentVersionModifications: []shared.ComponentModification{
			shared.BlobResource(shared.Resource{
				Name: "config",
				Data: string(configContent),
				Type: "configdata.ocm.software",
			}),
			shared.BlobResource(shared.Resource{
				Name: "schema",
				Data: string(schemaContent),
				Type: "PlainText",
			}),
			shared.ImageRefResource("ghcr.io/stefanprodan/podinfo:6.2.0", shared.Resource{
				Name:    "image",
				Version: "6.2.0",
				Type:    "ociImage",
			}),
			shared.BlobResource(shared.Resource{
				Name: "manifests",
				Data: string(manifestContent),
				Type: "kustomize.ocm.fluxcd.io",
			}),
		},
	}
}

func podinfoFrontend(t *testing.T) setup.Component {
	t.Helper()

	configContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "frontend", "config.yaml"))
	if err != nil {
		t.Fatal("failed to read config file: %w", err)
	}

	manifestContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "frontend", "manifests.tar"))
	if err != nil {
		t.Fatal("failed to read manifest file: %w", err)
	}

	schemaContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "frontend", "schema.cue"))
	if err != nil {
		t.Fatal("failed to read schema file: %w", err)
	}

	return setup.Component{
		Component: shared.Component{
			Name:    "mpas.ocm.software/podinfo/frontend",
			Version: "1.0.0",
		},
		ComponentVersionModifications: []shared.ComponentModification{
			shared.BlobResource(shared.Resource{
				Name: "config",
				Data: string(configContent),
				Type: "configdata.ocm.software",
			}),
			shared.BlobResource(shared.Resource{
				Name: "schema",
				Data: string(schemaContent),
				Type: "PlainText",
			}),
			shared.ImageRefResource("ghcr.io/stefanprodan/podinfo:6.2.0", shared.Resource{
				Name:    "image",
				Version: "6.2.0",
				Type:    "ociImage",
			}),
			shared.BlobResource(shared.Resource{
				Name: "manifests",
				Data: string(manifestContent),
				Type: "kustomize.ocm.fluxcd.io",
			}),
		},
	}
}

func podinfoRedis(t *testing.T) setup.Component {
	t.Helper()

	configContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "redis", "config.yaml"))
	if err != nil {
		t.Fatal("failed to read config file: %w", err)
	}

	manifestContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "redis", "manifests.tar"))
	if err != nil {
		t.Fatal("failed to read manifest file: %w", err)
	}

	schemaContent, err := os.ReadFile(filepath.Join("testdata", "podinfo", "redis", "schema.cue"))
	if err != nil {
		t.Fatal("failed to read schema file: %w", err)
	}

	return setup.Component{
		Component: shared.Component{
			Name:    "mpas.ocm.software/redis",
			Version: "1.0.0",
		},
		ComponentVersionModifications: []shared.ComponentModification{
			shared.BlobResource(shared.Resource{
				Name: "config",
				Data: string(configContent),
				Type: "configdata.ocm.software",
			}),
			shared.BlobResource(shared.Resource{
				Name: "schema",
				Data: string(schemaContent),
				Type: "PlainText",
			}),
			shared.ImageRefResource("redis:6.0.1", shared.Resource{
				Name:    "image",
				Version: "6.2.0",
				Type:    "ociImage",
			}),
			shared.BlobResource(shared.Resource{
				Name: "manifests",
				Data: string(manifestContent),
				Type: "kustomize.ocm.fluxcd.io",
			}),
		},
	}
}
