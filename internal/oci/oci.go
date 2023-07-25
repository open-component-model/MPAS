// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	credentials "github.com/oras-project/oras-credentials-go"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Repository struct {
	RepositoryURL string
	Username      string
	Password      string
	PlainHTTP     bool
}

// PushArtifact pushes the artifact to the given repository.
func (r *Repository) PushArtifact(ctx context.Context, src, version string) error {
	fs, err := file.New("")
	if err != nil {
		return err
	}
	defer fs.Close()
	mediaType := ocispec.MediaTypeImageLayerGzip
	fileDescriptors := make([]ocispec.Descriptor, 0, 1)
	fileDescriptor, err := fs.Add(ctx, src, mediaType, "")
	if err != nil {
		return err
	}
	fileDescriptors = append(fileDescriptors, fileDescriptor)

	artifactType := "ocm.software/bundle"
	manifestDescriptor, err := oras.Pack(ctx, fs, artifactType, fileDescriptors, oras.PackOptions{
		PackImageManifest: true,
	})

	if err != nil {
		return err
	}

	if err = fs.Tag(ctx, manifestDescriptor, version); err != nil {
		return err
	}

	reg, repo, err := repoRef(r.RepositoryURL)
	if err != nil {
		return err
	}

	repo.PlainHTTP = r.PlainHTTP

	creds, err := resolveCredentials(r.Username, r.Password, reg)
	if err != nil {
		return err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.DefaultCache,
		Credential: creds,
	}
	_, err = oras.Copy(ctx, fs, version, repo, version, oras.DefaultCopyOptions)
	return err
}

// PullArtifact pulls the artifact from the given repository.
func (r *Repository) PullArtifact(ctx context.Context, version string) (name string, err error) {
	fs, err := file.New(".")
	if err != nil {
		return "", fmt.Errorf("failed to create file store: %w", err)
	}
	defer fs.Close()
	fs.AllowPathTraversalOnWrite = true

	reg, repo, err := repoRef(r.RepositoryURL)
	if err != nil {
		return "", err
	}

	repo.PlainHTTP = r.PlainHTTP

	creds, err := resolveCredentials(r.Username, r.Password, reg)
	if err != nil {
		return "", err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.DefaultCache,
		Credential: creds,
	}

	copyOptions := oras.DefaultCopyOptions
	copyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		n, ok := desc.Annotations[ocispec.AnnotationTitle]
		if ok {
			name = n
		}
		return nil
	}

	_, err = oras.Copy(ctx, repo, version, fs, version, copyOptions)
	if err != nil {
		return "", err
	}

	return
}

func resolveCredentials(username string, password string, reg string) (func(context.Context, string) (auth.Credential, error), error) {
	var creds func(context.Context, string) (auth.Credential, error)
	if username != "" && password != "" {
		creds = auth.StaticCredential(reg, auth.Credential{
			Username: username,
			Password: password,
		})
	} else {
		store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{
			AllowPlaintextPut: true,
		})
		if err != nil {
			return nil, err
		}
		creds = credentials.Credential(store)
	}
	return creds, nil
}

// GetLatestVersion returns the latest version of the component with the given name.
func (r *Repository) GetLatestVersion(ctx context.Context) (string, error) {
	reg, repo, err := repoRef(r.RepositoryURL)
	if err != nil {
		return "", err
	}

	repo.PlainHTTP = r.PlainHTTP

	creds, err := resolveCredentials(r.Username, r.Password, reg)
	if err != nil {
		return "", err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.DefaultCache,
		Credential: creds,
	}

	var version *semver.Version
	err = repo.Tags(ctx, "", func(tags []string) error {
		vs := make([]*semver.Version, len(tags))
		for i, tag := range tags {
			v, err := semver.NewVersion(tag)
			if err != nil {
				return err
			}
			vs[i] = v
		}
		sort.Sort(semver.Collection(vs))
		version = vs[len(vs)-1]
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	return version.Original(), nil
}

func repoRef(repositoryURL string) (string, *remote.Repository, error) {
	if !strings.Contains(repositoryURL, "https") && !strings.Contains(repositoryURL, "http") {
		repositoryURL = "https://" + repositoryURL
	}
	u, err := url.Parse(repositoryURL)
	if err != nil {
		return "", nil, err
	}

	reg := u.Host
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", reg, strings.TrimPrefix(u.Path, "/")))
	if err != nil {
		return "", nil, err
	}
	return reg, repo, nil
}
