//go:build e2e
// +build e2e

// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/open-component-model/mpas/cmd/mpas/bootstrap"
	"github.com/open-component-model/mpas/internal/env"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
