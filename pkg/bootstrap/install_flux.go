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

	"github.com/Masterminds/semver/v3"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/containers/image/v5/pkg/compression"
	flux "github.com/fluxcd/flux2/v2/pkg/bootstrap"
	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	syncOpts "github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	"github.com/fluxcd/pkg/kustomize"
	rateoption "github.com/fluxcd/pkg/runtime/client"
	"github.com/open-component-model/mpas/pkg/kubeutils"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/accessmethods/ociartifact"
	"k8s.io/apimachinery/pkg/types"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	_                   Installer = &fluxInstall{}
	defaultKubeAPIQPS             = 50.0
	defaultKubeAPIBurst           = 300
)

type fluxOptions struct {
	gitClient  repository.Client
	kubeClient client.Client

	restClientGetter genericclioptions.RESTClientGetter

	url       string
	branch    string
	target    string
	namespace string
	token     string
	dir       string

	interval time.Duration
	timeout  time.Duration

	signature             git.Signature
	commitMessageAppendix string
	gpgKeyRing            openpgp.EntityList
	gpgPassphrase         string
	gpgKeyID              string
}

// fluxOption is a function that configures a fluxInstall.
type fluxOption func(*fluxOptions)

type fluxInstall struct {
	componentName string
	version       string
	repository    ocm.Repository
	components    []string
	*flux.PlainGitBootstrapper
	fluxOptions

	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

type nameTag struct {
	Name string
	Tag  string
}

func withInterval(interval time.Duration) fluxOption {
	return func(o *fluxOptions) {
		o.interval = interval
	}
}

func withTimeout(timeout time.Duration) fluxOption {
	return func(o *fluxOptions) {
		o.timeout = timeout
	}
}

func withToken(token string) fluxOption {
	return func(o *fluxOptions) {
		o.token = token
	}
}

func withKubeConfig(kubeconfig genericclioptions.RESTClientGetter) fluxOption {
	return func(o *fluxOptions) {
		o.restClientGetter = kubeconfig
	}
}

func withKubeClient(kubeClient client.Client) fluxOption {
	return func(o *fluxOptions) {
		o.kubeClient = kubeClient
	}
}

func withURL(url string) fluxOption {
	return func(o *fluxOptions) {
		o.url = url
	}
}

func withBranch(branch string) fluxOption {
	return func(o *fluxOptions) {
		o.branch = branch
	}
}

func withTarget(target string) fluxOption {
	return func(o *fluxOptions) {
		o.target = target
	}
}

func withNamespace(namespace string) fluxOption {
	return func(o *fluxOptions) {
		o.namespace = namespace
	}
}

func withDir(dir string) fluxOption {
	return func(o *fluxOptions) {
		o.dir = dir
	}
}

func withSignature(signature git.Signature) fluxOption {
	return func(o *fluxOptions) {
		o.signature = signature
	}
}

func withCommitMessageAppendix(commitMessageAppendix string) fluxOption {
	return func(o *fluxOptions) {
		o.commitMessageAppendix = commitMessageAppendix
	}
}

func withGPGKeyRing(gpgKeyRing openpgp.EntityList) fluxOption {
	return func(o *fluxOptions) {
		o.gpgKeyRing = gpgKeyRing
	}
}

func withGPGPassphrase(gpgPassphrase string) fluxOption {
	return func(o *fluxOptions) {
		o.gpgPassphrase = gpgPassphrase
	}
}

func withGPGKeyID(gpgKeyID string) fluxOption {
	return func(o *fluxOptions) {
		o.gpgKeyID = gpgKeyID
	}
}

func NewFluxInstall(name, version, owner string, repository ocm.Repository, opts ...fluxOption) (*fluxInstall, error) {
	f := &fluxInstall{
		componentName: name,
		version:       version,
		repository:    repository,
	}
	for _, o := range opts {
		o(&f.fluxOptions)
	}

	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage(), gogit.WithFallbackToDefaultKnownHosts()}
	gitClient, err := gogit.NewClient(f.dir, &git.AuthOptions{
		Transport: git.HTTPS,
		Username:  owner,
		Password:  f.token,
	}, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create a Git client: %w", err)
	}

	p, err := flux.NewPlainGitProvider(gitClient, f.kubeClient,
		flux.WithBranch(f.branch),
		flux.WithRepositoryURL(f.url),
		flux.WithLogger(log.NopLogger{}),
		flux.WithKubeconfig(f.restClientGetter, &rateoption.Options{QPS: float32(defaultKubeAPIQPS), Burst: defaultKubeAPIBurst}),
	)
	if err != nil {
		return nil, err
	}

	f.gitClient = gitClient
	f.PlainGitBootstrapper = p
	return f, nil
}

func (f *fluxInstall) Install(ctx context.Context) error {
	cv, err := GetComponentVersion(f.repository, f.componentName, f.version)
	if err != nil {
		return fmt.Errorf("failed to get component version: %w", err)
	}

	fluxResource, ocmConfig, imagesResources, comps, err := getResources(cv)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	f.components = comps

	if fluxResource == nil || ocmConfig == nil {
		return fmt.Errorf("flux or ocm-config resource not found")
	}

	kfile, kus, kconfig, err := f.generateKustomization(fluxResource, ocmConfig)
	if err != nil {
		return err
	}

	res, err := f.generateGOTKComponent(kconfig, imagesResources, kus, kfile)
	if err != nil {
		return err
	}

	err = f.reconcileComponents(ctx, fmt.Sprintf("%s/%s/%s", f.target, f.namespace, "gotk-components.yaml"), string(res))
	if err != nil {
		return fmt.Errorf("failed to reconcile components: %w", err)
	}

	secretOpts := sourcesecret.Options{
		Name:         f.namespace,
		Namespace:    f.namespace,
		TargetPath:   f.target,
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
		Username:     "git",
		Password:     f.token,
	}

	if err := f.ReconcileSourceSecret(ctx, secretOpts); err != nil {
		return err
	}

	syncOpts := syncOpts.Options{
		Interval:          f.interval,
		Name:              f.namespace,
		Namespace:         f.namespace,
		URL:               f.url,
		Branch:            f.branch,
		Secret:            secretOpts.Name,
		TargetPath:        f.target,
		ManifestFile:      syncOpts.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: false,
	}

	if err := f.ReconcileSyncConfig(ctx, syncOpts); err != nil {
		return fmt.Errorf("failed to reconcile sync config: %w", err)
	}

	var healthErr error
	if err := f.ReportKustomizationHealth(ctx, syncOpts, f.interval, f.timeout); err != nil {
		healthErr = errors.Join(healthErr, err)
	}

	installOpts := install.Options{
		Namespace:  f.namespace,
		Components: f.components,
		LogLevel:   "info",
	}
	if err := f.ReportComponentsHealth(ctx, installOpts, f.timeout); err != nil {
		healthErr = errors.Join(healthErr, err)
	}
	if healthErr != nil {
		return fmt.Errorf("bootstrap failed with errors: %w", healthErr)
	}

	return nil
}

func (f *fluxInstall) generateGOTKComponent(kconfig *cfd.ConfigData, imagesResources map[string]nameTag, kus kustypes.Kustomization, kfile string) ([]byte, error) {
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
		return nil, err
	}

	err = os.WriteFile(kfile, manifest, os.ModePerm)
	if err != nil {
		return nil, err
	}

	fs := filesys.MakeFsOnDisk()

	f.mu.Lock()
	defer f.mu.Unlock()

	m, err := kustomize.Build(fs, f.dir)
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	res, err := m.AsYaml()
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}
	return res, nil
}

func (f *fluxInstall) generateKustomization(fluxResource []byte, ocmConfig []byte) (string, kustypes.Kustomization, *cfd.ConfigData, error) {
	if err := os.WriteFile(filepath.Join(f.dir, "gotk-components.yaml"), fluxResource, os.ModePerm); err != nil {
		return "", kustypes.Kustomization{}, nil, err
	}

	kfile, err := generateKustomizationFile(f.dir, "./gotk-components.yaml")
	if err != nil {
		return "", kustypes.Kustomization{}, nil, err
	}

	data, err := os.ReadFile(kfile)
	if err != nil {
		return "", kustypes.Kustomization{}, nil, err
	}

	kus := kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
	}

	if err := yaml.Unmarshal(data, &kus); err != nil {
		return "", kustypes.Kustomization{}, nil, err
	}

	kconfig, err := unMarshallConfig(ocmConfig)
	if err != nil {
		return "", kustypes.Kustomization{}, nil, err
	}
	return kfile, kus, kconfig, nil
}

func (f *fluxInstall) Cleanup(ctx context.Context) error {
	return nil
}

func (f *fluxInstall) reconcileComponents(ctx context.Context, path, content string) error {
	err := f.cloneRepository(ctx)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	// Write generated files and make a commit
	err = f.commitAndPushComponents(ctx, path, content)
	if err != nil {
		return fmt.Errorf("failed to commit and push components: %w", err)
	}

	// Conditionally install manifests
	if f.mustInstallManifests(ctx) {
		componentsYAML := filepath.Join(f.gitClient.Path(), path)
		kfile := filepath.Join(filepath.Dir(componentsYAML), konfig.DefaultKustomizationFileName())
		if _, err := os.Stat(kfile); err == nil {
			// Apply the components and their patches
			if _, err := kubeutils.Apply(ctx, f.restClientGetter, f.gitClient.Path(), kfile); err != nil {
				return fmt.Errorf("failed to apply components: %w", err)
			}
		} else {
			// Apply the CRDs and controllers
			if _, err := kubeutils.Apply(ctx, f.restClientGetter, f.gitClient.Path(), componentsYAML); err != nil {
				return fmt.Errorf("failed to apply components: %w", err)
			}
		}
	}
	return nil
}

func (f *fluxInstall) mustInstallManifests(ctx context.Context) bool {
	namespacedName := types.NamespacedName{
		Namespace: f.namespace,
		Name:      f.namespace,
	}
	var k kustomizev1.Kustomization
	if err := f.kubeClient.Get(ctx, namespacedName, &k); err != nil {
		return true
	}
	return k.Status.LastAppliedRevision == ""
}

func (f *fluxInstall) commitAndPushComponents(ctx context.Context, path string, content string) (err error) {
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

func (f *fluxInstall) cloneRepository(ctx context.Context) error {
	if _, err := f.gitClient.Head(); err != nil {
		if err != git.ErrNoGitRepository {
			return err
		}
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
			return nil
		}); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}
	return nil
}

// cleanGitRepoDir cleans the directory meant for the Git repo.
func (f *fluxInstall) cleanGitRepoDir() (err error) {
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

func getResources(cv ocm.ComponentVersionAccess) ([]byte, []byte, map[string]nameTag, []string, error) {
	resources := cv.GetResources()
	var (
		fluxResource    []byte
		ocmConfig       []byte
		imagesResources = make(map[string]nameTag, 0)
		comps           = make([]string, 0)
		err             error
	)
	for _, resource := range resources {
		switch resource.Meta().GetName() {
		case "flux":
			fluxResource, err = getResourceContent(resource)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		case "ocm-config":
			ocmConfig, err = getResourceContent(resource)
			if err != nil {
				return nil, nil, nil, nil, err
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
				comps = append(comps, resource.Meta().GetName())
			}
		}
	}
	return fluxResource, ocmConfig, imagesResources, comps, nil
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
		keyID = strings.TrimPrefix(keyID, "0x")
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
