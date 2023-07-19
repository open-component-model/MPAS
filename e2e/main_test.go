//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-component-model/ocm-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

var (
	mpasManagementRepoName = "mpas-mgmt"
	mpasNamespace          = "mpas-system"
)

func TestHappyPath(t *testing.T) {
	t.Log("running mpas happy path tests")

	projectName := getYAMLField("project.yaml", "metadata.name")
	projectRepoName := prefix + getYAMLField("project.yaml", "metadata.name")
	gitCredentialName := getYAMLField("project.yaml", "spec.git.credentials.secretRef.name")
	gitRepoUrl := getYAMLField("project.yaml", "spec.git.domain")
	targetName := getYAMLField("target.yaml", "metadata.name")
	componentSubscriptionName := getYAMLField("subscription.yaml", "metadata.name")

	gitCredentialData := map[string]string{
		"token":    shared.TestUserToken,
		"username": shared.Owner,
		"password": shared.TestUserToken,
	}

	setupComponent := createTestComponentVersion(t)

	management := features.New("Configure Management Repository").
		Setup(setup.AddScheme(v1alpha1.AddToScheme)).
		Setup(setup.AddScheme(sourcev1.AddToScheme)).
		Setup(setup.AddScheme(kustomizev1.AddToScheme)).
		Setup(shared.CreateSecret(gitCredentialName, nil, gitCredentialData, mpasNamespace)).
		Assess(fmt.Sprintf("management namespace %s exists", mpasNamespace), checkIsNamespaceReady(mpasNamespace))

	project := newProjectFeature(projectName, projectRepoName, gitRepoUrl)

	target := features.New("Add a target").
		Setup(setup.AddFilesToGitRepository(
			setup.File{
				RepoName:       projectRepoName,
				SourceFilepath: "target.yaml",
				DestFilepath:   "targets/ingress-target.yaml",
			},
		)).
		Assess(fmt.Sprintf("target resource %s has been created", targetName),
			checkIfTargetExists(targetName, getYAMLField("target.yaml", "metadata.namespace")))

	subscription := features.New("Create a subscription").
		Setup(setup.AddFilesToGitRepository(
			setup.File{
				RepoName:       projectRepoName,
				SourceFilepath: "subscription.yaml",
				DestFilepath:   "subscriptions/podinfo.yaml",
			},
		)).
		Assess(fmt.Sprintf("componentsubscription resource %s has been created", componentSubscriptionName),
			checkIfSubscriptionExists(componentSubscriptionName, getYAMLField("subscription.yaml", "metadata.namespace")))

	product := newProductFeature(projectRepoName)

	testEnv.Test(t,
		setupComponent.Feature(),
		management.Feature(),
		project.Feature(),
		target.Feature(),
		subscription.Feature(),
		product.Feature(),
	)
}

func checkIsNamespaceReady(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if namespace with name: %s exists", name)
		gr := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*corev1.Namespace)
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

func checkIfTargetExists(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if rolebinding with name: %s exists", name)
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

func checkIfSubscriptionExists(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if rolebinding with name: %s exists", name)
		gr := &rcv1alpha1.ComponentSubscription{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rcv1alpha1.ComponentSubscription)
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

//func checkNamespaceReady(ns string) features.Func {
//	return func(ctx context.Context, t *testing.T, env *envconf.Config) context.Context {
//		t.Helper()
//		t.Logf("checking if namespace %s exists...", ns)
//		r, err := resources.New(env.Client().RESTConfig())
//		if err != nil {
//			t.Error(err)
//			return ctx
//		}
//		if err := r.Get(ctx, ns, ns, &corev1.Namespace{}); err != nil {
//			t.Error(err)
//			return ctx
//		}
//		t.Logf("namespace %s exists.", ns)
//		return ctx
//	}
//}
