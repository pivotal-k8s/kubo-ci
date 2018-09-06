#!/bin/bash

set -exu -o pipefail


export BOSH_ENVIRONMENT="$(jq -r .target gcs-source-json/source.json)"
export BOSH_CLIENT="$(jq -r .client gcs-source-json/source.json)"
export BOSH_CLIENT_SECRET="$(jq -r .client_secret gcs-source-json/source.json)"
export BOSH_CA_CERT="$(jq -r .ca_cert gcs-source-json/source.json)"

export BOSH_LOG_LEVEL=debug
export BOSH_LOG_PATH="$PWD/bosh.log"

set +x
bosh -d "${DEPLOYMENT_NAME}" -n delete-deployment --force
