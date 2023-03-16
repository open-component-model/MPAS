# Concept Paper: Subscriptions

Table of contents:
- [Context](#context)
- [Proposal](#proposal)
- [Replication Controller Changes](#replication-controller-changes)
- [Unresolved Issues](#unresolved-issues)
- [Appendix A: APIs](#appendix-a-apis)
- [Appendix B: Glossary](#appendix-b-glossary)

## Context

User story: https://github.com/open-component-model/MPAS/issues/1

The purpose of a Subscription is to replicate OCM components containing a particular Product from a delivery registry to a registry in the MPAS customer's environment. It should be possible to create both cluster-wide and namespace-scoped subscriptions. MPAS Operators may create cluster-wide subscriptions, whereas MPAS Users may create namespaced Subscriptions only accessible in the context of their project.

## Proposal

We will use the existing [replication-controller](https://github.com/open-component-model/replication-controller) with some modifications in order to satisfy the requirements of MPAS subscriptions.

The `replication-controller` already offers a `ComponentSubscription` resource which performs the following operations:
- authenticate to source and destination registries
- transfer component versions (following version constraint) from source to destination registry (copying resources by value)
- verify the component after transfer

`ComponentSubscriptions` can be created globally or within the context of a `Project`, to install a product from a `ComponentSubscription` then it is necessary to create `ProjectSubscriptionBinding`. This is described in detail in the `Project` Concept Document. Effectively, a `ProjectSubscriptionBinding` creates an **instance** of a particular **Subscription** in a **Project** and we call that instance a **Product**. This allows us to have multiple instances of a single **Subscription** in a **Project**.

## Replication Controller Changes
One requirement of MPAS Projects is that it is possible to specify project OCI repositories when creating the project. 
These OCI repositories can then be used when creating subscriptions for the project. 

This presents us with two issues given the current design of the `ComponentSubscription` api:

- 1. Both source and destination registries are specified directly in the Subscription spec, along with a reference to access credentials
- 2. Access credentials need to be copied from "somewhere" to the namespace of the Subscription

To address these issues we can make two changes to the replication controller.

- 1. We create a new API object `OCIRegistry`, which contains the url of the oci registry. We update the `ComponentSubscription` API to include a `registryRef` field. This enables us to define registries in our `Project` manifest and refer to them whenever we need to create a subscription.
- 2. We introduce support for registry authentication via Kubernetes `ServiceAccount`.  This means that we can use the project's service account to authenticate to an OCI registry. The Flux project already supports contextual authorization for AWS `ecr` and `gcr` via annotations on the `ServiceAccount`.

These two changes will require incrementing the API version of the `ComponentSubscription` to `v1alpha2`.

## Unresolved Issues

The question remains how to handle the supply of keys for verification. Should this be the responsibility of the control plane, the project or the individual subscription?

Ideally the public key should be provided by the same source which publishes the component. This means that when discovering components the public key should also be visible.

- The mechanism by which MPAS is configured allows for the specification of verification keys
- We extend the `Project` API to include the specification of verification keys
- We extend the `ComponentSubscription` to support passing the public key directly in the manifest.

## Appendix A: APIs

### A.1. Component Subscription Kubernetes API Object (v1alpha2)

The API for a `ComponentSubscription` will be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha2
kind: ComponentSubscription
metadata:
  name: string
  namespace: string
spec:
  component: string
  semver: string
  source:
    registryRef:
      name: string
      namespace: string
  destination:
    registryRef:
      name: string
      namespace: string
  serviceAccountName: string
  verify:
  - name: string
    secretRef:
      name: string
  # possible change
  # publicKey: string
```

## Appendix B: Glossary

#### Product

An MPAS Product is software that has been packaged using the Open Component Model. It contains both configuration resources, technical artifacts (such as Kubernetes manifests, docker images), localization & configuration rules and instructions that can be used by the MPAS to create a generate a product deployment pipeline.

#### Product Deployment

The `ProductDeployment` is a Kubernetes API object that specifies the information necessary to generate a product deployment pipeline. `ProductDeployment`'s are consumed by the Product controller.

#### Project

An MPAS Project is a collection of Products. Products are installed using OCM and GitOps. Products can be stored in one or more Git repositories that are managed by the Project.

The `Project` Kubernetes API object that contains the metadata required to create and manage an individual MPAS Project.

#### Subscription

A Subscription is request for an OCM component to be replicated from an delivery registry to a customer registry.

The `Subscription` Kubernetes API object specifies the details of the component, version constraints and the delivery & customer registries.

#### Project Subscription Binding

A Project Subscription Binding creates a relation between a Project and a component Subscription.

The `ProjectSubscriptionBinding` object contains a references to `Project` and `Subscription` objects. The Project controller watches `ProjectSubscriptionBinding` objects and creates `ProductDeployment` manifests.

#### Repository

A Repository is a Git repository that is created to house products installed as part of an MPAS project.

Because the Flux Source controller already defines an object named `GitRepository`, for the MPAS we shall define a Kubernetes API object named `Repository` that is describes project git repositories.

