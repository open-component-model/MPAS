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
	DefaultFluxVer = "v2.1.0"
	// DefaultOcmControllerVer is the default version of the ocm-controller component.
	DefaultOcmControllerVer = "v0.12.0"
	// DefaultGitControllerVer is the default version of the git-controller component.
	DefaultGitControllerVer = "v0.7.0"
	// DefaultReplicationVer is the default version of the replication-controller component.
	DefaultReplicationVer = "v0.6.1"
	// DefaultMpasProductControllerVer is the default version of the mpas-product-controller component.
	DefaultMpasProductControllerVer = "v0.3.2"
	// DefaultMpasProjectControllerVer is the default version of the mpas-project-controller component.
	DefaultMpasProjectControllerVer = "v0.1.1"
	// DefaultOcmCliVer is the default version of the ocm-cli component.
	DefaultOcmCliVer = "v0.3.0"
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
	// DefautBootstrapBundleLocation is the default location of the bootstrap bundle.
	DefautBootstrapBundleLocation = DefaultBootstrapComponentLocation + "-bundle"
	// DefaultFluxHost is the default host for the flux components.
	DefaultFluxHost = "ghrc.io/fluxcd"
	// DefaultOCMInstallPath is the default path to install the ocm components.
	DefaultOCMInstallPath = "ocm-system"
	// DefaultFluxNamespace is the default namespace to install the flux components.
	DefaultFluxNamespace = "flux-system"
	// RegistryTLSSecretName is the name of the secret in which we store the TLS creds for the in-cluster registry.
	RegistryTLSSecretName = "ocm-registry-tls-certs"
)

const (
	// DefaultsNamespace is the mpas-system namespace.
	DefaultsNamespace = "mpas-system"
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
)

var (
	// Components is the list of components to install or package.
	Components = []string{
		OcmControllerName,
		FluxName,
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
