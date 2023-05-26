// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package e2ecli

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/open-component-model/mpas/cmd/mpas/bootstrap"
)

func TestBootstrap_github(t *testing.T) {
	owner, repository, token, err := retrieveBootStrapConfigVars()
	if err != nil {
		t.Fatalf("failed to retrieve bootstrap config vars: %v", err)
	}
	bootstrapGithubCmd, err := bootstrapGithub(owner, repository, token)
	if err != nil {
		t.Fatalf("failed to bootstrap mpas-management repository: %v", err)
	}
	if bootstrapGithubCmd == nil {
		t.Fatalf("bootstrapGithubCmd is nil")
	}

	// cleanup
	ctx := context.Background()
	err = bootstrapGithubCmd.Cleanup(ctx)
	if err != nil {
		t.Fatalf("failed to cleanup mpas-management repository: %v", err)
	}

}

func retrieveBootStrapConfigVars() (string, string, string, error) {
	owner := os.Getenv(ownerVar)
	repository := os.Getenv(repoVar)
	token := os.Getenv(defaultghTokenVar)
	if token != "" || owner == "" || repository == "" {
		return "", "", "", fmt.Errorf("owner, repository and token must be set")
	}
	return owner, repository, token, nil
}

func bootstrapGithub(owner, repository, token string) (*bootstrap.BootstrapGithubCmd, error) {
	bootstrapGithubCmd := bootstrap.BootstrapGithubCmd{
		Owner:              owner,
		Repository:         repository,
		Token:              token,
		DestructiveActions: true,
	}

	err := bootstrapGithubCmd.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to execute bootstrapGithubCmd: %w", err)
	}
	return &bootstrapGithubCmd, nil
}
