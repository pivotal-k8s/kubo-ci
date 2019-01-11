#!/bin/bash
set -exu -o pipefail

signal="$(basename $(find gcs-shipable-version ! -name 'url' ! -name 'generation' -type f))"
cp gcs-shipable-version/$signal gcs-shipable-version-output/shipable
tar -xzf kubo-release/kubo-*.tgz --directory kubo-release-untarred
grep "commit_hash" release.MF | awk -F ' ' '{print $2}' >> gcs-shipable-version-output/shipable