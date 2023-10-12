package bootstrap

import (
	"context"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/ocm-controller/pkg/fakes"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
)

type mockRepository struct {
	ocm.Repository
	name    string
	version string

	cv []*mockComponentAccess
}

func (m *mockRepository) LookupComponent(name string) (ocm.ComponentAccess, error) {
	for _, ca := range m.cv {
		if ca.name == name {
			return ca, nil
		}
	}

	return nil, fmt.Errorf("%s not found in configured components", name)
}

var _ ocm.Repository = &mockRepository{}

// ************** Mock Component Access Values and Functions **************

type mockComponentAccess struct {
	ocm.ComponentAccess

	versions []string
	name     string
	cva      map[string]*fakes.Component
}

func (m *mockComponentAccess) ListVersions() ([]string, error) {
	return m.versions, nil
}

func (m *mockComponentAccess) LookupVersion(version string) (ocm.ComponentVersionAccess, error) {
	cva, ok := m.cva[version]
	if !ok {
		return nil, fmt.Errorf("component with version %s not found in mocks", version)
	}

	return cva, nil
}

var _ ocm.ComponentAccess = &mockComponentAccess{}

type mockGitRepository struct {
	gitprovider.UserRepository

	commitClient gitprovider.CommitClient
}

var _ gitprovider.UserRepository = &mockGitRepository{}

func (m *mockGitRepository) Commits() gitprovider.CommitClient {
	return m.commitClient
}

type mockCommitClient struct {
	gitprovider.CommitClient

	commit gitprovider.Commit

	calledWidth [][]any
}

var _ gitprovider.CommitClient = &mockCommitClient{}

func (m *mockCommitClient) Create(ctx context.Context, branch string, message string, files []gitprovider.CommitFile) (gitprovider.Commit, error) {
	m.calledWidth = append(m.calledWidth, []any{branch, message, files})

	return m.commit, nil
}

type mockCommit struct {
	sha string

	gitprovider.Commit
}

func (m *mockCommit) Get() gitprovider.CommitInfo {
	return gitprovider.CommitInfo{
		Sha: m.sha,
	}
}

var _ gitprovider.Commit = &mockCommit{}

type mockKustomizer struct {
	out []byte
	err error
}

var kustomizedDeployment = []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: git-controller
  namespace: ocm-system
spec:
  selector:
    matchLabels:
      app: git-controller
  replicas: 1
  template:
    metadata:
      labels:
        app: git-controller
    spec:
      containers:
      - name: manager
        image: ghcr.io/user/git-controller:v1.0.0
`)

func (m *mockKustomizer) GenerateKustomizedResourceData(component string) ([]byte, error) {
	return m.out, m.err
}

var _ Kustomizer = &mockKustomizer{}
