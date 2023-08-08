# MPAS

[![REUSE status](https://api.reuse.software/badge/github.com/open-component-model/<repo-name>)](https://api.reuse.software/info/github.com/open-component-model/<repo-name>)

# MPAS - Multi Platform Automation System

## About this project

MPAS (Multi Platform Automation System) enables the release of complex software
systems in a fully automated way. It is based on the [Open component model](https://github.com/open-component-model/ocm)
and uses [Kubernetes](https://kubernetes.io/) as a runtime environment.

# Quick start

## Prerequisites

* [kubernetes](https://kubernetes.io/) cluster
* [git](https://git-scm.com/)
* [oci registry](https://docs.docker.com/registry/spec/api/)


## Installation

### Install the mpas command line tool

```bash
curl -sfL https://raw.githubusercontent.com/open-component-model/mpas/main/install.sh | sh -
```

or  with brew

```bash
brew install open-component-model/tap/mpas
```

## Usage

### Bootstrap a kubernetes cluster

In order to bootstrap a `kubernetes` cluster, you may use the `bootstrap` command.
This will install a number of resources packaged in an `ocm component`. The component is
hosted as an `oci artifact` in a registry.

It is possible to override the default registry by using the `--registry` option.
And it is possible to install the component from a local file archive by using the `--from-file` option.

#### From a personal repository

```bash
mpas bootstrap github --owner <owner> --repository <my-repository> --personal --path clusters/my-cluster
```

#### From an organization repository

```bash
mpas bootstrap github --owner <owner> --repository <my-repository> --path clusters/my-cluster
```

#### Bootstrap from a local component bundle

It is possible to download the component bundle without installing it by using the `--export` option.
This may be useful to inspect the component bundle before installing it or to transport
it to another environment.

```bash
mpas bootstrap --export --export-path /tmp
```

It is then possible to install the component bundle from the local file system by using the `--from-file` option
and the exported file.

The `--registry` option is required to install the component bundle from a local file system.
It will first transfer the component bundle to the registry before installing it,
performing any configuration transformation required.

```bash
mpas bootstrap github --owner <owner> --repository <my-repository>  --registry <my-registry> --from-file /tmp/mpas-bundle.tar.gz --path clusters/my-cluster
```

## Licensing

Copyright 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
Please see our [LICENSE](LICENSE) for copyright and license information.
Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/open-component-model/<repo-name>).
