//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	gcv1alpha1 "github.com/open-component-model/git-controller/apis/mpas/v1alpha1"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	ocmv1alpha1 "github.com/open-component-model/ocm-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	notifv1 "github.com/fluxcd/notification-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

type kustomization struct {
	name          string
	path          string
	sourceRefKind string
	sourceRefName string
}

var (
	fluxNamespace      = "flux-system"
	prefix             = "mpas-"
	projectClusterRole = "mpas-projects-clusterrole"
	clusterRoleSuffix  = "-clusterrole"
	hookSecretName     = "receiver-token"
	hookSecretToken    = "supersecrettoken"
	gitCredentialName  = getYAMLField("project.yaml", "spec.git.credentials.secretRef.name")
)

func newProjectFeature(projectName, projectRepoName, gitRepoUrl string) *features.FeatureBuilder {
	projects := prefix + projectName
	return features.New("Create Project").
		Setup(setup.AddGitRepository(mpasManagementRepoName)).
		Assess(fmt.Sprintf("management git repository %s has been created", mpasManagementRepoName), assess.CheckRepoExists(mpasManagementRepoName)).
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       mpasManagementRepoName,
			SourceFilepath: "project.yaml",
			DestFilepath:   "projects/test-001.yaml",
		})).
		Setup(setup.AddFluxSyncForRepo(mpasManagementRepoName, "projects/", namespace)).
		Assess(fmt.Sprintf("flux::gitRepository has been created in %s namespace", fluxNamespace), checkFluxGitRepositoryReady(mpasManagementRepoName,
			fluxNamespace)).
		Assess(fmt.Sprintf("flux::kustomizations are configured in %s namespace", fluxNamespace), checkKustomizationsConfiguration(fluxNamespace,
			kustomization{
				name:          mpasManagementRepoName,
				path:          "projects/",
				sourceRefKind: "GitRepository",
				sourceRefName: mpasManagementRepoName,
			})).
		Assess(fmt.Sprintf("1.1 project namespace %s has been created", projects), checkIsNamespaceReady(projects)).
		Assess(fmt.Sprintf("1.2 projects ClusterRole %s exists", projectClusterRole), checkIfClusterRoleExists(projectClusterRole)).
		Assess(fmt.Sprintf("1.3 project service account %s has been created", projects), checkIfServiceAccountExists(projects)).
		Assess(fmt.Sprintf("1.4 project role %s has been created", projects), checkIfRoleExists(projects)).
		Assess(fmt.Sprintf("1.5 project RoleBindings %s has been created in namespace %s", projects, projects), checkIfRoleBindingsExist(projects, projects)).
		Assess(fmt.Sprintf("1.6 project RoleBindings %s has been created in namespace %s", projects+clusterRoleSuffix, projects), checkIfRoleBindingsExist(projects, projects)).
		Assess(fmt.Sprintf("1.7 project RoleBindings %s has been created in namespace %s", projects, mpasNamespace), checkIfRoleBindingsExist(projects, mpasNamespace)).
		Assess(fmt.Sprintf("1.8 project SA %s can list target and componentsubscription resources in %s namespace", projects, projects),
			checkSACanListResourcesInNamespace(projects, projects,
				&prodv1alpha1.TargetList{}, &rcv1alpha1.ComponentSubscriptionList{},
			)).
		Assess(fmt.Sprintf("1.9 project SA %s can create resources in %s namespace", projects, projects), checkSACanCreateResources(
			projects,
			&corev1.Secret{},
			&gcv1alpha1.Repository{},
			&prodv1alpha1.Target{},
			&prodv1alpha1.ProductDeployment{},
			&prodv1alpha1.ProductDeploymentGenerator{},
			&prodv1alpha1.ProductDeploymentPipeline{},
			&rcv1alpha1.ComponentSubscription{},
			&ocmv1alpha1.ComponentVersion{},
			&ocmv1alpha1.Localization{},
			&ocmv1alpha1.Configuration{},
			&sourcev1.OCIRepository{},
			&kustomizev1.Kustomization{},
		)).
		Assess(fmt.Sprintf("1.10 project SA %s can list target and componentsubscription resources in %s namespace", projects, projects),
			checkSACanListResourcesInNamespace(projects, projects,
				&prodv1alpha1.TargetList{}, &rcv1alpha1.ComponentSubscriptionList{},
			)).
		Assess(fmt.Sprintf("1.11 project repository %s/%s/%s has been created", shared.BaseURL, shared.Owner, projectRepoName), assess.CheckRepoExists(projectRepoName)).
		Assess("1.12 check files are created in project repo", checkRepoFileContent(projectRepoName)).
		Assess(fmt.Sprintf("1.13 flux resources have been created in %s namespace", fluxNamespace), checkFluxGitRepositoryReady(projects, mpasNamespace)).
		Assess(fmt.Sprintf("1.14 flux::GitRepository is configured correctly in %s namespace", mpasNamespace), checkGitRepositoryConfiguration(projects, strings.Join([]string{gitRepoUrl,
			shared.Owner, projects}, "/"), "main")).
		Assess(fmt.Sprintf("1.15 flux::kustomizations are configured correctly in %s namespace", mpasNamespace), checkKustomizationsConfiguration(mpasNamespace,
			kustomization{
				name:          projects + "-subscriptions",
				path:          "subscriptions",
				sourceRefKind: "GitRepository",
				sourceRefName: projects,
			},
			kustomization{
				name:          projects + "-targets",
				path:          "targets",
				sourceRefKind: "GitRepository",
				sourceRefName: projects,
			},
			kustomization{
				name:          projects + "-products",
				path:          "products",
				sourceRefKind: "GitRepository",
				sourceRefName: projects,
			},
			kustomization{
				name:          projects + "-generators",
				path:          "generators",
				sourceRefKind: "GitRepository",
				sourceRefName: projects,
			},
		))

}

func checkIfClusterRoleExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
			return ctx
		}
		t.Logf("checking if cluster role with name: %s exists", name)
		gr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rbacv1.ClusterRole)
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

func checkIfServiceAccountExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
			return ctx
		}
		t.Logf("checking if service account with name: %s exists", name)
		gr := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*corev1.ServiceAccount)
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

func checkIfRoleExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
			return ctx
		}
		t.Logf("checking if role with name: %s exists", name)
		gr := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rbacv1.Role)
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

func checkIfRoleBindingsExist(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
			return ctx
		}
		t.Logf("checking if rolebinding with name: %s exists", name)
		gr := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rbacv1.RoleBinding)
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

func checkSACanCreateResources(name string, res ...k8s.Object) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()
		cfg := c.Client().RESTConfig()
		cfg.Impersonate.UserName = fmt.Sprintf("system:serviceaccount:%s:%s", name, name)

		r, err := resources.New(cfg)
		if err != nil {
			t.Fatal(err)
			return ctx
		}

		for _, re := range res {
			re.SetName("can-sa-create-resource-test")
			re.SetNamespace(name)
			t.Logf("checking if service account %s:%s can create %s resources in namespace %s", name, name, reflect.TypeOf(re), name)
			err := r.Create(ctx, re)
			// The API should attempt to authorize the request first, before validating the object schema
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Fatal(err)
			}

			err = r.Delete(ctx, re)
			// The API should attempt to authorize the request first, before validating the object schema
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkSACanListResourcesInNamespace(name, namespace string, res ...k8s.ObjectList) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		cfg := c.Client().RESTConfig()
		cfg.Impersonate.UserName = fmt.Sprintf("system:serviceaccount:%s:%s", name, name)

		r, err := resources.New(cfg)
		if err != nil {
			t.Fatal(err)
			return ctx
		}

		for _, re := range res {
			t.Logf("checking if service account %s:%s can list %s resources in namespace %s", name, name, reflect.TypeOf(re), namespace)
			err := r.WithNamespace(namespace).List(ctx, re)
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

func checkRepoFileContent(projectRepoName string) features.Func {
	return assess.CheckRepoFileContent(
		assess.File{
			Repository: projectRepoName,
			Path:       "CODEOWNERS",
			Content:    "alice.bobb\nbob.alisson",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "generators/.keep",
			Content:    "",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "products/.keep",
			Content:    "",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "subscriptions/.keep",
			Content:    "",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "targets/.keep",
			Content:    "",
		},
	)
}

func checkFluxGitRepositoryReady(name, namespace string) features.Func {
	return checkObjectCondition(&sourcev1.GitRepository{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}},
		func(obj fconditions.Getter) bool {
			return fconditions.IsTrue(obj, meta.ReadyCondition)
		})
}

func checkGitRepositoryConfiguration(name, url, branch string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if git repository with name: %s is ready", name)
		gr := &sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: mpasNamespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			obj, ok := object.(*sourcev1.GitRepository)
			if !ok {
				return false
			}
			if obj.Spec.URL != url {
				t.Errorf("expected GitRepository %s to have URL %s, got %s", name, url, gr.Spec.URL)
				return false
			}

			if obj.Spec.Reference.Branch != branch {
				t.Errorf("expected GitRepository %s to have branch %s, got %s", name, branch, gr.Spec.Reference.Branch)
				return false
			}
			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkKustomizationsConfiguration(namespace string, kustomizations ...kustomization) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}

		for _, kustomization := range kustomizations {

			t.Logf("checking if Kustomization %s in namespace %s is ready", kustomization.name, namespace)
			gr := &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{Name: kustomization.name, Namespace: namespace},
			}
			err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
				obj, ok := object.(*kustomizev1.Kustomization)
				if !ok {
					return false
				}
				if obj.Spec.SourceRef.Kind != kustomization.sourceRefKind {
					t.Errorf("expected Kustomization %s to have sourceRef kind %s, got %s", kustomization.name, kustomization.sourceRefKind, obj.Spec.SourceRef.Kind)
					return false
				}

				if obj.Spec.SourceRef.Name != kustomization.sourceRefName {
					t.Errorf("expected Kustomization %s to have sourceRef name %s, got %s", kustomization.name, kustomization.sourceRefName, obj.Spec.SourceRef.Name)
					return false
				}

				if obj.Spec.Path != kustomization.path {
					t.Errorf("expected Kustomization %s to have path %s, got %s", kustomization.name, kustomization.path, obj.Spec.Path)
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

func prepareReceiver(name, namespace string) (map[string]interface{}, error) {
	gr := &notifv1.Receiver{
		TypeMeta: metav1.TypeMeta{
			Kind:       notifv1.ReceiverKind,
			APIVersion: "notification.toolkit.fluxcd.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: notifv1.ReceiverSpec{
			Type:     notifv1.GitHubReceiver,
			Interval: &metav1.Duration{Duration: time.Second * 5},
			Events:   []string{"ping", "push"},
			Resources: []notifv1.CrossNamespaceObjectReference{{
				APIVersion: "source.toolkit.fluxcd.io/v1",
				Kind:       sourcev1.GitRepositoryKind,
				Name:       name,
				Namespace:  mpasNamespace,
			}},
			SecretRef: meta.LocalObjectReference{Name: hookSecretName},
		},
	}

	return runtime.DefaultUnstructuredConverter.ToUnstructured(gr)
}

func createReceiver(projectName string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}
		clientset, err := dynamic.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}
		client, err := cfg.NewClient()
		if err != nil {
			t.Error(err)
			return ctx
		}
		unstructuredObj, err := prepareReceiver(projectName, projectName)
		if err != nil {
			t.Fatal(err)
			return ctx
		}

		fmt.Println(unstructuredObj)
		_, err = clientset.Resource(schema.GroupVersionResource{
			Group:    "notification.toolkit.fluxcd.io",
			Version:  "v1",
			Resource: "receivers",
		}).Namespace(projectName).Create(ctx, &unstructured.Unstructured{Object: unstructuredObj}, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
			return ctx
		}

		t.Logf("checking if Reciever %s exists...", projectName)
		receiver := &notifv1.Receiver{}
		if err := r.Get(ctx, projectName, projectName, receiver); err != nil {
			t.Fatal(err)
			return ctx
		}

		err = wait.For(conditions.New(client.Resources()).ResourceMatch(receiver, func(object k8s.Object) bool {
			obj, ok := object.(*notifv1.Receiver)
			if !ok {
				return false
			}
			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}