# Kube-Trash

## Project Overview

Kube-Trash is a measure designed to record and prevent accidental deletion of resources on Kubernetes. It only requires pre-configuration of the resources to be "protected" and the storage object (which only needs to be S3-compatible).

## Operation Demonstration

Client Operation

```shell
$ kubectl apply -f deployment.yaml
deployment.apps/nginx-deployment created
$ kubectl delete -f deployment.yaml
deployment.apps "nginx-deployment" deleted
```

We check the rescue status of this resource from the server side

```shell
Extracted API server from kubeconfig: https://127.0.0.1:63740
Using API server: https://127.0.0.1:63740
W0918 15:09:32.194231   69337 warnings.go:70] v1 ComponentStatus is deprecated in v1.19+
W0918 15:09:32.195482   69337 warnings.go:70] v1 ComponentStatus is deprecated in v1.19+
2024/09/18 15:09:41 Delete default Deployment nginx-deployment
2024/09/18 15:09:41 Processing item: nginx-deployment
2024/09/18 15:09:41 Successfully uploaded default/nginx-deployment_Deployment_2024-09-18_150941.yaml of size 7853
2024/09/18 15:09:41 Successfully uploaded default/nginx-deployment_Deployment_2024-09-18_150941.yaml to MinIO
```

Download the backed-up file for inspection

```shell
$ curl -O http://127.0.0.1:9000/browser/kube-trash/default%2Fnginx-deployment_Deployment_2024-09-18_142316.yaml
```

## Note
If you encounter errors like Failed to watch *unstructured.Unstructured in the logs, you can ignore them. Many resources do not define their Kind according to the standard, but this does not affect our processing logic.