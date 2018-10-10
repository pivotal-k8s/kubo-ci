#!/bin/bash
set -eux -o pipefail
ci_dir="$(cd "$(dirname "$0")"; pwd)"

print_usage() {
  echo "Usage:"
  echo "    $0 <pipeline name> <gcp> [branch name] "
  echo ""
  echo "    valid pipeline names:"
  for name in "${ci_dir}"/configs/*.yml; do
    local pipeline_name
    pipeline_name="$(basename "${name}")"
    echo "        - ${pipeline_name%.yml}"
  done
  echo
  echo 'Use following command to set all pipelines'
  echo 'find configs/* -maxdepth 0 -name \*.yml -exec ./set_pipeline $(basename {}) \;'
}

extract_pipeline_name() {
  local pipeline_name="$1"
  local pipeline_filename="${ci_dir}/configs/${pipeline_name}.yml"
  if [ ! -f "${pipeline_filename}" ]; then
    pipeline_filename="${ci_dir}/configs/${pipeline_name}"
    if [ ! -f "${pipeline_filename}" ]; then
      echo "Unknown pipeline name ${pipeline_name}"
      print_usage
      exit 1
    fi
  fi
  pipeline_name=$(basename "${pipeline_filename}")
  echo -n "${pipeline_name%.*}"
}

main() {
  local pipeline_name pipeline_config env iaas
  if [ "$#" == "0" ]; then
    print_usage
    exit 1
  fi
  templated_pipeline=$(mktemp)
  trap 'rm -f ${templated_pipeline}' EXIT
  pipeline_name=$(extract_pipeline_name "${1}")
  iaas="${2}"
  local pipeline_properties="${ci_dir}/configs/${pipeline_name}.yml"

  # pipeline_config="$(erb "$ci_dir"/templates/template.yml)" # > "$templated_pipeline"
  local pipeline_ops_file="${ci_dir}/templates/ops-files/${pipeline_name}.yml"
  if [ -f "${pipeline_ops_file}" ]; then
    pipeline_config=$(bosh int "$ci_dir"/templates/template.yml --ops-file "${pipeline_ops_file}")
  fi

  echo "$pipeline_config"

  bosh int <(echo "${pipeline_config}") \
    --ops-file "$ci_dir"/templates/ops-files/"$iaas"/default.yml \
    --vars-file "$pipeline_properties" \
    --var-errs-unused \
    > "$templated_pipeline"

  fly --target kubo sync > /dev/null
  fly --target kubo set-pipeline \
    --config "$templated_pipeline" \
    --pipeline "${pipeline_name}"
  #   --load-vars-from "${pipeline_properties}"
}
pushd "${ci_dir}" > /dev/null
main "$@"
popd > /dev/null
