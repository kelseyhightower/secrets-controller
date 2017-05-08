# Secrets Controller

The secrets-controller is example Kubernetes controller that syncs a single vault secret to Kubernetes.

## Usage

```
kubectl create -f secrets-controller-rs.yaml
``` 

## Build Container

```
gcloud container builds submit --config cloudbuild.yaml .
```
