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

	cgen "github.com/open-component-model/mpas/pkg/componentsgen"
	"github.com/open-component-model/mpas/pkg/ocm"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	archivePathPrefix = "mpas-bootstrap-component"
	fluxBinURL        = "https://github.com/fluxcd/flux2/releases/download"
	ocmBinURL         = "https://github.com/open-component-model/ocm/releases/download"
)

var (
	fluxLocalizationtemplate = `- name: %s
file: gotk-components.yaml
image: spec.template.spec.containers[0].image
resource:
  name: %s
`
	ocmLocalizationtemplate = `- name: %s
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
	releaseAPIURL = "https://api.github.com/repos/open-component-model/%s/releases"
	releaseURL    = "https://github.com/open-component-model/%s/releases"
)

// Releaser releases the bootstrap component and its dependencies.
type Releaser struct {
	octx          om.Context
	username      string
	token         string
	tmpDir        string
	repositoryURL string
	target        om.Repository
}

// New creates a new Releaser.
func New(octx om.Context, username, token, tmpDir, repositoryURL string, target om.Repository) *Releaser {
	return &Releaser{
		octx:          octx,
		username:      username,
		token:         token,
		tmpDir:        tmpDir,
		repositoryURL: repositoryURL,
		target:        target,
	}
}

// ReleaseBootstrapComponent releases the bootstrap component.
func (r *Releaser) ReleaseBootstrapComponent(ctx context.Context, components map[string]*ocm.Component, bootstrapVersion string) error {
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/bootstrap",
		bootstrapVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, "bootstrap"))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return fmt.Errorf("failed to create component archive: %w", err)
	}

	for ref, comp := range components {
		if err := component.AddResource(ocm.WithResourceName(ref),
			ocm.WithResourceType("componentReference"),
			ocm.WithComponentName(comp.Name),
			ocm.WithResourceVersion(comp.Version)); err != nil {
			return fmt.Errorf("failed to add resource flux: %w", err)
		}
	}

	if err := component.Transfer(r.target); err != nil {
		return fmt.Errorf("failed to transfer component: %w", err)
	}

	return nil
}

// ReleaseOCMControllerComponent releases the ocm-controller component.
func (r *Releaser) ReleaseOcmControllerComponent(ctx context.Context, ocmVersion, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "ocm-controller", ocmVersion, r.tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ocm-controller manifests: %v", err)
	}
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/ocm-controller",
		ocmVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := r.release(ctx, component, &o, "ocm-controller", ocmLocalizationtemplate); err != nil {
		return nil, fmt.Errorf("failed to release ocm-controller component: %w", err)
	}

	return component, nil
}

// ReleaseGitControllerComponent releases the git-controller component.
func (r *Releaser) ReleaseGitControllerComponent(ctx context.Context, gitVersion, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "git-controller", gitVersion, r.tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git-controller manifests: %v", err)
	}
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/git-controller",
		gitVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := r.release(ctx, component, &o, "git-controller", ocmLocalizationtemplate); err != nil {
		return nil, fmt.Errorf("failed to release git-controller component: %w", err)
	}

	return component, nil
}

// ReleaseReplicationControllerComponent releases the replication-controller component.
func (r *Releaser) ReleaseReplicationControllerComponent(ctx context.Context, replicationVersion, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "replication-controller", replicationVersion, r.tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate replication-controller manifests: %v", err)
	}
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/replication-controller",
		replicationVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := r.release(ctx, component, &o, "replication-controller", ocmLocalizationtemplate); err != nil {
		return nil, fmt.Errorf("failed to release replication-controller component: %w", err)
	}

	return component, nil
}

// ReleaseMpasProductControllerComponent releases the mpas-product-controller component.
func (r *Releaser) ReleaseMpasProductControllerComponent(ctx context.Context, mpasProductVersion, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "mpas-product-controller", mpasProductVersion, r.tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mpas-product-controller manifests: %v", err)
	}
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/mpas-product-controller",
		mpasProductVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := r.release(ctx, component, &o, "mpas-product-controller", ocmLocalizationtemplate); err != nil {
		return nil, fmt.Errorf("failed to release mpas-product-controller component: %w", err)
	}

	return component, nil
}

// ReleaseMpasProjectControllerComponent releases the mpas-project-controller component.
func (r *Releaser) ReleaseMpasProjectControllerComponent(ctx context.Context, mpasProjectVersion, comp string) (*ocm.Component, error) {
	o, err := generateController(ctx, "mpas-project-controller", mpasProjectVersion, r.tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mpas-project-controller manifests: %v", err)
	}
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/mpas-project-controller",
		mpasProjectVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := r.release(ctx, component, &o, "mpas-project-controller", ocmLocalizationtemplate); err != nil {
		return nil, fmt.Errorf("failed to release mpas-project-controller component: %w", err)
	}

	return component, nil
}

// ReleaseFluxComponent releases flux with all its components
func (r *Releaser) ReleaseFluxComponent(ctx context.Context, fluxVersion, comp string) (*ocm.Component, error) {
	f, err := generateFlux(ctx, fluxVersion, r.tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate flux manifests: %v", err)
	}
	component := ocm.NewComponent(r.octx,
		"github.com/mpas/flux",
		fluxVersion,
		ocm.WithProvider("fluxcd"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := r.release(ctx, component, &f, "flux", fluxLocalizationtemplate); err != nil {
		return nil, fmt.Errorf("failed to release flux component: %w", err)
	}

	return component, nil
}

// ReleaseFluxCliComponent releases flux-cli.
func (r *Releaser) ReleaseFluxCliComponent(ctx context.Context, fluxVersion, comp, targetOS, targetArch string) (*ocm.Component, error) {
	if fluxVersion == "" {
		return nil, fmt.Errorf("flux version is empty")
	}
	ver := strings.TrimPrefix(fluxVersion, "v")

	binURL := fmt.Sprintf("%s/v%s/flux_%s_%s_%s.tar.gz", fluxBinURL, ver, ver, targetOS, targetArch)
	hashURL := fmt.Sprintf("%s/v%s/flux_%s_checksums.txt", fluxBinURL, ver, ver)
	b, err := getBinary(ctx, fluxVersion, r.tmpDir, binURL, hashURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get flux-cli binary: %v", err)
	}

	component := ocm.NewComponent(r.octx,
		"github.com/mpas/flux-cli",
		fluxVersion,
		ocm.WithProvider("fluxcd"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return nil, fmt.Errorf("failed to create component archive: %w", err)
	}

	if err := component.AddResource(ocm.WithResourceName("flux-cli"),
		ocm.WithResourcePath(path.Join(r.tmpDir, b.Path)),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource flux: %w", err)
	}

	if err := component.Transfer(r.target); err != nil {
		return nil, fmt.Errorf("failed to transfer component: %w", err)
	}

	return component, nil
}

// ReleaseOCMCliComponent releases ocm-cli.
func (r *Releaser) ReleaseOCMCliComponent(ctx context.Context, ocmCliVersion, comp, targetOS, targetArch string) (*ocm.Component, error) {
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
	b, err := getBinary(ctx, ocmCliVersion, r.tmpDir, binURL, hashURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get ocm-cli binary: %v", err)
	}

	component := ocm.NewComponent(r.octx,
		"github.com/mpas/ocm-cli",
		ocmCliVersion,
		ocm.WithProvider("ocm"),
		ocm.WithUsername(r.username),
		ocm.WithToken(r.token),
		ocm.WithArchivePath(path.Join(r.tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp))),
		ocm.WithRepositoryURL(r.repositoryURL))

	if err := component.CreateComponentArchive(); err != nil {
		return nil, fmt.Errorf("failed to create component archive: %w", err)
	}

	if err := component.AddResource(ocm.WithResourceName("ocm-cli"),
		ocm.WithResourcePath(path.Join(r.tmpDir, b.Path)),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return nil, fmt.Errorf("failed to add resource flux: %w", err)
	}

	if err := component.Transfer(r.target); err != nil {
		return nil, fmt.Errorf("failed to transfer component: %w", err)
	}

	return component, nil
}

func (r *Releaser) release(ctx context.Context, component *ocm.Component, gen cgen.Generator, name, loc string) error {
	if err := component.CreateComponentArchive(); err != nil {
		return fmt.Errorf("failed to create component archive: %w", err)
	}

	tmpl, err := gen.GenerateLocalizationFromTemplate(localizationTemplateHeader, loc)
	if err != nil {
		return fmt.Errorf("failed to generate localization from template: %w", err)
	}
	images, err := gen.GenerateImages()
	if err != nil {
		return fmt.Errorf("failed to generate images: %w", err)
	}
	err = os.WriteFile(path.Join(r.tmpDir, "config.yaml"), []byte(tmpl), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config.yaml: %w", err)
	}

	if err := component.AddResource(ocm.WithResourceName(name),
		ocm.WithResourcePath(path.Join(r.tmpDir, gen.GetPath())),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return fmt.Errorf("failed to add resource %s: %w", name, err)
	}

	if err := component.AddResource(ocm.WithResourceName("ocm-config"),
		ocm.WithResourcePath(path.Join(r.tmpDir, "config.yaml")),
		ocm.WithResourceType("file"),
		ocm.WithResourceVersion(component.Version)); err != nil {
		return fmt.Errorf("failed to add resource ocm-config: %w", err)
	}

	for image, nameVersion := range images {
		if err := component.AddResource(ocm.WithResourceName(nameVersion[0]),
			ocm.WithResourceType("ociImage"),
			ocm.WithResourceImage(image),
			ocm.WithResourceVersion(nameVersion[1])); err != nil {
			return fmt.Errorf("failed to add resource %s: %w", image, err)
		}
	}
	if err := component.Transfer(r.target); err != nil {
		return fmt.Errorf("failed to transfer component: %w", err)
	}

	return nil
}

func getBinary(ctx context.Context, version, tmpDir, binURL, hashURL string) (cgen.Binary, error) {
	b := cgen.Binary{
		Version: version,
		BinURL:  binURL,
		HashURL: hashURL,
	}
	err := b.Get(ctx, tmpDir)
	return b, err
}

func generateFlux(ctx context.Context, version, tmpDir string) (cgen.Flux, error) {
	if version == "" {
		return cgen.Flux{}, fmt.Errorf("flux version is empty")
	}

	f := cgen.Flux{Version: version}
	err := f.GenerateManifests(ctx, tmpDir)
	return f, err
}

func generateController(ctx context.Context, name, version, tmpDir string) (cgen.Controller, error) {
	if version == "" {
		return cgen.Controller{}, fmt.Errorf("contoller version is empty")
	}

	o := cgen.Controller{
		Name:          name,
		Version:       version,
		ReleaseURL:    fmt.Sprintf(releaseURL, name),
		ReleaseAPIURL: fmt.Sprintf(releaseAPIURL, name),
	}
	err := o.GenerateManifests(ctx, tmpDir)
	return o, err
}
