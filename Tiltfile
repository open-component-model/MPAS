# -*- mode: Starlark -*-

# generate developer certificates for the ocm registry
load('ext://namespace', 'namespace_create', 'namespace_inject')
namespace_create('ocm-system') # make sure it exists at this point
namespace_create('mpas-system') # make sure it exists at this point

print('applying certificate to ocm-system namespace')
k8s_yaml('../ocm-controller/hack/certs/registry_certs_secret.yaml', allow_duplicates = True)

print('applying certificate to mpas-system namepsace')
k8s_yaml(namespace_inject(read_file('../ocm-controller/hack/certs/registry_certs_secret.yaml'), 'mpas-system'), allow_duplicates = True)

include('../replication-controller/Tiltfile')
include('../git-controller/Tiltfile')
include('../ocm-controller/Tiltfile')
include('../mpas-project-controller/Tiltfile')
include('../mpas-product-controller/Tiltfile')
