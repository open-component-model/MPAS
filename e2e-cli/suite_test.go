// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

var (
	testEnv           env.Environment
	kindClusterName   string
	defaultghTokenVar = "GITHUB_TOKEN"
	ownerVar          = "MPAS_MANAGEMENT_REPO_OWNER"
	repoVar           = "MPAS_MANAGEMENT_REPO"
)

func TestMain(m *testing.M) {
	// "starting e2e-cli test suite"

	cfg, _ := envconf.NewFromFlags()
	testEnv = env.NewWithConfig(cfg)
	kindClusterName = envconf.RandomName("mpas-e2e-cli", 32)

	testEnv.Setup(
		envfuncs.CreateKindCluster(kindClusterName),
	)

	testEnv.Finish(
		envfuncs.DestroyKindCluster(kindClusterName),
	)

	os.Exit(testEnv.Run(m))
}
