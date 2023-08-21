// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"k8s.io/utils/pointer"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var (
	//go:embed patches/kustomize_ocm-controller.yaml
	ocmControllerPatch []byte
	//go:embed patches/kustomize_git-controller.yaml
	gitControllerPatch []byte
	//go:embed patches/kustomize_replication-controller.yaml
	replicationControllerPatch []byte
	//go:embed patches/kustomize_mpas-product-controller.yaml
	mpasProductControllerPatch []byte
	//go:embed patches/kustomize_mpas-project-controller.yaml
	mpasProjectControllerPatch []byte
)

// patchMap maps all patches to the controllers.
var patchMap = map[string][]byte{
	"ocm-controller":          ocmControllerPatch,
	"git-controller":          gitControllerPatch,
	"replication-controller":  replicationControllerPatch,
	"mpas-project-controller": mpasProjectControllerPatch,
	"mpas-product-controller": mpasProductControllerPatch,
}

const (
	defaultRegistry   = "ghcr.io/open-component-model"
	secretPlaceholder = "<SECRET-NAME>"
)

// Controller is a component that generates manifests for a controller,
// localization files from a template, and images for a given controller.
type Controller struct {
	// Name is the name of the controller.
	Name string
	// Version is the version of the controller.
	Version string
	// Registry is the registry to get the controller image from.
	Registry string
	// Path is the path to the manifests.
	Path string
	// ReleaseURL is the URL to the release page.
	ReleaseURL string
	// ReleaseAPIURL is the URL to the release API.
	ReleaseAPIURL string
	// Content is the content of the install.yaml file.
	Content *string
	// CertificateSecretName defines the name of the secret that stores the registry certificates.
	CertificateSecretName string
}

// GenerateManifests downloads the install.yaml file and writes it to a temporary directory.
// It validates the version and returns an error if the version does not exist.
func (o *Controller) GenerateManifests(ctx context.Context, tmpDir string) error {
	if err := o.validateVersion(ctx); err != nil {
		return err
	}

	if err := o.fetch(ctx); err != nil {
		return fmt.Errorf("failed to download install.yaml file: %w", err)
	}

	// TODO: Make the certificate secret name configurable.
	if err := o.applyCertificatePatch(); err != nil {
		return fmt.Errorf("failed to apply patch to install.yaml file: %w", err)
	}

	if tmpDir != "" {
		path, err := o.writeFile(tmpDir)
		if err != nil {
			return fmt.Errorf("failed to write manifests to temporary directory: %w", err)
		}
		o.Path = path
	}

	o.Registry = defaultRegistry
	return nil
}

func (o *Controller) validateVersion(ctx context.Context) error {
	ver := o.Version
	if ver == "" {
		return fmt.Errorf("version is empty")
	}

	if !strings.HasPrefix(ver, "v") && ver != "latest" {
		ver = "v" + ver
	}

	if ver == "latest" {
		latest, err := getLatestVersion(ctx, o.ReleaseAPIURL)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest version for %s: %s", o.Name, err)
		}
		o.Version = latest
		return nil
	}

	ghURL := fmt.Sprintf(o.ReleaseAPIURL+"/tags/%s", ver)
	resp, err := getFrom(ctx, ghURL)
	if err != nil {
		return err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("target version %s does not exist for %s", ver, o.Name)
	default:
		return fmt.Errorf("error while validating version %s for %s: %s", ver, o.Name, resp.Status)
	}
}

func (o *Controller) fetch(ctx context.Context) error {
	ghURL := fmt.Sprintf("%s/download/%s/install.yaml", o.ReleaseURL, o.Version)
	resp, err := getFrom(ctx, ghURL)
	if err != nil {
		return err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download manifests.tar.gz from %s, status: %s", ghURL, resp.Status)
	}

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return fmt.Errorf("failed to write to buffer: %s", err)
	}

	data := buf.String()
	o.Content = &data
	return nil
}

func (o *Controller) writeFile(rootDir string) (string, error) {
	path := filepath.Join(o.Name, "install.yaml")
	err := writeFile(rootDir, path, *o.Content)
	if err != nil {
		return "", err
	}
	return path, nil
}

// GenerateLocalizationFromTemplate generates localization files from a template.
func (o *Controller) GenerateLocalizationFromTemplate(tmpl, loc string) (string, error) {
	// add localization
	tmpl += fmt.Sprintf(loc, o.Name, o.Name)
	return tmpl, nil
}

// GenerateImages returns a map of images from the install.yaml file.
func (o *Controller) GenerateImages() (map[string][]string, error) {
	var images = make(map[string][]string)
	index := strings.Index(*o.Content, fmt.Sprintf("%s/%s", o.Registry, o.Name))
	var image string
	for i := index; i < len(*o.Content); i++ {
		v := string((*o.Content)[i])
		if v == "\n" {
			break
		}
		image += v
	}

	if im := strings.Split(image, ":"); len(im) != 2 {
		image += ":" + o.Version
	} else {
		image = im[0] + ":" + o.Version
	}
	images[image] = []string{
		o.Name,
		o.Version,
	}

	return images, nil
}

// GetPath returns the path to the manifests.
func (o *Controller) GetPath() string {
	return o.Path
}

func (o *Controller) applyCertificatePatch() (err error) {
	patch, ok := patchMap[o.Name]
	if !ok {
		return fmt.Errorf("no patch exists for controller with name: %s", o.Name)
	}

	patch = o.setCertificateSecretName(patch)

	fs := filesys.MakeFsInMemory()
	if err := fs.WriteFile("kustomization.yaml", patch); err != nil {
		return fmt.Errorf("failed to create kustomization file: %w", err)
	}

	if err := fs.WriteFile("install.yaml", []byte(*o.Content)); err != nil {
		return fmt.Errorf("failed to create install file: %w", err)
	}

	kustomizer := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	result, err := kustomizer.Run(fs, ".")
	if err != nil {
		return fmt.Errorf("failed to run kustomize for controller %s: %w", o.Name, err)
	}

	asYaml, err := result.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to create yaml from kustomize result: %w", err)
	}

	o.Content = pointer.String(string(asYaml))

	return nil
}

func (o *Controller) setCertificateSecretName(patch []byte) []byte {
	return bytes.ReplaceAll(patch, []byte(secretPlaceholder), []byte(o.CertificateSecretName))
}
