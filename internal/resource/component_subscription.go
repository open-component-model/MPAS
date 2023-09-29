// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	rep1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resource is an interface for Kubernetes resources.
var _ Resource = (*ComponentSubscription)(nil)

// ComponentSubscription is a wrapper around prj1alpha1.ComponentSubscription.
type ComponentSubscription struct {
	rep1alpha1.ComponentSubscription
}

// ToClientObject returns the component subscription as a client.Object.
func (c *ComponentSubscription) ToClientObject() client.Object {
	return &c.ComponentSubscription
}

// GetObservedGeneration returns the observed generation of the component subscription.
func (c *ComponentSubscription) GetObservedGeneration() int64 {
	return c.Status.ObservedGeneration
}

// GetGeneration returns the generation of the component subscription.
func (c *ComponentSubscription) GetGeneration() int64 {
	return c.Generation
}

// ToYamlExport returns the component subscription as a YAML string.
// It can be used to export the component subscription to a file or to pass to kubectl apply.
func (c *ComponentSubscription) ToYamlExport() (string, error) {
	sub := c.DeepCopy()
	gvk := rep1alpha1.GroupVersion.WithKind("ComponentSubscription")
	sub.SetGroupVersionKind(gvk)
	return toYamlExport(sub)
}
