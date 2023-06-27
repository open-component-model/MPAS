// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubeutils

import (
	"fmt"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	productv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	projectv1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	ocmv1alpha1 "github.com/open-component-model/ocm-controller/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
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

	return cfg, nil
}

// KubeClient returns a new Kubernetes client.
func KubeClient(rcg genericclioptions.RESTClientGetter) (client.WithWatch, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	scheme := NewScheme()
	kubeClient, err := client.NewWithWatch(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kube client: %w", err)
	}

	return kubeClient, nil
}
