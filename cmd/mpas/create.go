// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/open-component-model/mpas/cmd/mpas/config"
	"github.com/open-component-model/mpas/cmd/mpas/create"
	"github.com/spf13/cobra"
)

// NewCreate returns a new cobra.Command to create resources
func NewCreate(cfg *config.MpasConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [resources] [flags]",
		Short: "create a resource in the Kubernetes cluster.",
		Long:  "create a resource in the Kubernetes cluster.",
	}

	cmd.AddCommand(NewCreateProject(cfg))
	cmd.AddCommand(NewCreateComponentSubscription(cfg))
	cmd.AddCommand(NewCreateProductDeploymentGenerator(cfg))

	return cmd
}

// NewCreateProject returns a new cobra.Command to create a project
func NewCreateProject(cfg *config.MpasConfig) *cobra.Command {
	c := &config.ProjectConfig{}
	cmd := &cobra.Command{
		Use:   "project [flags]",
		Short: "Create a project resource.",
		Example: `  - Create a project in namespace my-namespace
    mpas create project my-project --owner=myUser --personal --provider=github, --secret-ref=github-secret --namespace my-namespace=my-project

    - Create a project with a private user repository
    mpas create project my-project --owner=myUser --personal --provider=github, --secret-ref=github-secret --visibility=private --namespace my-namespace=my-project

    - Create a project an export the project to a file
    mpas create project my-project --owner=myUser --personal --provider=github, --secret-ref=github-secret --namespace my-namespace=my-project --export > my-project.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]
			if name == "" {
				return fmt.Errorf("no project name specified, see mpas create project --help for more information")
			}

			p := create.NewProjectCmd(name, *c)
			return p.Execute(cmd.Context(), cfg)
		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
}

// NewCreateComponentSubscription returns a new cobra.Command to create a component subscription
func NewCreateComponentSubscription(cfg *config.MpasConfig) *cobra.Command {
	c := &config.ComponentSubscriptionConfig{}
	cmd := &cobra.Command{
		Use:     "component-subscription [flags]",
		Aliases: []string{"cs"},
		Short:   "Create a component subscription resource.",
		Example: `  - Create a component subscription in namespace my-namespace
    mpas create component-subscription my-subscription --component=mpas.ocm.software/podinfo --semver=">=v1.0.0" --source-url=ghcr.io/open-component-model/mpas --source-secret-ref=github-access --namespace=my-namespace my-project

    - Create a component subscription an export the project to a file
    mpas create component-subscription my-subscription --component=mpas.ocm.software/podinfo --semver=">=v1.0.0" --source-url=ghcr.io/open-component-model/mpas --source-secret-ref=github-access --namespace my-namespace=my-project --export > my-subscription.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]
			if name == "" {
				return fmt.Errorf("no component subscription name specified, see mpas create component-subscription --help for more information")
			}

			p := create.NewComponentSubscriptionCmd(name, *c)
			return p.Execute(cmd.Context(), cfg)
		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
}

// NewCreateProductDeploymentGenerator returns a new cobra.Command to create a product deployment generator
func NewCreateProductDeploymentGenerator(cfg *config.MpasConfig) *cobra.Command {
	c := &config.ProductDeploymentGeneratorConfig{}
	cmd := &cobra.Command{
		Use:     "product-deployment-generator [flags]",
		Aliases: []string{"pdg"},
		Short:   "Create a product deployment generator resource.",
		Example: `  - Create a product deployment generator in namespace my-namespace
    mpas create product-deployment-generator my-product --service-account=my-sa --subscription-name=my-subscription --subscription-namespace=my-project  --namespace=my-project

    - Create a product deployment generator an export the project to a file
    mpas create product-deployment-generator my-product --service-account=my-sa --subscription-name=my-subscription --subscription-namespace=my-project  --namespace=my-project --export > my-product-generator.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]
			if name == "" {
				return fmt.Errorf("no product deployment generator name specified, see mpas create product-deployment-generator --help for more information")
			}

			p := create.NewProductDeploymentGeneratorCmd(name, *c)
			return p.Execute(cmd.Context(), cfg)
		},
	}

	c.AddFlags(cmd.Flags())

	return cmd
}
