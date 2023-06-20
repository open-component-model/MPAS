// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"github.com/containers/image/v5/pkg/compression"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

const (
	defaultFluxHost   = "ghrc.io/fluxcd"
	localizationField = "localization"
	fileField         = "file"
	imageField        = "image"
	resourceField     = "resource/name"
)

var (
	_ Installer = &FluxInstall{}
)

type FluxInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
}

func NewFluxInstall(name, version string, repository ocm.Repository) *FluxInstall {
	return &FluxInstall{
		componentName: name,
		version:       version,
		repository:    repository,
	}
}

func (f *FluxInstall) Install(ctx context.Context) error {
	fmt.Println("Installing Flux...", f.componentName)
	c, err := f.repository.LookupComponent(f.componentName)
	if err != nil {
		return err
	}
	vnames, err := c.ListVersions()
	if err != nil {
		return err
	}
	constraint, err := semver.NewConstraint(f.version)
	if err != nil {
		return err
	}
	var (
		ver   *semver.Version
		valid bool
	)
	for _, vname := range vnames {
		v, err := semver.NewVersion(vname)
		if err != nil {
			return err
		}
		if constraint.Check(v) {
			ver = v
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("no matching version found for constraint %q", f.version)
	}

	cv, err := c.LookupVersion(ver.Original())
	if err != nil {
		return err
	}

	resources := cv.GetResources()
	var (
		fluxResource    []byte
		ocmConfig       []byte
		imagesResources = make(map[string]struct {
			Name string
			Tag  string
		}, 0)
	)
	for _, resource := range resources {
		switch resource.Meta().GetName() {
		case "flux":
			fluxResource, err = getResourceContent(resource)
			if err != nil {
				return err
			}
		case "ocm-config":
			ocmConfig, err = getResourceContent(resource)
			if err != nil {
				return err
			}
		default:
			name, version := getResourceRef(resource)
			imagesResources[resource.Meta().GetName()] = struct {
				Name string
				Tag  string
			}{
				Name: name,
				Tag:  version,
			}
		}
	}

	if fluxResource == nil || ocmConfig == nil {
		return fmt.Errorf("flux or ocm-config resource not found")
	}

	dir, err := os.MkdirTemp("", "flux-install")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "gotk-components.yaml"), fluxResource, os.ModePerm); err != nil {
		return err
	}

	kfile, err := generateKustomizationFile(dir, "gotk-components.yaml")
	if err != nil {
		return err
	}

	data, err := os.ReadFile(kfile)
	if err != nil {
		return err
	}

	kus := kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
	}

	if err := yaml.Unmarshal(data, &kus); err != nil {
		return err
	}

	kconfig, err := unMarshallConfig(ocmConfig)
	if err != nil {
		return err
	}

	for _, loc := range kconfig.Localization {
		image := imagesResources[loc.Resource.Name]
		kus.Images = append(kus.Images, kustypes.Image{
			Name:    fmt.Sprintf("%s/%s", defaultFluxHost, loc.Resource.Name),
			NewName: fmt.Sprintf("%s/%s", defaultFluxHost, image.Name),
			NewTag:  image.Tag,
		})
	}

	manifest, err := yaml.Marshal(kus)
	if err != nil {
		return err
	}

	fmt.Println(string(manifest))

	return nil
}

func (f *FluxInstall) Cleanup(ctx context.Context) error {
	return nil
}

func generateKustomizationFile(path, resource string) (string, error) {
	kfile := filepath.Join(path, konfig.DefaultKustomizationFileName())
	f, err := os.Create(kfile)
	if err != nil {
		return "", err
	}
	f.Close()
	kus := &kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
		Resources: []string{resource},
	}
	kd, err := yaml.Marshal(kus)
	if err != nil {
		os.Remove(kfile)
		return "", err
	}
	return kfile, os.WriteFile(kfile, kd, os.ModePerm)
}

func getResourceContent(resource ocm.ResourceAccess) ([]byte, error) {
	access, err := resource.AccessMethod()
	if err != nil {
		return nil, err
	}

	reader, err := access.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decompressedReader, decompressed, err := compression.AutoDecompress(reader)
	if err != nil {
		return nil, err
	}
	if decompressed {
		defer decompressedReader.Close()
	}
	return io.ReadAll(decompressedReader)
}

func getResourceRef(resource ocm.ResourceAccess) (string, string) {
	name := resource.Meta().Name
	version := resource.Meta().Version
	return name, version
}

func unMarshallConfig(data []byte) (*cfd.ConfigData, error) {
	fmt.Println(string(data))
	k := &cfd.ConfigData{}
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))
	err := decoder.Decode(k)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config data: %w", err)
	}
	return k, nil
}
