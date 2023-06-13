// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/open-component-model/mpas/cmd/release-bootstrap-component/release"
	"github.com/open-component-model/mpas/pkg/ocm"
	flag "github.com/spf13/pflag"
)

const (
	tokenVar                = "GITHUB_TOKEN"
	defaultFluxVer          = "v2.0.0-rc.5"
	defaultOcmControllerVer = "v0.8.3"
	defaultOcmCliVer        = "v0.2.0"
)

var (
	components = []string{
		"ocm-controller",
		"flux",
		"git-controller",
		"replication-controller",
		"mpas-product-controller",
		"mpas-project-controller",
	}
	binaryComponents = []string{
		"flux-cli",
		"ocm-cli",
	}
)

func main() {
	var (
		// The version of the flux component to use.
		fluxVersion string
		// The version of the ocm-controller component to use.
		ocmControllerVersion string
		// The version of the ocm-cli component to use.
		ocmCliVersion string
		// The repository URL to use.
		repositoryURL string
		// The username to use.
		username string
		// The target os.
		targetOS string
		// The target arch.
		targetArch string
	)

	flag.StringVar(&fluxVersion, "flux-version", defaultFluxVer, "The version of the flux component to use.")
	flag.StringVar(&ocmControllerVersion, "ocm-controller-version", defaultOcmControllerVer, "The version of the ocm-controller component to use.")
	flag.StringVar(&ocmCliVersion, "ocm-cli-version", defaultOcmCliVer, "The version of the ocm-cli component to use.")
	flag.StringVar(&repositoryURL, "repository-url", "", "The repository URL to use.")
	flag.StringVar(&username, "username", "", "The username to use.")
	flag.StringVar(&targetOS, "target-os", "linux", "The target OS to use.")
	flag.StringVar(&targetArch, "target-arch", "amd64", "The target arch to use.")

	flag.Parse()

	token := os.Getenv(tokenVar)
	if token == "" {
		fmt.Println("token must be provided via environment variable")
		os.Exit(1)
	}

	if repositoryURL == "" {
		fmt.Println("repository URL must be provided")
		os.Exit(1)
	}

	if username == "" {
		fmt.Println("username must be provided")
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Println("We are going to package the bootstrap component and ship it as an OCM component.")
	tmpDir, err := os.MkdirTemp("", "mpas-bootstrap")
	if err != nil {
		fmt.Println("Failed to create temporary directory: ", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	var generatedComponents []*ocm.Component
	for _, comp := range components {
		var component *ocm.Component
		switch comp {
		case "ocm-controller":
			component, err = release.ReleaseOcmControllerComponent(ctx, ocmControllerVersion, username, token, tmpDir, repositoryURL, comp)
			if err != nil {
				fmt.Println("Failed to release ocm-controller component: ", err)
				os.Exit(1)
			}
		case "flux":
			component, err = release.ReleaseFluxComponent(ctx, fluxVersion, username, token, tmpDir, repositoryURL, comp)
			if err != nil {
				fmt.Println("Failed to release flux component: ", err)
				os.Exit(1)
			}
		}
		generatedComponents = append(generatedComponents, component)
	}
	for _, comp := range binaryComponents {
		var component *ocm.Component
		switch comp {
		case "flux-cli":
			component, err = release.ReleaseFluxCliComponent(ctx, fluxVersion, username, token, tmpDir, repositoryURL, comp, targetOS, targetArch)
			if err != nil {
				fmt.Println("Failed to release flux-cli component: ", err)
				os.Exit(1)
			}
		case "ocm-cli":
			component, err = release.ReleaseOCMCliComponent(ctx, ocmCliVersion, username, token, tmpDir, repositoryURL, comp, targetOS, targetArch)
			if err != nil {
				fmt.Println("Failed to release ocm-cli component: ", err)
				os.Exit(1)
			}
		}
		generatedComponents = append(generatedComponents, component)
	}
}
