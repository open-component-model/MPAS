//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"

	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	gitv1alphav1 "github.com/open-component-model/git-controller/apis/delivery/v1alpha1"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-controller/api/v1alpha1"
	"strings"
	"testing"
	"time"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
	"github.com/sourcegraph/conc/pool"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func newProductFeature(projectRepoName string) *features.FeatureBuilder {
	prodDepGenName := getYAMLField("product_deployment_generator.yaml", "metadata.name")

	// Product deployment
	return features.New("3 Reconcile product deployment").
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       projectRepoName,
			SourceFilepath: "product_description.yaml",
			DestFilepath:   "products/product_description.yaml",
		})).
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       projectRepoName,
			SourceFilepath: "product_deployment_generator.yaml",
			DestFilepath:   "generators/product_deployment_generator.yaml",
		})).
		Assess("3.2 check status of product deployment generator", checkIfProductDeploymentGeneratorReady(prodDepGenName, projectRepoName)).
		//do for description
		//Assess("3.1 check if product deployment generator exists", assess.ResourceWasCreated(assess.Object{
		//	Name:      prodDepGenName,
		//	Namespace: mpasNamespace,
		//	Obj:       &prodv1alpha1.ProductDeploymentGenerator{},
		//})).
		Assess("3.3 wait for product objects to be Ready", checkSnapshotsSyncExist(prodDepGenName, projectRepoName)).

		//
		//// Check if repository contains stuff under `products` folder.
		//// The stuff under product folder should be the result of the Sync request from the Snapshot to the repository.
		//

		Assess("3.4 Check if Pull Request was created for product files", assess.CheckIfPullRequestExists(projectRepoName, 1)).
		Assess("3.5 Merge PR", setup.MergePullRequest(projectRepoName, 1)).
		Assess("3.6 check if product files has been created", assess.CheckFileInRepoExists(
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/README.md",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/kustomization.yaml",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/product-deployment.yaml",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/values.yaml",
			},
		))

	//
	//// Wait for flux to pick it up and apply it to the cluster.
	//
	//// Once it's validated and reconciled, check if a `ProductDeployment` is created.
	//Assess("wait for ProductDeployment to exist", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	//	t.Helper()
	//
	//	client, err := cfg.NewClient()
	//	if err != nil {
	//		t.Fail()
	//	}
	//
	//	productDeployment := &prodv1alpha1.ProductDeployment{
	//		ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
	//	}
	//
	//	if err := wait.For(conditions.New(client.Resources()).ResourceMatch(productDeployment, func(object k8s.Object) bool {
	//		return true
	//	}), wait.WithTimeout(time.Minute*2)); err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	return ctx
	//}).
	//
	//// Wait for ComponentVersion, Loc, Configuration, Flux OCI
	//Assess("wait for ComponentVersion, Localization, Configuration and Flux OCI to exist",
	//	func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	//		t.Helper()
	//
	//		client, err := cfg.NewClient()
	//		if err != nil {
	//			t.Fail()
	//		}
	//
	//		componentVersion := &v1alpha1.ComponentVersion{
	//			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
	//		}
	//		localization := &v1alpha1.Localization{
	//			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
	//		}
	//		configuration := &v1alpha1.Configuration{
	//			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
	//		}
	//		fluxOci := &v1beta2.OCIRepository{
	//			ObjectMeta: metav1.ObjectMeta{Name: "podinfo", Namespace: mpasNamespace},
	//		}
	//
	//		objs := []k8s.Object{
	//			componentVersion, localization, configuration, fluxOci,
	//		}
	//
	//		p := pool.New().WithErrors()
	//
	//		for _, obj := range objs {
	//			objCopy := obj
	//			p.Go(func() error {
	//				return wait.For(conditions.New(client.Resources()).ResourceMatch(objCopy, func(object k8s.Object) bool {
	//					return true
	//				}), wait.WithTimeout(time.Minute*2))
	//			})
	//		}
	//
	//		if err := p.Wait(); err != nil {
	//			t.Fatal(err)
	//		}
	//
	//		return ctx
	//	}).
	//
	//// Validate podinfo deployment at target?
	//Assess("wait for podinfo deployment to become ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	//	client, err := cfg.NewClient()
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	// check backend, frontend, redis?
	//	dep := appsv1.Deployment{
	//		ObjectMeta: metav1.ObjectMeta{Name: "podinfo-backend", Namespace: cfg.Namespace()},
	//	}
	//	// wait for the deployment to finish becoming available
	//	err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, v1.ConditionTrue), wait.WithTimeout(time.Minute*10))
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	return ctx
	//})
}
func checkIfProductDeploymentGeneratorReady(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		t.Log("waiting for condition ready on the product deployment generator")
		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}
		productGenerator := &prodv1alpha1.ProductDeploymentGenerator{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}

		err = wait.For(conditions.New(client.Resources()).ResourceMatch(productGenerator, func(object k8s.Object) bool {
			obj, ok := object.(*prodv1alpha1.ProductDeploymentGenerator)
			if !ok {
				return false
			}

			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		}), wait.WithTimeout(time.Minute*2))
		if err != nil {
			t.Fatal(err)
		}

		return ctx
	}
}

func checkSnapshotsSyncExist(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		t.Log("waiting for snapshot, git sync")

		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		snapshotForGeneratorContent := &v1alpha1.Snapshot{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		}
		syncRequest := &gitv1alphav1.Sync{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		}

		objs := []k8s.Object{
			snapshotForGeneratorContent, syncRequest,
		}

		snapshotList := &v1alpha1.SnapshotList{}
		syncList := &gitv1alphav1.SyncList{}

		err = client.Resources().List(ctx, snapshotList)
		if err != nil {
			t.Fatal(err)
		}
		for _, snapshot := range snapshotList.Items {
			if snapshot.ObjectMeta.Namespace == namespace && strings.Contains(snapshot.ObjectMeta.Name, name) {
				snapshotForGeneratorContent.ObjectMeta.Name = snapshot.Name
			}
		}
		if len(snapshotForGeneratorContent.ObjectMeta.Name) == 0 {
			t.Fatal("Snapshot for Generator Content not Found")
		}

		err = client.Resources().List(ctx, syncList)
		if err != nil {
			t.Fatal(err)
		}
		for _, sync := range syncList.Items {
			if sync.ObjectMeta.Namespace == namespace && strings.Contains(sync.ObjectMeta.Name, name+"-sync") {
				syncRequest.ObjectMeta.Name = sync.Name
			}
		}

		if len(syncRequest.ObjectMeta.Name) == 0 {
			t.Fatal("Sync for Product not Found")
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
	}
}
func reasonMatches(from fconditions.Getter, reason string) bool {
	conditions_ := from.GetConditions()
	for _, condition := range conditions_ {
		if condition.Reason == reason {
			return true
		}
	}
	return false
}