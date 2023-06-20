// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"os"
	"testing"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/pkg/printer"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

var (
	testEnv               env.Environment
	kindClusterName       string
	defaultghTokenVar     = "GITHUB_TOKEN"
	defaultgiteatTokenVar = "GITEA_TOKEN"
	ownerVar              = "MPAS_MANAGEMENT_REPO_OWNER"
	repository            = "mpas-management-test"
	hostnameVar           = "MPAS_MANAGEMENT_REPO_HOSTNAME"
	namespace             = "mpas-cli-testns"
	cfg                   = config.MpasConfig{Printer: printer.Newprinter("", nil), Timeout: "5m"}
)

func TestMain(m *testing.M) {
	// "starting e2e-cli test suite"

	cfg, _ := envconf.NewFromFlags()
	testEnv = env.NewWithConfig(cfg)
	kindClusterName = envconf.RandomName("mpas-e2e-cli", 32)

	stopChannelGitea := make(chan struct{}, 1)

	testEnv.Setup(
		envfuncs.CreateKindCluster(kindClusterName),
		envfuncs.CreateNamespace(namespace),
		shared.StartGitServer(namespace),
		shared.ForwardPortForAppName("gitea", 3000, stopChannelGitea),
	)

	testEnv.Finish(
		shared.RemoveGitServer(namespace),
		shared.ShutdownPortForward(stopChannelGitea),
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyKindCluster(kindClusterName),
	)

	os.Exit(testEnv.Run(m))
}
