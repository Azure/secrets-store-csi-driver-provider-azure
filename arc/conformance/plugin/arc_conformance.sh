#!/usr/bin/env bash

set -e
# disable globbing
set -f 

results_dir="${RESULTS_DIR:-/tmp/results}"
keyvault_name="arc-akv-secrets-$(openssl rand -hex 2)"
keyvault_resource_group="akv-conformance-test-$(openssl rand -hex 2)"
keyvault_location="${KEYVAULT_LOCATION:-westus2}"

function waitForResources {
    available=false
    max_retries=60
    RESOURCE=$1
    NAMESPACE=$2
    for i in $(seq 1 $max_retries)
    do
    echo "Waiting for $RESOURCE in $NAMESPACE to be available. Attempt# $i of $max_retries"
    if [[ $(kubectl wait --for=condition=available "${RESOURCE}" --all --namespace "${NAMESPACE}") ]]; then
        available=true
        break
    fi
    done
    
    echo "$available"
}


saveResult() {
  # prepare the results for handoff to the Sonobuoy worker.
  cd "${results_dir}"
  # Sonobuoy worker expects a tar file.
  tar czf results.tar.gz *
  # Signal the worker by writing out the name of the results file into a "done" file.
  printf "%s"/results.tar.gz, "${results_dir}" > "${results_dir}"/done
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
  az group create \
  --name "$keyvault_resource_group" \
  --location "$keyvault_location" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # create keyvault
  az keyvault create \
  --name "$keyvault_name" \
  --resource-group "$keyvault_resource_group" \
  --location "$keyvault_location" \
  --sku premium 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
  export KEYVAULT_NAME=$keyvault_name

  # set access policy for keyvault
  az keyvault set-policy \
  --name "$keyvault_name" \
  --resource-group "$keyvault_resource_group" \
  --spn "${AZURE_CLIENT_ID}" \
  --key-permissions get create \
  --secret-permissions get set \
  --certificate-permissions get create import 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py

  # create keyvault secret
  secret_value=$(openssl rand -hex 6)
  az keyvault secret set \
  --vault-name "$keyvault_name" \
  --name secret1 \
  --value "$secret_value" 2> "${results_dir}"/error || python3 /arc/setup_failure_handler.py
  export SECRET_NAME=secret1
  export SECRET_VALUE=$secret_value

  # create keyvault key
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

# wait for resources in ARC namespace
waitSuccessArc="$(waitForResources deployment azure-arc)"
if [ "${waitSuccessArc}" == false ]; then
    echo "ERROR: deployment is not avilable in namespace - azure-arc" > "${results_dir}"/error
    python3 /arc/setup_failure_handler.py
    exit 1
fi

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
        'secrets-store-csi-driver.syncSecret.enabled=true'

# wait for secrets store csi driver and provider pods
kubectl wait pod -n kube-system --for=condition=Ready -l app=secrets-store-csi-driver
kubectl wait pod -n kube-system --for=condition=Ready -l app=csi-secrets-store-provider-azure

/arc/e2e -ginkgo.v -ginkgo.skip="${GINKGO_SKIP}"

# clean up test resources
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