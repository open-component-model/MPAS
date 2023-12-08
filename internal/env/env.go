// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"fmt"
	"time"
)

// The default versions of the components. Please update this list when a new version is released.
const (
	// DefaultFluxVer is the default version of the flux component.
	DefaultFluxVer = "v2.1.2"
	// DefaultCertManagerVer is the default version of the cert-manager component.
	DefaultCertManagerVer = "v1.13.2"
	// DefaultExternalSecretsVer is the default version of the external secrets component.
	DefaultExternalSecretsVer = "v0.9.9"
	// DefaultOcmControllerVer is the default version of the ocm-controller component.
	DefaultOcmControllerVer = "v0.18.1"
	// DefaultGitControllerVer is the default version of the git-controller component.
	DefaultGitControllerVer = "v0.11.0"
	// DefaultReplicationVer is the default version of the replication-controller component.
	DefaultReplicationVer = "v0.12.2"
	// DefaultMpasProductControllerVer is the default version of the mpas-product-controller component.
	DefaultMpasProductControllerVer = "v0.10.0"
	// DefaultMpasProjectControllerVer is the default version of the mpas-project-controller component.
	DefaultMpasProjectControllerVer = "v0.5.0"
	// DefaultOcmCliVer is the default version of the ocm-cli component.
	DefaultOcmCliVer = "v0.4.1"
)

const (
	// FluxBinURL is the URL to download the flux binary.
	FluxBinURL = "https://github.com/fluxcd/flux2/releases/download"
	// OcmBinURL is the URL to download the ocm binary.
	OcmBinURL = "https://github.com/open-component-model/ocm/releases/download"
	// ComponentNamePrefix is the prefix for the component names.
	ComponentNamePrefix = "ocm.software/mpas"
	// DefaultOCMHost is the default host for the ocm components.
	DefaultOCMHost = "ghcr.io/open-component-model"
	// DefaultBootstrapComponentLocation is the default location of the bootstrap component.
	DefaultBootstrapComponentLocation = "ghcr.io/open-component-model/mpas-bootstrap-component"
	// DefaultBootstrapBundleLocation is the default location of the bootstrap bundle.
	DefaultBootstrapBundleLocation = DefaultBootstrapComponentLocation + "-bundle"
	// DefaultFluxHost is the default host for the flux components.
	DefaultFluxHost = "ghcr.io/fluxcd"
	// DefaultCertManagerHost is the default host for the cert-manager components.
	DefaultCertManagerHost = "quay.io/jetstack"
	// DefaultExternalSecretsHost is the default host for the external-secrets components.
	DefaultExternalSecretsHost = "ghcr.io/external-secrets"
	// DefaultFluxNamespace is the default namespace to install the flux components.
	DefaultFluxNamespace = "flux-system"
	// DefaultCertManagerNamespace is the default namespace to install the cert-manager components.
	DefaultCertManagerNamespace = "cert-manager"
	// DefaultExternalSecretsNamespace is the default namespace to install the external secrets components.
	DefaultExternalSecretsNamespace = "default"
)

const (
	// DefaultOCMNamespace is the default path to install the ocm components.
	DefaultOCMNamespace = "ocm-system"
	// DefaultMPASNamespace is the mpas-system namespace.
	DefaultMPASNamespace = "mpas-system"
	// GithubTokenVar is the name of the environment variable to use to get the github token.
	GithubTokenVar = "GITHUB_TOKEN"
	// GiteaTokenVar is the name of the environment variable to use to get the gitea token.
	GiteaTokenVar = "GITEA_TOKEN"
)

const (
	// ProviderGithub is the github provider.
	ProviderGithub = "github"
	// ProviderGitea is the gitea provider.
	ProviderGitea = "gitea"
)

const (
	OcmControllerName         = "ocm-controller"
	GitControllerName         = "git-controller"
	ReplicationControllerName = "replication-controller"
	MpasProductControllerName = "mpas-product-controller"
	MpasProjectControllerName = "mpas-project-controller"
	FluxName                  = "flux"
	CertManagerName           = "cert-manager"
	ExternalSecretsName       = "external-secrets-operator"
)

var (
	// InstallComponents is the list of components to install or package.
	InstallComponents = []string{
		OcmControllerName,
		FluxName,
		CertManagerName,
		GitControllerName,
		ReplicationControllerName,
		MpasProductControllerName,
		MpasProjectControllerName,
	}
	// BootstrapComponents is the list of components for creating a package.
	// Note, these components might contain more than what is installed later on using InstallComponents.
	BootstrapComponents = []string{
		OcmControllerName,
		FluxName,
		CertManagerName,
		ExternalSecretsName,
		GitControllerName,
		ReplicationControllerName,
		MpasProductControllerName,
		MpasProjectControllerName,
	}
	// BinaryComponents is the list of components that are binaries.
	BinaryComponents = []string{
		"flux-cli",
		"ocm-cli",
	}
	// DefaultBootstrapComponent is the default bootstrap component fqdn.
	DefaultBootstrapComponent = fmt.Sprintf("%s/bootstrap", ComponentNamePrefix)
)

var (
	// DefaultKubeAPIQPS is the default QPS for the kube API.
	DefaultKubeAPIQPS float32 = 50.0
	// DefaultKubeAPIBurst is the default burst for the kube API.
	DefaultKubeAPIBurst = 300
	// DefaultPollInterval is the default poll interval.
	DefaultPollInterval = 2 * time.Second
)
