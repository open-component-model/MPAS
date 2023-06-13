// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package release

import (
	"context"
	"fmt"
	"os"
	"path"

	mgen "github.com/open-component-model/mpas/pkg/manifestsgen"
	"github.com/open-component-model/mpas/pkg/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/clictx"
)

const (
	archivePathPrefix = "mpas-bootstrap-component"
)

var (
	fluxLocalizationTemplate = `- name: %s
file: gotk-components.yaml
image: spec.template.spec.containers[0].image
resource:
  name: %s
`
	fluxLocalizationTemplateHeader = `apiVersion: config.ocm.software/v1alpha1
kind: ConfigData
metadata:
  name: ocm-config
localization:
`
)

func ReleaseOcmControllerComponent(ctx context.Context, ocmVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	o, err := generateOcmController(ctx, ocmVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ocm-controller manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/souleb/ocm-controller",
		ocmVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return nil, fmt.Errorf("failed to create component archive: %w", err)
	}

	tmpl, err := o.GenerateLocalizationFromTemplate(fluxLocalizationTemplateHeader, fluxLocalizationTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate localization from template: %w", err)
	}
	images, err := o.GenerateImages()
	if err != nil {
		return nil, fmt.Errorf("failed to generate images: %w", err)
	}
	err = os.WriteFile(path.Join(tmpDir, "config.yaml"), []byte(tmpl), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write config.yaml: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("ocm-controller"),
		ocm.WithResourcePath(path.Join(tmpDir, o.Path)),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource ocm-controller: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("ocm-config"),
		ocm.WithResourcePath(path.Join(tmpDir, "config.yaml")),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource ocm-config: %w", err)
	}

	for image, nameVersion := range images {
		if err := component.AddResource(username, token, ocm.WithResourceName(nameVersion[0]),
			ocm.WithResourceType("ociImage"),
			ocm.WithResourceImage(image),
			ocm.WithResourceVersion(nameVersion[1])); err != nil {
			return nil, fmt.Errorf("failed to add resource %s: %w", image, err)
		}
	}
	if err := component.Transfer(); err != nil {
		return nil, fmt.Errorf("failed to transfer component: %w", err)
	}

	return component, nil
}

func ReleaseFluxComponent(ctx context.Context, fluxVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	f, err := generateFlux(ctx, fluxVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate flux manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/souleb/flux",
		fluxVersion,
		ocm.WithProvider("fluxcd"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return nil, fmt.Errorf("failed to create component archive: %w", err)
	}

	tmpl, err := f.GenerateLocalizationFromTemplate(fluxLocalizationTemplateHeader, fluxLocalizationTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate localization from template: %w", err)
	}
	images, err := f.GenerateImages()
	if err != nil {
		return nil, fmt.Errorf("failed to generate images: %w", err)
	}
	err = os.WriteFile(path.Join(tmpDir, "config.yaml"), []byte(tmpl), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write config.yaml: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("flux"),
		ocm.WithResourcePath(path.Join(tmpDir, f.Path)),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource flux: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("ocm-config"),
		ocm.WithResourcePath(path.Join(tmpDir, "config.yaml")),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource ocm-config: %w", err)
	}

	for image, nameVersion := range images {
		if err := component.AddResource(username, token, ocm.WithResourceName(nameVersion[0]),
			ocm.WithResourceType("ociImage"),
			ocm.WithResourceImage(image),
			ocm.WithResourceVersion(nameVersion[1])); err != nil {
			return nil, fmt.Errorf("failed to add resource %s: %w", image, err)
		}
	}
	if err := component.Transfer(); err != nil {
		return nil, fmt.Errorf("failed to transfer component: %w", err)
	}

	return component, nil
}

func generateFlux(ctx context.Context, version, tmpDir string) (mgen.Flux, error) {
	if version == "" {
		return mgen.Flux{}, fmt.Errorf("flux version is empty")
	}

	f := mgen.Flux{Version: version}
	err := f.GenerateManifests(ctx, tmpDir)
	return f, err
}

func generateOcmController(ctx context.Context, version, tmpDir string) (mgen.OcmController, error) {
	if version == "" {
		return mgen.OcmController{}, fmt.Errorf("flux version is empty")
	}

	o := mgen.OcmController{Version: version}
	err := o.GenerateManifests(ctx, tmpDir)
	return o, err
}
