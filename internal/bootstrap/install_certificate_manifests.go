// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/open-component-model/mpas/internal/env"
)

var (
	//go:embed certmanager/cluster_issuer.yaml
	clusterIssuer []byte
	//go:embed certmanager/ocm_system_certificate.yaml
	ocmCertificate []byte
	//go:embed certmanager/mpas_system_certificate.yaml
	mpasCertificate []byte
)

type certManifestOptions struct {
	gitRepository         gitprovider.UserRepository
	branch                string
	targetPath            string
	provider              string
	timeout               time.Duration
	commitMessageAppendix string
}

// certManifestInstall is used to install to install cert-manager custom-resources objects
type certManifestInstall struct {
	*certManifestOptions
}

// newCertManifestInstall returns a new component install
func newCertManifestInstaller(opts *certManifestOptions) *certManifestInstall {
	return &certManifestInstall{
		certManifestOptions: opts,
	}
}

func (c *certManifestInstall) Install(ctx context.Context) (string, error) {
	clusterIssuerPath := filepath.Join(c.targetPath, "cert-manager", "cluster_issuer.yaml")
	mpasCertificatePath := filepath.Join(c.targetPath, "mpas-system", "mpas_certificate.yaml")
	ocmCertificatePath := filepath.Join(c.targetPath, "ocm-system", "ocm_certificate.yaml")
	commitMsg := "Add cluster issuer and namespace certificates"
	if c.commitMessageAppendix != "" {
		commitMsg = commitMsg + "\n\n" + c.commitMessageAppendix
	}

	clusterIssuerData := string(clusterIssuer)
	if c.provider == env.ProviderGitea {
		clusterIssuerData = base64.StdEncoding.EncodeToString(clusterIssuer)
	}

	mpasCertificateData := string(mpasCertificate)
	if c.provider == env.ProviderGitea {
		mpasCertificateData = base64.StdEncoding.EncodeToString(mpasCertificate)
	}

	ocmCertificateData := string(ocmCertificate)
	if c.provider == env.ProviderGitea {
		ocmCertificateData = base64.StdEncoding.EncodeToString(ocmCertificate)
	}

	commit, err := c.gitRepository.Commits().Create(ctx,
		c.branch,
		commitMsg,
		[]gitprovider.CommitFile{
			{
				Path:    &clusterIssuerPath,
				Content: &clusterIssuerData,
			},
			{
				Path:    &mpasCertificatePath,
				Content: &mpasCertificateData,
			},
			{
				Path:    &ocmCertificatePath,
				Content: &ocmCertificateData,
			},
		})
	if err != nil {
		return "", fmt.Errorf("failed to add commit for certificate data: %w", err)
	}

	return commit.Get().Sha, nil
}
