# Deprecation Notice

This repo is now deprecated and no further changes will be made. The
current alternative for installing Argo CD extensions is by defining
an init-container in the Argo CD API server using the
argocd-extension-installer image provided by the following repo:

* https://github.com/argoproj-labs/argocd-extension-installer

This is an example about how to use it:

* https://github.com/argoproj-labs/argocd-extension-metrics#install-ui-extension

# Argo CD Extensions

To enable Extensions for your Argo CD cluster will require just a single `kubectl apply`.

Here we provide a way to extend Argo CD such that it can provide resource-specific visualizations, capabilities and interactions in the following ways:

- Richer and context-sensitive UI components can be displayed in the user interface about custom resources.
- Custom health checks can be configured to assess the health of the resource.
- Custom actions could be performed to manipulate resources in predefined ways.

## Motivation

Argo CD is commonly used as a dashboard to Kubernetes applications. The current UI is limited in that it only displays very general information about Kubernetes objects. Any special visualizations can currently only be done native Kubernetes kinds.

For custom resources, Argo CD does not by default have any special handling or understanding of CRs, such as how to assess health of the object or visualizations. When examining a resource, a user can only see a YAML view of the object, which is not helpful unless they are familiar with the object's spec and status information.

Note that Argo CD does currently have a resource customizations feature, which allows operators to define health checks and actions via lua scripts in the argocd-cm ConfigMap. However, the current mechanism of configuring resource customizations is difficult and highly error prone.

This proposal would allow operators to more easily configure Argo CD to understand custom resources, as well as provide more powerful visualization of objects.'

## Goals

- Enable new visualizations in the UI for resources that do not have baked-in support
- Extensions can be configured by operators at runtime, without a feature being built directly into Argo CD, and with no need to recompile UI code.
- Extensions should be easy to develop and install (via an `ArgoCDExtension` CR)
- Replace current resource customizations in argocd-cm ConfigMap with extensions

## Getting Started

The simplest way to install the extension controller is to use Kustomize to bundle Argo CD
and the extensions controller manifests together:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
# base Argo CD components
- https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/ha/install.yaml

components:
# extensions controller component
- https://github.com/argoproj-labs/argocd-extensions/manifests
```

Store the YAML above into kustomization.yaml file and use the following command to install manifests:

```bash
kubectl create ns argocd && kustomize build . | kubectl apply -f - -n argocd
```
