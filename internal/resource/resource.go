// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Resource is an interface for Kubernetes resources.
type Resource interface {
	ToClientObject() client.Object
	GetObservedGeneration() int64
	GetGeneration() int64
	GetConditions() []metav1.Condition
}

// ApplyAndWait takes an object implementing client.Object and a function to mutate the object.
// It creates or updates the object and returns the operation performed and an error if one occurred.
// It then waits for the object to reach the desired generation and ready condition.
func ApplyAndWait(ctx context.Context, kubeClient client.Client, resource Resource, interval, timeout time.Duration, mutateFn func() error) (string, error) {
	name := types.NamespacedName{
		Namespace: resource.ToClientObject().GetNamespace(),
		Name:      resource.ToClientObject().GetName(),
	}
	op, err := controllerutil.CreateOrUpdate(ctx, kubeClient, resource.ToClientObject(), mutateFn)
	if err != nil {
		return string(op), fmt.Errorf("failed to create or update %s: %w", name, err)
	}

	err = wait.PollWithContext(ctx, interval, timeout, func(ctx context.Context) (done bool, err error) {
		err = kubeClient.Get(ctx, name, resource.ToClientObject())
		if err != nil {
			return false, fmt.Errorf("failed to get %s: %w", name, err)
		}

		if resource.GetGeneration() == resource.GetObservedGeneration() {
			if c := apimeta.FindStatusCondition(resource.GetConditions(), meta.ReadyCondition); c != nil {
				switch c.Status {
				case metav1.ConditionTrue:
					return true, nil
				case metav1.ConditionFalse:
					return false, fmt.Errorf(c.Message)
				}
			}
		}

		return false, nil
	})

	return string(op), err
}
