package e2e

import (
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

func newProductFeature(mpasRepoName, mpasNamespace, projectRepoName string) *features.FeatureBuilder {
	// Product deployment
	product := features.New("Reconcile product deployment")

	// Setup
	product.
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       mpasRepoName,
			SourceFilepath: "product_deployment_generator.yaml",
			DestFilepath:   "generators/mpas-podinfo-001.yaml",
		})).
		Assess("management repository has been created", assess.CheckRepoExists(mpasRepoName)).
		Assess("management namespace has been created", checkNamespaceReady(mpasNamespace)).Feature()

	// Pre-flight check
	product.
		Assess("project repository has been created", assess.CheckRepoExists(projectRepoName)).
		Assess("check files are created in project repo", assess.CheckRepoFileContent(
			assess.File{
				Repository: projectRepoName,
				Path:       "CODEOWNERS",
				Content:    "@alive.bobb\n@bob.alisson",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "generators/.gitkeep",
				Content:    "",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "products/.gitkeep",
				Content:    "",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "subscriptions/.gitkeep",
				Content:    "",
			},
			assess.File{
				Repository: projectRepoName,
				Path:       "targets/.gitkeep",
				Content:    "",
			},
		))

	// Validate K8s resources for project created correctly

	// Apply Product Deployment Generator

	// Check Creation of Product Deployment object

	// Validate podinfo deployment

	return product
}
