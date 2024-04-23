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

	flux "github.com/fluxcd/flux2/v2/pkg/bootstrap"
	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	syncOpts "github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	rateoption "github.com/fluxcd/pkg/runtime/client"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/mpas/internal/kubeutils"
	cfd "github.com/open-component-model/ocm-controller/pkg/configdata"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type fluxOptions struct {
	gitClient             repository.Client
	kubeClient            client.Client
	restClientGetter      genericclioptions.RESTClientGetter
	url                   string
	testURL               string
	transport             string
	branch                string
	targetPath            string
	namespace             string
	token                 string
	dir                   string
	commitMessageAppendix string
	interval              time.Duration
	timeout               time.Duration
	caFile                []byte
}

type fluxInstall struct {
	componentName    string
	version          string
	repository       ocm.Repository
	components       []string
	fluxBootstrapper *flux.PlainGitBootstrapper
	*fluxOptions
	// mu is used to synchronize access to the kustomization file
	mu sync.Mutex
}

func newFluxInstall(name, version, owner string, repository ocm.Repository, opts *fluxOptions) (*fluxInstall, error) {
	f := &fluxInstall{
		componentName: name,
		version:       version,
		repository:    repository,
		fluxOptions:   opts,
	}

	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage(), gogit.WithFallbackToDefaultKnownHosts()}
	gitOptions := &git.AuthOptions{Transport: git.HTTPS, Username: owner, Password: f.token}
	if f.transport == "http" {
		clientOpts = append(clientOpts, gogit.WithInsecureCredentialsOverHTTP())
		gitOptions.Transport = git.HTTP
	}
	gitClient, err := gogit.NewClient(f.dir, gitOptions, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create a Git client: %w", err)
	}

	p, err := flux.NewPlainGitProvider(gitClient, f.kubeClient,
		flux.WithBranch(f.branch),
		flux.WithRepositoryURL(f.url),
		flux.WithLogger(log.NopLogger{}),
		flux.WithKubeconfig(f.restClientGetter, &rateoption.Options{QPS: env.DefaultKubeAPIQPS, Burst: env.DefaultKubeAPIBurst}),
	)
	if err != nil {
		return nil, err
	}

	f.gitClient = gitClient
	f.fluxBootstrapper = p
	return f, nil
}

func (f *fluxInstall) Install(ctx context.Context, component string) error {
	cv, err := getComponentVersion(f.repository, f.componentName, f.version)
	if err != nil {
		return fmt.Errorf("failed to get component version: %w", err)
	}

	resources, err := getResources(cv, component)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	f.components = resources.componentList

	if resources.componentResource == nil || resources.ocmConfig == nil {
		return fmt.Errorf("flux or ocm-config resource not found")
	}

	kfile, kus, err := f.generateKustomization(resources.componentResource)
	if err != nil {
		return err
	}

	kconfig, err := unMarshallConfig(resources.ocmConfig)
	if err != nil {
		return err
	}

	res, err := f.generateGOTKComponent(kconfig, resources.imagesResources, kus, kfile)
	if err != nil {
		return err
	}

	err = f.reconcileComponents(ctx, fmt.Sprintf("%s/%s/%s", f.targetPath, f.namespace, "gotk-components.yaml"), string(res))
	if err != nil {
		return fmt.Errorf("failed to reconcile components: %w", err)
	}

	secretOpts := sourcesecret.Options{
		Name:         f.namespace,
		Namespace:    f.namespace,
		TargetPath:   f.targetPath,
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
		Username:     "git",
		Password:     f.token,
		CAFile:       f.caFile,
	}

	if err := f.fluxBootstrapper.ReconcileSourceSecret(ctx, secretOpts); err != nil {
		return err
	}

	syncOpts := syncOpts.Options{
		Interval:          f.interval,
		Name:              f.namespace,
		Namespace:         f.namespace,
		URL:               f.url,
		Branch:            f.branch,
		Secret:            secretOpts.Name,
		TargetPath:        f.targetPath,
		ManifestFile:      syncOpts.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: false,
	}

	if f.testURL != "" {
		syncOpts.URL = f.testURL
	}

	if err := f.fluxBootstrapper.ReconcileSyncConfig(ctx, syncOpts); err != nil {
		return fmt.Errorf("failed to reconcile sync config: %w", err)
	}

	var healthErr error
	if err := f.fluxBootstrapper.ReportKustomizationHealth(ctx, syncOpts, env.DefaultPollInterval, f.timeout); err != nil {
		healthErr = errors.Join(healthErr, err)
	}

	installOpts := install.Options{
		Namespace:  f.namespace,
		Components: f.components,
	}
	if err := f.fluxBootstrapper.ReportComponentsHealth(ctx, installOpts, f.timeout); err != nil {
		healthErr = errors.Join(healthErr, err)
	}
	if healthErr != nil {
		return fmt.Errorf("failed to report health, please try again later: %w", healthErr)
	}

	return nil
}

func (f *fluxInstall) generateGOTKComponent(kconfig *cfd.ConfigData, imagesResources map[string]nameTag, kus kustypes.Kustomization, kfile string) ([]byte, error) {
	for _, loc := range kconfig.Localization {
		image := imagesResources[loc.Resource.Name]
		kus.Images = append(kus.Images, kustypes.Image{
			Name:    fmt.Sprintf("%s/%s", env.DefaultFluxHost, loc.Resource.Name),
			NewName: image.Name,
			NewTag:  image.Tag,
		})
	}

	return buildKustomization(kus, kfile, f.dir, &f.mu)
}

func (f *fluxInstall) generateKustomization(fluxResource []byte) (string, kustypes.Kustomization, error) {
	if err := os.WriteFile(filepath.Join(f.dir, "gotk-components.yaml"), fluxResource, os.ModePerm); err != nil {
		return "", kustypes.Kustomization{}, err
	}

	return genKus(f.dir, "./gotk-components.yaml")
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
	return kubeutils.MustInstallKustomization(ctx, f.kubeClient, f.namespace, f.namespace)
}

func (f *fluxInstall) commitAndPushComponents(ctx context.Context, path string, content string) (err error) {
	commitMsg := fmt.Sprintf("Add Flux %s component manifests", f.version)
	if f.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + f.commitMessageAppendix
	}

	_, err = f.gitClient.Commit(git.Commit{
		Author:  git.Signature{Name: "Flux"},
		Message: commitMsg,
	}, repository.WithFiles(map[string]io.Reader{
		path: strings.NewReader(content),
	}))
	if err != nil && !errors.Is(err, git.ErrNoStagedFiles) {
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
		if !errors.Is(err, git.ErrNoGitRepository) {
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

func genKus(dir string, resourceName string) (string, kustypes.Kustomization, error) {
	kfile, err := generateKustomizationFile(dir, resourceName)
	if err != nil {
		return "", kustypes.Kustomization{}, err
	}

	data, err := os.ReadFile(kfile)
	if err != nil {
		return "", kustypes.Kustomization{}, err
	}

	kus := kustypes.Kustomization{
		TypeMeta: kustypes.TypeMeta{
			APIVersion: kustypes.KustomizationVersion,
			Kind:       kustypes.KustomizationKind,
		},
	}

	if err := yaml.Unmarshal(data, &kus); err != nil {
		return "", kustypes.Kustomization{}, err
	}

	return kfile, kus, nil
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

func unMarshallConfig(data []byte) (*cfd.ConfigData, error) {
	k := &cfd.ConfigData{}
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))
	err := decoder.Decode(k)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config data: %w", err)
	}
	return k, nil
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
