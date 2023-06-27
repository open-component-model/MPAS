// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
)

const (
	// DefaultBootstrapComponents is the default bootstrap components
	defaultBootstrapComponent = "ocm.software/mpas/bootstrap"
)

func (b *Bootstrap) fetchBootstrapComponentReferences(ociRepo ocm.Repository) (map[string]compdesc.ComponentReference, error) {
	var references map[string]compdesc.ComponentReference
	c, err := ociRepo.LookupComponent(defaultBootstrapComponent)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup component %q: %w", defaultBootstrapComponent, err)
	}
	vnames, err := c.ListVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to list versions of component %q: %w", defaultBootstrapComponent, err)
	}
	vs := make([]*semver.Version, len(vnames))
	for i, vname := range vnames {
		v, err := semver.NewVersion(vname)
		if err != nil {
			return nil, err
		}
		vs[i] = v
	}
	sort.Sort(semver.Collection(vs))
	ver := vs[len(vs)-1]
	cv, err := c.LookupVersion(ver.Original())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup version %q of component %q: %w", ver.String(), defaultBootstrapComponent, err)
	}

	references = make(map[string]compdesc.ComponentReference, len(b.components))
	for _, component := range b.components {
		ref, err := cv.GetReference(metav1.Identity{
			"name": component,
		})
		if err != nil {
			return nil, err
		}
		references[component] = ref
	}

	return references, nil
}

func makeOCIRepository(octx ocm.Context, repositoryURL, dockerconfigPath string) (ocm.Repository, error) {
	if !strings.Contains(repositoryURL, "https://") && !strings.Contains(repositoryURL, "http://") {
		repositoryURL = "https://" + repositoryURL
	}
	regURL, err := url.Parse(repositoryURL)
	if err != nil {
		return nil, err
	}

	spec := dockerconfig.NewRepositorySpec(dockerconfigPath, true)
	// attach the repository to the context, this propagates the consumer ids.
	_, err = octx.CredentialsContext().RepositoryForSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("cannot access docker config: %w", err)
	}

	meta := ocireg.NewComponentRepositoryMeta(strings.TrimPrefix(regURL.Path, "/"), ocireg.OCIRegistryURLPathMapping)
	targetSpec := ocireg.NewRepositorySpec(regURL.Host, meta)
	target, err := octx.RepositoryForSpec(targetSpec)
	if err != nil {
		return nil, err
	}
	return target, nil
}
