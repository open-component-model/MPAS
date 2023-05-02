# Concept Paper: Projects

Table of contents:
- [Context](#context)
- [Proposal](#proposal)
  - [Project Controller](#project-controller)
  - [Git Repositories](#git-repositories)
  - [Subscriptions](#subscriptions)
  - [Targets](#targets)
  - [Product Installation](#installing-products-within-projects)
  - [Role Based Access Control](#role-based-access-control)
- [Appendix A: APIs](#appendix-a-apis)
- [Appendix B: Glossary](#appendix-b-glossary)

## Context

User story: https://github.com/open-component-model/MPAS/issues/4

It must be possible for an MPAS End user to create new "MPAS Projects" (or edit a existing ones). An MPAS Project must contain the following set of configuration data:

- Project Name
- OCI Registry Name / URL / Credentials
- Git Repository
- Product Subscriptions

## Proposal

We will introduce a Custom Resource named `Project` that will extend the Kubernetes API to provide an API object responsible for managing the lifecycle of MPAS Projects. The `Project` object will be consumed by a Custom Controller in order to execute the functionality required to setup the project.

The functions required to be carried out by the Project controller are:
- creating a Kubernetes namespace for the project
- creating a dedicated ServiceAccount and RBAC for the project
- creating project Git repositories
- ensuring project Git repositories have the appropriate directory structure
- creating repository deploy keys and persisting them for access in cluster
- ensuring repositories have the appropriate maintainers configured
- ensuring Flux is configured to reconcile the project Git repository

Because MPAS is a GitOps system `Project` manifests should be stored in a dedicated "management repository".

This repository shall be reconciled by Flux and the `Project` manifests applied to the cluster via a `Kustomization` impersonating a Service Account with appropriate permissions to administer the MPAS system.

Global `Subscriptions` and `Targets` for use within all projects can be stored in the "management repository".

### Project Controller

To implement the functionality required for MPAS Projects we will create a dedicated Kubernetes controller. The controller will act as a coordinator, it's primary function being to translate the `Project` Custom Resource into lower level resources which will perform the specific operations.

The reconciliation process for the project controller will be as follows:
- verify git & oci credentials exist and are valid
- create the project `Namespace`
- create the project `ServiceAccount`
- generate the `Role`, `RoleBinding` & `ClusterRoleBinding` for the Project `ServiceAccount`
- create/update/delete `Repository` object for `spec.git.repository`
- create/update/delete deploy (SSH) key for `Repository` for `spec.git.repository`
- create/update/delete Kubernetes secret containing deploy key for `Repository` for each repository in `spec.git.repositories`
- create/update/delete Flux 'GitRepository' object targeting `spec.git.repository`
- create/update/delete Flux 'Kustomization' objects targeting directories in `spec.git.repository` configured to use the project `ServiceAccount`

The process for on-boarding a new project is as follows:

![image](https://user-images.githubusercontent.com/4415593/227281052-d4c31de3-c0c5-4bd6-8639-33602c6a31b0.png)

### Git Repositories

The Git related functionality described above will not be performed by the `Project` controller. Instead, this functionality shall be added to the existing [git-sync-controller](https://github.com/open-component-model/git-sync-controller). The `Project` controller will create `Repository` objects that will be consumed by the `git-sync-controller`.

The directory structure of the Project Repository should be as follows:

```bash
├── generators # contains subscription and target bindings for the project
├── products # contains the installed products
├── subscriptions # contains subscriptions created within the project
└── targets # contains targets created within the project
```

This directory structure shall be created by the controller with responsibility for provisioning the project git repository.

Relevant information, such as commit message and commit user, will be part of the `Project` CRD.

### Installing Products within Projects

To deploy an instance of a product from a subscription to a target requires generating a `ProductDeployment`. To enable this we shall introduce the `ProductDeploymentGenerator`. The `ProductDeploymentGenerator` creates a relationship between a `Subscription` and a set of `Target`s. The `ProductDeploymentGenerator` is a namespaced resource that exists in the context of a project.

The Project controller will watch the Kubernetes API for `ProductDeploymentGenerator` events.

When a new `ProductDeploymentGenerator` is created, the Project controller should fetch `ProductDescription` resource from the component associated with the subscription in the `ProductDeploymentGenerator`'s '`spec.subscriptionRef` field.

Using the `ProductDescription` resource, the Project controller will generate a `ProductDeployment` manifest, a product configuration file (if necessary) and a production configuration README.

These files will then be committed to the project repository on a new branch under the `products` directory in the repository. If the `GitRepository` has automatic pull-request creation enabled then a pull-request will be created.

Finally the Project controller will create a `ProductValidator` object that executes the `Validator` specified in the `ProductDescription` resource against the pull-request and mark it as valid using appropriate means per provider.

### Role Based Access Control

In order to support multi-tenancy and enforce project segregation it is necessary to configure Kubernetes RBAC objects when creating a Project.

This will require creating a dedicated project namespace and project service account. The project service account will have an associated role granting the appropriate project level permissions.

The project service account will be used:
- by the OCM controller to authenticate with the project OCI registry (via image pull secrets)
- by the MPAS control plane when generating resources for the Project
- when reconciling resources with a Flux Kustomization

## Unresolved issues

- How to configure credentials appropriately on the Project Service Account? Ideally we would use a mechanism similar to EKS IRSA. This would enable us to avoid storing long-lived credentials in cluster.

## Appendix A: APIs

### A.1. Project Kubernetes API Object

The API for a `Project` will be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: Project
metadata:
  name: string # the project name
  namespace: string # the storage namespace of the project (likely mpas-system or similar)
spec:
  # The spec.git object contains the configuration that will
  # be used to create and configure the project Git repositories
  git:
    commitTemplate:
      email: <email>
      message: "Commit made by the project controller"
      name: Joe Commit
    provider: string # github or gitlab (required)
    domain: string # the domain of the git provider (optional)
    owner: string # required
    isOrganisation: boolean
    # this is the repository that
    # will be used to administer the project
    # it is required
    # the following will be created:
    # - GitRepository API Object with $name
    # - CODEOWNERS file with maintainers list
    # - directories for (subscriptions, installations, deployments, targets)
    # - repository deploy key
    # - Kubernetes secret containing repository deploy key
    # - Flux GitRepository source in the project namespace
    # - Flux Kustomization configured with spec.serviceAccountName
    repositoryName: string # name of project repository that will be created ( required )
    defaultBranch: string # default branch for the project repository (default: main)
    maintainers: []string # identites of maintainers added to the project repository
    visibility: string
    exisitingRepositoryPolicy: string # adopt | fail (controls whether an existing repository with the same name should be adopted and used or fail, causing the reconciliation to stall)
    # credentials for access to the git provider's api, secret for MVP but
    # ultimately should be OAuth
    credentials:
      secretRef:
        name: string # (required)
```

### A.2 ProductDeploymentGenerator Kubernetes API Object

The API for a project ProductDeploymentGenerator should be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: ProductDeploymentGenerator
metadata:
  name: string
  namespace: string
spec:
  subscriptionRef:
    name: string
    namespace: string
  targetSelector:
    matchLabels:
      string: string
    matchExpressions:
      - { key: string, operator: In, values: [string] }
      - { key: string, operator: NotIn, values: [string] }
  # repository is an optional field specifying the git repository to use
  # in the case that the deployment should be generated in a
  # different repository
  repositoryRef: # (optional)
    name: string
```

### A.3 Repository Kubernetes API Object

The API for a project Git Repository should be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: Repository
metadata:
  name: string
  namespace: string
spec:
  provider: string # github | gitlab (required)
  owner: string # required
  isOrganisation: boolean # (default: true)
  name: string # name of project repository that will be created ( required )
  maintainers: []string # identites of maintainers added to the project repository
  visibility: string # (default: private)
  domain: string # the git provider domain
  # credentials for access to the git provider's api, secret for MVP but
  # ultimately should be OAuth
  credentials:
    secretRef:
      name: string # (required)
  automaticPullRequestCreation: true # (optional: default true)
  exisitingRepositoryPolicy: string # adopt | fail (controls whether an existing repository with the same name should be adopted and used or fail, causing the reconciliation to stall)
```

## Appendix B: Glossary

#### Project

An MPAS Project is a collection of Products. Products are installed using OCM and GitOps. Products can be stored in one or more Git repositories that a `Project` manages.

The `Project` Kubernetes API object that contains the metadata required to create and manage an individual MPAS Project.

#### Subscription

A Subscription is a request for an OCM component to be replicated from a delivery registry to a customer registry.

The `Subscription` Kubernetes API object specifies the details of the component, version constraints and the delivery & customer registries.

#### ProductDeploymentGenerator

A ProductDeploymentGenerator creates a relation between a Subscription and a set of Targets.

The `ProductDeploymentGenerator` object contains references to a `Subscription` object. The Product controller watches `ProductDeploymentGenerator` objects and creates `ProductDeployment` manifests.

#### Target

A Target is a deployment environment for the final result of an MPAS product generation pipeline.

The `Target` Kubernetes API object specifies the access details for the target. Creating or managing the target infrastructure is not the responsibility of the MPAS system.

#### Repository

A Repository is a Git repository that is created to house products installed as part of an MPAS project.

Because the Flux Source controller already defines an object named `GitRepository`, for the MPAS we shall define a Kubernetes API object called `Repository` that describes project git repositories.
