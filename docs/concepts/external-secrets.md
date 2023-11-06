# Concept Paper: Using external-secrets and cert-manager for distributing confidential information in the cluster

Table of contents:
- [Premise](#premise)
- [Proposal](#proposal)
  - [Generating certificates for the internal registry](#generating-certificates-for-the-internal-registry)
  - [Replicating secrets inside all namespaces including new ones](#replicating-secrets-inside-all-namespaces-including-new-ones)
    - [ClusterSecretStore](#clustersecretstore)
    - [ClusterExternalSecret](#clusterexternalsecret)
- [Installation using bootstrapper](#installation-using-bootstrapper)
- [Testing](#testing)

## Premise

The distribution of secrets in the cluster can be difficult to pull of securely, especially since we are dealing with
multiple namespaces that require the same access information; i.e.: pulling components, fetching images, accessing the
management repository, etc.

These secrets need to exist in all of the provided namespaces including every new namespace created by the project
controller. It needs to be done in a secure manner without requiring adminstrators of the cluster to do leg-work every
time a new project is needed.

This can be achieved by combining industry standard solutions with a bit of help from the project controller.

## Proposal

There are two aspects of secrets that need to be covered. One is the certificate generation for the in-cluster registry
that is running using HTTPS, and the other is replicating access to all namespaces, including every potential new one.

The trick here is the new namespaces which we know nothing about and don't even know their names either.

### Generating certificates for the internal registry

For certificates, we are going to use the first industry standard in this domain, [cert-manager](https://cert-manager.io/).

Cert manager can be used to generate self-signed certificates using a `ClusterIssuer` such as this:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: mpas-bootstrap-issuer
spec:
  selfSigned: {}
```

Next, we are going to create a self-signed root certificate using this issuer with a `Certificate`.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: mpas-bootstrap-certificate
  namespace: cert-manager
spec:
  isCA: true
  secretName: ocm-registry-tls-certs
  dnsNames:
    - registry.ocm-system.svc.cluster.local
  privateKey:
    algorithm: ECDSA
    size: 256
  issuerRef:
    name: mpas-bootstrap-issuer
    kind: ClusterIssuer
    group: cert-manager.io
```

Once this is done, we can create another `ClusterIssuer` which will use this rootCA as a CA for every new certificate
it generates. This will allows the usage in e2e test suits to easily prime a cluster, then download this certificate and access
the internal registry using https to create new components for testing.

The new `ClusterIssuer` looks like this:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: mpas-certificate-issuer
spec:
  ca:
    secretName: ocm-registry-tls-certs
```

Now, what's left, is to deal with any new project namespaces that might pop up. However, since mpas-project-controller
already creates resources in the project namespace, it also creates a `Certificate` resource for this using this specific
`ClusterIssuer`.

Which means, every new namespace will automatically get a certificate secret generated for the OciRepositories to use
to access the internal registry.

### Replicating secrets inside all namespaces including new ones

Next is the access to various facilities outside the cluster, like the management repository.
For this, we offer an integration with [external-secrets](https://external-secrets.io/). External secrets are a
convenient approach in getting secrets into the cluster and then distributing them to all namespaces.

To achieve this, we have to get acquainted with two new concepts/objects. [ClusterSecretStore](https://external-secrets.io/latest/api/clustersecretstore/) and
[ClusterExternalSecret](https://external-secrets.io/latest/api/clusterexternalsecret/).

#### ClusterSecretStore

ClusterSecretStore defines from WHERE the secrets are coming. External Secrets support for a wast number of secret stores
from which it can fetch the secrets. All store configurations will need to be managed by the cluster administrator. Secrets
that contain access for these stores can be put into the cluster or fetched via other means.

For example, if secrets are already stored in the management cluster, we could create a cluster store like this:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: secret-store-name
spec:
  provider:
    kubernetes:
      # This is the namespace in which it will look for the secrets to be distributed.
      # Meaning, all secrets are applied in here. This might be some secret-namespace for distribution purposes.
      remoteNamespace: ocm-system
      auth:
        serviceAccount:
          namespace: "default"
          # This would be the project service account which has the right permissions to create secrets.
          name: "default"
      server:
        caProvider: # we are connecting to our own cluster.
          namespace: "default"
          type: ConfigMap
          name: kube-root-ca.crt
          key: ca.crt
  # Conditions about namespaces in which the ClusterSecretStore is usable for ExternalSecrets
  conditions:
    - namespaces:
        - "ocm-system"
        - "mpas-system"
        - "flux-system"
    - namespaceSelector:
        matchExpressions:
          - key: "mpas.ocm.system/project"
            operator: exists

```

The important part of all of this is the following bit, that _EVERY_ cluster store _MUST_ have:

```yaml
  conditions:
    - namespaces:
        - "ocm-system"
        - "mpas-system"
        - "flux-system"
    - namespaceSelector:
        matchExpressions:
          - key: "mpas.ocm.system/project"
            operator: exists
```

That last `namespaceSelector` bit enables the secrets to be automatically created in EVERY new namespace that the
project controller creates. The project controller creates namespaces that have this annotation on them with the value
that is the name of the `Project` resource created in the cluster. Here, we only care that the annotation `exists`.

#### ClusterExternalSecret

Once the store exists, we will need to create `ClusterExternalSecret` objects, one for each secret in the cluster is required.
For example, consider a pull secret for components and images. We need that secret to exist in the `ocm-system`
namespace, the `mpas-system` namespace and every new namespace that the project controller might create.

The `ClusterExternalSecret` object's job is to create `Secret`s in each namespace that it is watching.

For example:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterExternalSecret
metadata:
  name: pull-creds
spec:
  # The name to be used on the ExternalSecrets
  externalSecretName: "pull-creds-es"

  # This is a basic label selector to select the namespaces to deploy ExternalSecrets to.
  # you can read more about them here https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#resources-that-support-set-based-requirements
  namespaceSelector:
    namespaces:
      - ocm-system
      - mpas-system
      - flux-system
    matchExpressions:
      - key: "mpas.ocm.system/project"
        operator: exists

  # How often the ClusterExternalSecret should reconcile itself
  # This will decide how often to check and make sure that the ExternalSecrets exist in the matching namespaces
  refreshTime: "1m"

  # This is the spec of the ExternalSecrets to be created
  # The content of this was taken from our ExternalSecret example
  externalSecretSpec:
    secretStoreRef:
      name: secret-store-name
      kind: ClusterSecretStore
    target:
      template:
        type: kubernetes.io/dockerconfigjson
        metadata:
          annotations:
            # this will make sure that this pull access is also put into the service account created by the project.
            mpas.ocm.system/secret.dockerconfig: managed
      name: pull-creds
      creationPolicy: Owner
    data:
    - secretKey: .dockerconfigjson
      remoteRef:
        key: pull-creds
        property: .dockerconfigjson
```

Note two things of importance:

First, the same `namespaceSelector` that defines where the secrets will be created. We use the same `exists` operator
here as in the store. This allows external secrets to replicate secrets into every future project that might be created
without the need of an administrator or K8s controller, to update this object with the name of the new namespace.

Second, we have an annotation that is applied to the secret here:

```yaml
        metadata:
          annotations:
            # this will make sure that this pull access is also put into the service account created by the project.
            mpas.ocm.system/secret.dockerconfig: managed
```

This annotation is for the project controller. The project controller makes sure that any secret with this
annotation is synced into the `ServiceAccount` of the project. Doing this, allows the controllers in the project
namespace to have access to pulling components and authentication against any remote repositories.

It is done automatically so admins don't have to bother with updating any service accounts by hand. Removing the secrets
detaches it from the service account as well.

## Installation using bootstrapper

Now that we have a better picture of how to set up access, we can move on to how to set up the components for this
system. Lucky for any cluster admin, the MPAS system contains a bootstrapping script that will install all necessary
or optional components into a cluster described [here](bootstrap.md).

To install all components simply launch MPAS boostrap script like this:

```bash
mpas bootstrap github --owner skarlso --repository=mpas-test-project --personal
```

In case you would like to disable external-secrets and use a different solution, use the following command:

```bash
mpas bootstrap github --owner skarlso --repository=mpas-test-project --personal --components=""
```

Notice the added `--components=""`. This clears the optional component list which by default contains
`external-secrets-component`.

## Testing

This scenario makes spinning up the bootstrap clusters a lot easier. No need to pre-generate any certificates. We still
use `mkcert` to install the generated root certificate locally so the e2e script has access to the internal https
registry; but we are no longer required to generate and create any secret by hand.

In the ocm-controller and the MPAS repository, there is a script under the `hack` folder to prime any clusters with a
certificate already installed.

The targets `make e2e` and `make e2e-verbose` use this script to prime a cluster with cert-manager and generate any
certs that could be required during running e2e tests. Using mkcert then we download the created CA and install it into
the local environment.
