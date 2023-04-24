package e2e

import (
	"context"
	"fmt"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	projv1alpha1 "github.com/open-component-model/mpas-project-controller/api/v1alpha1"
	ocmv1alpha1 "github.com/open-component-model/ocm-controller/api/v1alpha1"
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

func newProjectFeature(mpasRepoName, mpasNamespace, projectName, projectRepoName string) features.Feature {
	// Setup and management resources
	fb := features.New("Create Project").
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       mpasRepoName,
			SourceFilepath: "project.yaml",
			DestFilepath:   "projects/test-001.yaml",
		})).
		Assess("management repository has been created", assess.CheckRepoExists(mpasRepoName)).
		Assess("management namespace has been created", checkNamespaceReady(mpasNamespace)).
		// The projects ClusterRole will provide permissions for all projects managed by the projects-controller
		// via a ClusterRoleBinding for each project ServiceAccount.
		Assess("projects ClusterRole exists", checkClusterRoleExists("mpas-project-clusterrole"))

	// Validate project repo created with correct file structure
	fb = fb.Assess("project repository has been created", assess.CheckRepoExists(projectRepoName)).
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
	fb = fb.Assess("project namespace has been created", checkNamespaceReady(projectName)).
		// Validate project RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding)
		Assess("project service account has been created", checkServiceAccountReady(projectName)).
		Assess("project ClusterRoleBinding has been created", checkClusterRoleBindingReady(projectName)).
		Assess("project SA can list target and subscription resources in MPAS namespace",
			checkSACanListResourcesInNamespace(mpasNamespace, projectName,
				&rcv1alpha1.ComponentSubscriptionList{}, &projv1alpha1.TargetList{},
			)).
		Assess("project SA can create resources in project namespace", checkSACanCreateResources(
			projectName,
			&rcv1alpha1.ComponentSubscription{},
			&prodv1alpha1.ProductDeploymentGenerator{},
			&prodv1alpha1.ProductDeployment{},
			&prodv1alpha1.ProductDeploymentPipeline{},
			&projv1alpha1.Target{},
			&ocmv1alpha1.ComponentVersion{},
			&ocmv1alpha1.Localization{},
			&ocmv1alpha1.Configuration{},
			&kustomizev1.Kustomization{},
			&sourcev1.OCIRepository{},
			&corev1.Secret{},
		))

	// Validate Flux resources are created correctly
	fb = fb.Assess("flux resources have been created", checkFluxResourcesReady(projectName)).
		Assess("flxu GitRepository is configured correctly", checkGitRepositoryConfiguration(projectName, projectRepoName, "main")).
		Assess("flux Kustomization (subscriptions) is configured correctly",
			checkKustomizationConfiguration(projectName+"-subscriptions", "./subscriptions", "GitRepository", projectName)).
		Assess("flux Kustomization (targets) is configured correctly",
			checkKustomizationConfiguration(projectName+"-targets", "./targets", "GitRepository", projectName)).
		Assess("flux Kustomization (products) is configured correctly",
			checkKustomizationConfiguration(projectName+"-products", "./products", "GitRepository", projectName)).
		Assess("flux Kustomization (generators) is configured correctly",
			checkKustomizationConfiguration(projectName+"-generators", "./generators", "GitRepository", projectName))

	return fb.Feature()
}

func checkServiceAccountReady(name string) features.Func {
	return func(ctx context.Context, t *testing.T, env *envconf.Config) context.Context {
		t.Helper()

		r, err := resources.New(env.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if service account %s exists...", name)
		sa := &corev1.ServiceAccount{}
		if err := r.Get(ctx, name, name, sa); err != nil {
			t.Error(err)
			return ctx
		}

		return ctx
	}
}

func checkClusterRoleExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if cluster role %s exists...", name)
		cr := &rbacv1.ClusterRole{}
		if err := r.Get(ctx, name, "", cr); err != nil {
			t.Error(err)
			return ctx
		}

		return ctx
	}
}

func checkClusterRoleBindingReady(name string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if cluster role binding %s exists...", name)
		crb := &rbacv1.ClusterRoleBinding{}
		if err := r.Get(ctx, name, "", crb); err != nil {
			t.Error(err)
			return ctx
		}

		return ctx
	}
}

func checkSACanListResourcesInNamespace(name, namespace string, res ...k8s.ObjectList) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		cfg := c.Client().RESTConfig()
		cfg.Impersonate.UserName = fmt.Sprintf("system:serviceaccount:%s:%s", name, name)

		r, err := resources.New(cfg)
		if err != nil {
			t.Error(err)
			return ctx
		}

		for _, re := range res {
			t.Logf("checking if service account %s:%s can list %s resources in namespace %s...", name, name, re.GetObjectKind().GroupVersionKind(), namespace)
			err := r.WithNamespace(namespace).List(ctx, re)
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Error(err)
			}
		}

		return ctx
	}
}

func checkSACanCreateResources(name string, res ...k8s.Object) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		cfg := c.Client().RESTConfig()
		cfg.Impersonate.UserName = fmt.Sprintf("system:serviceaccount:%s:%s", name, name)

		r, err := resources.New(cfg)
		if err != nil {
			t.Error(err)
			return ctx
		}

		for _, re := range res {
			re.SetName(name)
			re.SetNamespace(name)
			t.Logf("checking if service account %s:%s can create %s resources...", name, name, re.GetObjectKind().GroupVersionKind())
			err := r.Create(ctx, re)
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Error(err)
			}
		}

		return ctx
	}
}

func checkGitRepositoryConfiguration(name string, url string, branch string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if GitRepository %s has been configured correctly...", name)
		gr := &sourcev1.GitRepository{}
		if err := r.Get(ctx, name, "flux-system", gr); err != nil {
			t.Error(err)
			return ctx
		}

		if gr.Spec.URL != url {
			t.Errorf("expected GitRepository %s to have URL %s, got %s", name, url, gr.Spec.URL)
		}

		if gr.Spec.Reference.Branch != branch {
			t.Errorf("expected GitRepository %s to have branch %s, got %s", name, branch, gr.Spec.Reference.Branch)
		}

		return ctx
	}
}

func checkKustomizationConfiguration(name string, path string, sourceRefKind string, sourceRefName string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if Kustomization %s has been configured correctly...", name)
		k := &kustomizev1.Kustomization{}
		if err := r.Get(ctx, name, "flux-system", k); err != nil {
			t.Error(err)
			return ctx
		}

		if k.Spec.SourceRef.Kind != sourceRefKind {
			t.Errorf("expected Kustomization %s to have sourceRef kind %s, got %s", name, sourceRefKind, k.Spec.SourceRef.Kind)
		}

		if k.Spec.SourceRef.Name != sourceRefName {
			t.Errorf("expected Kustomization %s to have sourceRef name %s, got %s", name, sourceRefName, k.Spec.SourceRef.Name)
		}

		if k.Spec.Path != path {
			t.Errorf("expected Kustomization %s to have path %s, got %s", name, path, k.Spec.Path)
		}

		return ctx
	}
}
