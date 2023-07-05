//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"fmt"
	"os"
	"testing"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	env2 "github.com/open-component-model/mpas/pkg/env"
	"github.com/open-component-model/mpas/pkg/printer"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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
	registry              = env2.DefaultBootstrapComponentLocation
	namespace             = "mpas-cli-testns"
	targetPath            = "clusters/my-cluster"
	cfg                   = config.MpasConfig{Timeout: "5m"}
	envConf               *envconf.Config
)

func setCfgPrinter() {
	printer, _ := printer.Newprinter(nil)
	cfg.Printer = printer
	cfg.KubeConfigArgs = genericclioptions.NewConfigFlags(false)
	cfg.DockerconfigPath = "~/.docker/config.json"
}

func TestMain(m *testing.M) {
	setCfgPrinter()
	// "starting e2e-cli test suite"
	var err error
	envConf, err = envconf.NewFromFlags()
	if err != nil {
		fmt.Println("failed to create config from flags: ", err)
		os.Exit(1)
	}
	testEnv = env.NewWithConfig(envConf)
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
