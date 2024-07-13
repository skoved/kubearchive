# Demo Script

## Before the demo
* Start the cluster
* Create `demo-tekton` and `demo-jobs` namespaces
* port forward the database and initialize. Stop port forwording the database
* port forward the api server so you can use curl to make http requests
* get the cert for the api server

## During the demo
* list kubearchive deployments if you want to explain the different components
```bash
kubectl get deployments -n kubearchive
```
* explain that you want KubeArchive to "archive" `PipelineRuns` in the `demo-tekton` namespace (you'll comeback to the other stuff on the yaml file)
```bash
vim conf-tekton.yaml
```
* make KubeArchive "archive" `PipelineRuns` in the `demo-tekton namespace
```bash
kubectl apply -f conf-tekton.yaml
```
* show the Tekton Pipeline you're going to run
```bash
vim tekton.yaml
```
* run a Tekton Pipeline
```bash
kubectl apply -f tekton.yaml
```
* explain that you want KubeArchive to "archive" `Jobs` in the `demo-jobs` namespace (you'll comeback to the other stuff in the yaml file)
```bash
vim conf-jobs.yaml
```
* make KubeArchive "archive" `Jobs` in the `demo-jobs` namespaces
```bash
kubectl apply -f conf-jobs.yaml
```
* show the `Job` that you're going to run
```bash
vim job.yaml
```
* run a `Job`
```bash
kubectl apply -f job.yaml
```
* explain that the KubeArchive Api Server queries Kubernetes to determine if a user has the correct permissions to read the requested resource (that's what the Roles, RoleBindings, and ServiceAccounts were for)
```bash
vim conf-jobs.yaml
```
```bash
vim conf-tekton.yaml
```
* show that the ServiceAccounts can't lookup other Resources than what they are give Roles for
```bash
kubectl auth can-i get jobs.batch --as=system:serviceaccount:demo-jobs:demo-jobs -n demo-jobs
```
```bash
kubectl auth can-i get pipelineruns.tekton.dev --as=system:serviceaccount:demo-jobs:demo-jobs -n demo-jobs
```
```bash
kubectl auth can-i get pipelineruns.tekton.dev --as=system:serviceaccount:demo-tekton:demo-tekton -n demo-tekton
```
```bash
kubectl auth can-i get jobs.batch --as=system:serviceaccount:demo-tekton:demo-tekton -n demo-tekton
```
* get the archived `PipelineRun` from the KubeArchive Api Server
```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-tekton -n demo-tekton) https://localhost:8081/apis/tekton.dev/v1beta1/pipelineruns | jq
```
* get the archived `Job` from the KubeArchive Api Server
```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-jobs -n demo-jobs) https://localhost:8081/apis/batch/v1/jobs | jq
```
* show that the KubeArchive Api Server responds with 401 Unauthorized if you don't have the role to view that resource
```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-jobs -n demo-jobs) https://localhost:8081/apis/tekton.dev/v1beta1/pipelineruns | jq
```
```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-tekton -n demo-tekton) https://localhost:8081/apis/batch/v1/jobs | jq
```
