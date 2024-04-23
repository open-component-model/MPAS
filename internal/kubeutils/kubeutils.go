// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubeutils

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/fluxcd/cli-utils/pkg/object"
	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/status"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/ssa/utils"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	productv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	projectv1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	"github.com/open-component-model/mpas/internal/env"
	ocmv1alpha1 "github.com/open-component-model/ocm-controller/api/v1alpha1"
	rep1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var apiList = []func(s *apiruntime.Scheme) error{}

func init() {
	apiList = append(apiList, apiextensionsv1.AddToScheme)
	apiList = append(apiList, corev1.AddToScheme)
	apiList = append(apiList, rbacv1.AddToScheme)
	apiList = append(apiList, appsv1.AddToScheme)
	apiList = append(apiList, networkingv1.AddToScheme)
	apiList = append(apiList, sourcev1.AddToScheme)
	apiList = append(apiList, kustomizev1.AddToScheme)
	apiList = append(apiList, ocmv1alpha1.AddToScheme)
	apiList = append(apiList, productv1alpha1.AddToScheme)
	apiList = append(apiList, projectv1alpha1.AddToScheme)
	apiList = append(apiList, rep1alpha1.AddToScheme)
}

// NewScheme creates the Scheme methods for serializing and deserializing API objects
func NewScheme() (scheme *apiruntime.Scheme, err error) {
	scheme = apiruntime.NewScheme()
	for _, api := range apiList {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}

	return scheme, nil
}

// KubeConfig returns a new Kubernetes rest config.
func KubeConfig(rcg genericclioptions.RESTClientGetter) (*rest.Config, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create kube config: %w", err)
	}

	cfg.QPS = env.DefaultKubeAPIQPS
	cfg.Burst = env.DefaultKubeAPIBurst

	return cfg, nil
}

// KubeClient returns a new Kubernetes client.
func KubeClient(rcg genericclioptions.RESTClientGetter) (client.WithWatch, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	cfg.QPS = env.DefaultKubeAPIQPS
	cfg.Burst = env.DefaultKubeAPIBurst

	scheme, err := NewScheme()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheme: %w", err)
	}

	kubeClient, err := client.NewWithWatch(cfg, client.Options{
		Scheme: scheme,
		WarningHandler: client.WarningHandlerOptions{
			SuppressWarnings: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kube client: %w", err)
	}

	return kubeClient, nil
}

// MustInstallKustomization returns true if the given kustomization is not installed.
func MustInstallKustomization(ctx context.Context, kubeClient client.Client, name, namespace string) bool {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var k kustomizev1.Kustomization
	if err := kubeClient.Get(ctx, namespacedName, &k); err != nil {
		return true
	}
	return k.Status.LastAppliedRevision == ""
}

// MustInstallNS returns true if the given namespace is not installed.
func MustInstallNS(ctx context.Context, kubeClient client.Client, namespace string) bool {
	namespacedName := types.NamespacedName{
		Name: namespace,
	}
	var ns corev1.Namespace
	if err := kubeClient.Get(ctx, namespacedName, &ns); err != nil {
		return true
	}
	return false
}

// ReconcileKustomization reconciles the given kustomization.
func ReconcileKustomization(ctx context.Context, kubeClient client.Client, name, namespace string) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var k kustomizev1.Kustomization
	if err := kubeClient.Get(ctx, namespacedName, &k); err != nil {
		return err
	}

	return reconcileObject(ctx, namespacedName, kubeClient, kustomizev1.GroupVersion.WithKind("Kustomization"))
}

// ReconcileGitrepository reconciles the given git repository.
func ReconcileGitrepository(ctx context.Context, kubeClient client.Client, name, namespace string) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var g sourcev1.GitRepository
	if err := kubeClient.Get(ctx, namespacedName, &g); err != nil {
		return err
	}
	return reconcileObject(ctx, namespacedName, kubeClient, sourcev1.GroupVersion.WithKind("GitRepository"))
}

func reconcileObject(ctx context.Context, namespacedName types.NamespacedName, kubeClient client.Client, gvk schema.GroupVersionKind) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		object := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		object.SetGroupVersionKind(gvk)
		if err := kubeClient.Get(ctx, namespacedName, object); err != nil {
			return err
		}
		patch := client.MergeFrom(object.DeepCopy())
		if ann := object.GetAnnotations(); ann == nil {
			object.SetAnnotations(map[string]string{
				meta.ReconcileRequestAnnotation: time.Now().Format(time.RFC3339Nano),
			})
		} else {
			ann[meta.ReconcileRequestAnnotation] = time.Now().Format(time.RFC3339Nano)
			object.SetAnnotations(ann)
		}
		return kubeClient.Patch(ctx, object, patch)
	})
}

// ReportGitrepositoryHealth reconciles the health of the given git repository.
func ReportGitrepositoryHealth(ctx context.Context, kubeClient client.Client, name, namespace, expectedRevision string, pollInterval, timeout time.Duration) error {
	objKey := client.ObjectKey{Name: name, Namespace: namespace}
	var o sourcev1.GitRepository
	if err := wait.PollImmediateWithContext(ctx, pollInterval, timeout, reconciledGitrepositoryHealth(
		ctx, kubeClient, objKey, &o, expectedRevision),
	); err != nil {
		return err
	}
	return nil
}

// ReportKustomizationHealth reconciles the health of the given kustomization.
func ReportKustomizationHealth(ctx context.Context, kubeClient client.Client, name, namespace, expectedRevision string, pollInterval, timeout time.Duration) error {
	objKey := client.ObjectKey{Name: name, Namespace: namespace}
	var k kustomizev1.Kustomization
	if err := wait.PollImmediateWithContext(ctx, pollInterval, timeout, reconciledKustomizationHealth(
		ctx, kubeClient, objKey, &k, expectedRevision),
	); err != nil {
		return err
	}
	return nil
}

func reconciledKustomizationHealth(ctx context.Context, kube client.Client, objKey client.ObjectKey,
	kustomization *kustomizev1.Kustomization, expectRevision string) func(context.Context) (bool, error) {

	return func(ctx context.Context) (bool, error) {
		if err := kube.Get(ctx, objKey, kustomization); err != nil {
			return false, err
		}

		// Detect suspended Kustomization, as this would result in an endless wait
		if kustomization.Spec.Suspend {
			return false, fmt.Errorf("kustomization is suspended")
		}

		// Confirm the state we are observing is for the current generation
		if kustomization.Generation != kustomization.Status.ObservedGeneration {
			return false, nil
		}

		// Confirm the given revision has been attempted by the controller
		if kustomization.Status.LastAttemptedRevision != expectRevision {
			return false, nil
		}

		// Confirm the resource is healthy
		if c := apimeta.FindStatusCondition(kustomization.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}

func reconciledGitrepositoryHealth(ctx context.Context, kube client.Client, objKey client.ObjectKey,
	gitrepo *sourcev1.GitRepository, expectedRevision string) func(context.Context) (bool, error) {

	return func(ctx context.Context) (bool, error) {
		if err := kube.Get(ctx, objKey, gitrepo); err != nil {
			return false, err
		}

		// Detect suspended Kustomization, as this would result in an endless wait
		if gitrepo.Spec.Suspend {
			return false, fmt.Errorf("GitRepository is suspended")
		}

		// Confirm the state we are observing is for the current generation
		if gitrepo.Generation != gitrepo.Status.ObservedGeneration {
			return false, nil
		}

		if gitrepo.Status.Artifact.Revision != expectedRevision {
			return false, nil
		}

		// Confirm the resource is healthy
		if c := apimeta.FindStatusCondition(gitrepo.Status.Conditions, meta.ReadyCondition); c != nil {
			switch c.Status {
			case metav1.ConditionTrue:
				return true, nil
			case metav1.ConditionFalse:
				return false, fmt.Errorf(c.Message)
			}
		}
		return false, nil
	}
}

// ReportComponentsHealth reconciles the health of the given components.
func ReportComponentsHealth(ctx context.Context, rcg genericclioptions.RESTClientGetter, timeout time.Duration, components []string, ns string) error {
	cfg, err := KubeConfig(rcg)
	if err != nil {
		return err
	}

	checker, err := status.NewStatusChecker(cfg, env.DefaultPollInterval, timeout, log.NopLogger{})
	if err != nil {
		return err
	}

	var identifiers []object.ObjMetadata
	for _, component := range components {
		identifiers = append(identifiers, object.ObjMetadata{
			Namespace: ns,
			Name:      component,
			GroupKind: schema.GroupKind{Group: "apps", Kind: "Deployment"},
		})
	}

	if err := checker.Assess(identifiers...); err != nil {
		return err
	}
	return nil
}

// YamlToUnstructructured converts the given yaml to a slice of unstructured objects.
func YamlToUnstructructured(data []byte) ([]*unstructured.Unstructured, error) {
	return utils.ReadObjects(bytes.NewReader(data))
}

func UnstructuredToYaml(objs []*unstructured.Unstructured) ([]byte, error) {
	var buf bytes.Buffer
	e := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil,
		json.SerializerOptions{Yaml: true, Pretty: true})

	for _, obj := range objs {
		buf.WriteString("---\n")
		if err := e.Encode(obj, &buf); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// FilterUnstructured returns a slice of unstructured objects that match the given filter.
func FilterUnstructured(objs []*unstructured.Unstructured, filter func(*unstructured.Unstructured) bool) []*unstructured.Unstructured {
	var filtered []*unstructured.Unstructured
	for _, obj := range objs {
		if filter(obj) {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}

// NSFilter returns a filter that filters out the given namespace.
func NSFilter(ns string) func(*unstructured.Unstructured) bool {
	return func(obj *unstructured.Unstructured) bool {
		return obj.GetKind() != "Namespace" && obj.GetName() != ns
	}
}
