# Estafette CI

The `estafette-ci-api` component is part of the Estafette CI system documented at https://estafette.io.

Please file any issues related to Estafette CI at https://github.com/estafette/estafette-ci-central/issues

## Estafette-ci-api

This component handles all api calls for github, bitbucket and slack integrations; it serves api calls for the web frontend; and it creates build jobs in Kubernetes doing the hard work.

## Installation

Prepare using Helm:

```
brew install kubernetes-helm
kubectl -n kube-system create serviceaccount tiller
kubectl create clusterrolebinding tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller --wait
```

Then install or upgrade with Helm:

```
helm repo add estafette https://helm.estafette.io
helm upgrade --install estafette-ci --namespace estafette-ci estafette/estafette-ci
```

## Development

To start development run

```bash
git clone git@github.com:estafette/estafette-ci-api.git
cd estafette-ci-api
go get github.com/golang/mock/mockgen
```

Before committing your changes run

```bash
go generate ./...
go test -short ./...
go mod tidy
```