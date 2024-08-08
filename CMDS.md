get all pipelineruns

```bash
curl -s --cacert ca.crt -H "Authorization: Bearer $(kubectl create token demo-tekton-cluster -n demo-tekton)" https://localhost:8081/apis/tekton.dev/v1beta1/pipelineruns | jq
```

get pipelineruns in a specific namepspace

```bash
curl -s --cacert ca.crt -H "Authorization: Bearer $(kubectl create token demo-tekton-namespace -n demo-tekton)" https://localhost:8081/apis/tekton.dev/v1beta1/namespaces/demo-tekton/pipelineruns | jq
```

get all jobs

```bash
curl -s --cacert ca.crt -H "Authorization: Bearer $(kubectl create token demo-jobs-cluster -n demo-jobs)" https://localhost:8081/apis/batch/v1/jobs | jq
```

get jobs in a specific namespace

```bash
curl -s --cacert ca.crt -H "Authorization: Bearer $(kubectl create token demo-jobs-namespace -n demo-jobs)" https://localhost:8081/apis/batch/v1/namespaces/demo-jobs/jobs | jq
```

cannot access resources cluster wide that you don't have permission to access

```bash
curl -s --cacert ca.crt -H "Authorization: Bearer $(kubectl create token demo-tekton-cluster -n demo-tekton)" https://localhost:8081/apis/batch/v1/jobs | jq
```

cannot access resources in namespaces where you don't have permissions

```bash
curl -s --cacert ca.crt -H "Authorization: Bearer $(kubectl create token demo-jobs-namespace -n demo-jobs)" https://localhost:8081/apis/tekton.dev/v1beta1/namespaces/other-jobs/pipelineruns | jq
```
