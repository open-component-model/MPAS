// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	env2 "github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/mpas/internal/printer"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
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
	defautHostname        = fmt.Sprintf("gitea.%s.svc.cluster.local:3000", namespace)
	targetPath            = "clusters/my-cluster"
	cfg                   = config.MpasConfig{
		Timeout:          "5m",
		PollInterval:     5 * time.Millisecond,
		DockerconfigPath: "~/.docker/config.json",
		KubeConfigArgs:   genericclioptions.NewConfigFlags(false),
		PlainHTTP:        true,
	}
	envConf *envconf.Config
)

func setCfgPrinter() {
	printer, _ := printer.Newprinter(nil)
	cfg.Printer = printer
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
		envfuncs.CreateNamespace(env2.DefaultOCMNamespace),
		envfuncs.CreateNamespace("mpas-system"),
		envfuncs.CreateNamespace(namespace),
		shared.StartGitServer(namespace),
		shared.ForwardPortForAppName("gitea", 3000, stopChannelGitea),
	)

	// set the kubeconfig namespace
	cfg.KubeConfigArgs.Namespace = &namespace

	testEnv.Finish(
		shared.RemoveGitServer(namespace),
		shared.ShutdownPortForward(stopChannelGitea),
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyKindCluster(kindClusterName),
	)

	// This is required because controller-runtime expects its consumers to
	// set a logger through log.SetLogger within 30 seconds of the program's
	// initalization. If not set, the entire debug stack is printed as an
	// error, see: https://github.com/kubernetes-sigs/controller-runtime/blob/ed8be90/pkg/log/log.go#L59
	// Since we have our own logging and don't care about controller-runtime's
	// logger, we configure it's logger to do nothing.
	ctrllog.SetLogger(logr.New(ctrllog.NullLogSink{}))

	os.Exit(testEnv.Run(m))
}
