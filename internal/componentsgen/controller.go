// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentsgen

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/open-component-model/mpas/internal/env"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

//go:embed patch/replication_controller_patch.yaml
var replicationControllerPatch []byte

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
	Content string
}

// GenerateManifests downloads the install.yaml file and writes it to a temporary directory.
// It validates the version and returns an error if the version does not exist.
func (o *Controller) GenerateManifests(ctx context.Context, tmpDir string) error {
	if o.Version == "latest" {
		latest, err := getLatestVersion(ctx, o.ReleaseAPIURL)
		if err != nil {
			return fmt.Errorf("failed to retrieve latest version for %s: %s", o.Name, err)
		}

		o.Version = latest
	}

	if err := validateVersion(ctx, o.Version, o.ReleaseAPIURL, o.Name); err != nil {
		return err
	}

	tmpDir = filepath.Join(tmpDir, o.Name)
	content, err := fetch(ctx, o.ReleaseURL, o.Version, tmpDir, "install.yaml")
	if err != nil {
		return fmt.Errorf("failed to download install.yaml file: %w", err)
	}

	if o.Name == env.ReplicationControllerName {
		if err := o.enableMpasForReplicationController(); err != nil {
			return fmt.Errorf("failed to update replication controller: %w", err)
		}
	}

	o.Path = filepath.Join(o.Name, "install.yaml")
	o.Registry = env.DefaultOCMHost
	o.Content = string(content)
	return nil
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
	index := strings.Index(o.Content, fmt.Sprintf("%s/%s", o.Registry, o.Name))
	var image string
	for i := index; i < len(o.Content); i++ {
		v := string((o.Content)[i])
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

func (o *Controller) enableMpasForReplicationController() (err error) {
	fs := filesys.MakeFsInMemory()
	if err := fs.WriteFile("kustomization.yaml", replicationControllerPatch); err != nil {
		return fmt.Errorf("failed to create kustomization file: %w", err)
	}

	if err := fs.WriteFile("install.yaml", []byte(o.Content)); err != nil {
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

	o.Content = string(asYaml)

	return nil
}
