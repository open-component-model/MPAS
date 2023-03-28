# Concept Paper: Products

Table of contents:
- [Context](#context)
- [Proposal](#proposal)
- [Product Controller](#product-controller)
- [Product Configuration](#product-configuration-values)
- [Appendix A: APIs](#appendix-a-apis)
- [Appendix B: Glossary](#appendix-b-glossary)

## Context

User story: https://github.com/open-component-model/MPAS/issues/4

Products are made available to the MPAS system as OCM Components via a Subscription. Multiple instances of a Product may be installed that refer to same Subscription.

## Proposal

![image](https://user-images.githubusercontent.com/4415593/226614131-3fd68575-3410-4865-9ba2-628bce1a3547.png)

We will introduce a Custom Resource named `ProductDeployment` that will provide an API object describing the product deployment. The `ProductDeployment` object will be consumed by a Custom Controller (`Product` controller) in order to execute the functionality required to install the product.

The `ProductDeployment` manifest will be committed to a Git repository by the MPAS control-plane and reconciled to the Kubernetes cluster using Flux.

Alongside the `ProductDeployment` manifest there may be a `values.yaml` file that contains site-specific configuration values, supplied by the customer. The configuration must always be accompanied by a `README.md` that describes the how the product can be configured.

### Installing Products

To deploy an instance of a product from a **subscription** to a *target* requires generating a `ProductDeployment` file. To do this we introduce the `ProductDeploymentGenerator`.

The Product controller will watch the Kubernetes API for `ProductDeploymentGenerator` events.

When a new `ProductDeploymentGenerator` is created, the MPAS control-plane should fetch the `ProductDescription` resource from the component associated with the subscription in the `ProductDeploymentGenerator`'s '`spec.subscriptionRef` field.

Using the `ProductDescription` resource, the Project controller will generate a `ProductDeployment` manifest, a product configuration file (if necessary) and a production configuration README.

The `ProductDescription` also specifies roles which describe the targets that are required for each pipeline in the product. A role defines the kind of target as well as the constraints used to select a particular target.

These files will then be committed to the project repository on a new branch under the `products` directory in the repository. If the `GitRepository` has automatic pull-request creation enabled then a pull-request will be created.

Finally, the Product controller will create a `ProductValidator` object that executes the `Validator` specified in the `ProductDescription` against the pull-request.

At this point the consumer can set configuration values and the validator will ensure any customer supplied changes are valid.

### Deploying Products

Once a `ProductDeployment` has been validated, approved and merged Flux will apply the `ProductDeployment` manifest. The `Product` controller will then begin the reconciliation process.

The reconciliation process for a `ProductDeployment` will be as follows:
- fetch `spec.component.registryRef` object and create the `ComponentVersion` CR
- for each pipeline in the `spec.pipelines` array:
  - 1. create the Resource CR
  - 2. create the Localization CR
  - 3. fetch the configuration values provided by the user and create the Configuration CR
  - 4. create a Flux Source pointing at the snapshot created in step 2
  - 5. bind to a target (this could be handled by a dedicated scheduler):
      - 5.1 get the target selector for the pipeline
      - 5.2 fetch the list of targets matching the target selector constraints
      - 5.3 if more than one target matches constraint, then select at random
      - 5.4 set the `spec.target` field on `ProductDeployment` to the target
  - 6. if the target field is non-empty create a Flux Kustomization configured with the target's KubeConfig which reconciles the Flux Source from step 4

### Product Configuration Values

As previously mentioned, the product directory may contain a `values.yaml` that contains customer supplied parameters. The `Product` controller must fetch this file from Git and pass the values to the Configuration Custom Resource. This can be done by fetching the artifact from the Project's Flux GitRepository and setting the values on the `Configuration` custom resource's `spec.values` field, which allows for values to be passed inline.

## Appendix A: APIs

### A.1. Product Deployment Kubernetes API Object

The API for a `ProductDeployment` will be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: ProductDeployment
metadata:
  name: string # the name of the product deployment
  namespace: string # the storage namespace of the product deployment
spec:
  # the component field is used to create a ComponentVersion custom resource
  component:
    name: string
    version: string
    registry:
      url: string
  pipelines:
  - name: string
    # will be used to create Localization Custom Resource
    localization:
    resource: # the ocm resource to be Localized
      name: string
      version: string
    rules: # the ocm resource containing the Localization rules
      name: string
        version: string
    # the configuration field will create a Configuration Custom Resource
    # it will also fetch the valuesFile
    # and pass them to the configuration
    # a Flux OCI Repository will also be created
    configuration:
      rules:
        name: string
        version: string
      valuesFile:
        path: string
    targetRole: #
      kind: string # (required) the kind of target: Kubernetes, CloudFoundry, OCI Repository, SSH
      selector: # (required)
        matchLabels:
          string: string
        matchExpressions:
          - { key: string, operator: In, values: [string] }
          - { key: string, operator: NotIn, values: [string] }   matchLabels:
    targetRef: # (optional) set by the product controller/scheduler once a target has been selected
      name: string
      namespace: string
```

### A.2. Product Deployment Generator Kubernetes API Object

The API for a `ProductDeploymentGenerator` will be as follows:

```yaml
apiVersion: mpas.ocm.software/v1alpha1
kind: ProductDeploymentGenerator
metadata:
  name: string # the name of the product deployment
  namespace: string # the storage namespace of the product deployment
spec:
  subscriptionRef:
    name: string
    namespace: string
  # repository is an optional field specifying the git repository to use
  # in the case that the deployment should be generated in a
  # different repository
  repositoryRef: # (optional)
    name: string
```

## Appendix B: Glossary

#### Product

An MPAS Product is software that has been packaged using the Open Component Model. It contains both configuration resources, technical artifacts (such as Kubernetes manifests, docker images), localization & configuration rules and instructions that can be used by the MPAS to create a generate a product deployment pipeline.

#### Product Deployment

The `ProductDeployment` is a Kubernetes API object that specifies the information necessary to generate a product deployment pipeline. `ProductDeployment`'s are consumed by the Product controller.

#### Product Deployment Generator

The `ProductDeploymentGenerator` is a Kubernetes API object that specifies the information necessary to generate a product deployment manifest.

#### Project

An MPAS Project is a collection of Products. Products are installed using OCM and GitOps. Products can be stored in one or more Git repositories that a `Project` manages.

The `Project` Kubernetes API object that contains the metadata required to create and manage an individual MPAS Project.

#### Subscription

A Subscription is a request for an OCM component to be replicated from a delivery registry to a customer registry.

The `Subscription` Kubernetes API object specifies the details of the component, version constraints and the delivery & customer registries.

#### Target

A Target is a deployment environment for the final result of an MPAS product generation pipeline.

The `Target` Kubernetes API object specifies the access details for the target. Creating or managing the target infrastructure is not the responsibility of the MPAS system.

#### Repository

A Repository is a Git repository that is created to house products installed as part of an MPAS project.

Because the Flux Source controller already defines an object named `GitRepository`, for the MPAS we shall define a Kubernetes API object called `Repository` that describes project git repositories.
