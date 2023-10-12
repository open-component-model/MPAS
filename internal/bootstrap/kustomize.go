// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fluxcd/pkg/kustomize"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

type kustomizerOptions struct {
	dir           string
	repository    ocm.Repository
	componentName string
	version       string
	host          string
}

type Kustomizer struct {
	*kustomizerOptions

	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

// NewKustomizer creates a new kustomizer based on mutation options.
func NewKustomizer(opts *kustomizerOptions) *Kustomizer {
	return &Kustomizer{
		kustomizerOptions: opts,
	}
}

func (k *Kustomizer) generateKustomizedResourceData(component string) ([]byte, error) {
	cv, err := getComponentVersion(k.repository, k.componentName, k.version)
	if err != nil {
		return nil, fmt.Errorf("failed to get component version: %w", err)
	}

	resources, err := getResources(cv, component)
	if err != nil {
		return nil, fmt.Errorf("failed to get resources: %w", err)
	}

	if resources.componentResource == nil || resources.ocmConfig == nil {
		return nil, fmt.Errorf("failed to get component resource or ocm config")
	}

	kfile, kus, err := k.generateKustomization(resources.componentResource)
	if err != nil {
		return nil, fmt.Errorf("failed to generate kustomization: %w", err)
	}

	kconfig, err := unMarshallConfig(resources.ocmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall config: %w", err)
	}

	return k.generateComponentYaml(kconfig, resources.imagesResources, kus, kfile)
}

func (k *Kustomizer) generateKustomization(componentResource []byte) (string, kustypes.Kustomization, error) {
	if err := os.WriteFile(filepath.Join(k.dir, fmt.Sprintf("%s.yaml", strings.Split(k.componentName, "/")[2])), componentResource, os.ModePerm); err != nil {
		return "", kustypes.Kustomization{}, err
	}

	return genKus(k.dir, fmt.Sprintf("./%s.yaml", strings.Split(k.componentName, "/")[2]))
}

func (k *Kustomizer) generateComponentYaml(kconfig *cfd.ConfigData, imagesResources map[string]nameTag, kus kustypes.Kustomization, kfile string) ([]byte, error) {
	for _, loc := range kconfig.Localization {
		image := imagesResources[loc.Resource.Name]
		kus.Images = append(kus.Images, kustypes.Image{
			Name:    fmt.Sprintf("%s/%s", k.host, loc.Resource.Name),
			NewName: image.Name,
			NewTag:  image.Tag,
		})
	}

	return buildKustomization(kus, kfile, k.dir, &k.mu)
}

func buildKustomization(kus kustypes.Kustomization, kfile, dir string, mu sync.Locker) ([]byte, error) {
	manifest, err := yaml.Marshal(kus)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(kfile, manifest, os.ModePerm)
	if err != nil {
		return nil, err
	}

	fs := filesys.MakeFsOnDisk()

	mu.Lock()
	defer mu.Unlock()

	m, err := kustomize.Build(fs, dir)
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	res, err := m.AsYaml()
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}
	return res, nil
}
