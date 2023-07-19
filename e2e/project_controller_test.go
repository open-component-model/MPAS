package e2e

import (
	"context"
	"fmt"
	"github.com/fluxcd/pkg/apis/meta"
	fconditions "github.com/fluxcd/pkg/runtime/conditions"
	gcv1alpha1 "github.com/open-component-model/git-controller/apis/mpas/v1alpha1"
	prodv1alpha1 "github.com/open-component-model/mpas-product-controller/api/v1alpha1"
	ocmv1alpha1 "github.com/open-component-model/ocm-controller/api/v1alpha1"
	"github.com/open-component-model/ocm-e2e-framework/shared"
	rcv1alpha1 "github.com/open-component-model/replication-controller/api/v1alpha1"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-component-model/ocm-e2e-framework/shared/steps/assess"
	"github.com/open-component-model/ocm-e2e-framework/shared/steps/setup"
)

type kustomization struct {
	name          string
	path          string
	sourceRefKind string
	sourceRefName string
}

var (
	fluxNamespace      = "flux-system"
	prefix             = "mpas-"
	projectClusterRole = "mpas-projects-clusterrole"
	clusterRoleSuffix  = "-clusterrole"
)

func newProjectFeature(projectName, projectRepoName, gitRepoUrl string) *features.FeatureBuilder {
	project := prefix + projectName
	return features.New("Create Project").
		// Required only for project.yaml file
		Setup(setup.AddGitRepository(mpasManagementRepoName)).
		Assess(fmt.Sprintf("management repository %s has been created", mpasManagementRepoName), assess.CheckRepoExists(mpasManagementRepoName)).
		Setup(setup.AddFilesToGitRepository(setup.File{
			RepoName:       mpasManagementRepoName,
			SourceFilepath: "project.yaml",
			DestFilepath:   "projects/test-001.yaml",
		})).
		Setup(setup.AddFluxSyncForRepo(mpasManagementRepoName, "projects/", namespace)).
		Assess(fmt.Sprintf("check flux gitRepository has been created in %s namespace for reconciling project.yaml", fluxNamespace), checkFluxGitRepositoryReady(mpasManagementRepoName, fluxNamespace)).
		Assess(fmt.Sprintf("check flux kustomizations are configured in %s namespace", fluxNamespace), checkKustomizationsConfiguration(fluxNamespace,
			kustomization{
				name:          mpasManagementRepoName,
				path:          "projects/",
				sourceRefKind: "GitRepository",
				sourceRefName: mpasManagementRepoName,
			})).
		// Validate Namespace
		Assess(fmt.Sprintf("1.1 project namespace %s has been created", project), checkIsNamespaceReady(project)).
		// Validate project RBAC (ServiceAccount, ClusterRole, Role, RoleBindings)
		Assess(fmt.Sprintf("1.2 projects ClusterRole %s exists", projectClusterRole), checkIfClusterRoleExists(projectClusterRole)).
		Assess(fmt.Sprintf("1.3 project service account %s has been created", project), checkIfServiceAccountExists(project)).
		Assess(fmt.Sprintf("1.4 project role %s has been created", project), checkIfRoleExists(project)).
		Assess(fmt.Sprintf("1.5 project RoleBindings %s has been created in namespace %s", project, project), checkIfRoleBindingsExist(project, project)).
		Assess(fmt.Sprintf("1.6 project RoleBindings %s has been created in namespace %s", project+clusterRoleSuffix, project), checkIfRoleBindingsExist(project, project)).
		Assess(fmt.Sprintf("1.7 project RoleBindings %s has been created in namespace %s", project, mpasNamespace), checkIfRoleBindingsExist(project, mpasNamespace)).
		Assess(fmt.Sprintf("1.8 project SA %s can list target and componentsubscription resources in %s namespace", project, mpasNamespace),
			checkSACanListResourcesInNamespace(project, mpasNamespace,
				&prodv1alpha1.TargetList{}, &rcv1alpha1.ComponentSubscriptionList{},
			)).
		Assess(fmt.Sprintf("1.9 project SA %s can create resources in %s namespace", project, project), checkSACanCreateResources(
			project,
			&corev1.Secret{},
			&gcv1alpha1.Repository{},
			&prodv1alpha1.Target{},
			&prodv1alpha1.ProductDeployment{},
			&prodv1alpha1.ProductDeploymentGenerator{},
			&prodv1alpha1.ProductDeploymentPipeline{},
			&rcv1alpha1.ComponentSubscription{},
			&ocmv1alpha1.ComponentVersion{},
			&ocmv1alpha1.Localization{},
			&ocmv1alpha1.Configuration{},
			&sourcev1.OCIRepository{},
			&kustomizev1.Kustomization{},
		)).
		Assess(fmt.Sprintf("1.10 project SA %s can list target and componentsubscription resources in %s namespace", project, project),
			checkSACanListResourcesInNamespace(project, project,
				&prodv1alpha1.TargetList{}, &rcv1alpha1.ComponentSubscriptionList{},
			)).
		// Validate Git Repository & structure
		Assess(fmt.Sprintf("1.11 project repository %s/%s/%s has been created", shared.BaseURL, shared.Owner, projectRepoName), assess.CheckRepoExists(projectRepoName)).
		Assess("1.12 check files are created in project repo", checkRepoFileContent(projectRepoName)).
		// Validate Flux resources for a project
		Assess(fmt.Sprintf("1.13 check flux resources have been created in %s namespace", fluxNamespace), checkFluxGitRepositoryReady(project, mpasNamespace)).
		Assess(fmt.Sprintf("1.14 check flux GitRepository is configured correctly in %s namespace", mpasNamespace), checkGitRepositoryConfiguration(project, strings.Join([]string{gitRepoUrl, shared.Owner, project}, "/"), "main")).
		Assess(fmt.Sprintf("1.15 check flux kustomizations are configured correctly in %s namespace", mpasNamespace), checkKustomizationsConfiguration(mpasNamespace,
			kustomization{
				name:          project + "-subscriptions",
				path:          "subscriptions",
				sourceRefKind: "GitRepository",
				sourceRefName: project,
			},
			kustomization{
				name:          project + "-targets",
				path:          "targets",
				sourceRefKind: "GitRepository",
				sourceRefName: project,
			},
			kustomization{
				name:          project + "-products",
				path:          "products",
				sourceRefKind: "GitRepository",
				sourceRefName: project,
			},
			kustomization{
				name:          project + "-generators",
				path:          "generators",
				sourceRefKind: "GitRepository",
				sourceRefName: project,
			},
		))
}

func checkFluxGitRepositoryReady(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()

		client, err := cfg.NewClient()
		if err != nil {
			t.Fail()
		}

		t.Logf("checking if GitRepository object %s in namespace %s is ready...", name, namespace)
		gr := &sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}

		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			obj, ok := object.(*sourcev1.GitRepository)
			if !ok {
				return false
			}

			return fconditions.IsTrue(obj, meta.ReadyCondition)
		}), wait.WithTimeout(time.Minute*1))

		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkIfServiceAccountExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if service account with name: %s exists", name)
		gr := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*corev1.ServiceAccount)
			if !ok {
				return false
			}
			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkIfClusterRoleExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if cluster role with name: %s exists", name)
		gr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rbacv1.ClusterRole)
			if !ok {
				return false
			}
			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkIfRoleExists(name string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if role with name: %s exists", name)
		gr := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: name},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rbacv1.Role)
			if !ok {
				return false
			}
			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkIfRoleBindingsExist(name string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if rolebinding with name: %s exists", name)
		gr := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			_, ok := object.(*rbacv1.RoleBinding)
			if !ok {
				return false
			}
			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
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
			t.Fatal(err)
			return ctx
		}

		for _, re := range res {
			t.Logf("checking if service account %s:%s can list %s resources in namespace %s...", name, name, re.GetObjectKind().GroupVersionKind(), namespace)
			err := r.WithNamespace(namespace).List(ctx, re)
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Fatal(err)
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
			// The API should attempt to authorize the request first, before validating the object schema
			if err != nil && (apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)) {
				t.Error(err)
			}
		}

		return ctx
	}
}

func checkRepoFileContent(projectRepoName string) features.Func {
	return assess.CheckRepoFileContent(
		assess.File{
			Repository: projectRepoName,
			Path:       "CODEOWNERS",
			Content:    "alice.bobb\nbob.alisson",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "generators/.keep",
			Content:    "",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "products/.keep",
			Content:    "",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "subscriptions/.keep",
			Content:    "",
		},
		assess.File{
			Repository: projectRepoName,
			Path:       "targets/.keep",
			Content:    "",
		},
	)
}

func checkGitRepositoryConfiguration(name, url, branch string) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("checking if git repository with name: %s is ready", name)
		gr := &sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: mpasNamespace},
		}
		err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
			obj, ok := object.(*sourcev1.GitRepository)
			if !ok {
				return false
			}
			if obj.Spec.URL != url {
				t.Errorf("expected GitRepository %s to have URL %s, got %s", name, url, gr.Spec.URL)
				return false
			}

			if obj.Spec.Reference.Branch != branch {
				t.Errorf("expected GitRepository %s to have branch %s, got %s", name, branch, gr.Spec.Reference.Branch)
				return false
			}
			return true
		}), wait.WithTimeout(time.Minute*1))
		if err != nil {
			t.Fatal(err)
		}
		return ctx
	}
}

func checkKustomizationsConfiguration(namespace string, kustomizations ...kustomization) features.Func {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		t.Helper()
		client, err := cfg.NewClient()
		if err != nil {
			t.Fatal(err)
		}

		for _, kustomization := range kustomizations {

			t.Logf("checking if Kustomization %s in namespace %s is ready", kustomization.name, namespace)
			gr := &kustomizev1.Kustomization{
				ObjectMeta: metav1.ObjectMeta{Name: kustomization.name, Namespace: namespace},
			}
			err = wait.For(conditions.New(client.Resources()).ResourceMatch(gr, func(object k8s.Object) bool {
				obj, ok := object.(*kustomizev1.Kustomization)
				if !ok {
					return false
				}
				if obj.Spec.SourceRef.Kind != kustomization.sourceRefKind {
					t.Errorf("expected Kustomization %s to have sourceRef kind %s, got %s", kustomization.name, kustomization.sourceRefKind, obj.Spec.SourceRef.Kind)
					return false
				}

				if obj.Spec.SourceRef.Name != kustomization.sourceRefName {
					t.Errorf("expected Kustomization %s to have sourceRef name %s, got %s", kustomization.name, kustomization.sourceRefName, obj.Spec.SourceRef.Name)
					return false
				}

				if obj.Spec.Path != kustomization.path {
					t.Errorf("expected Kustomization %s to have path %s, got %s", kustomization.name, kustomization.path, obj.Spec.Path)
					return false
				}
				return true
			}), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
		}
		return ctx
	}
}

//func checkClusterRoleExists(name string) features.Func {
//	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
//		t.Helper()
//
//		r, err := resources.New(c.Client().RESTConfig())
//		if err != nil {
//			t.Error(err)
//			return ctx
//		}
//
//		t.Logf("checking if cluster role %s exists...", name)
//		cr := &rbacv1.ClusterRole{}
//		if err := r.Get(ctx, name, "", cr); err != nil {
//			t.Error(err)
//			return ctx
//		}
//
//		return ctx
//	}
//}

//func checkClusterRoleBindingReady(name string) features.Func {
//	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
//		t.Helper()
//
//		r, err := resources.New(c.Client().RESTConfig())
//		if err != nil {
//			t.Error(err)
//			return ctx
//		}
//
//		t.Logf("checking if cluster role binding %s exists...", name)
//		crb := &rbacv1.ClusterRoleBinding{}
//		if err := r.Get(ctx, name, "", crb); err != nil {
//			t.Error(err)
//			return ctx
//		}
//
//		return ctx
//	}
//}

//func checkGitRepositoryConfiguration(name string, url string, branch string) features.Func {
//	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
//		t.Helper()
//
//		r, err := resources.New(c.Client().RESTConfig())
//		if err != nil {
//			t.Error(err)
//			return ctx
//		}
//
//		t.Logf("checking if GitRepository %s has been configured correctly...", name)
//		gr := &sourcev1.GitRepository{}
//		if err := r.Get(ctx, name, mpasNamespace, gr); err != nil {
//			t.Error(err)
//			return ctx
//		}
//
//		if gr.Spec.URL != url {
//			t.Errorf("expected GitRepository %s to have URL %s, got %s", name, url, gr.Spec.URL)
//		}
//
//		if gr.Spec.Reference.Branch != branch {
//			t.Errorf("expected GitRepository %s to have branch %s, got %s", name, branch, gr.Spec.Reference.Branch)
//		}
//		return ctx
//	}
//}

//	func checkKustomizationsConfiguration(kustomiations ...kustomization) features.Func {
//		return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
//			t.Helper()
//
//			r, err := resources.New(c.Client().RESTConfig())
//			if err != nil {
//				t.Error(err)
//				return ctx
//			}
//
//			for _, ku := range kustomiations {
//				t.Logf("checking if Kustomization %s has been configured correctly...", ku.name)
//				k := &kustomizev1.Kustomization{}
//				if err := r.Get(ctx, ku.name, "flux-system", k); err != nil {
//					t.Error(err)
//					return ctx
//				}
//
//				if k.Spec.SourceRef.Kind != ku.sourceRefKind {
//					t.Errorf("expected Kustomization %s to have sourceRef kind %s, got %s", ku.name, ku.sourceRefKind, k.Spec.SourceRef.Kind)
//				}
//
//				if k.Spec.SourceRef.Name != ku.sourceRefName {
//					t.Errorf("expected Kustomization %s to have sourceRef name %s, got %s", ku.name, ku.sourceRefName, k.Spec.SourceRef.Name)
//				}
//
//				if k.Spec.Path != ku.path {
//					t.Errorf("expected Kustomization %s to have path %s, got %s", ku.name, ku.path, k.Spec.Path)
//				}
//			}
//
//			return ctx
//		}
//	}

//	func checkServiceAccountReady(name string) features.Func {
//		return func(ctx context.Context, t *testing.T, env *envconf.Config) context.Context {
//			t.Helper()
//			r, err := resources.New(env.Client().RESTConfig())
//			if err != nil {
//				t.Error(err)
//				return ctx
//			}
//			t.Logf("checking if service account %s exists...", name)
//			sa := &corev1.ServiceAccount{}
//			if err := r.Get(ctx, name, name, sa); err != nil {
//				t.Error(err)
//				return ctx
//			}
//
//			return ctx
//		}
//	}
