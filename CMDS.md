```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-tekton-cluster -n demo-tekton) https://localhost:8081/apis/tekton.dev/v1beta1/pipelineruns | jq
```

```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-tekton-namespace -n demo-tekton) https://localhost:8081/apis/tekton.dev/v1beta1/namespace/demo-tekton/pipelineruns | jq
```

```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-jobs-cluster -n demo-jobs) https://localhost:8081/apis/batch/v1/jobs | jq
```

```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-jobs-namespace -n demo-jobs) https://localhost:8081/apis/batch/v1/namespace/demo-jobs/jobs | jq
```

```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-tekton-cluster -n demo-tekton) https://localhost:8081/apis/batch/v1/jobs | jq
```

```bash
curl -s --cacert ca.crt -H Authorization: Bearer $(kubectl create token demo-jobs-namespace -n demo-jobs) https://localhost:8081/apis/tekton.dev/v1beta1/namespace/demo-jobs/pipelineruns | jq
```
