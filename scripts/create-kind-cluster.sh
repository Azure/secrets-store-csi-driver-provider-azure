#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

: "${SERVICE_ACCOUNT_ISSUER:?Environment variable empty or not defined.}"

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1

SERVICE_ACCOUNT_SIGNING_KEY_FILE="$(pwd)/sa.key"
SERVICE_ACCOUNT_KEY_FILE="$(pwd)/sa.pub"

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"

create_kind_cluster() {
  # create a kind cluster
  cat <<EOF | kind create cluster --name "${KIND_CLUSTER_NAME}" --image "kindest/node:${KIND_K8S_VERSION:-v1.22.4}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
    - hostPath: ${SERVICE_ACCOUNT_KEY_FILE}
      containerPath: /etc/kubernetes/pki/sa.pub
    - hostPath: ${SERVICE_ACCOUNT_SIGNING_KEY_FILE}
      containerPath: /etc/kubernetes/pki/sa.key
    # Load environment json into kind node to enable custom cloud coverage
    - hostPath: test/custom_environment.json
      containerPath: /etc/kubernetes/custom_environment.json
      readOnly: true
      propagation: None
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
    taints:
    - key: "kubeadmNode"
      value: "master"
      effect: "NoSchedule"
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        service-account-issuer: ${SERVICE_ACCOUNT_ISSUER}
        service-account-key-file: /etc/kubernetes/pki/sa.pub
        service-account-signing-key-file: /etc/kubernetes/pki/sa.key
    controllerManager:
      extraArgs:
        service-account-private-key-file: /etc/kubernetes/pki/sa.key
EOF

  kubectl wait node "${KIND_CLUSTER_NAME}-control-plane" --for=condition=Ready --timeout=90s
}

download_service_account_keys() {
  if [[ -z "${SERVICE_ACCOUNT_KEYVAULT_NAME:-}" ]]; then
    return
  fi
  az keyvault secret show --vault-name "${SERVICE_ACCOUNT_KEYVAULT_NAME}" --name workload-identity-sa-pub | jq -r .value | base64 -d > "${SERVICE_ACCOUNT_KEY_FILE}"
  az keyvault secret show --vault-name "${SERVICE_ACCOUNT_KEYVAULT_NAME}" --name workload-identity-sa-key | jq -r .value | base64 -d > "${SERVICE_ACCOUNT_SIGNING_KEY_FILE}"
}

download_service_account_keys
create_kind_cluster
