# Concept Paper: Product Description

Table of contents:
- [Context](#context)
- [Proposal](#proposal)
- [Structure](#structure)
- [Miscellaneous](#miscellaneous)
- [Appendix A: APIs](#appendix-a-apis)

## Context

User story: https://github.com/open-component-model/MPAS/issues/5

Packaging a product in an MPAS-compatible way requires a specific component structure and essential resources. This has to be described and documented, so that Solution Designers / Software Architects are able deliver MPAS-compatible products.

## Proposal

We propose the use of a dedicated `ProductDescription` manifest file.

The purpose of the `ProductDescription` manifest is to specify a set of resources that can be used to build a product.

It should also provide a textual description of a product that can be used for discovery.

Finally (and out of scope for MVP), the product description may define a set of dependencies that are required to be installed before the product is deployed.

The final product deployment artifacts will be rendered by the `ocm-controller` via a pipeline that is dynamically generated by the MPAS control plane.

Resources may be part of the same component as the `ProductDescription` (local) or may be accessed via a reference to a remote component (remote).

The `ProductDescription` manifest should be an OCM resource with the type: "productdescription.mpas.ocm.software"

## Structure

### Description

The `spec.description` field contains a textual description of the product that can be used in content discovery systems.

### Pipeline Specification

The field `spec.pipelines` defines pipelines which must be put in place to deploy the product. Multiple pipelines may be specified, each consisting of the following elements:
- `source` (required): the resource containing the deployment artifact (Kubernetes manifests, Flux HelmRelease etc...)
- `localization` (required): the resource containing localization rules that can be used to localize the `source`
- `configuration` (optional):
  - a resource containing configuration rules that can be used to configure the localized `source`
  - a resource containing a README file instructing the operator what parameters are available for configuration
- `validation` (required): the resource which can be executed to validate the product has been configured correctly and ready for deployment
- `targetRoleName` (optional): a target class to which the pipeline result should be deployed. If no target class is specified the pipeline result will be deployed to any available target.

### Target Specification

A Target Role is a logical grouping of features. The field `spec.targetRoles` defines the targets roles that are available for use by the pipeline(s). Target roles consist of a name, kind and a set of label selectors. It is up to the consumer to ensure that targets with the appropriate labels exist.

Labels should be namespaced and have a well-defined taxonomy, for example:

```yaml
# devices
"target.mpas.ocm.software/gpu"
"target.mpas.ocm.software/hsm"
"target.mpas.ocm.software/fpga"

# networking
"target.mpas.ocm.software/public-internet"
"target.mpas.ocm.software/private-network"
"target.mpas.ocm.software/enclave-network"

# machine types
"target.mpas.ocm.software/memory-optimized"
"target.mpas.ocm.software/cpu-optimized"
"target.mpas.ocm.software/network-optimized"
```

## Miscellaneous

A CLI tool could be provided be used to validate if the component contains a conformant `ProductDescription` manifest and to aid with generating the `ProductDescription`.

Documentation should be provided that describes the available parameters for the `ProductDescription` manifest and provides several usage examples.

## Appendix A: APIs

### A.1. Product Description Manifest

The API for a `ProductDescription` shall be as follows:

```yaml
apiVersion: meta.mpas.ocm.software/v1alpha1
kind: ProductDescription
metadata:
  name: string
spec:
  description: string
  pipelines:
  - name: string # (required)
    targetRoleName: string # (optional)
    source: # (required)
      name: string # resource name
      version: string # resource version
      referencePath: # (optional) if provided the resource is retrieved from this component
        name: string
    localization: # (required)
      name: string # resource name
      version: string # resource version
      referencePath: # (optional) if provided the resource is retrieved from this component
        name: string
    configuration: # (optional)
      rules:
        name: string # resource name
        version: string # resource version
        referencePath: # (optional) if provided the resource is retrieved from this component
          name: string
      readme:
        name: string # resource name
        version: string # resource version
        referencePath: # (optional) if provided the resource is retrieved from this component
          name: string
    validation: # (required)
      name: string # resource name
      version: string # resource version
      referencePath: # (optional) if provided the resource is retrieved from this component
        name: string
  # targetRoles defines a list of target classes that
  # may be selected by a pipeline
  # selector defines labels that a given target should have in order
  # to be considered a member of the class
  targetRoles: # (optional)
  - name: string # (required) the name of the target
    type: string # (required) the type of target
    selector: # (required)
      matchLabels:
        string: string
      matchExpressions:
        - { key: string, operator: In, values: [string] }
        - { key: string, operator: NotIn, values: [string] }
# out of scope
# dependsOn:
# - component: string
#  version: string
```
