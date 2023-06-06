# End-to-end tests

This directory contains the end-to-end tests for Argo CD Extensions as custom health checks. The
tests are implemented using [kuttl](https://kuttl.dev) and require some
prerequisites.

**This is work-in-progress at a very early stage**. The end-to-end tests are
not yet expected to work flawlessly, and they require an opinionated setup to
run. If you are going to use the end-to-end tests, it is expected that you are
prepared to hack on them. Do not ask for support, please.

# Components

The end-to-end tests are comprised of the following components:

* A local, vanilla K8s cluster that is treated as volatile. The tests only
  support k3s as a cluster at the moment.
* A dedicated Argo CD installation. No other Argo CD must be installed to
  the test cluster.
* A Git repository, containing resources to be consumed by Argo CD.
  This will be deployed on demand to the test cluster, with test data that
  is provided by the end-to-end tests.
* A Docker registry, holding the container images we use for testing.
  This will be deployed on demand to the test cluster.

## Local cluster

## Pre-requisites

1. Run `make install-prereqs` to setup the test environment with all the pre-requisites on your local cluster.
2. Get the IP address for the e2e-git-repo service installed in the previos step and update the same in all the integration test cases listed under path test/e2e/suite
3. The e2e-git-repo service also includes a public server key located at path test/e2e/ssh-host-keys/ssh_host_ed25519_key.pub. For argocd-extension to be able to verify the git server while connecting via ssh, we would also need to update the ssh known hosts entry. This could be done from the argocd UI by adding ssh known hosts entry like below under settings.
`[10.96.25.65]:2222 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDzQclfSGaf5Txma20q9qEj5vbDCnRvVWEzB6Gx1EYzm`
Note: The IP in this entry should be the IP address of the e2e-git-repo service retrieved in step 2.
4. You also needs to deploy a secret to be able to access the repository on e2e-git-repo service.
```
apiVersion: v1
data:
  git_token: Z2l0
  git_user: Z2l0
  insecure: dHJ1ZQ==
  sshkey: LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUFNd0FBQUF0emMyZ3RaVwpReU5UVXhPUUFBQUNCeFBxMHE0U1c5RGJmY0oxL09jdE9RajZ2NDNaTUpVckZqc1JXYTJMaVFsQUFBQUtCbUJERlZaZ1F4ClZRQUFBQXR6YzJndFpXUXlOVFV4T1FBQUFDQnhQcTBxNFNXOURiZmNKMS9PY3RPUWo2djQzWk1KVXJGanNSV2EyTGlRbEEKQUFBRURCT3g5RmVIc3ZTcjBSdzhVcEIwM2VPOU8wRlN0bHNPOWRGSzZ4cGJKREUzRStyU3JoSmIwTnQ5d25YODV5MDVDUApxL2pka3dsU3NXT3hGWnJZdUpDVUFBQUFIWEp0YjNWQVVtRm9kV3h6TFUxaFkwSnZiMnN0VUhKdkxteHZZMkZzCi0tLS0tRU5EIE9QRU5TU0ggUFJJVkFURSBLRVktLS0tLQo=
  type: Z2l0
kind: Secret
metadata:
  name: e2e-git-local
type: Opaque
```
4. The test cases for ssh and http are located at 
`test/e2e/suite/`
Example command to execute the test suite
`kubectl kuttl test <path_to_test_suite>`
