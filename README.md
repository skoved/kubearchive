# kubearchive

## Overview
KubeArchive is a utility that stores Kubernetes resources off of the
Kubernetes cluster. This enables users to delete those resources from
the cluster without losing the information contained in those resources.
KubeArchive will provide an API so users can retrieve stored resources
for inspection.

The main users of KubeArchive are projects that use Kubernetes resources
for one-shot operations and want to inspect those resources in the long-term.
For example, users using Jobs on Kubernetes that want to track the success
rate over time, but need to remove completed Jobs for performance/storage
reasons, would benefit from KubeArchive. Another example would be users
that run build systems on top of Kubernetes (Shipwright, Tekton) that use
resources for one-shot builds and want to keep track of those builds over time.

* [Code of Conduct](./CODE_OF_CONDUCT.md)
* [Documentation](https://kubearchive.github.io/kubearchive/main/index.html)
