#!/usr/bin/env bash

start_gcs_migrator()
{
  retry=1
  while true; do
    if [[ $retry -gt 5 ]]; then
      >&2 echo "could not start gcs-migrator after 5 retries, backing off"
      exit 1
    fi
    /gcs-migrator
    sleep "$retries"
    >&2 echo "Somthing went wrong, restarting gcs-migrator"
    retry=$((retry+1))
  done
}

start_gcs_migrator &

/estafette-ci-api
