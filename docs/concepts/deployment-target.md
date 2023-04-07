# Concept Paper: Deployment Targets

Table of contents:
- [Context](#context)
- [Proposal](#proposal)
- [Project Target Binding](#project-target-binding)
- [Unresolved Issues](#unresolved-issues)
- [Appendix A: APIs](#appendix-a-apis)
- [Appendix B: Glossary](#appendix-b-glossary)

## Context

User story: https://github.com/open-component-model/MPAS/issues/2

Deployment Targets represent some piece of infrastructure that will be used to deploy the end result(s) of an MPAS pipeline. Examples of deployment targets might include:

- Kubernetes clusters
- Virtual Machines (not in scope for MVP)
- OCI Repositories (not in scope for MVP)
- Git Repositories (not in scope for MVP)

Targets will be consumed by the MPAS control plane when rendering the `ProductDeployment`. Therefore a Target should provide a reference to the appropriate access credentials for the targetd environment.

## Proposal

We will introduce a new Kubernetes Custom Resource named `Target`.  The primary purpose of the `Target` is to describe an access method for the target; therefore it is not necessary to have separate Kubernetes Custom Resources for each type.

The `Target` Custom Resource specifies the `type` of target and contains a reference to a secret that provides access credentials for the target. When creating a `Target` is imperative that the correct kind of access credentials are present in the secret.

For a `Target` with type `kubernetes`, the `Product` controller will generate a Flux `Kustomization` and configure the `spec.kubeConfig` field to point at the `Target`'s `spec.access` secret. Flux will then deploy the end-result of the `ProductDeployment` pipeline to the target cluster.

For a `Target` with type `ssh`, the `Product` controller will generate a `MachineManager` CustomResource and configure the credentials appropriately based on the `Target`'s access.

## Scheduling Product Deployments to Targets

The `ProductDescription` may define the kind and features of the `Target`'s it requires. Because of this it is necessary for the MPAS control-plane to make a scheduling decision of the basis of the information provided in the `ProductDescription` and the `Target`'s available within the MPAS system. Once a target is chosen then it is assigned to the `ProductDeployment` and from this point onwards is immutable.

## Appendix A: APIs

### A.1. Target Kubernetes API Object

The API for a `Target` will be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: Target
metadata:
  name: string # the name of the product deployment
  namespace: string # the storage namespace of the product deployment
spec:
  type: string # required can be one of: kubernetes, ssh, ociRepository
  access:
    secretRef:
      name: string
      namespace: string
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

#### Target

A Target is a deployment environment for the final result of an MPAS product generation pipeline.

The `Target` Kubernetes API object specifies the access details for the target. Creating or managing the target infrastructure is not the responsibility of the MPAS system.

#### Repository

A Repository is a Git repository that is created to house products installed as part of an MPAS project.

Because the Flux Source controller already defines an object named `GitRepository`, for the MPAS we shall define a Kubernetes API object named `Repository` that is describes project git repositories.
