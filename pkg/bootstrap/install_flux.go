// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	syncOpts "github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"

	"github.com/Masterminds/semver/v3"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/containers/image/v5/pkg/compression"
	flux "github.com/fluxcd/flux2/v2/pkg/bootstrap"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	"github.com/fluxcd/pkg/kustomize"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
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

	gitClient repository.Client

	url    string
	branch string
	target string

	dir string

	signature             git.Signature
	commitMessageAppendix string
	gpgKeyRing            openpgp.EntityList
	gpgPassphrase         string
	gpgKeyID              string

	*flux.PlainGitBootstrapper

	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

func NewFluxInstall(name, version string, repository ocm.Repository, url, branch, owner, token, dir, target string) (*FluxInstall, error) {
	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage(), gogit.WithFallbackToDefaultKnownHosts()}
	gitClient, err := gogit.NewClient(dir, &git.AuthOptions{
		Transport: git.HTTPS,
		Username:  owner,
		Password:  token,
	}, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create a Git client: %w", err)
	}
	p, err := flux.NewPlainGitProvider(gitClient, nil,
		flux.WithBranch(branch),
		flux.WithRepositoryURL(url))
	if err != nil {
		return nil, err
	}
	f := &FluxInstall{
		componentName:        name,
		version:              version,
		repository:           repository,
		PlainGitBootstrapper: p,
		dir:                  dir,
		gitClient:            gitClient,
		url:                  url,
		branch:               branch,
		target:               target,
	}
	return f, nil
}

func (f *FluxInstall) Install(ctx context.Context) error {
	fmt.Println("Installing Flux...", f.componentName)
	cv, err := GetComponentVersion(f.repository, f.componentName, f.version)
	if err != nil {
		return fmt.Errorf("failed to get component version: %w", err)
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
			if resource.Meta().GetType() == "ociImage" {
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
	}

	if fluxResource == nil || ocmConfig == nil {
		return fmt.Errorf("flux or ocm-config resource not found")
	}

	if err := os.WriteFile(filepath.Join(f.dir, "gotk-components.yaml"), fluxResource, os.ModePerm); err != nil {
		return err
	}

	kfile, err := generateKustomizationFile(f.dir, "./gotk-components.yaml")
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
			NewName: image.Name,
			NewTag:  image.Tag,
		})
	}

	manifest, err := yaml.Marshal(kus)
	if err != nil {
		return err
	}

	err = os.WriteFile(kfile, manifest, os.ModePerm)
	if err != nil {
		return err
	}

	fs := filesys.MakeFsOnDisk()

	// acuire the lock
	f.mu.Lock()
	defer f.mu.Unlock()

	m, err := kustomize.Build(fs, f.dir)
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	res, err := m.AsYaml()
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	err = f.ReconcileComponents(ctx, f.target+"/flux-system/gotk-components.yaml", string(res))
	if err != nil {
		return fmt.Errorf("failed to reconcile components: %w", err)
	}

	syncOpts := syncOpts.Options{
		Interval:          5 * time.Minute,
		Name:              "flux-system",
		Namespace:         "flux-system",
		Branch:            f.branch,
		Secret:            "test-secret",
		TargetPath:        f.target + "/flux-system/gotk-sync.yaml",
		ManifestFile:      syncOpts.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: false,
	}
	if err := f.ReconcileSyncConfig(ctx, syncOpts); err != nil {
		return fmt.Errorf("failed to reconcile sync config: %w", err)
	}

	return nil
}

// GetComponentVersion returns the component version matching the given version constraint.
func GetComponentVersion(repository ocm.Repository, componentName, version string) (ocm.ComponentVersionAccess, error) {
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

func (f *FluxInstall) Cleanup(ctx context.Context) error {
	return nil
}

func (f *FluxInstall) ReconcileComponents(ctx context.Context, path, content string) error {
	cloned, err := f.cloneRepository(ctx)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	if cloned {
		fmt.Println("Repository cloned")
	}
	// Write generated files and make a commit
	err = f.commitAndPushComponents(path, content, ctx)
	if err != nil {
		return fmt.Errorf("failed to commit and push components: %w", err)
	}
	return nil
}

func (f *FluxInstall) commitAndPushComponents(path string, content string, ctx context.Context) (err error) {
	var signer *openpgp.Entity
	if f.gpgKeyRing != nil {
		signer, err = getOpenPgpEntity(f.gpgKeyRing, f.gpgPassphrase, f.gpgKeyID)
		if err != nil {
			return fmt.Errorf("failed to generate OpenPGP entity: %w", err)
		}
	}
	commitMsg := fmt.Sprintf("Add Flux %s component manifests", f.version)
	if f.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + f.commitMessageAppendix
	}

	_, err = f.gitClient.Commit(git.Commit{
		Author:  f.signature,
		Message: commitMsg,
	}, repository.WithFiles(map[string]io.Reader{
		path: strings.NewReader(content),
	}), repository.WithSigner(signer))
	if err != nil && err != git.ErrNoStagedFiles {
		return fmt.Errorf("failed to commit sync manifests: %w", err)
	}

	if err == nil {
		if err = f.gitClient.Push(ctx, repository.PushConfig{}); err != nil {
			return fmt.Errorf("failed to push manifests: %w", err)
		}
	}
	return nil
}

func (f *FluxInstall) cloneRepository(ctx context.Context) (bool, error) {
	if _, err := f.gitClient.Head(); err != nil {
		if err != git.ErrNoGitRepository {
			return true, err
		}
		var cloned bool
		if err = retry(1, 2*time.Second, func() error {
			if err := f.cleanGitRepoDir(); err != nil {
				return fmt.Errorf("failed to clean git repository directory: %w", err)
			}
			_, err = f.gitClient.Clone(ctx, f.url, repository.CloneConfig{
				CheckoutStrategy: repository.CheckoutStrategy{
					Branch: f.branch,
				},
			})
			if err != nil {
				return err
			}
			if err == nil {
				cloned = true
			}
			return nil
		}); err != nil {
			return false, fmt.Errorf("failed to clone repository: %w", err)
		}
		if cloned {
			return true, nil
		}
	}
	return false, nil
}

// cleanGitRepoDir cleans the directory meant for the Git repo.
func (f *FluxInstall) cleanGitRepoDir() (err error) {
	dirs, er := os.ReadDir(f.gitClient.Path())
	if er != nil {
		return er
	}

	for _, dir := range dirs {
		if er := os.RemoveAll(filepath.Join(f.gitClient.Path(), dir.Name())); er != nil {
			err = errors.Join(err, er)
		}
	}
	return
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
	a, err := resource.Access()
	if err != nil {
		return "", ""
	}
	spec := a.(*ociartifact.AccessSpec)
	im := spec.ImageReference
	name, version := strings.Split(im, ":")[0], strings.Split(im, ":")[1]
	return name, version
}

func unMarshallConfig(data []byte) (*cfd.ConfigData, error) {
	k := &cfd.ConfigData{}
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))
	err := decoder.Decode(k)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config data: %w", err)
	}
	return k, nil
}

func getOpenPgpEntity(keyRing openpgp.EntityList, passphrase, keyID string) (*openpgp.Entity, error) {
	if len(keyRing) == 0 {
		return nil, fmt.Errorf("empty GPG key ring")
	}

	var entity *openpgp.Entity
	if keyID != "" {
		if strings.HasPrefix(keyID, "0x") {
			keyID = strings.TrimPrefix(keyID, "0x")
		}
		if len(keyID) != 16 {
			return nil, fmt.Errorf("invalid GPG key id length; expected %d, got %d", 16, len(keyID))
		}
		keyID = strings.ToUpper(keyID)

		for _, ent := range keyRing {
			if ent.PrimaryKey.KeyIdString() == keyID {
				entity = ent
			}
		}

		if entity == nil {
			return nil, fmt.Errorf("no GPG keyring matching key id '%s' found", keyID)
		}
		if entity.PrivateKey == nil {
			return nil, fmt.Errorf("keyring does not contain private key for key id '%s'", keyID)
		}
	} else {
		entity = keyRing[0]
	}

	err := entity.PrivateKey.Decrypt([]byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt GPG private key: %w", err)
	}

	return entity, nil
}

func retry(retries int, wait time.Duration, fn func() error) (err error) {
	for i := 0; ; i++ {
		err = fn()
		if err == nil {
			return
		}
		if i >= retries {
			break
		}
		time.Sleep(wait)
	}
	return err
}
