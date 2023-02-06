#!/usr/bin/env bash

if pgrep gcs-migrator; then
  echo "gcs-migrator is running"
else
  exit 1
fi
