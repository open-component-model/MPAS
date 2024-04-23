//go:build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/e2e-framework/klient/conf"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"

	"github.com/open-component-model/ocm-e2e-framework/shared"

	mpasenv "github.com/open-component-model/mpas/internal/env"
)

var (
	testEnv env.Environment
	//kindClusterName string
	namespace string
)

func TestMain(m *testing.M) {
	setupLog("starting e2e test suite")

	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	testEnv = env.NewWithConfig(cfg)
	namespace = mpasenv.DefaultOCMNamespace

	stopChannelRegistry := make(chan struct{}, 1)
	stopChannelGitea := make(chan struct{}, 1)

	testEnv.Setup(
		//envfuncs.CreateKindCluster(kindClusterName),
		envfuncs.CreateNamespace(namespace),
		shared.StartGitServer(namespace),
		shared.InstallFlux("latest"),
		RunLocalTilt(),
		shared.ForwardPortForAppName("registry", 5000, stopChannelRegistry),
		shared.ForwardPortForAppName("gitea", 3000, stopChannelGitea),
	)

	testEnv.Finish(
		shared.RemoveGitServer(namespace),
		shared.ShutdownPortForward(stopChannelRegistry),
		shared.ShutdownPortForward(stopChannelGitea),
		envfuncs.DeleteNamespace(namespace),
		//envfuncs.DestroyKindCluster(kindClusterName),
	)
	ctrllog.SetLogger(logr.New(ctrllog.NullLogSink{}))
	os.Exit(testEnv.Run(m))
}
