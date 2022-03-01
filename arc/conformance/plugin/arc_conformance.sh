#!/usr/bin/env bash

set -e

results_dir="${RESULTS_DIR:-/tmp/results}"
keyvault_name="arc-akv-secrets-$(openssl rand -hex 2)"
keyvault_resource_group="akv-conformance-test-$(openssl rand -hex 2)"
keyvault_location="${KEYVAULT_LOCATION:-westus2}"

waitForArc() {
    ready=false
    max_retries=60
    sleep_seconds=20

    for i in $(seq 1 $max_retries)
    do
    status=$(helm ls -a -A -o json | jq '.[]|select(.name=="azure-arc").status' -r)
    if [ "$status" == "deployed" ]; then
        echo "helm release successful"
        ready=true
        break
    elif [ "$status" == "failed" ]; then
        echo "helm release failed"
        break
    else
        echo "waiting for helm release to be successful. Status - ${status}. Attempt# $i of $max_retries"
        sleep ${sleep_seconds}
    fi
    done
}

saveResult() {
  # prepare the results for handoff to the Sonobuoy worker.
  cd "${results_dir}"
  # Sonobuoy worker expects a tar file.
  tar czf results.tar.gz ./*
  # Signal the worker by writing out the name of the results file into a "done" file.
  printf "%s/results.tar.gz" "${results_dir}" > "${results_dir}"/done
}

# Ensure that we tell the Sonobuoy worker we are done regardless of results.
trap saveResult EXIT

# initial environment variables for the plugin
setEnviornmentVariables() {
  export JUNIT_OUTPUT_FILEPATH=/tmp/results/junit.xml
  export IS_ARC_TEST=true
  export CI_KIND_CLUSTER=true
  export ACK_GINKGO_DEPRECATIONS=1.16.4 # remove this when we move to ginkgo 2.0
}

# initialize keyvault for conformance test
setupKeyVault() {
  # create resource group
  echo "INFO: Creating resource group $keyvault_resource_group"
  az group create \
  --name "$keyvault_resource_group" \
  --location "$keyvault_location" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # create keyvault
  echo "INFO: Creating key vault $keyvault_name"
  az keyvault create \
  --name "$keyvault_name" \
  --resource-group "$keyvault_resource_group" \
  --location "$keyvault_location" \
  --sku premium 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
  export KEYVAULT_NAME=$keyvault_name

  # set access policy for keyvault
  echo "INFO: Setting up key vault access policies"
  az keyvault set-policy \
  --name "$keyvault_name" \
  --resource-group "$keyvault_resource_group" \
  --spn "${AZURE_CLIENT_ID}" \
  --key-permissions get create \
  --secret-permissions get set \
  --certificate-permissions get create import 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # create keyvault secret
  echo "INFO: Creating secret in key vault"
  secret_value=$(openssl rand -hex 6)
  az keyvault secret set \
  --vault-name "$keyvault_name" \
  --name secret1 \
  --value "$secret_value" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
  export SECRET_NAME=secret1
  export SECRET_VALUE=$secret_value

  # create keyvault key
  echo "INFO: Creating keys in key vault"
  # RSA key
  key_name=key1
  az keyvault key create \
  --vault-name "$keyvault_name" \
  --name $key_name \
  --kty RSA \
  --size 2048 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  az keyvault key download \
  --vault-name "$keyvault_name" \
  --name $key_name \
  -e PEM \
  -f publicKey.pem 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  keyVersion=$(az keyvault key show --vault-name "$keyvault_name" --name $key_name --query "key.kid" | tr -d '"' | sed 's#.*/##')
  publicKeyContent=$(cat publicKey.pem)
  export KEY_NAME=$key_name
  export KEY_VALUE=$publicKeyContent
  export KEY_VERSION=$keyVersion

  # RSA-HSM Key
  az keyvault key create \
  --vault-name "$keyvault_name" \
  --name rsahsmkey1 \
  --kty RSA-HSM \
  --size 2048 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # EC-HSM Key
  az keyvault key create \
  --vault-name "$keyvault_name" \
  --name echsmkey1 \
  --kty EC-HSM \
  --curve P-256 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py


  # create keyvault certificate
  echo "INFO: Importing certificates in key vault"
  # PEM and PKCS12 certificates
  step certificate create test.domain.com test.crt test.key \
  --profile self-signed \
  --subtle \
  --san test.domain.com \
  --kty RSA \
  --not-after 86400h \
  --no-password \
  --insecure 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  openssl pkcs12 -export -in test.crt -inkey test.key -out test.pfx -passout pass: 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  az keyvault certificate import \
  --vault-name "$keyvault_name" \
  --name pemcert1 \
  --file test.pfx 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  az keyvault certificate import \
  --vault-name "$keyvault_name" \
  --name pkcs12cert1 \
  --file test.pfx 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # ECC certificates
  step certificate create test.domain.com testec.crt testec.key \
  --profile self-signed \
  --subtle \
  --san test.domain.com \
  --kty EC \
  --not-after 86400h \
  --no-password \
  --insecure 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
  
  openssl pkcs12 -export -in testec.crt -inkey testec.key -out testec.pfx -passout pass: 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  az keyvault certificate import \
  --vault-name "$keyvault_name" \
  --name ecccert1 \
  --file testec.pfx 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
}

# setup kubeconfig for conformance test
setupKubeConfig() {
  KUBECTL_CONTEXT=azure-arc-akv-test
  APISERVER=https://kubernetes.default.svc/
  TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
  cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt > ca.crt

  kubectl config set-cluster ${KUBECTL_CONTEXT} \
    --embed-certs=true \
    --server=${APISERVER} \
    --certificate-authority=./ca.crt 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  kubectl config set-credentials ${KUBECTL_CONTEXT} --token="${TOKEN}" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # Delete previous rolebinding if exists. And ignore the error if not found.
  kubectl delete clusterrolebinding clusterconnect-binding --ignore-not-found
  kubectl create clusterrolebinding clusterconnect-binding --clusterrole=cluster-admin --user="${OBJECT_ID}" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  kubectl config set-context ${KUBECTL_CONTEXT} \
    --cluster=${KUBECTL_CONTEXT} \
    --user=${KUBECTL_CONTEXT} \
    --namespace=default 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  kubectl config use-context ${KUBECTL_CONTEXT} 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
  echo "INFO: KubeConfig setup complete"
}

# validate enviorment variables
if [ -z "${TENANT_ID}" ]; then
  echo "ERROR: parameter TENANT_ID is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

if [ -z "${SUBSCRIPTION_ID}" ]; then
  echo "ERROR: parameter SUBSCRIPTION_ID is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

if [ -z "${AZURE_CLIENT_ID}" ]; then
  echo "ERROR: parameter AZURE_CLIENT_ID is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

if [ -z "${AZURE_CLIENT_SECRET}" ]; then
  echo "ERROR: parameter AZURE_CLIENT_SECRET is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

if [ -z "${ARC_CLUSTER_NAME}" ]; then
  echo "ERROR: parameter ARC_CLUSTER_NAME is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

if [ -z "${ARC_CLUSTER_RG}" ]; then
  echo "ERROR: parameter ARC_CLUSTER_RG is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

# OBJECT_ID is an id of the Service Principal created in conformance test subscription.
if [ -z "${OBJECT_ID}" ]; then
  echo "ERROR: parameter OBJECT_ID is required." > "${results_dir}"/error
  python3 /arc/setup_failure_handler.py
fi

# add az cli extensions 
az extension add --name aks-preview
az extension add --name k8s-extension

# login with service principal
az login --service-principal \
  -u "${AZURE_CLIENT_ID}" \
  -p "${AZURE_CLIENT_SECRET}" \
  --tenant "${TENANT_ID}" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

az account set --subscription "${SUBSCRIPTION_ID}" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

# set environment variables
setEnviornmentVariables

# setup keyvault
setupKeyVault

# setup Kubeconfig
setupKubeConfig

# Wait for resources in ARC agents to come up
echo "INFO: Waiting for ConnectedCluster to come up"
waitSuccessArc="$(waitForArc)"
if [ "${waitSuccessArc}" == false ]; then
    echo "helm release azure-arc failed" > "${results_dir}"/error
    python3 /arc/setup_failure_handler.py
    exit 1
else
    echo "INFO: ConnectedCluster is available"
fi

echo "INFO: Creating extension"
az k8s-extension create \
      --name arc-akv-conformance \
      --extension-type Microsoft.AzureKeyVaultSecretsProvider \
      --scope cluster \
      --cluster-name "${ARC_CLUSTER_NAME}" \
      --resource-group "${ARC_CLUSTER_RG}" \
      --cluster-type connectedClusters \
      --release-train preview \
      --release-namespace kube-system \
      --configuration-settings 'secrets-store-csi-driver.enableSecretRotation=true' \
        'secrets-store-csi-driver.rotationPollInterval=30s' \
        'secrets-store-csi-driver.syncSecret.enabled=true' 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

# wait for secrets store csi driver and provider pods
kubectl wait pod -n kube-system --for=condition=Ready -l app=secrets-store-csi-driver
kubectl wait pod -n kube-system --for=condition=Ready -l app=csi-secrets-store-provider-azure

/arc/e2e -ginkgo.v -ginkgo.skip="${GINKGO_SKIP}" -ginkgo.focus="${GINKGO_FOCUS}"

# clean up test resources
echo "INFO: cleaning up test resources" 
az k8s-extension delete \
  --name arc-akv-conformance \
  --resource-group "${ARC_CLUSTER_RG}" \
  --cluster-type connectedClusters \
  --cluster-name "${ARC_CLUSTER_NAME}" \
  --force \
  --yes \
  --no-wait

az group delete \
  --name "$keyvault_resource_group" \
  --yes \
  --no-wait
