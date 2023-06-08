// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path"

	mgen "github.com/open-component-model/mpas/pkg/manifestsgen"
	"github.com/open-component-model/mpas/pkg/ocm"
	"github.com/open-component-model/ocm/pkg/contexts/clictx"
	flag "github.com/spf13/pflag"
)

const (
	archivePathPrefix = "mpas-bootstrap-component"
	tokenVar          = "GITHUB_TOKEN"
)

var components = []string{
	"ocm-controller",
	"flux",
	"flux-cli",
	"ocm-cli",
	"git-controller",
	"replication-controller",
	"mpas-product-controller",
	"mpas-project-controller",
}

func main() {
	var (
		// The version of the flux component to use.
		fluxVersion string
		// The version of the ocm-controller component to use.
		ocmControllerVersion string
		// The repository URL to use.
		repositoryURL string
	)

	flag.StringVar(&fluxVersion, "flux-version", "", "The version of the flux component to use.")
	flag.StringVar(&ocmControllerVersion, "ocm-controller-version", "", "The version of the ocm-controller component to use.")
	flag.StringVar(&repositoryURL, "repository-url", "", "The repository URL to use.")

	flag.Parse()

	token := os.Getenv(tokenVar)
	if token == "" {
		fmt.Println("token must be provided via environment variable")
		os.Exit(1)
	}

	fmt.Println("We are going to package the bootstrap component and ship it as an OCM component.")
	tmpDir, err := os.MkdirTemp("", "mpas-bootstrap")
	if err != nil {
		fmt.Println("Failed to create temporary directory: ", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	var generatedComponents []*ocm.Component
	for _, component := range components {
		fmt.Println("Component: ", component)
		switch component {
		case "ocm-controller":
		case "flux":
			manifests, err := generateFlux(fluxVersion, tmpDir)
			if err != nil {
				fmt.Println("Failed to generate flux manifests: ", err)
				os.Exit(1)
			}
			component := &ocm.Component{
				Context:       clictx.DefaultContext(),
				Name:          "github.com/mpas/flux",
				Version:       fluxVersion,
				Provider:      "fluxcd",
				ArchivePath:   path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, component)),
				RepositoryURL: repositoryURL,
			}
			if err := component.CreateComponentArchive(); err != nil {
				fmt.Println("Failed to create component archive: ", err)
				os.Exit(1)
			}

			if err := component.AddResource(ocm.WithResourceName("flux"),
				ocm.WithResourcePath(path.Join(tmpDir, manifests)),
				ocm.WithResourceType("file"),
				ocm.WithResourceVersion(component.Version)); err != nil {
				fmt.Println("Failed to add resource: ", err)
				os.Exit(1)
			}
			if err := component.ConfigureCredentials(token); err != nil {
				fmt.Println("Failed to configure credentials: ", err)
				os.Exit(1)
			}
			if err := component.Transfer(); err != nil {
				fmt.Println("Failed to transfer component: ", err)
				os.Exit(1)
			}
			generatedComponents = append(generatedComponents, component)
		}
	}
}

func generateFlux(version, tmpDir string) (string, error) {
	if version == "" {
		return "", fmt.Errorf("flux version is empty")
	}

	ver := mgen.FluxVersion(version)
	return ver.GenerateFluxManifests(tmpDir)
}
