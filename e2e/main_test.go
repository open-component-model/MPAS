//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	notifv1 "github.com/fluxcd/notification-controller/api/v1"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	gitv1alphav1 "github.com/open-component-model/git-controller/apis/delivery/v1alpha1"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
	"time"

	"github.com/open-component-model/ocm-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

var (
	mpasManagementRepoName = "mpas-mgmt"
	mpasNamespace          = "mpas-system"
)

func TestMpasE2e(t *testing.T) {
	t.Log("running mpas happy path tests")

	projectName := getYAMLField("project.yaml", "metadata.name")
	projects := prefix + projectName
	projectRepoName := prefix + getYAMLField("project.yaml", "metadata.name")
	gitRepoUrl := getYAMLField("project.yaml", "spec.git.domain")
	gitCredentialName = getYAMLField("project.yaml", "spec.git.credentials.secretRef.name")
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
		Setup(setup.AddScheme(notifv1.AddToScheme)).
		Setup(shared.CreateSecret(gitCredentialName, nil, gitCredentialData, mpasNamespace)).
		Assess(fmt.Sprintf("management namespace %s exists", mpasNamespace), checkIsNamespaceReady(mpasNamespace))

	project := newProjectFeature(projectName, projectRepoName, gitRepoUrl)

	intermediateSetup := features.New("2.1 Create Receiver for gitea hook").
		WithStep("Create git credentials ", 1, shared.CreateSecret(gitCredentialName, nil, gitCredentialData, projectRepoName)).
		WithStep("Create gitea hook secret", 2, shared.CreateSecret(hookSecretName, nil, map[string]string{"token": hookSecretToken}, projects)).
		WithStep("Create registry-certs for target namespace", 2, replicateRegistryCerts(getYAMLField("target.yaml", "spec.access.targetNamespace"))).
		WithStep("Create Receiver", 3, createReceiver(projects)).
		WithStep("Create Web Hook", 4, setup.CreateWebhookAPI(projects, hookSecretToken))

	product := newProductFeature(projectRepoName)

	testEnv.Test(t,
		setupComponent.Feature(),
		management.Feature(),
		project.Feature(),
		intermediateSetup.Feature(),
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

func replicateRegistryCerts(namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		name := "registry-certs"
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

func checkObjectCondition(conditionObject k8s.Object, condition func(getter fconditions.Getter) bool) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("checking readiness of object with name: %s", conditionObject.GetName())

		err = wait.For(conditions.New(client.Resources()).ResourceMatch(conditionObject, func(object k8s.Object) bool {
			obj, ok := object.(fconditions.Getter)
			if !ok {
				return false
			}
			return condition(obj)
		}), wait.WithTimeout(time.Minute*1))

		if err != nil {
			t.Fatal(err)
		}

		return ctx
	}
}
func reasonMatches(from fconditions.Getter, reason string) bool {
	for _, condition := range from.GetConditions() {
		if condition.Reason == reason {
			return true
		}
	}
	return false
}