// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-component-model/ocm-controller/api/v1alpha1"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

func TestProjectCreation(t *testing.T) {
	t.Log("running project creation test")

	repositoryName := envconf.RandomName("mpas", 32)

	f := features.New("Create Project").
		Setup(setup.AddSchemeAndNamespace(v1alpha1.AddToScheme, namespace)).
		Assess("check that repository has been created", assess.CheckRepoExists(repositoryName))

	testEnv.Test(t, f.Feature())
}
