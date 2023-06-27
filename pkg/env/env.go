// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package env

import "fmt"

const (
	DefaultFluxVer                  = "v2.0.0-rc.5"
	DefaultOcmControllerVer         = "v0.8.3"
	DefaultGitControllerVer         = "v0.4.1"
	DefaultReplicationVer           = "v0.3.0"
	DefaultMpasProductControllerVer = "v0.1.0"
	DefaultMpasProjectControllerVer = "v0.1.1"
	DefaultOcmCliVer                = "v0.2.0"
	DefaultKubeAPIQPS               = 50.0
	DefaultKubeAPIBurst             = 300
)

const (
	FluxBinURL            = "https://github.com/fluxcd/flux2/releases/download"
	OcmBinURL             = "https://github.com/open-component-model/ocm/releases/download"
	ComponentNamePrefix   = "ocm.software/mpas"
	DefaultOCMHost        = "ghcr.io/open-component-model"
	DefaultFluxHost       = "ghrc.io/fluxcd"
	DefaultOCMInstallPath = "ocm-system"
)

const (
	DefaultsNamespace = "mpas-system"
	GithubTokenVar    = "GITHUB_TOKEN"
	GiteaTokenVar     = "GITEA_TOKEN"
)

var (
	Components = []string{
		"ocm-controller",
		"flux",
		"git-controller",
		"replication-controller",
		"mpas-product-controller",
		"mpas-project-controller",
	}
	BinaryComponents = []string{
		"flux-cli",
		"ocm-cli",
	}
	DefaultBootstrapComponent = fmt.Sprintf("%s/bootstrap", ComponentNamePrefix)
)
