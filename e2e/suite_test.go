//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"github.com/go-logr/logr"
	"os"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"

	"github.com/open-component-model/ocm-e2e-framework/shared"
)

var (
	testEnv         env.Environment
	kindClusterName string
	namespace       string
)

func TestMain(m *testing.M) {
	setupLog("starting e2e test suite")

	cfg, _ := envconf.NewFromFlags()
	testEnv = env.NewWithConfig(cfg)
	kindClusterName = envconf.RandomName("mpas-e2e", 32)
	namespace = "ocm-system"

	stopChannelRegistry := make(chan struct{}, 1)
	stopChannelGitea := make(chan struct{}, 1)

	testEnv.Setup(
		envfuncs.CreateKindCluster(kindClusterName),
		envfuncs.CreateNamespace(namespace),
		shared.StartGitServer(namespace),
		shared.InstallFlux("latest"),
		shared.RunTiltForControllers("ocm-controller", "git-controller", "replication-controller", "mpas-project-controller", "mpas-product-controller"),
		shared.ForwardPortForAppName("registry", 5000, stopChannelRegistry),
		shared.ForwardPortForAppName("gitea", 3000, stopChannelGitea),
	)

	testEnv.Finish(
		//shared.RemoveGitServer(namespace),
		shared.ShutdownPortForward(stopChannelRegistry),
		shared.ShutdownPortForward(stopChannelGitea),
		//envfuncs.DeleteNamespace(namespace),
		//envfuncs.DestroyKindCluster(kindClusterName),
	)
	ctrllog.SetLogger(logr.New(ctrllog.NullLogSink{}))
	os.Exit(testEnv.Run(m))
}
