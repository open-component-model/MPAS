# -*- mode: Starlark -*-

# generate developer certificates for the ocm registry
load('ext://namespace', 'namespace_create', 'namespace_inject')
namespace_create('ocm-system') # make sure it exists at this point
namespace_create('mpas-system') # make sure it exists at this point

#load('ext://cert_manager', 'deploy_cert_manager')
#deploy_cert_manager(version = 'v1.13.1')

print('install certificate bootstrap')
k8s_yaml(read_file('e2e/certmanager/bootstrap.yaml'))

include('../replication-controller/Tiltfile')
include('../git-controller/Tiltfile')
include('../mpas-project-controller/Tiltfile')
include('../ocm-controller/Tiltfile')
include('../mpas-product-controller/Tiltfile')
