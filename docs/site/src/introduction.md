# Introduction

MPAS (Multi Product Pipeline Automation System) enables the release of complex software
systems in a fully automated way. It is based on the [Open component model](https://github.com/open-component-model/ocm)
and uses [Kubernetes](https://kubernetes.io/) as a runtime environment.

MPAS provides a cloud-native operational model for packaging and running applications. It takes an opinionated approach that considers the packaging unit to be an Open Component Model (OCM) component, Git to be the configuration interface and GitOps to be the deployment methodology. 

Therefore an environment in which MPAS can be bootstrapped requires only the following cloud-native primitives:

- Kubernetes Cluster
- OCI Registry
- Git Provider (currently we support Gitea and GitHub)

MPAS has a strong focus on tooling and our CLI helps with automating all of the hard stuff.

Take a look at the [getting started guide](./getting_started.md) to get going straight away or peruse the [architecture](./architecture.md) documentation to get a technical overview of how MPAS functions.


