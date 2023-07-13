// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocm

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/open-component-model/mpas/pkg/env"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/credentials"
	"github.com/open-component-model/ocm/pkg/contexts/credentials/repositories/dockerconfig"
	"github.com/open-component-model/ocm/pkg/contexts/oci/identity"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc"
	metav1 "github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/meta/v1"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ctf"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
)

// FetchLatestComponent fetches the latest version of the component with the given name.
// It returns the component version access and an error if the component cannot be fetched.
func FetchLatestComponent(repo ocm.Repository, name string) (ocm.ComponentVersionAccess, error) {
	c, err := repo.LookupComponent(name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup component %q: %w", name, err)
	}
	ver, err := fetchLatestComponentVersion(c, name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version of component %q: %w", name, err)
	}
	cv, err := c.LookupVersion(ver.Original())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup version %q of component %q: %w", ver.String(), env.DefaultBootstrapComponent, err)
	}

	return cv, nil
}

func fetchLatestComponentVersion(c ocm.ComponentAccess, name string) (*semver.Version, error) {
	vnames, err := c.ListVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to list versions of component %q: %w", name, err)
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
	return ver, nil
}

// FetchComponentReferences fetches the component references from the given component version.
func FetchComponentReferences(cv ocm.ComponentVersionAccess, components []string) (map[string]compdesc.ComponentReference, error) {
	references := make(map[string]compdesc.ComponentReference, len(components))
	for _, component := range components {
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

func RepositoryFromCTF(path string) (ocm.Repository, error) {
	octx := ocm.DefaultContext()
	repo, err := ctf.Open(octx, accessobj.ACC_READONLY, path, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open component archive %q: %w", path, err)
	}
	return repo, nil
}

// MakeRepositoryWithDockerConfig creates a repository, and use tge given dockerconfigPath
// to configure the credentials.
func MakeRepositoryWithDockerConfig(repositoryURL, dockerconfigPath string) (ocm.Repository, error) {
	octx := ocm.DefaultContext()

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

	return makeOCIRepository(octx, regURL.Host, regURL.Path)
}

// MakeOCIRepository creates a repository for the given repositoryURL.
func MakeOCIRepository(octx ocm.Context, repositoryURL string) (ocm.Repository, error) {
	regURL, err := parseURL(repositoryURL)
	if err != nil {
		return nil, err
	}

	return makeOCIRepository(octx, regURL.Host, regURL.Path)
}

func makeOCIRepository(octx ocm.Context, host, path string) (ocm.Repository, error) {
	meta := ocireg.NewComponentRepositoryMeta(strings.TrimPrefix(path, "/"), ocireg.OCIRegistryURLPathMapping)
	targetSpec := ocireg.NewRepositorySpec(host, meta)
	target, err := octx.RepositoryForSpec(targetSpec)
	if err != nil {
		return nil, err
	}
	return target, nil
}

func (c *Component) configureCredentials() error {
	regURL, err := parseURL(c.repositoryURL)
	if err != nil {
		return err
	}

	consumerID := credentials.NewConsumerIdentity(identity.CONSUMER_TYPE,
		identity.ID_HOSTNAME, regURL.Host,
		identity.ID_PATHPREFIX, c.username,
	)

	creds := credentials.DirectCredentials{
		credentials.ATTR_USERNAME:       c.username,
		credentials.ATTR_IDENTITY_TOKEN: c.token,
	}

	c.Context.CredentialsContext().SetCredentialsForConsumer(consumerID, creds)
	return nil
}

// parseURL parses a url and adds the scheme if missing.
// It returns an error if the url is invalid.
func parseURL(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url %s: %w", target, err)
	}
	if u.Host == "" {
		target = fmt.Sprintf("https://%s", target)
		u, err = url.Parse(target)
		if err != nil {
			return nil, fmt.Errorf("failed to parse url %s: %w", target, err)
		}
	}
	return u, nil
}
