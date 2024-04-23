package bootstrap

import (
	"bytes"
	"testing"

	"github.com/open-component-model/ocm-controller/pkg/fakes"
	"github.com/open-component-model/ocm/pkg/contexts/ocm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfigData = []byte(`apiVersion: config.ocm.software/v1alpha1
kind: ConfigData
metadata:
  name: ocm-config
localization:
- name: git-controller
  file: gotk-components.yaml
  image: spec.template.spec.containers[0].image
  resource:
    name: git-controller
`)

var testComponentData = []byte(`apiVersion: apps/v1
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

func TestKustomize(t *testing.T) {
	tmp := t.TempDir()
	componentName := "ocm.software/mpas/git-controller"
	repo := &mockRepository{
		name:    "test-repo",
		version: "v1.0.0",
		cv: []*mockComponentAccess{
			{
				versions: []string{"v1.0.0"},
				name:     componentName,
				cva: map[string]*fakes.Component{
					"v1.0.0": {
						Name:    componentName,
						Version: "v1.0.0",
						Resources: []*fakes.Resource[*ocm.ResourceMeta]{
							{
								Name:    "ocm-config",
								Version: "v1.0.0",
								Data:    testConfigData,
								Kind:    "localBlob",
								Type:    "ociBlob",
							},
							{
								Name:    componentName,
								Version: "v1.0.0",
								Data:    testComponentData,
								Kind:    "localBlob",
								Type:    "ociBlob",
							},
							{
								Name:     "git-controller",
								Version:  "v1.0.0",
								Kind:     "ociArtifact",
								Type:     "ociImage",
								Relation: "external",
								AccessOptions: []fakes.AccessOptionFunc{
									func(m map[string]any) {
										m["imageReference"] = "ghcr.io/new-user/git-controller:v1.0.0"
									},
									func(m map[string]any) {
										m["type"] = "ociArtifact"
									},
								},
							},
						},
					},
				},
			},
		},
	}
	kustomizer := NewKustomizer(&kustomizerOptions{
		dir:           tmp,
		repository:    repo,
		componentName: componentName,
		version:       "v1.0.0",
		host:          "ghcr.io/user",
	})

	out, err := kustomizer.GenerateKustomizedResourceData(componentName)
	require.NoError(t, err)
	assert.True(t, bytes.Contains(out, []byte("ghcr.io/new-user/git-controller:v1.0.0")), "expected localized image to be present in output")
}
