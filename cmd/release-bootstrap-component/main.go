// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/open-component-model/mpas/cmd/release-bootstrap-component/release"
	"github.com/open-component-model/mpas/pkg/env"
	"github.com/open-component-model/mpas/pkg/fs"
	"github.com/open-component-model/mpas/pkg/oci"
	"github.com/open-component-model/mpas/pkg/ocm"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	om "github.com/open-component-model/ocm/pkg/contexts/ocm"
	flag "github.com/spf13/pflag"
)

const (
	Version = "v0.0.1"
)

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

func main() {
	flag.StringVar(&fluxVersion, "flux-version", env.DefaultFluxVer, "The version of the flux component to use.")
	flag.StringVar(&ocmControllerVersion, "ocm-controller-version", env.DefaultOcmControllerVer, "The version of the ocm-controller component to use.")
	flag.StringVar(&gitControllerVersion, "git-controller-version", env.DefaultGitControllerVer, "The version of the git-controller component to use.")
	flag.StringVar(&replicationControllerVersion, "replication-controller-version", env.DefaultReplicationVer, "The version of the replication-controller component to use.")
	flag.StringVar(&mpasProductControllerVersion, "mpas-product-controller-version", env.DefaultMpasProductControllerVer, "The version of the mpas-product-controller component to use.")
	flag.StringVar(&mpasProjectControllerVersion, "mpas-project-controller-version", env.DefaultMpasProjectControllerVer, "The version of the mpas-project-controller component to use.")
	flag.StringVar(&ocmCliVersion, "ocm-cli-version", env.DefaultOcmCliVer, "The version of the ocm-cli component to use.")
	flag.StringVar(&repositoryURL, "repository-url", "", "The oci repository URL to use.Must be of format <host>/<path>.")
	flag.StringVar(&username, "username", "", "The username to use.")
	flag.StringVar(&targetOS, "target-os", "linux", "The target OS to use.")
	flag.StringVar(&targetArch, "target-arch", "amd64", "The target arch to use.")

	flag.Parse()

	token := os.Getenv(env.GithubTokenVar)
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

	fmt.Println("Releasing bootstrap component...")
	tmpDir, err := os.MkdirTemp("", "mpas-bootstrap")
	if err != nil {
		fmt.Println("Failed to create temporary directory: ", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	target, err := ocm.MakeOCIRepository(octx, repositoryURL)
	if err != nil {
		fmt.Println("Failed to create target: ", err)
		os.Exit(1)
	}
	defer target.Close()

	ctfPath := fmt.Sprintf("%s/%s", tmpDir, "ctf")
	if err := releaseComponents(ctx, octx, token, tmpDir, ctfPath, target); err != nil {
		fmt.Println("Failed to release components: ", err)
		os.Exit(1)
	}

	src, err := fs.CreateArchive(ctfPath, "mpas-bundle.tar.gz")
	if err != nil {
		fmt.Println("Failed to create bundle archive: ", err)
		os.Exit(1)
	}

	if err := oci.PushArtifact(ctx, repositoryURL+"-bundle", src, username, token, Version); err != nil {
		fmt.Println("Failed to push bundle: ", err)
		os.Exit(1)
	}

	// if err := oci.PullArtifact(ctx, "ghcr.io/souleb/mpas-bundle", username, token, Version); err != nil {
	// 	fmt.Println("Failed to pull bundle: ", err)
	// 	os.Exit(1)
	// }

	fmt.Println("Release of bootstrap component successful.")
}

func releaseComponents(ctx context.Context, octx om.Context, token, tmpDir, ctfPath string, target om.Repository) error {
	// create transport archive
	ctf, err := ocm.CreateCTF(octx, ctfPath, accessio.FormatDirectory)
	if err != nil {
		fmt.Println("Failed to create CTF: ", err)
		os.Exit(1)
	}
	defer ctf.Close()

	r := release.New(octx, username, token, tmpDir, repositoryURL, ctf)

	generatedComponents := make(map[string]*ocm.Component)
	for _, comp := range env.Components {
		var component *ocm.Component
		switch comp {
		case env.OcmControllerName:
			component, err = r.ReleaseOcmControllerComponent(ctx, ocmControllerVersion, comp)
			if err != nil {
				fmt.Printf("Failed to release %s component: %v\n", comp, err)
				os.Exit(1)
			}
		case env.FluxName:
			component, err = r.ReleaseFluxComponent(ctx, fluxVersion, comp)
			if err != nil {
				fmt.Printf("Failed to release %s component: %v\n", comp, err)
				os.Exit(1)
			}
		case env.GitControllerName:
			component, err = r.ReleaseGitControllerComponent(ctx, gitControllerVersion, comp)
			if err != nil {
				fmt.Printf("Failed to release %s component: %v\n", comp, err)
				os.Exit(1)
			}
		case env.ReplicationControllerName:
			component, err = r.ReleaseReplicationControllerComponent(ctx, replicationControllerVersion, comp)
			if err != nil {
				fmt.Printf("Failed to release %s component: %v\n", comp, err)
				os.Exit(1)
			}
		case env.MpasProductControllerName:
			component, err = r.ReleaseMpasProductControllerComponent(ctx, mpasProductControllerVersion, comp)
			if err != nil {
				fmt.Printf("Failed to release %s component: %v\n", comp, err)
				os.Exit(1)
			}
		case env.MpasProjectControllerName:
			component, err = r.ReleaseMpasProjectControllerComponent(ctx, mpasProjectControllerVersion, comp)
			if err != nil {
				fmt.Printf("Failed to release %s component: %v\n", comp, err)
				os.Exit(1)
			}
		}
		generatedComponents[comp] = component
	}
	for _, comp := range env.BinaryComponents {
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
	return nil
}
