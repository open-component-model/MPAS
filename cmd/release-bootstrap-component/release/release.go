// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package release

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	mgen "github.com/open-component-model/mpas/pkg/manifestsgen"
	"github.com/open-component-model/mpas/pkg/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/clictx"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	archivePathPrefix = "mpas-bootstrap-component"
	fluxBinURL        = "https://github.com/fluxcd/flux2/releases/download"
	ocmBinURL         = "https://github.com/open-component-model/ocm/releases/download"
)

var (
	localizationtemplate = `- name: %s
file: gotk-components.yaml
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
	releaseAPIURL = "https://api.github.com/repos/open-component-model/%s/releases"
	releaseURL    = "https://github.com/open-component-model/%s/releases"
)

func ReleaseBootstrapComponent(ctx context.Context, components []*ocm.Component, bootstrapVersion, username, token, tmpDir, repositoryURL string) error {
	return nil
}

func ReleaseOcmControllerComponent(ctx context.Context, ocmVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "ocm-controller", ocmVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ocm-controller manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/ocm-controller",
		ocmVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := release(ctx, component, &o, username, token, tmpDir, "ocm-controller"); err != nil {
		return nil, fmt.Errorf("failed to release ocm-controller component: %w", err)
	}

	return component, nil
}

func ReleaseGitControllerComponent(ctx context.Context, gitVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "git-controller", gitVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git-controller manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/git-controller",
		gitVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := release(ctx, component, &o, username, token, tmpDir, "git-controller"); err != nil {
		return nil, fmt.Errorf("failed to release git-controller component: %w", err)
	}

	return component, nil
}

func ReleaseReplicationControllerComponent(ctx context.Context, replicationVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "replication-controller", replicationVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate replication-controller manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/replication-controller",
		replicationVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := release(ctx, component, &o, username, token, tmpDir, "replication-controller"); err != nil {
		return nil, fmt.Errorf("failed to release replication-controller component: %w", err)
	}

	return component, nil
}

func ReleaseMpasProductControllerComponent(ctx context.Context, mpasProductVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "mpas-product-controller", mpasProductVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mpas-product-controller manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/mpas-product-controller",
		mpasProductVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := release(ctx, component, &o, username, token, tmpDir, "mpas-product-controller"); err != nil {
		return nil, fmt.Errorf("failed to release mpas-product-controller component: %w", err)
	}

	return component, nil
}

func ReleaseMpasProjectControllerComponent(ctx context.Context, mpasProjectVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "mpas-project-controller", mpasProjectVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mpas-project-controller manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/mpas-project-controller",
		mpasProjectVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := release(ctx, component, &o, username, token, tmpDir, "mpas-project-controller"); err != nil {
		return nil, fmt.Errorf("failed to release mpas-project-controller component: %w", err)
	}

	return component, nil
}

func ReleaseFluxComponent(ctx context.Context, fluxVersion, username, token, tmpDir, repositoryURL, comp string) (*ocm.Component, error) {
	f, err := generateFlux(ctx, fluxVersion, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate flux manifests: %v", err)
	}
	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/flux",
		fluxVersion,
		ocm.WithProvider("fluxcd"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := release(ctx, component, &f, username, token, tmpDir, "flux"); err != nil {
		return nil, fmt.Errorf("failed to release flux component: %w", err)
	}

	return component, nil
}

func ReleaseFluxCliComponent(ctx context.Context, fluxVersion, username, token, tmpDir, repositoryURL, comp, targetOS, targetArch string) (*ocm.Component, error) {
	if fluxVersion == "" {
		return nil, fmt.Errorf("flux version is empty")
	}
	ver := strings.TrimPrefix(fluxVersion, "v")

	binURL := fmt.Sprintf("%s/v%s/flux_%s_%s_%s.tar.gz", fluxBinURL, ver, ver, targetOS, targetArch)
	hashURL := fmt.Sprintf("%s/v%s/flux_%s_checksums.txt", fluxBinURL, ver, ver)
	b, err := getBinary(ctx, fluxVersion, tmpDir, binURL, hashURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get flux-cli binary: %v", err)
	}

	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/flux-cli",
		fluxVersion,
		ocm.WithProvider("fluxcd"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return nil, fmt.Errorf("failed to create component archive: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("flux-cli"),
		ocm.WithResourcePath(path.Join(tmpDir, b.Path)),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource flux: %w", err)
	}

	if err := component.Transfer(); err != nil {
		return nil, fmt.Errorf("failed to transfer component: %w", err)
	}

	return component, nil
}

func ReleaseOCMCliComponent(ctx context.Context, ocmCliVersion, username, token, tmpDir, repositoryURL, comp, targetOS, targetArch string) (*ocm.Component, error) {
	if ocmCliVersion == "" {
		return nil, fmt.Errorf("ocm version is empty")
	}
	ver := strings.TrimPrefix(ocmCliVersion, "v")
	caseEng := cases.Title(language.Dutch)
	targetOS = caseEng.String(targetOS)
	if targetArch == "amd64" {
		targetArch = "x86_64"
	}
	binURL := fmt.Sprintf("%s/v%s/ocm_%s_%s.tar.gz", ocmBinURL, ver, targetOS, targetArch)
	hashURL := fmt.Sprintf("%s/v%s/checksums.txt", ocmBinURL, ver)
	b, err := getBinary(ctx, ocmCliVersion, tmpDir, binURL, hashURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get ocm-cli binary: %v", err)
	}

	component := ocm.NewComponent(clictx.DefaultContext(),
		"github.com/mpas/ocm-cli",
		ocmCliVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(username),
		ocm.WithToken(token),
		ocm.WithArchivePath(path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return nil, fmt.Errorf("failed to create component archive: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("ocm-cli"),
		ocm.WithResourcePath(path.Join(tmpDir, b.Path)),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource flux: %w", err)
	}

	if err := component.Transfer(); err != nil {
		return nil, fmt.Errorf("failed to transfer component: %w", err)
	}

	return component, nil
}

func release(ctx context.Context, component *ocm.Component, gen mgen.Generator, username, token, tmpDir, name string) error {
	if err := component.CreateComponentArchive(); err != nil {
		return fmt.Errorf("failed to create component archive: %w", err)
	}

	tmpl, err := gen.GenerateLocalizationFromTemplate(localizationTemplateHeader, localizationtemplate)
	if err != nil {
		return fmt.Errorf("failed to generate localization from template: %w", err)
	}
	images, err := gen.GenerateImages()
	if err != nil {
		return fmt.Errorf("failed to generate images: %w", err)
	}
	err = os.WriteFile(path.Join(tmpDir, "config.yaml"), []byte(tmpl), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config.yaml: %w", err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName(name),
		ocm.WithResourcePath(path.Join(tmpDir, gen.GetPath())),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return fmt.Errorf("failed to add resource %s: %w", name, err)
	}

	if err := component.AddResource(username, token, ocm.WithResourceName("ocm-config"),
		ocm.WithResourcePath(path.Join(tmpDir, "config.yaml")),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return fmt.Errorf("failed to add resource ocm-config: %w", err)
	}

	for image, nameVersion := range images {
		if err := component.AddResource(username, token, ocm.WithResourceName(nameVersion[0]),
			ocm.WithResourceType("ociImage"),
			ocm.WithResourceImage(image),
			ocm.WithResourceVersion(nameVersion[1])); err != nil {
			return fmt.Errorf("failed to add resource %s: %w", image, err)
		}
	}
	if err := component.Transfer(); err != nil {
		return fmt.Errorf("failed to transfer component: %w", err)
	}

	return nil
}

func getBinary(ctx context.Context, version, tmpDir, binURL, hashURL string) (mgen.Binary, error) {
	b := mgen.Binary{
		Version: version,
		BinURL:  binURL,
		HashURL: hashURL,
	}
	err := b.Get(ctx, tmpDir)
	return b, err
}

func generateFlux(ctx context.Context, version, tmpDir string) (mgen.Flux, error) {
	if version == "" {
		return mgen.Flux{}, fmt.Errorf("flux version is empty")
	}

	f := mgen.Flux{Version: version}
	err := f.GenerateManifests(ctx, tmpDir)
	return f, err
}

func generateController(ctx context.Context, name, version, tmpDir string) (mgen.Controller, error) {
	if version == "" {
		return mgen.Controller{}, fmt.Errorf("contoller version is empty")
	}

	o := mgen.Controller{
		Name:          name,
		Version:       version,
		ReleaseURL:    fmt.Sprintf(releaseURL, name),
		ReleaseAPIURL: fmt.Sprintf(releaseAPIURL, name),
	}
	err := o.GenerateManifests(ctx, tmpDir)
	return o, err
}
