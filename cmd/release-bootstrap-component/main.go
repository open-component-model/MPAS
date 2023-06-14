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
	tokenVar                        = "GITHUB_TOKEN"
	defaultFluxVer                  = "v2.0.0-rc.5"
	defaultOcmControllerVer         = "v0.8.3"
	defaultGitControllerVer         = "v0.4.1"
	defaultReplicationVer           = "v0.3.0"
	defaultMpasProductControllerVer = "v0.1.0"
	defaultMpasProjectControllerVer = "v0.1.1"
	defaultOcmCliVer                = "v0.2.0"
	defaultBoostrapVer              = "v0..O.O-dev.0"
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
		// The version of the git-controller component to use.
		gitControllerVersion string
		// The version of the replication-controller component to use.
		replicationControllerVersion string
		// The version of the mpas-product-controller component to use.
		mpasProductControllerVersion string
		// The version of the mpas-project-controller component to use.
		mpasProjectControllerVersion string
		// The version of the ocm-cli component to use.
		ocmCliVersion string
		// The version of the bootstrap component to use.
		bootstrapVersion string
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
	flag.StringVar(&gitControllerVersion, "git-controller-version", defaultGitControllerVer, "The version of the git-controller component to use.")
	flag.StringVar(&replicationControllerVersion, "replication-controller-version", defaultReplicationVer, "The version of the replication-controller component to use.")
	flag.StringVar(&mpasProductControllerVersion, "mpas-product-controller-version", defaultMpasProductControllerVer, "The version of the mpas-product-controller component to use.")
	flag.StringVar(&mpasProjectControllerVersion, "mpas-project-controller-version", defaultMpasProjectControllerVer, "The version of the mpas-project-controller component to use.")
	flag.StringVar(&ocmCliVersion, "ocm-cli-version", defaultOcmCliVer, "The version of the ocm-cli component to use.")
	flag.StringVar(&bootstrapVersion, "bootstrap-version", defaultBoostrapVer, "The version of the bootstrap component to use.")
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
		case "git-controller":
			component, err = release.ReleaseGitControllerComponent(ctx, gitControllerVersion, username, token, tmpDir, repositoryURL, comp)
			if err != nil {
				fmt.Println("Failed to release git-controller component: ", err)
				os.Exit(1)
			}
		case "replication-controller":
			component, err = release.ReleaseReplicationControllerComponent(ctx, replicationControllerVersion, username, token, tmpDir, repositoryURL, comp)
			if err != nil {
				fmt.Println("Failed to release replication-controller component: ", err)
				os.Exit(1)
			}
		case "mpas-product-controller":
			component, err = release.ReleaseMpasProductControllerComponent(ctx, mpasProductControllerVersion, username, token, tmpDir, repositoryURL, comp)
			if err != nil {
				fmt.Println("Failed to release mpas-product-controller component: ", err)
				os.Exit(1)
			}
		case "mpas-project-controller":
			component, err = release.ReleaseMpasProjectControllerComponent(ctx, mpasProjectControllerVersion, username, token, tmpDir, repositoryURL, comp)
			if err != nil {
				fmt.Println("Failed to release mpas-project-controller component: ", err)
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

	if err := release.ReleaseBootstrapComponent(ctx, generatedComponents, bootstrapVersion, username, token, tmpDir, repositoryURL); err != nil {
		fmt.Println("Failed to release bootstrap component: ", err)
		os.Exit(1)
	}

	fmt.Println("Release of bootstrap component successful.")
}
