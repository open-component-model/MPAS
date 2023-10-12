package bootstrap

import (
	"context"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
)

func TestCertManagerInstall(t *testing.T) {
	temp := t.TempDir()
	mc := &mockCommitClient{
		commit: &mockCommit{
			sha: "sha",
		},
	}
	repo := &mockGitRepository{
		commitClient: mc,
	}
	c := &certManagerInstall{
		componentName: "ocm.software/mpas/test-component",
		version:       "v1.0.1",
		repository:    nil, // gone through mocks
		certManagerOptions: &certManagerOptions{
			gitRepository: repo,
			dir:           temp,
			branch:        "main",
			targetPath:    "target",
			namespace:     "ocm-system",
			provider:      "gitea",
		},
		kustomizer: &mockKustomizer{
			out: kustomizedDeployment,
		},
	}

	sha, err := c.Install(context.Background(), "ocm.software/mpas/test-component")
	require.NoError(t, err)
	assert.Equal(t, "sha", sha)

	require.Lenf(t, mc.calledWidth, 1, "exactly one call expected from mock client, but was %d", len(mc.calledWidth))
	args := mc.calledWidth[0]
	assert.Equal(t, "main", args[0])
	assert.Equal(t, "Add ocm.software/mpas/test-component v1.0.1 manifests", args[1])
	assert.Equal(t, []gitprovider.CommitFile{
		{
			Path:    pointer.String("target/ocm-system/test-component.yaml"),
			Content: pointer.String("YXBpVmVyc2lvbjogYXBwcy92MQpraW5kOiBEZXBsb3ltZW50Cm1ldGFkYXRhOgogIG5hbWU6IGdpdC1jb250cm9sbGVyCiAgbmFtZXNwYWNlOiBvY20tc3lzdGVtCnNwZWM6CiAgc2VsZWN0b3I6CiAgICBtYXRjaExhYmVsczoKICAgICAgYXBwOiBnaXQtY29udHJvbGxlcgogIHJlcGxpY2FzOiAxCiAgdGVtcGxhdGU6CiAgICBtZXRhZGF0YToKICAgICAgbGFiZWxzOgogICAgICAgIGFwcDogZ2l0LWNvbnRyb2xsZXIKICAgIHNwZWM6CiAgICAgIGNvbnRhaW5lcnM6CiAgICAgIC0gbmFtZTogbWFuYWdlcgogICAgICAgIGltYWdlOiBnaGNyLmlvL3VzZXIvZ2l0LWNvbnRyb2xsZXI6djEuMC4wCg=="),
		},
	}, args[2])
}
