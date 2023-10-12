package bootstrap

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
)

// getComponentVersion returns the component version matching the given version constraint.
func getComponentVersion(repository ocm.Repository, componentName, version string) (ocm.ComponentVersionAccess, error) {
	c, err := repository.LookupComponent(componentName)
	if err != nil {
		return nil, err
	}
	vnames, err := c.ListVersions()
	if err != nil {
		return nil, err
	}
	constraint, err := semver.NewConstraint(version)
	if err != nil {
		return nil, err
	}
	var ver *semver.Version
	for _, vname := range vnames {
		v, err := semver.NewVersion(vname)
		if err != nil {
			return nil, err
		}
		if constraint.Check(v) {
			ver = v
			break
		}
	}

	if ver == nil {
		return nil, errors.New("no matching version found")
	}

	cv, err := c.LookupVersion(ver.Original())
	if err != nil {
		return nil, err
	}
	return cv, nil
}

// resources contains the resources extracted from the component version
type resources struct {
	componentResource []byte
	ocmConfig         []byte
	imagesResources   map[string]nameTag
	componentList     []string
}

type nameTag struct {
	Name string
	Tag  string
}

func getResources(cv ocm.ComponentVersionAccess, componentName string) (resources, error) {
	res := cv.GetResources()
	var (
		componentResource []byte
		ocmConfig         []byte
		imagesResources   = make(map[string]nameTag, 0)
		comps             = make([]string, 0)
		err               error
	)
	for _, resource := range res {
		switch resource.Meta().GetName() {
		case componentName:
			componentResource, err = getResourceContent(resource)
			if err != nil {
				return resources{}, err
			}
		case "ocm-config":
			ocmConfig, err = getResourceContent(resource)
			if err != nil {
				return resources{}, err
			}
		default:
			if resource.Meta().GetType() == "ociImage" {
				name, version, err := getResourceRef(resource)
				if err != nil {
					return resources{}, fmt.Errorf("failed to get resource reference: %w", err)
				}
				imagesResources[resource.Meta().GetName()] = struct {
					Name string
					Tag  string
				}{
					Name: name,
					Tag:  version,
				}
				comps = append(comps, resource.Meta().GetName())
			}
		}
	}
	return resources{
		componentResource: componentResource,
		ocmConfig:         ocmConfig,
		imagesResources:   imagesResources,
		componentList:     comps,
	}, nil
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

	decompressedReader, _, err := compression.AutoDecompress(reader)
	if err != nil {
		return nil, err
	}
	defer decompressedReader.Close()

	return io.ReadAll(decompressedReader)
}

func getResourceRef(resource ocm.ResourceAccess) (string, string, error) {
	a, err := resource.Access()
	if err != nil {
		return "", "", err
	}
	spec, ok := a.(*ociartifact.AccessSpec)
	if !ok {
		return "", "", fmt.Errorf("access spec was of type %+v; expected ociartifact", a)
	}

	im := spec.ImageReference
	split := strings.Split(im, ":")
	if len(split) != 2 {
		return "", "", fmt.Errorf("expected image format of domain:verion but was: %s", im)
	}

	name, version := split[0], split[1]
	return name, version, nil
}
