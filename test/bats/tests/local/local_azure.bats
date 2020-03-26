#!/usr/bin/env bats

load ../../helpers
BATS_LOCAL_TESTS_DIR=test/bats/tests/local
WAIT_TIME=60
SLEEP_TIME=1
IMAGE_TAG=e2e-$(git rev-parse --short HEAD)

setup() {
  if [[ -z "${AZURE_CLIENT_ID}" ]] || [[ -z "${AZURE_CLIENT_SECRET}" ]]; then
    echo "Error: Azure service principal is not provided" >&2
    return 1
  fi

}

@test "install driver helm chart" {
  run helm install csi-secrets-store ../secrets-store-csi-driver/charts/secrets-store-csi-driver --namespace dev
  assert_success
}

@test "install azure provider with e2e image" {
  yq w deployment/provider-azure-installer.yaml "spec.template.spec.containers[0].image" "${PROVIDER_TEST_IMAGE}:${IMAGE_TAG}" \
   | yq w - spec.template.spec.containers[0].imagePullPolicy "IfNotPresent" | kubectl apply -n dev -f -
}

@test "create azure k8s secret" {
  run kubectl create secret generic secrets-store-creds --from-literal clientid=${AZURE_CLIENT_ID} --from-literal clientsecret=${AZURE_CLIENT_SECRET}
  assert_success
}

@test "CSI inline volume test" {
  envsubst < $BATS_LOCAL_TESTS_DIR/nginx-pod-secrets-store-inline-volume-local.yaml | kubectl apply -f -

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline
  assert_success
}

@test "CSI inline volume test - read azure kv object #1 from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$KEYVAULT_OBJECT_NAME1)
  [[ "$result" -eq "test" ]]
}

@test "CSI inline volume test - read azure kv object #2 from pod" {
  KEY_VALUE_CONTAINS=$KEYVAULT_OBJECT_VALUE2

  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$KEYVAULT_OBJECT_NAME2)
  [[ "$result" == *"${KEY_VALUE_CONTAINS}"* ]]
}

@test "secretproviderclasses crd is established" {
  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get crd/secretproviderclasses.secrets-store.csi.x-k8s.io
  assert_success
}

@test "deploy azure secretproviderclass crd" {
  envsubst < $BATS_LOCAL_TESTS_DIR/azure_v1alpha1_secretproviderclass_local.yaml | kubectl apply -f -

  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  cmd="kubectl get secretproviderclasses.secrets-store.csi.x-k8s.io/azure -o yaml | grep azure"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"
}

@test "CSI inline volume test with pod portability" {
  run kubectl apply -f $BATS_LOCAL_TESTS_DIR/nginx-pod-secrets-store-inline-volume-crd-local.yaml
  assert_success

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-crd
  assert_success
}

@test "CSI inline volume test with pod portability - read azure kv object #1 from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/$KEYVAULT_OBJECT_NAME1)
  [[ "$result" -eq "test" ]]
}

@test "CSI inline volume test with pod portability - read azure kv object #2 from pod" {
  KEY_VALUE_CONTAINS=$KEYVAULT_OBJECT_VALUE2
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/$KEYVAULT_OBJECT_NAME2)
  [[ "$result" == *"${KEY_VALUE_CONTAINS}"* ]]
}

@test "CSI inline volume test with pod portability - read azure kv object #3  if alias present from pod" {
  KEY_VAULT_CONTAINS=$KEYVAULT_OBJECT_VALUE3
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/$KEYVAULT_OBJECT_ALIAS3)
}
