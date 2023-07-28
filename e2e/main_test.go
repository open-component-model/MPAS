//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	gitv1alphav1 "github.com/open-component-model/git-controller/apis/delivery/v1alpha1"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
	"time"

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
	projects := prefix + projectName
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

	setupComponent := createTestComponentVersion(t) //createTestComponentVersion(t)

	management := features.New("Configure Management Repository").
		Setup(setup.AddScheme(v1alpha1.AddToScheme)).
		Setup(setup.AddScheme(sourcev1.AddToScheme)).
		Setup(setup.AddScheme(kustomizev1.AddToScheme)).
		Setup(setup.AddScheme(gitv1alphav1.AddToScheme)).
		Setup(setup.AddScheme(prodv1alpha1.AddToScheme)).
		Setup(setup.AddScheme(rcv1alpha1.AddToScheme)).
		Setup(shared.CreateSecret(gitCredentialName, nil, gitCredentialData, mpasNamespace)).
		Assess(fmt.Sprintf("management namespace %s exists", mpasNamespace), checkIsNamespaceReady(mpasNamespace))

	project := newProjectFeature(projectName, projectRepoName, gitRepoUrl)

	targetAndSubscription := features.New("2.1 Add a target & subscription").
		WithStep("2.2 Add Target", 1, setup.AddFilesToGitRepository(
			setup.File{
				RepoName:       projectRepoName,
				SourceFilepath: "target.yaml",
				DestFilepath:   "targets/ingress-target.yaml",
			},
		)).
		WithStep("2.3 Add Subscription", 1, setup.AddFilesToGitRepository(
			setup.File{
				RepoName:       projectRepoName,
				SourceFilepath: "subscription.yaml",
				DestFilepath:   "subscriptions/podinfo.yaml",
			},
		)).
		Assess(fmt.Sprintf("2.4 target resource %s has been created", targetName),
			checkIfTargetExists(targetName, projects)).
		Assess(fmt.Sprintf("2.5 componentsubscription resource %s has been created", componentSubscriptionName),
			checkIfSubscriptionExists(componentSubscriptionName, projects)).
		Setup(shared.CreateSecret(gitCredentialName, nil, gitCredentialData, projects))

	assessTargetSubscription := features.New("2.6 Validate Target & subscription").
		Assess(fmt.Sprintf("2.7 target resource %s has been created", targetName), assess.ResourceWasCreated(assess.Object{
			Name:      targetName,
			Namespace: projects,
			Obj:       &prodv1alpha1.Target{},
		})).
		Assess(fmt.Sprintf("2.8 componentsubscription resource %s has been created", componentSubscriptionName), assess.ResourceWasCreated(assess.Object{
			Name:      componentSubscriptionName,
			Namespace: projects,
			Obj:       &rcv1alpha1.ComponentSubscription{},
		}))

	product := newProductFeature(projectRepoName)

	testEnv.Test(t,
		setupComponent.Feature(),
		management.Feature(),
		project.Feature(),
		targetAndSubscription.Feature(),
		assessTargetSubscription.Feature(),
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
		}), wait.WithTimeout(time.Second*10))
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
		t.Logf("checking if subscription %s in namespace %s exists", name, namespace)
		gr := &rcv1alpha1.ComponentSubscription{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			obj, ok := object.(*rcv1alpha1.ComponentSubscription)
			if !ok {
				return false
			}
			return fconditions.IsTrue(obj, meta.ReadyCondition) && reasonMatches(obj, meta.SucceededReason)
		}), wait.WithTimeout(time.Second*10))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}