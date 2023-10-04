// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/bootstrap"
	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/cmd/mpas/create"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var prjYaml = `---
apiVersion: mpas.ocm.software/v1alpha1
kind: Project
metadata:
  name: test
  namespace: %s
spec:
  flux:
    interval: 5m0s
  git:
    credentials:
      secretRef:
        name: test-secret
    interval: 5m0s
    isOrganization: false
    owner: %s
    provider: gitea
  interval: 5m0s
`

var prdYaml = `---
apiVersion: mpas.ocm.software/v1alpha1
kind: ProductDeploymentGenerator
metadata:
  name: test
  namespace: %s
spec:
  interval: 5m0s
  serviceAccountName: test-sa
  subscriptionRef:
    name: test
    namespace: %s
`

var csubYaml = `---
apiVersion: delivery.ocm.software/v1alpha1
kind: ComponentSubscription
metadata:
  name: test
  namespace: %s
spec:
  component: mpas.ocm.software/podinfo
  interval: 5m0s
  semver: '>=v1.0.0'
  source:
    secretRef:
      name: test-secret
    url: ghcr.io/open-component-model/mpas
`

func TestBootstrap_github(t *testing.T) {
	owner, token, err := retrieveBootStrapConfigVars()
	require.NoError(t, err)
	bootstrapGithubCmd, err := bootstrapGithub(owner, token)
	require.NoError(t, err)
	assert.NotNil(t, bootstrapGithubCmd)

	// cleanup
	ctx := context.Background()
	err = bootstrapGithubCmd.Cleanup(ctx)
	require.NoError(t, err)
}

func TestBootstrap_gitea(t *testing.T) {
	owner := os.Getenv(ownerVar)
	if owner == "" {
		owner = shared.Owner
	}
	token := os.Getenv(defaultgiteatTokenVar)
	if token == "" {
		token = shared.TestUserToken
	}
	hostname := os.Getenv(hostnameVar)
	if hostname == "" {
		hostname = shared.BaseURL
	}
	bootstrapGiteaCmd, err := bootstrapGitea(owner, token, hostname)
	require.NoError(t, err)
	assert.NotNil(t, bootstrapGiteaCmd)

	//set export falg
	cfg.Export = true

	prjOut := strings.Builder{}
	cfg.Printer.SetOutput(&prjOut)
	prjCmd := createProject("test", "gitea", owner, "test-secret", true)
	err = prjCmd.Execute(context.Background(), &cfg)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(prjYaml, namespace, owner), prjOut.String())

	csubOut := strings.Builder{}
	cfg.Printer.SetOutput(&csubOut)
	csubCmd := createComponentSubscription("test", "test-secret")
	err = csubCmd.Execute(context.Background(), &cfg)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(csubYaml, namespace), csubOut.String())

	prdOut := strings.Builder{}
	cfg.Printer.SetOutput(&prdOut)
	pdgCmd := createProductDeploymentGenerator("test", "test-sa")
	err = pdgCmd.Execute(context.Background(), &cfg)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(prdYaml, namespace, namespace), prdOut.String())

	// cleanup
	ctx := context.Background()
	err = bootstrapGiteaCmd.Cleanup(ctx)
	require.NoError(t, err)
}

func retrieveBootStrapConfigVars() (string, string, error) {
	owner := os.Getenv(ownerVar)
	token := os.Getenv(defaultghTokenVar)
	if token != "" || owner == "" {
		return "", "", fmt.Errorf("owner and token must be set")
	}
	return owner, token, nil
}

func bootstrapGithub(owner, token string) (*bootstrap.GithubCmd, error) {
	ctx := context.Background()
	bootstrapGithubCmd := bootstrap.GithubCmd{
		Owner:              owner,
		Repository:         repository,
		Token:              token,
		Path:               targetPath,
		Registry:           registry,
		Components:         env.Components,
		DockerconfigPath:   cfg.DockerconfigPath,
		DestructiveActions: true,
	}

	// set kubeconfig
	kubeconfig := envConf.KubeconfigFile()
	fmt.Println("kubeconfig: ", kubeconfig)
	cfg.KubeConfigArgs.KubeConfig = &kubeconfig

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, err
	}

	interval, err := time.ParseDuration("1s")
	if err != nil {
		return nil, err
	}

	bootstrapGithubCmd.Timeout = timeout
	bootstrapGithubCmd.Interval = interval

	err = bootstrapGithubCmd.Execute(ctx, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bootstrapGithubCmd: %w", err)
	}
	return &bootstrapGithubCmd, nil
}

func bootstrapGitea(owner, token, hostname string) (*bootstrap.GiteaCmd, error) {
	ctx := context.Background()
	bootstrapGiteaCmd := bootstrap.GiteaCmd{
		Owner:              owner,
		Repository:         repository,
		Token:              token,
		Hostname:           hostname,
		Personal:           true,
		Path:               targetPath,
		Registry:           registry,
		TestURL:            fmt.Sprintf("http://%s/%s/%s", defautHostname, owner, repository),
		Components:         env.Components,
		DockerconfigPath:   cfg.DockerconfigPath,
		DestructiveActions: true,
	}

	// set kubeconfig
	kubeconfig := envConf.KubeconfigFile()
	cfg.KubeConfigArgs.KubeConfig = &kubeconfig

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, err
	}

	interval, err := time.ParseDuration("1s")
	if err != nil {
		return nil, err
	}

	bootstrapGiteaCmd.Timeout = timeout
	bootstrapGiteaCmd.Interval = interval

	err = bootstrapGiteaCmd.Execute(ctx, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bootstrapGiteaCmd: %w", err)
	}
	return &bootstrapGiteaCmd, nil
}

func createProject(name, provider, owner, secret string, personal bool) *create.ProjectCmd {
	return create.NewProjectCmd(name, config.ProjectConfig{
		Provider:  provider,
		Owner:     owner,
		SecretRef: secret,
		Personal:  personal,
		CreateConfig: config.CreateConfig{
			Interval: "5m",
		},
	})
}

func createComponentSubscription(name, secret string) *create.ComponentSubscriptionCmd {
	return create.NewComponentSubscriptionCmd(name, config.ComponentSubscriptionConfig{
		Component:       "mpas.ocm.software/podinfo",
		Semver:          ">=v1.0.0",
		SourceUrl:       "ghcr.io/open-component-model/mpas",
		SourceSecretRef: secret,
		CreateConfig: config.CreateConfig{
			Interval: "5m",
		},
	})
}

func createProductDeploymentGenerator(name, sa string) *create.ProductDeploymentGeneratorCmd {
	return create.NewProductDeploymentGeneratorCmd(name, config.ProductDeploymentGeneratorConfig{
		SubscriptionName:      name,
		SubscriptionNamespace: namespace,
		ServiceAccount:        sa,
		CreateConfig: config.CreateConfig{
			Interval: "5m",
		},
	})
}
