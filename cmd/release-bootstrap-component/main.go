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

var (
	components = []string{
		"ocm-controller",
		"flux",
		"flux-cli",
		"ocm-cli",
		"git-controller",
		"replication-controller",
		"mpas-product-controller",
		"mpas-project-controller",
	}
	fluxLocalizationTemplate = `- name: %s
file: gotk-components.yaml
image: spec.template.spec.containers[0].image
resource:
  name: %s
`
	fluxLocalizationTemplateHeader = `apiVersion: config.ocm.software/v1alpha1
kind: ConfigData
metadata:
  name: ocm-config
localization:
`
)

func main() {
	var (
		// The version of the flux component to use.
		fluxVersion string
		// The version of the ocm-controller component to use.
		ocmControllerVersion string
		// The repository URL to use.
		repositoryURL string
		// The username to use.
		username string
	)

	flag.StringVar(&fluxVersion, "flux-version", "", "The version of the flux component to use.")
	flag.StringVar(&ocmControllerVersion, "ocm-controller-version", "", "The version of the ocm-controller component to use.")
	flag.StringVar(&repositoryURL, "repository-url", "", "The repository URL to use.")
	flag.StringVar(&username, "username", "", "The username to use.")

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

	fmt.Println("We are going to package the bootstrap component and ship it as an OCM component.")
	tmpDir, err := os.MkdirTemp("", "mpas-bootstrap")
	if err != nil {
		fmt.Println("Failed to create temporary directory: ", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	var generatedComponents []*ocm.Component
	for _, comp := range components {
		fmt.Println("Component: ", comp)
		var component *ocm.Component
		switch comp {
		case "ocm-controller":
		case "flux":
			f, err := generateFlux(fluxVersion, tmpDir)
			if err != nil {
				fmt.Println("Failed to generate flux manifests: ", err)
				os.Exit(1)
			}
			component = &ocm.Component{
				Context:       clictx.DefaultContext(),
				Name:          "github.com/souleb/flux",
				Version:       fluxVersion,
				Provider:      "fluxcd",
				ArchivePath:   path.Join(tmpDir, fmt.Sprintf("%s-%s", archivePathPrefix, comp)),
				RepositoryURL: repositoryURL,
			}
			if err := component.CreateComponentArchive(); err != nil {
				fmt.Println("Failed to create component archive: ", err)
				os.Exit(1)
			}

			tmpl, err := f.GenerateLocalizationFromTemplate(fluxLocalizationTemplateHeader, fluxLocalizationTemplate)
			if err != nil {
				fmt.Println("Failed to generate localization from template: ", err)
				os.Exit(1)
			}
			images, err := f.GenerateImages()
			if err != nil {
				fmt.Println("Failed to generate images: ", err)
				os.Exit(1)
			}
			fmt.Println(tmpl)
			err = os.WriteFile(path.Join(tmpDir, "config.yaml"), []byte(tmpl), 0644)
			if err != nil {
				fmt.Println("Failed to write config.yaml: ", err)
				os.Exit(1)
			}

			if err := component.AddResource(username, token, ocm.WithResourceName("flux"),
				ocm.WithResourcePath(path.Join(tmpDir, f.Path)),
				ocm.WithResourceType("file"),
				ocm.WithResourceVersion(component.Version)); err != nil {
				fmt.Println("Failed to add resource flux: ", err)
				os.Exit(1)
			}

			if err := component.AddResource(username, token, ocm.WithResourceName("ocm-config"),
				ocm.WithResourcePath(path.Join(tmpDir, "config.yaml")),
				ocm.WithResourceType("file"),
				ocm.WithResourceVersion(component.Version)); err != nil {
				fmt.Println("Failed to add resource ocm-config: ", err)
				os.Exit(1)
			}

			for image, nameVersion := range images {
				fmt.Println("image: ", image)
				fmt.Println("name: ", nameVersion[0])
				fmt.Println("Version: ", nameVersion[1])
				if err := component.AddResource(username, token, ocm.WithResourceName(nameVersion[0]),
					ocm.WithResourceType("ociImage"),
					ocm.WithResourceImage(image),
					ocm.WithResourceVersion(nameVersion[1])); err != nil {
					fmt.Printf("Failed to add resource %s: %v", image, err)
					os.Exit(1)
				}
			}
			fmt.Println("time to transfer")
			if err := component.Transfer(username, token); err != nil {
				fmt.Println("Failed to transfer component: ", err)
				os.Exit(1)
			}
		}
		generatedComponents = append(generatedComponents, component)
	}
}

func generateFlux(version, tmpDir string) (mgen.Flux, error) {
	if version == "" {
		return mgen.Flux{}, fmt.Errorf("flux version is empty")
	}

	f := mgen.Flux{Version: version, Registry: "", Components: nil}
	err := f.GenerateFluxManifests(tmpDir)
	return f, err
}
