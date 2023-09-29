// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	prd1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	prj1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resource is an interface for Kubernetes resources.
var _ Resource = (*ProductDeploymentGenerator)(nil)

// ProductDeploymentGenerator is a wrapper around prj1alpha1.ProductDeploymentGenerator.
type ProductDeploymentGenerator struct {
	prd1alpha1.ProductDeploymentGenerator
}

// ToClientObject returns the project as a client.Object.
func (p *ProductDeploymentGenerator) ToClientObject() client.Object {
	return &p.ProductDeploymentGenerator
}

// GetObservedGeneration returns the observed generation of the project.
func (p *ProductDeploymentGenerator) GetObservedGeneration() int64 {
	return p.Status.ObservedGeneration
}

// GetGeneration returns the generation of the project.
func (p *ProductDeploymentGenerator) GetGeneration() int64 {
	return p.Generation
}

// ToYamlExport returns the project as a YAML string.
// It can be used to export the project to a file or to pass to kubectl apply.
func (p *ProductDeploymentGenerator) ToYamlExport() (string, error) {
	proj := p.DeepCopy()
	gvk := prj1alpha1.GroupVersion.WithKind("ProductDeploymentGenerator")
	proj.SetGroupVersionKind(gvk)
	return toYamlExport(proj)
}
