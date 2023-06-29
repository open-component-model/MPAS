// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubeutils

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/status"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/ssa"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	productv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	projectv1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	"github.com/open-component-model/mpas/pkg/env"
	ocmv1alpha1 "github.com/open-component-model/ocm-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Create the Scheme, methods for serializing and deserializing API objects
// which can be shared by tests.
func NewScheme() *apiruntime.Scheme {
	scheme := apiruntime.NewScheme()
	_ = apiextensionsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)
	_ = ocmv1alpha1.AddToScheme(scheme)
	_ = productv1alpha1.AddToScheme(scheme)
	_ = projectv1alpha1.AddToScheme(scheme)

	return scheme
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

	scheme := NewScheme()
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

func ReconcileKustomization(ctx context.Context, kubeClient client.Client, name, namespace string) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	var k kustomizev1.Kustomization
	if err := kubeClient.Get(ctx, namespacedName, &k); err != nil {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		object := &metav1.PartialObjectMetadata{}
		object.SetGroupVersionKind(kustomizev1.GroupVersion.WithKind("Kustomization"))
		object.SetName(namespacedName.Name)
		object.SetNamespace(namespacedName.Namespace)
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

func ReportHealth(ctx context.Context, rcg genericclioptions.RESTClientGetter, timeout time.Duration, components []string, ns string) error {
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

func YamlToUnstructructured(data []byte) ([]*unstructured.Unstructured, error) {
	return ssa.ReadObjects(bytes.NewReader(data))
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

func FilterUnstructured(objs []*unstructured.Unstructured, filter func(*unstructured.Unstructured) bool) []*unstructured.Unstructured {
	var filtered []*unstructured.Unstructured
	for _, obj := range objs {
		if filter(obj) {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}

func NSFilter(ns string) func(*unstructured.Unstructured) bool {
	return func(obj *unstructured.Unstructured) bool {
		return obj.GetKind() != "Namespace" && obj.GetName() != ns
	}
}
