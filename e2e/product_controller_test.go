//go:build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/sourcegraph/conc/pool"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
)

func newProductFeature(projectRepoName string) *features.FeatureBuilder {
	prodDepGenName := getYAMLField("product_deployment_generator.yaml", "metadata.name")
	targetName := getYAMLField("target.yaml", "metadata.name")
	targetNamespace := getYAMLField("target.yaml", "spec.access.targetNamespace")
	componentSubscriptionName := getYAMLField("subscription.yaml", "metadata.name")
	pipelineNames := []string{
		getYAMLField("product_description.yaml", "spec.pipelines[0].name"),
		getYAMLField("product_description.yaml", "spec.pipelines[1].name"),
		getYAMLField("product_description.yaml", "spec.pipelines[2].name")}

	return features.New("Reconcile Product Deployment").
		WithSetup("Create ocm-registry-tls-certs for target namespace", replicateRegistryCerts(getYAMLField("target.yaml", "spec.access.targetNamespace"))).
		WithSetup("Add Target to project git repository", setup.AddFilesToGitRepository(
			setup.File{
				RepoName:       projectRepoName,
				SourceFilepath: "target.yaml",
				DestFilepath:   "targets/ingress-target.yaml",
			})).
		WithSetup("Add Subscription to project git repository", setup.AddFilesToGitRepository(
			setup.File{
				RepoName:       projectRepoName,
				SourceFilepath: "subscription.yaml",
				DestFilepath:   "subscriptions/podinfo.yaml",
			})).
		WithSetup("Add Product Deployment Generator to project git repository", setup.AddFilesToGitRepository(setup.File{
			RepoName:       projectRepoName,
			SourceFilepath: "product_deployment_generator.yaml",
			DestFilepath:   "generators/product_deployment_generator.yaml",
		})).
		Assess(fmt.Sprintf("Target %s has been created in namespace %s", targetName, projectRepoName), checkIfTargetExists(targetName, projectRepoName)).
		Assess(fmt.Sprintf("ComponentSubscription %s has been created in namespace %s", componentSubscriptionName, projectRepoName), checkIfComponentSubscriptionExists(componentSubscriptionName,
			projectRepoName)).
		Assess(fmt.Sprintf("ProductDeploymentGenerator %s has been created in namespace %s", projectRepoName, projectRepoName), checkIfProductDeploymentGeneratorReady(prodDepGenName, projectRepoName)).
		Assess(fmt.Sprintf("Snapshot, Sync %s have been created in namespace %s", prodDepGenName, projectRepoName), checkSnapshotsSyncExist(prodDepGenName, projectRepoName)).
		Assess("PR was created for product files in project repository", assess.CheckIfPullRequestExists(projectRepoName, 1)).
		Assess("Merge PR in project repository", setup.MergePullRequest(projectRepoName, 1)).
		Assess("product files have been created in project git repository", assess.CheckFileInRepoExists(
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/README.md"},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/kustomization.yaml"},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/product-deployment.yaml"},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/" + prodDepGenName + "/values.yaml"},
		)).
		Assess(fmt.Sprintf("ProductDeployment %s exists in namespace %s", prodDepGenName, projectRepoName), checkIfProductDeploymentExists(prodDepGenName, projectRepoName)).
		Assess(fmt.Sprintf("ProductDeploymentPipelines exist in namespace %s", projectRepoName), checkIfProductDeploymentPipelinesExist(projectRepoName, pipelineNames)).
		Assess(fmt.Sprintf("ComponentVersion %s is created in namespace %s", prodDepGenName, projectRepoName), checkIsComponentVersionReady(prodDepGenName+"component-version", projectRepoName)).
		Assess(fmt.Sprintf("Localization is Ready in namespace %s", projectRepoName), checkLocalizationReady(projectRepoName, pipelineNames)).
		Assess(fmt.Sprintf("Configuration is Ready in namespace %s", projectRepoName), checkConfigurationReady(projectRepoName, pipelineNames)).
		Assess(fmt.Sprintf("OCIRepository is Ready in namespace %s", projectRepoName), checkOCIRepositoryReady(projectRepoName, pipelineNames)).
		Assess(fmt.Sprintf("Kustomization is Ready in namespace %s", projectRepoName), checkKustomizationReady(projectRepoName, pipelineNames)).
		Assess(fmt.Sprintf("FLuxDeployer is Ready in namespace %s", projectRepoName), checkFluxDeployerReady(projectRepoName, pipelineNames)).
		Assess(fmt.Sprintf("Deployment is Ready in namespace %s", targetNamespace), checkDeploymentsReady(targetNamespace, pipelineNames))
}

func checkIfTargetExists(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
			return ctx
		}
		t.Logf("checking if target %s in namespace %s exists", name, namespace)
		gr := &prodv1alpha1.Target{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*prodv1alpha1.Target)
			if !ok {
				return false
			}
			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkIfComponentSubscriptionExists(name, namespace string) features.Func {
	return checkObjectCondition(&rcv1alpha1.ComponentSubscription{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		func(obj fconditions.Getter) bool {
			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		})
}

func checkIfProductDeploymentGeneratorReady(name, namespace string) features.Func {
	return checkObjectCondition(&prodv1alpha1.ProductDeploymentGenerator{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		func(obj fconditions.Getter) bool {
			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		})
}

func checkSnapshotsSyncExist(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		t.Log("waiting for snapshot, git sync")

		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
			return ctx
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
			return ctx
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
			return ctx
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
				}), wait.WithTimeout(time.Minute*1))
			})
		}

		if err := p.Wait(); err != nil {
			t.Fatal(err)
		}

		return ctx
	}
}

func checkIfProductDeploymentPipelinesExist(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}

		for _, obj := range pipelineNames {
			resource := &prodv1alpha1.ProductDeploymentPipeline{
				ObjectMeta: metav1.ObjectMeta{Name: obj, Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*prodv1alpha1.ProductDeploymentPipeline)
				if !ok {
					return false
				}
				return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkIsComponentVersionReady(name, namespace string) features.Func {
	return checkObjectCondition(&v1alpha1.ComponentVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}, func(obj fconditions.Getter) bool {
		return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
	})
}

func checkIfProductDeploymentExists(name, namespace string) features.Func {
	return checkObjectCondition(&prodv1alpha1.ProductDeployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		func(obj fconditions.Getter) bool {
			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		})
}

func checkLocalizationReady(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		for _, obj := range pipelineNames {
			resource := &v1alpha1.Localization{
				ObjectMeta: metav1.ObjectMeta{Name: obj + "-localization", Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*v1alpha1.Localization)
				if !ok {
					return false
				}
				return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkConfigurationReady(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		for _, obj := range pipelineNames {
			resource := &v1alpha1.Configuration{
				ObjectMeta: metav1.ObjectMeta{Name: obj + "-configuration", Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*v1alpha1.Configuration)
				if !ok {
					return false
				}
				return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkFluxDeployerReady(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		for _, obj := range pipelineNames {
			resource := &v1alpha1.FluxDeployer{
				ObjectMeta: metav1.ObjectMeta{Name: obj + "-kustomization", Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*v1alpha1.FluxDeployer)
				if !ok {
					return false
				}
				return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkOCIRepositoryReady(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		for _, obj := range pipelineNames {
			resource := &sourcev1.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{Name: obj + "-kustomization", Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*sourcev1.OCIRepository)
				if !ok {
					return false
				}
				return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkKustomizationReady(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		for _, obj := range pipelineNames {
			resource := &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{Name: obj + "-kustomization", Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*kustomizev1.Kustomization)
				if !ok {
					return false
				}
				return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, "ReconciliationSucceeded")
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkDeploymentsReady(namespace string, pipelineNames []string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		for _, obj := range pipelineNames {
			resource := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: obj, Namespace: namespace},
			}

			err = wait.For(conditions.New(client.Resources()).ResourceMatch(resource, func(object k8s.Object) bool {
				obj, ok := object.(*appsv1.Deployment)
				if !ok {
					return false
				}
				return obj.Status.ReadyReplicas > 0
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}
func replicateRegistryCerts(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		name := "ocm-registry-tls-certs"
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		clientset, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatal(err)
			return ctx
		}
		t.Logf("checking if secret with with name: %s exists", name)
		gr := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ocm-system"},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			obj, ok := object.(*corev1.Secret)
			if !ok {
				return false
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"caFile":   obj.Data["caFile"],
					"certFile": obj.Data["certFile"],
					"keyFile":  obj.Data["keyFile"]},
			}

			_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				fmt.Println(err)
				t.Fatal(err)
			}

			newSecret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			}
			err = wait.For(conditions.New(cfg.Client().Resources()).ResourceMatch(&newSecret, func(object k8s.Object) bool {
				_, ok := object.(*corev1.Secret)
				if !ok {
					return false
				}
				return true
			}), wait.WithTimeout(time.Minute*1))

			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}
