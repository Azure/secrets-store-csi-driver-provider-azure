#!/bin/bash

set -eo pipefail

target_dir="${TARGET_DIR}"

if [[ -z "${target_dir}" ]];then
  echo "target dir is not set. please set TARGET_DIR env var"
  exit 1 # if not set this will put the pod in crash loop
fi

azure_provider_dir="${target_dir}/azure"
mkdir -p ${azure_provider_dir}

cp /bin/secrets-store-csi-driver-provider-azure ${azure_provider_dir}/provider-azure

#https://github.com/kubernetes/kubernetes/issues/17182
# if we are running on kubernetes cluster as a daemon set we should
# not exit otherwise, container will restart and goes into crashloop (even if exit code is 0)
while true; do echo "install done, daemonset sleeping" && sleep 60; done