// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	//go:embed certmanager/cluster_issuer.yaml
	clusterIssuer []byte
	//go:embed certmanager/ocm_system_certificate.yaml
	ocmCertificate []byte
	//go:embed certmanager/mpas_system_certificate.yaml
	mpasCertificate []byte
)

type certificateManifestOptions struct {
	gitRepository         gitprovider.UserRepository
	branch                string
	targetPath            string
	provider              string
	timeout               time.Duration
	commitMessageAppendix string
	kubeClient            client.Client
}

// certManifestInstall is used to install cert-manager objects
type certificateManifestsInstall struct {
	*certificateManifestOptions
}

// newCertificateManifestInstaller returns a new certificate installer
func newCertificateManifestInstaller(opts *certificateManifestOptions) *certificateManifestsInstall {
	return &certificateManifestsInstall{
		certificateManifestOptions: opts,
	}
}

func (c *certificateManifestsInstall) Install(ctx context.Context) (string, error) {
	mpasCertificatePath := filepath.Join(c.targetPath, "mpas-system", "mpas_certificate.yaml")
	ocmCertificatePath := filepath.Join(c.targetPath, env.DefaultOCMNamespace, "ocm_certificate.yaml")
	mpasCertificateData := SetProviderDataFormat(c.provider, mpasCertificate)
	ocmCertificateData := SetProviderDataFormat(c.provider, ocmCertificate)
	commitMsg := "Add cluster issuer and namespace certificates"

	if c.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + c.commitMessageAppendix
	}

	files := []gitprovider.CommitFile{
		{
			Path:    &mpasCertificatePath,
			Content: &mpasCertificateData,
		},
		{
			Path:    &ocmCertificatePath,
			Content: &ocmCertificateData,
		},
	}

	ok, err := c.addClusterIssuerIfAbsent(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to check if cluster issuer exists: %w", err)
	}

	if ok {
		clusterIssuerPath := filepath.Join(c.targetPath, "cert-manager", "cluster_issuer.yaml")
		clusterIssuerData := SetProviderDataFormat(c.provider, clusterIssuer)

		files = append(files, gitprovider.CommitFile{
			Path:    &clusterIssuerPath,
			Content: &clusterIssuerData,
		})
	}

	commit, err := c.gitRepository.Commits().Create(ctx, c.branch, commitMsg, files)
	if err != nil {
		return "", fmt.Errorf("failed to add commit for certificate data: %w", err)
	}

	return commit.Get().Sha, nil
}

func (c *certificateManifestsInstall) addClusterIssuerIfAbsent(ctx context.Context) (bool, error) {
	// Avoid having to import certmanager and add it to the client scheme. We just want to check if it exists or not.
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("cert-manager.io/v1")
	obj.SetKind("ClusterIssuer")
	if err := c.kubeClient.Get(ctx, types.NamespacedName{
		Name: "mpas-bootstrap-issuer", // matches name with what is in internal/boostrap/certmanager/cluster_issuer.yaml
	}, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, fmt.Errorf("failed to get ClusterIssuer mpas-bootstra-issuer: %w", err)
	}

	return false, nil
}
