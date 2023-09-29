// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"bytes"
	"strings"

	prj1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Resource is an interface for Kubernetes resources.
var _ Resource = (*Project)(nil)

// Project is a wrapper around prj1alpha1.Project.
type Project struct {
	prj1alpha1.Project
}

// ToClientObject returns the project as a client.Object.
func (p *Project) ToClientObject() client.Object {
	return &p.Project
}

// GetObservedGeneration returns the observed generation of the project.
func (p *Project) GetObservedGeneration() int64 {
	return p.Status.ObservedGeneration
}

// GetGeneration returns the generation of the project.
func (p *Project) GetGeneration() int64 {
	return p.Generation
}

// ToYamlExport returns the project as a YAML string.
// It can be used to export the project to a file or to pass to kubectl apply.
func (p *Project) ToYamlExport() (string, error) {
	proj := p.DeepCopy()
	gvk := prj1alpha1.GroupVersion.WithKind("Project")
	proj.SetGroupVersionKind(gvk)
	return toYamlExport(proj)
}

func toYamlExport(obj interface{}) (string, error) {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	b = bytes.Replace(b, []byte("  creationTimestamp: null\n"), []byte(""), 1)
	b = bytes.Replace(b, []byte("status: {}\n"), []byte(""), 1)
	b = bytes.TrimSpace(b)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(b)

	return sb.String(), nil
}
