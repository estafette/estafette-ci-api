# Estafette CI

The `estafette-ci-api` component is part of the Estafette CI system documented at https://estafette.io.

Please file any issues related to Estafette CI at https://github.com/estafette/estafette-ci-central/issues

## Estafette-ci-api

This component handles all api calls for github, bitbucket and slack integrations; it serves api calls for the web frontend; and it creates build jobs in Kubernetes doing the hard work.

## Development

To start development run

```bash
git clone git@github.com:estafette/estafette-ci-api.git
cd estafette-ci-api
```

Before committing your changes run

```bash
go test
go mod tidy
```