// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/open-component-model/mpas/cmd/release-bootstrap-component/release"
	"github.com/open-component-model/mpas/pkg/ocm"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/repositories/ocireg"
	flag "github.com/spf13/pflag"
)

const (
	tokenVar                        = "GITHUB_TOKEN"
	defaultFluxVer                  = "v2.0.0-rc.5"
	defaultOcmControllerVer         = "v0.8.4"
	defaultGitControllerVer         = "v0.4.1"
	defaultReplicationVer           = "v0.3.1"
	defaultMpasProductControllerVer = "v0.1.0"
	defaultMpasProjectControllerVer = "v0.1.1"
	defaultOcmCliVer                = "v0.2.0"
	Version                         = "v0.1.0"
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
	flag.StringVar(&repositoryURL, "repository-url", "", "The oci repository URL to use.Must be of format <host>/<path>.")
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
	octx := om.New(datacontext.MODE_SHARED)
	target, err := makeTarget(octx, repositoryURL)
	if err != nil {
		fmt.Println("Failed to create target: ", err)
		os.Exit(1)
	}
	defer target.Close()

	fmt.Println("Releasing bootstrap component...")
	tmpDir, err := os.MkdirTemp("", "mpas-bootstrap")
	if err != nil {
		fmt.Println("Failed to create temporary directory: ", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// create transport archive
	ctf, err := ocm.CreateCTF(octx, fmt.Sprintf("%s/%s", tmpDir, "ctf"), accessio.FormatDirectory)
	if err != nil {
		fmt.Println("Failed to create CTF: ", err)
		os.Exit(1)
	}
	defer ctf.Close()

	r := release.New(octx, username, token, tmpDir, repositoryURL, ctf)

	generatedComponents := make(map[string]*ocm.Component)
	for _, comp := range components {
		var component *ocm.Component
		switch comp {
		case "ocm-controller":
			component, err = r.ReleaseOcmControllerComponent(ctx, ocmControllerVersion, comp)
			if err != nil {
				fmt.Println("Failed to release ocm-controller component: ", err)
				os.Exit(1)
			}
		case "flux":
			component, err = r.ReleaseFluxComponent(ctx, fluxVersion, comp)
			if err != nil {
				fmt.Println("Failed to release flux component: ", err)
				os.Exit(1)
			}
		case "git-controller":
			component, err = r.ReleaseGitControllerComponent(ctx, gitControllerVersion, comp)
			if err != nil {
				fmt.Println("Failed to release git-controller component: ", err)
				os.Exit(1)
			}
		case "replication-controller":
			component, err = r.ReleaseReplicationControllerComponent(ctx, replicationControllerVersion, comp)
			if err != nil {
				fmt.Println("Failed to release replication-controller component: ", err)
				os.Exit(1)
			}
		case "mpas-product-controller":
			component, err = r.ReleaseMpasProductControllerComponent(ctx, mpasProductControllerVersion, comp)
			if err != nil {
				fmt.Println("Failed to release mpas-product-controller component: ", err)
				os.Exit(1)
			}
		case "mpas-project-controller":
			component, err = r.ReleaseMpasProjectControllerComponent(ctx, mpasProjectControllerVersion, comp)
			if err != nil {
				fmt.Println("Failed to release mpas-project-controller component: ", err)
				os.Exit(1)
			}
		}
		generatedComponents[comp] = component
	}
	for _, comp := range binaryComponents {
		var component *ocm.Component
		switch comp {
		case "flux-cli":
			component, err = r.ReleaseFluxCliComponent(ctx, fluxVersion, comp, targetOS, targetArch)
			if err != nil {
				fmt.Println("Failed to release flux-cli component: ", err)
				os.Exit(1)
			}
		case "ocm-cli":
			component, err = r.ReleaseOCMCliComponent(ctx, ocmCliVersion, comp, targetOS, targetArch)
			if err != nil {
				fmt.Println("Failed to release ocm-cli component: ", err)
				os.Exit(1)
			}
		}
		generatedComponents[comp] = component
	}

	if err := r.ReleaseBootstrapComponent(ctx, generatedComponents, Version); err != nil {
		fmt.Println("Failed to release bootstrap component: ", err)
		os.Exit(1)
	}

	if err := ocm.Transfer(octx, ctf, target); err != nil {
		fmt.Println("Failed to transfer CTF: ", err)
		os.Exit(1)
	}

	fmt.Println("Release of bootstrap component successful.")
}

func makeTarget(octx om.Context, repositoryURL string) (om.Repository, error) {
	regURL, err := ocm.ParseURL(repositoryURL)
	if err != nil {
		return nil, err
	}

	meta := ocireg.NewComponentRepositoryMeta(strings.TrimPrefix(regURL.Path, "/"), ocireg.OCIRegistryURLPathMapping)
	targetSpec := ocireg.NewRepositorySpec(regURL.Host, meta)
	target, err := octx.RepositoryForSpec(targetSpec)
	if err != nil {
		return nil, err
	}
	return target, nil
}
