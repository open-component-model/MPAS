// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package kubeutils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fluxcd/cli-utils/pkg/kstatus/polling"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/kustomization"
	"github.com/fluxcd/pkg/ssa"
	"github.com/fluxcd/pkg/ssa/utils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"
)

// The file is copied from https://github.com/fluxcd/flux2/blob/main/internal/utils/apply.go
// As it is an internal package, we need to copy it here to use it.

// Apply is the equivalent of 'kubectl apply --server-side -f'.
// If the given manifest is a kustomization.yaml, then apply performs the equivalent of 'kubectl apply --server-side -k'.
func Apply(ctx context.Context, rcg genericclioptions.RESTClientGetter, root, manifestPath string) (string, error) {
	objs, err := readObjects(root, manifestPath)
	if err != nil {
		return "", err
	}

	if len(objs) == 0 {
		return "", fmt.Errorf("no Kubernetes objects found at: %s", manifestPath)
	}

	if err := ssa.SetNativeKindsDefaults(objs); err != nil {
		return "", err
	}

	changeSet := ssa.NewChangeSet()

	// contains only CRDs and Namespaces
	var stageOne []*unstructured.Unstructured

	// contains all objects except for CRDs and Namespaces
	var stageTwo []*unstructured.Unstructured

	for _, u := range objs {
		if utils.IsClusterDefinition(u) {
			stageOne = append(stageOne, u)
		} else {
			stageTwo = append(stageTwo, u)
		}
	}

	if len(stageOne) > 0 {
		cs, err := applySet(ctx, rcg, stageOne)
		if err != nil {
			return "", err
		}
		changeSet.Append(cs.Entries)
	}

	if len(changeSet.Entries) > 0 {
		if err := waitForSet(rcg, changeSet); err != nil {
			return "", fmt.Errorf("failed to wait for changeset: %w", err)
		}
	}

	if len(stageTwo) > 0 {
		cs, err := applySet(ctx, rcg, stageTwo)
		if err != nil {
			return "", err
		}
		changeSet.Append(cs.Entries)
	}

	return changeSet.String(), nil
}

func readObjects(root, manifestPath string) ([]*unstructured.Unstructured, error) {
	fi, err := os.Lstat(manifestPath)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("expected %q to be a file", manifestPath)
	}

	if isRecognizedKustomizationFile(manifestPath) {
		resources, err := kustomization.BuildWithRoot(root, filepath.Dir(manifestPath))
		if err != nil {
			return nil, err
		}
		return utils.ReadObjects(bytes.NewReader(resources))
	}

	ms, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer ms.Close()

	return utils.ReadObjects(bufio.NewReader(ms))
}

func applySet(ctx context.Context, rcg genericclioptions.RESTClientGetter, objects []*unstructured.Unstructured) (*ssa.ChangeSet, error) {
	man, err := newManager(rcg)
	if err != nil {
		return nil, err
	}

	return man.ApplyAll(ctx, objects, ssa.DefaultApplyOptions())
}

func waitForSet(rcg genericclioptions.RESTClientGetter, changeSet *ssa.ChangeSet) error {
	man, err := newManager(rcg)
	if err != nil {
		return err
	}
	return man.WaitForSet(changeSet.ToObjMetadataSet(), ssa.WaitOptions{Interval: 2 * time.Second, Timeout: time.Minute})
}

func isRecognizedKustomizationFile(path string) bool {
	base := filepath.Base(path)
	for _, v := range konfig.RecognizedKustomizationFileNames() {
		if base == v {
			return true
		}
	}
	return false
}

func newManager(rcg genericclioptions.RESTClientGetter) (*ssa.ResourceManager, error) {
	cfg, err := KubeConfig(rcg)
	if err != nil {
		return nil, err
	}
	restMapper, err := rcg.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	scheme, err := NewScheme()
	if err != nil {
		return nil, err
	}

	kubeClient, err := client.New(cfg, client.Options{Mapper: restMapper, Scheme: scheme})
	if err != nil {
		return nil, err
	}
	kubePoller := polling.NewStatusPoller(kubeClient, restMapper, polling.Options{})

	return ssa.NewResourceManager(kubeClient, kubePoller, ssa.Owner{
		Field: "flux",
		Group: "fluxcd.io",
	}), nil

}
