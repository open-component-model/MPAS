//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	"github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/sourcegraph/conc/pool"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	gitv1alphav1 "github.com/open-component-model/git-controller/apis/delivery/v1alpha1"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-controller/api/v1alpha1"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

func newProductFeature(mpasRepoName, mpasNamespace, projectRepoName string) *features.FeatureBuilder {
	// Product deployment
	product := features.New("Reconcile product deployment")

	// Setup
	product = product.
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       mpasRepoName,
			SourceFilepath: "product_deployment_generator.yaml",
			DestFilepath:   "generators/mpas-podinfo-001.yaml",
		})).
		Assess("management repository has been created", assess.CheckRepoExists(mpasRepoName)).
		Assess("management namespace has been created", checkNamespaceReady(mpasNamespace))

	// Pre-flight check
	product = product.
		Assess("project repository has been created", assess.CheckRepoExists(projectRepoName)).
		Assess("check files are created in project repo", assess.CheckRepoFileContent(
			assess.File{
				Repository: projectRepoName,
				Path:       "CODEOWNERS",
				Content:    "@alive.bobb\n@bob.alisson",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "generators/.gitkeep",
				Content:    "",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/.gitkeep",
				Content:    "",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "subscriptions/.gitkeep",
				Content:    "",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "targets/.gitkeep",
				Content:    "",
			},
		))

	product = product.
		Assess("check if product deployment generator exists", assess.ResourceWasCreated(assess.Object{
			Name:      "podinfo",
			Namespace: mpasNamespace,
			Obj:       &prodv1alpha1.ProductDeploymentGenerator{},
		})).
		Assess("check status of product deployment generator", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Helper()
			t.Log("waiting for condition ready on the product deployment generator")
			client, err := cfg.NewClient()
			if err != nil {
				t.Fail()
			}

			productGenerator := &prodv1alpha1.ProductDeploymentGenerator{
				ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(productGenerator, func(object k8s.Object) bool {
				cvObj, ok := object.(*prodv1alpha1.ProductDeploymentGenerator)
				if !ok {
					return false
				}

				return fconditions.IsTrue(cvObj, meta.ReadyCondition)
			}), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		})

	product = product.Assess("wait for product objects to be created", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		t.Log("waiting for snapshot, git sync and product deployment")

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		productGenerator := &prodv1alpha1.ProductDeploymentGenerator{
			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
		}
		snapshotForGeneratorContent := &v1alpha1.Snapshot{
			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
		}
		syncRequest := &gitv1alphav1.Sync{
			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
		}

		objs := []k8s.Object{
			productGenerator, snapshotForGeneratorContent, syncRequest,
		}

		p := pool.New().WithErrors()

		for _, obj := range objs {
			objCopy := obj
			p.Go(func() error {
				return wait.For(conditions.New(client.Resources()).ResourceMatch(objCopy, func(object k8s.Object) bool {
					return true
				}), wait.WithTimeout(time.Minute*2))
			})
		}

		if err := p.Wait(); err != nil {
			t.Fatal(err)
		}

		return ctx
	})

	// Check if repository contains stuff under `products` folder.
	// The stuff under product folder should be the result of the Sync request from the Snapshot to the repository.
	product = product.
		Assess("check if product files has been created", assess.CheckRepoFileContent(
			assess.File{
				Repository: projectRepoName,
				Path:       "products/<pending>",
				Content:    "",
			},
		))

	// Wait for flux to pick it up and apply it to the cluster.

	// Once it's validated and reconciled, check if a `ProductDeployment` is created.
	product = product.Assess("wait for ProductDeployment to exist", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		productDeployment := &prodv1alpha1.ProductDeployment{
			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
		}

		if err := wait.For(conditions.New(client.Resources()).ResourceMatch(productDeployment, func(object k8s.Object) bool {
			return true
		}), wait.WithTimeout(time.Minute*2)); err != nil {
			t.Fatal(err)
		}

		return ctx
	})

	// Wait for ComponentVersion, Loc, Configuration, Flux OCI
	product = product.Assess("wait for ComponentVersion, Localization, Configuration and Flux OCI to exist",
		func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Helper()

			client, err := cfg.NewClient()
			if err != nil {
				t.Fail()
			}

			componentVersion := &v1alpha1.ComponentVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
			}
			localization := &v1alpha1.Localization{
				ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
			}
			configuration := &v1alpha1.Configuration{
				ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
			}
			fluxOci := &v1beta2.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
			}

			objs := []k8s.Object{
				componentVersion, localization, configuration, fluxOci,
			}

			p := pool.New().WithErrors()

			for _, obj := range objs {
				objCopy := obj
				p.Go(func() error {
					return wait.For(conditions.New(client.Resources()).ResourceMatch(objCopy, func(object k8s.Object) bool {
						return true
					}), wait.WithTimeout(time.Minute*2))
				})
			}

			if err := p.Wait(); err != nil {
				t.Fatal(err)
			}

			return ctx
		})

	// Validate podinfo deployment at target?
	product = product.Assess("wait for podinfo deployment to become ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		// check backend, frontend, redis?
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "podinfo-backend", Namespace: cfg.Namespace()},
		}
		// wait for the deployment to finish becoming available
		err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, v1.ConditionTrue), wait.WithTimeout(time.Minute*10))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	})

	return product
}
