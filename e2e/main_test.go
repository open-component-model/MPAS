// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-component-model/ocm-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

var (
	mpasRepoName  = "mpas-mgmt"
	mpasNamespace = "mpas-system"
)

func TestHappyPath(t *testing.T) {
	t.Log("running mpas happy path tests")

	projectName := getYAMLField("project.yaml", "metadata.name")
	projectRepoName := getYAMLField("project.yaml", "spec.git.repository.name")

	management := features.New("Configure Management Repository").
		Setup(setup.AddScheme(v1alpha1.AddToScheme)).
		Setup(setup.AddScheme(sourcev1.AddToScheme)).
		Setup(setup.AddScheme(kustomizev1.AddToScheme)).
		Setup(setup.AddGitRepository(mpasRepoName)).
		//TODO: change this to add single gitsource and
		// multiple kustomizations
		Setup(setup.AddFluxSyncForRepo(mpasRepoName, "projects/", namespace)).
		Assess("project flux resources have been created", checkFluxResourcesReady(mpasRepoName)).
		Setup(setup.CreateNamespace(mpasNamespace))

	project := newProjectFeature(mpasRepoName, mpasNamespace, projectName, projectRepoName)

	target := features.New("Add a target").
		Setup(setup.AddFileToGitRepository(mpasRepoName, "target.yaml", "targets/ingress-target.yaml"))

	subscription := features.New("Create a subscription").
		Setup(setup.AddFileToGitRepository(mpasRepoName, "subscription.yaml", "subscriptions/podinfo.yaml"))

	product := features.New("Install a product").
		Setup(setup.AddFileToGitRepository(projectRepoName, "podinfo_product_generator.yaml", "generators/podinfo.yaml"))

	testEnv.Test(t,
		management.Feature(),
		project,
		target.Feature(),
		subscription.Feature(),
		product.Feature())
}

func checkNamespaceReady(ns string) features.Func {
	return func(ctx context.Context, t *testing.T, env *envconf.Config) context.Context {
		t.Helper()
		t.Logf("checking if namespace %s exists...", ns)

		r, err := resources.New(env.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		if err := r.Get(ctx, ns, ns, &corev1.Namespace{}); err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("namespace %s exists.", ns)
		return ctx
	}
}

func checkRBACReady(name string) features.Func {
	return func(ctx context.Context, t *testing.T, env *envconf.Config) context.Context {
		t.Helper()

		r, err := resources.New(env.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if service account %s exists...", name)
		sa := &corev1.ServiceAccount{}
		if err := r.Get(ctx, name, name, sa); err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if role binding %s exists...", name)
		rb := &rbacv1.RoleBinding{}
		if err := r.Get(ctx, name, name, rb); err != nil {
			t.Error(err)
			return ctx
		}

		return ctx
	}
}

func checkFluxResourcesReady(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		t.Logf("checking if GitRepository object %s is ready...", name)
		gr := &sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "flux-system"},
		}

		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			obj, ok := object.(*sourcev1.GitRepository)
			if !ok {
				return false
			}

			return fconditions.IsTrue(obj, meta.ReadyCondition)
		}), wait.WithTimeout(time.Minute*1))

		if err != nil {
			t.Fatal(err)
		}

		t.Logf("checking if Kustomization object %s is ready...", name)
		kust := &kustomizev1.Kustomization{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "flux-system"},
		}

		err = wait.For(conditions.New(client.Resources()).ResourceMatch(kust, func(object k8s.Object) bool {
			obj, ok := object.(*kustomizev1.Kustomization)
			if !ok {
				return false
			}

			return fconditions.IsTrue(obj, meta.ReadyCondition)
		}), wait.WithTimeout(time.Minute*1))

		if err != nil {
			t.Fatal(err)
		}

		return ctx
	}
}

func checkProductReady(ctx context.Context, t *testing.T, env *envconf.Config) context.Context {
	t.Fail()
	return ctx
}
