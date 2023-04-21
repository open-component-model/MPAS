package e2e

import (
	"context"
	"fmt"
	"testing"

	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

func newProjectFeature(mpasRepoName, mpasNamespace, projectName, projectRepoName string) features.Feature {
	feature := features.New("Create Project").
		Setup(setup.AddFileToGitRepository(mpasRepoName, "project.yaml", "projects/test-001.yaml")).
		Assess("management repository has been created", assess.CheckRepoExists(mpasRepoName)).
		Assess("management namespace has been created", checkNamespaceReady(mpasNamespace)).
		// The projects ClusterRole will provide permissions for all projects managed by the projects-controller
		// via a ClusterRoleBinding for each project ServiceAccount.
		Assess("projects ClusterRole exists", checkClusterRoleExists("mpas-project-clusterrole")).
		Assess("project namespace has been created", checkNamespaceReady(projectName)).
		// Validate project repo created with correct file structure.
		Assess("project repository has been created", assess.CheckRepoExists(projectRepoName)).
		Assess("CODEOWNERS file exists with maintainers",
			assess.CheckRepoFileContent(projectRepoName, "CODEOWNERS", "@alive.bobb\n@bob.alisson")).
		Assess("generators folder exists in repo",
			assess.CheckRepoFileContent(projectRepoName, "generators/.gitkeep", "")).
		Assess("products folder exists in repo",
			assess.CheckRepoFileContent(projectRepoName, "products/.gitkeep", "")).
		Assess("subscriptions folder exists in repo",
			assess.CheckRepoFileContent(projectRepoName, "subscriptions/.gitkeep", "")).
		Assess("targets folder exists in repo",
			assess.CheckRepoFileContent(projectRepoName, "targets/.gitkeep", "")).
		// Validate project RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding)
		Assess("project service account has been created", checkServiceAccountReady(projectName)).
		Assess("project ClusterRoleBinding has been created", checkClusterRoleBindingReady(projectName)).
		Assess("project SA can list resources in MPAS namespace",
			canSAListResourcesInNamespace(mpasNamespace, projectName)).
		Assess("flux resources have been created", checkFluxResourcesReady(projectName)).
		Feature()

	return feature
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

func canSAListResourcesInNamespace(namespace, saName string) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		t.Helper()

		cfg := c.Client().RESTConfig()
		cfg.Impersonate.UserName = fmt.Sprintf("system:serviceaccount:%s:%s", saName, saName)

		r, err := resources.New(cfg)
		if err != nil {
			t.Error(err)
			return ctx
		}

		t.Logf("checking if service account %s:%s can list resources in namespace %s...", saName, saName, namespace)
		pods := &corev1.PodList{}
		if err := r.WithNamespace(namespace).List(ctx, pods); err != nil {
			t.Error(err)
			return ctx
		}

		return ctx
	}
}
