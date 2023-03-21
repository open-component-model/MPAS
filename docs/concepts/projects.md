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
- creating project Git repositories
- ensuring project Git repositories have the appropriate directory structure
- creating repository deploy keys and persisting them for access in cluster
- ensuring the repositories have the appropriate maintainers configured
- ensuring Flux is configured to reconcile the project Git repository

Because MPAS is a GitOps system `Project` manifests should be stored in a dedicated "management repository". This repository would be reconciled by Flux and the `Project` manifests applied to the cluster using a `Kustomization` using a service account with the necessary permissions. Global `Subscriptions` and `Targets` for use within all projects can be stored in the "management repository" also.


### Project Controller

To implement the functionality required for MPAS Projects we will create a dedicated Kubernetes controller. The controller will act as a coordinator, it's primary function being to translate the `Project` Custom Resource into lower level resources which will perform the specific operations.

The reconciliation process for the project controller will be as follows:
- verify git & oci credentials exist and are valid
- create the project `Namespace`
- create the project `ServiceAccount`
- generate the `Role`, `RoleBinding` & `RoleBindings` for the Project `ServiceAccount`
- create/update/delete `Repository` object(s) for each repository in `spec.git.repositories`
- create/update/delete deploy (SSH) key for `Repository` for each repository in `spec.git.repositories`
- create/update/delete Kubernetes secret containing deploy key for `Repository` for each repository in `spec.git.repositories`
- create/update/delete `OCIRegistry` object(s) using `spec.oci.registries`
- create/update/delete Flux 'GitRepository' object(s) targeting each repository in `spec.git.repositories`
- create/update/delete Flux 'Kustomization' object(s) targeting each repository in `spec.git.repositories` configured to use the project `ServiceAccount`
- watch for `ProjectSubscriptionBindings`
- for any new `ProjectSubscriptionBindings` generate a `ProductDeployment` manifest and commit to the project repository

The process for onboarding a new project is as follows:

![image](https://user-images.githubusercontent.com/4415593/225442259-c4b7c78a-66e7-41de-9f04-1dd5ab881d8e.png)

### Git Repositories

The git related functionality described above will not be part of the `Project` controller. Instead we shall add this functionality to the existing [git-sync-controller](https://github.com/open-component-model/git-sync-controller); the `Project` controller will create `Repository` objects that will be consumed by the `git-sync-controller`.

The directory structure of the Project Repository should be as follows:

```bash
├── bindings # contains subscription and target bindings for the project
│   ├── subscriptions
│   └── targets
├── products # contains the installed products
├── subscriptions # contains subscriptions created within the project
└── targets # contains targets created within the project
```

This directory structure should be created by the controller with responsibility for provisioning the project git repository.

Targets bindings should always be reconciled before subscription bindings so it is important to ensure that the Flux Kustomizations are configured to achieve this behaviour.

### Subscriptions

Subscriptions need to be associated with a project however they will not be managed by the project controller itself. To this end we will leverage the functionality of the existing [replication-controller](https://github.com/open-component-model/replication-controller) to perform the mechanics of transferring components to the customer registry from the delivery registry.

Subscriptions will be linked to the Project using a `ProjectSubscriptionBinding` object. Any number of `ProjectSubscriptionBindings` can be created between a given Project or Subscription.

An additional benefit of following the `ProjectSubscriptionBinding` approach is that it is possible to have both cluster-wide (global) subscriptions that can be used by any project, as well as project specific subscriptions. This can be enforced using Kubernetes RBAC.

A reference to `ProjectSubscriptionBindings` should be stored in the `Project` status field.

### Targets

Targets shall be linked to the Project using a `ProjectTargetBinding` object. Any number of `ProjectTargetBinding`'s' can be created between a given Project or Target.

A `ProjectTargetBinding` may ignore particular projects using the ignore field. Otherwise targets will be consumed used by all products in the `Project`.

Similar to `ProjectSubscriptionBinding`, the `ProjectTargetBinding` makes it possible to define both global and project specific targets.

A reference to `ProjectTargetBindings` should be stored in the `Project` status field.

### Installing Products within Projects

The Project controller will watch the Kubernetes API for `ProjectSubscriptionBinding` events.

When a new `ProjectSubscriptionBinding` created the Project controller should fetch `ProductonInstallation` resource from the component associated with the subscription in the `ProjectSubscriptionBinding`'s '`spec.subscriptionRef` field.

Using the `ProductionInstallation` resource the Project controller will generate a `ProductDeployment` manifest the project controller, product configuration file (if necessary) & production configuration README. The Project controller will also check for any available `ProjectTargetBindings` and write these to the `ProductDeployment` manifest.

These files will then be committed to the project repository on a new branch under the `products` directory in the repository. If the `GitRepository` has automatic pull-request creation enabled then a pull-request will be created.

Finally the Project controller will create a `ProductValidator` object that executes the `Validator` specified in the `ProductInstallation` resource against the pull-request and mark it as valid using appropriate means per provider.

An update to a `ProjectTargetBinding` should trigger a reconcile of the named `Project` to ensure that newly created `ProjectTargetBindings` result in an updated `ProductDeployment`.

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
    provider: string # github or gitlab (required)
    org: string # required
    # for each entry in the spec.git.repositories list
    # we will create:
    # - GitRepository API Object with $name
    # - CODEOWNERS file with maintainers list
    # - empty directories for (subscriptions, bindings, products, targets)
    # - repository deploy key
    # - Kubernetes secret containing repository deploy key
    # - Flux GitRepository source in the project namespace
    # - Flux Kustomization configured with spec.serviceAccountName
    repositories:
    - name: string # name of project repository that will be created ( required )
      maintainers: []string # identites of maintainers added to the project repository
    # credentials for access to the git provider's api, secret for MVP but
    # ultimately should be OAuth
    credentials:
      secretRef:
        name: string # (required)
  # The spec.oci object contains the configuration detailing oci
  # registries that the project has access to
  # note: MPAS will not create the oci registry, it must already exist
  oci: # (optional)
    # for each entry in the registries list we will
    # create an OCIRegistry API Object that can be used to fetch the repositories
    # details
    registries:
    - name: string # (requried)
      url: string # (requried)
      # credentials for access to the registry
      # should be provided via the project service account
      credentials:
        secretRef:
          name: string # (required)
status:
  # subscriptions associated with the project are
  # stored in the status. subscriptions are linked to the project
  # by creating a project-subscription-binding, a reference to the binding
  # is stored in the status.subscriptions in the form "namespace/name"
  subscriptions: []string
  targets: []string
  serviceAccountName: string
```

### A.2 ProjectSubscriptionBinding Kubernetes API Object

The API for a project ProjectSubscriptionBinding should be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: ProjectSubscriptionBinding
metadata:
  name: string
  namespace: string
spec:
  projectRef:
    name: string
    namespace: string
    # repository is an optional field specifying the project repository to use
    # in the case that there is more than one project repository
    repository: # (optional)
      name: string
  subscriptionRef:
    name: string
    namespace: string
```

### A.3 ProjectTargetBinding Kubernetes API Object

The API for a project ProjectTargetBinding should be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: ProjectTargetBinding
metadata:
  name: string
  namespace: string
spec:
  projectRef:
    name: string
    namespace: string
    # ignore products is an optional field that denotes
    # products that should not be deployed to this target
    # it's behaviour could be similar to the sourceignore field
    # which supports wildcards and exclusion semantics (*, !)
    ignoreProducts: # (optional)
    - string
  targetRef:
    name: string
    namespace: string
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
  provider: string # github or gitlab (required)
  org: string # required
  name: string # name of project repository that will be created ( required )
  maintainers: []string # identites of maintainers added to the project repository
  # credentials for access to the git provider's api, secret for MVP but
  # ultimately should be OAuth
  credentials:
    secretRef:
      name: string # (required)
  automaticPullRequestCreation: true # (optional: default true)
```

### A.4 OCIRegistry Kubernetes API Object

The API for a project OCI Repository should be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: OCIRegistry
metadata:
  name: string
  namespace: string
spec:
  url: string
```

## Appendix B: Glossary

#### Project

An MPAS Project is a collection of Products. Products are installed using OCM and GitOps. Products can be stored in one or more Git repositories that a `Project` manages.

The `Project` Kubernetes API object that contains the metadata required to create and manage an individual MPAS Project.

#### Subscription

A Subscription is a request for an OCM component to be replicated from a delivery registry to a customer registry.

The `Subscription` Kubernetes API object specifies the details of the component, version constraints and the delivery & customer registries.

#### Project Subscription Binding

A Project Subscription Binding creates a relation between a Project and a component Subscription.

The `ProjectSubscriptionBinding` object contains a references to `Project` and `Subscription` objects. The Project controller watches `ProjectSubscriptionBinding` objects and creates `ProductDeployment` manifests.

#### Target

A Target is a deployment environment for the final result of an MPAS product generation pipeline.

The `Target` Kubernetes API object specifies the access details for the target. Creating or managing the target infrastructure is not the responsibility of the MPAS system.

#### Project Target Binding

A Project Target Binding creates a relation between a Project and a deployment Target.

The `ProjectTargetBinding` object contains a references to `Project` and `Target` objects. The Project controller will look for available `ProjectTargetBinding`'s whenever it is creating a `ProductDeployment`.

#### Repository

A Repository is a Git repository that is created to house products installed as part of an MPAS project.

Because the Flux Source controller already defines an object named `GitRepository`, for the MPAS we shall define a Kubernetes API object named `Repository` that is describes project git repositories.
