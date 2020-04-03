#!/usr/bin/env bats

load helpers

BATS_TESTS_DIR=test/bats/tests
WAIT_TIME=60
SLEEP_TIME=1
IMAGE_TAG=e2e-$(git rev-parse --short HEAD)
PROVIDER_TEST_IMAGE=${PROVIDER_TEST_IMAGE:-"upstreamk8sci.azurecr.io/public/k8s/csi/secrets-store/provider-azure"}
SECRETS_STORE_CSI_DRIVER_PATH=${SECRETS_STORE_CSI_DRIVER_PATH:-"$GOPATH/src/k8s.io"}

export SECRET_NAME=${KEYVAULT_SECRET_NAME:-secret1}
export SECRET_ALIAS=${KEYVAULT_SECRET_ALIAS:-SECRET_1}
export SECRET_TYPE=${KEYVAULT_SECRET_TYPE:-secret}
export KEY_NAME=${KEYVAULT_KEY_NAME:-key1}
export KEY_ALIAS=${KEYVAULT_KEY_ALIAS:-KEY_1}
export KEY_TYPE=${KEYVAULT_KEY_TYPE:-key}
export SECRET_VERSION=""
export KEY_VALUE=${KEYVAULT_KEY_VALUE:-uiPCav0xdIq}


export CERT1_NAME=${CERT1_NAME:-pemcert1}
export CERT2_NAME=${CERT2_NAME:-pkcs12cert1}
export CERT1_VERSION=${CERT1_VERSION:-""}
export CERT2_VERSION=${CERT2_VERSION:-""}
export CERT1_VALUE=${CERT1_VALUE:-"LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRQ3hEbHkyS0d6RUQ0RjIKdU9BMStGMXdyQ3Bic0hKNmtDdk15SnNSeWk4WXBCUFdBUmR6OHVKK1hsQXl5MTFyajRZWUN2cXo2UWIrcnhSKwovSmM2WTM3UHVMVnhqMUZkYkluTldOZEdLTUdxTVkya1NzV1BxSjZPSGcxSlZib0pENTdkZ1ZtOEVNZlhSc2t2CnFTWDNhWjN6ZU1MSFR6MUwxcTRhZy9HY2VwOWFKZ1U1M3d2c00zWWFaT1lYUzdxZlI1UGV6WmozNGFVSUxLZDIKR2NKaWFYWU13ODliK25oVG9jUVhuZHNaMUVVcnB4T2cvVzJXUGNzVk8wVmVYUVFWdXhFUUhJQlRXTE1ES1BiRgo4bkpvYitZWTdMY3hodFlpMGU3QWVuUmJoQlBYUUVsc0hIUU5ydmpNVS9UN3dPRHcyNWpXRmJTL0ZyYzIreWg4Ck4xR2RjNkdCQWdNQkFBRUNnZ0VBQW1laHE0MjEwMk5abmlBOGJxT1hSSlVCVTQwSUJLaVpvMVV2MUMrMXNLOU8KampYSTh2MklyZXBDMXQ2QlFBSDV6NW1FQiszT2puZ3ZlMzJFVmxGRzBiUkVPa1E0dnJ6SDQwZUlUNHFZRlBxUAptSERrRjhaM3FxbDlNZVFkbVJDNTdrRVZHY3dNWG5RMm9UMDZPVWtaUmVLSFZJMlkxV1pHQktXS3hWWUEyaXVJCjRLRm9TYi9zRU9yVkFXWGhMWm44Q1R6aUs2TjFpaVhmSGtVNExxN0U1aXg2L01GUEkxV0JKMlhmM3dTc2JOWnUKalFXSlJuenBKT24rYUU1eU5HbTBvRk5nNEFEOVFPWW1LRzRFZFl6dExUSTBTVGJ5WGkvMEZNNVRHeHRVU3F5RQpBZ2pRWXR6VnFPTW03M2dTRzJabDVYdDFmTElmZEl0VU4wenhZR2ZXd3dLQmdRRGUzc1p1V3VaeE83eEZJcm1VCkVQTTJXQnN6MlZXUEdKeGRnUjZ1R0FvbWtBVi9VOXhTOTQxYkFOMkZhY3llWVhnYm1PQWpDRXIxbFIvdnplcXEKVjhQTmVjbTJ6Qk9NM0lVSXF6dFMxYTlBQjBUdWIzbUJsb0kxT0hiVkRFVXNRSmtCUm9CWk9kOTE1MDR4TlZIdwpmVFZBMXA2V25RQTBCbWFSMERtZ0lEa0xvd0tCZ1FETFlDYlpTa2hUb0RiWFdYakxTZTBrM3YrKzNvVU52MnF4CnI2YS9xU1kyVUJpVnFyaDFoTWt4eCtCWnQvcHM0QWVXVUltYWY5UTRJWW9WZzR3Rm1HVWRVL0NIUlhNWEprMW0KUlZWd0dBSUdVQ25aS1VqV3h4SlpxVFRRUmVEV2FRVXVtZzhnZ0pzSUY2NDNhMTQ4bnZVdGRIcnVQanBKc0JMbQpOUkE3UjBGd2l3S0JnUUN6OWVEMnhSR2t4MTVyMlBGTzNTejJhY2gxWW4zUzBVV1p2eVE5NFkxNHUveWtadHZXClpxeE9tbkZGUkR3RWU2SFhidWMxZ29HOHNkQ2ErNFFNVGxmOTkrUm9aWHMzMSt6WUppUDk3Q3Zab01VSlh4d1gKQnFoWFB5TzlQbTR3b0d5citmaXprNmFiOXMxTnNNZGNVRTRLOEFJWWplZlhHb0FDSjhnUVExU3N6d0tCZ0NIdQpPczBKM2FOR0dhQTRKelVUY21NeWFVeTQ1MDN4MzZVaGZ4cCs2QWNydWM1T20xUFFBWmt5bGJXaVFqK2o2T0FsCk02LzVIN2oxcjRvRFZuc2dmODR5MFBCZ24rRCszTzd4SmwzN1EyczJPS1VvaENTQk5naUxlR28vSGxIblY1djgKekFWS0w1TmNFQTdpOU9mOFJUOStMWHhPR1g5dHh0bHRoUFcrMzZZZEFvR0FKTGRuRVRIZ0RHSFJuTDJiK1hKTwpaNHZPSFpMQU1STE5nU085b25RWWU3SjJZQWc5ZmozL3M3dTY5c2pzSzJZVXF5YmVTbDJEd1ZySlpwcVlscHpWCnlCSjFoMXhaa3ZYbi9aWWYvbWVJRHBjRmtmSTBSQ2FucFNOR05FV0tFN0xaMkxlNG9meWZTWm1Xbk53bmlTQ0QKNWNUZXd6aS9OdkozakxqTmp2NkhoQzg9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURPakNDQWlLZ0F3SUJBZ0lRVG9sL3VmZTZTSE92amlkUDIxYlhNVEFOQmdrcWhraUc5dzBCQVFzRkFEQWEKTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3SGhjTk1qQXdOVEExTVRZeU1EQTBXaGNOTWpJdwpOVEExTVRZek1EQTBXakFhTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3Z2dFaU1BMEdDU3FHClNJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUN4RGx5MktHekVENEYydU9BMStGMXdyQ3Bic0hKNmtDdk0KeUpzUnlpOFlwQlBXQVJkejh1SitYbEF5eTExcmo0WVlDdnF6NlFiK3J4UisvSmM2WTM3UHVMVnhqMUZkYkluTgpXTmRHS01HcU1ZMmtTc1dQcUo2T0hnMUpWYm9KRDU3ZGdWbThFTWZYUnNrdnFTWDNhWjN6ZU1MSFR6MUwxcTRhCmcvR2NlcDlhSmdVNTN3dnNNM1lhWk9ZWFM3cWZSNVBlelpqMzRhVUlMS2QyR2NKaWFYWU13ODliK25oVG9jUVgKbmRzWjFFVXJweE9nL1cyV1Bjc1ZPMFZlWFFRVnV4RVFISUJUV0xNREtQYkY4bkpvYitZWTdMY3hodFlpMGU3QQplblJiaEJQWFFFbHNISFFOcnZqTVUvVDd3T0R3MjVqV0ZiUy9GcmMyK3loOE4xR2RjNkdCQWdNQkFBR2pmREI2Ck1BNEdBMVVkRHdFQi93UUVBd0lGb0RBSkJnTlZIUk1FQWpBQU1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUIKQmdnckJnRUZCUWNEQWpBZkJnTlZIU01FR0RBV2dCUzJrRVNzcnkzd3NqSnRiRy9wOTM4RUlVUmExekFkQmdOVgpIUTRFRmdRVXRwQkVySzh0OExJeWJXeHY2ZmQvQkNGRVd0Y3dEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBRzhhCnlJNEwzR1d4OTVVTHB2NE5TdE4zWW8vc3J4WTdVN1RTdlJUMTZOak5QNkM1NGkwVVpEUjJWSUxENEZ5dFp4TEkKZ1M5VTA3Z1pRUXpyamd1M1ZtWERtM2o5ZCtUT1lrbFlLLzIrRW5rR2tZanU5NldxbWhja1V1VDI4VVd1LytqbQozRkh4S3pLQTkwRzhsOXhuM0xyTnA5ekUxS3lESU5RN0xzSDd5OEplNXNIZ0t2aTZ3VkFZK01vSUVJajBjMnEvCi9OcndrSnBTK1RSU2JiRUZ2RGpjdThwaG5GTHA4MldCU0o2ZVFQaWo5cDMwMTlnOExZYWxkMXJ1MXlCVWZoRzQKbTF5aGFuQkd2K1g4aVE1UHNvRDNqeVJscEQ3bnJsWkVrekpJYmtYcXMwUy82Sjg5MWN1bFdVV3hocjV3dm1ubgp0ZmQ4U0JoUktVSE1UcDJTTXR3PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t"}
export CERT2_VALUE=${CERT2_VALUE:-"LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRRE9oNVBLQ3ZZa1lzVUEKNzBCNUZlSGpXcGR0QlpHeWI4OFhMcjFoOXcvSXl5MUJpODBDSldOd0I1dlNaMG8ycTlhaTJMRGtESU9hVmsrUAp0bE9wTUREZUowNkNtQTJIY1UzZTFxb2VYb0NSV2tWZkZwRGlVaTNPa0xXczNTNE9OOU56ZUZGdW0xWWp1ejRLCnlxS0FlUmZYY1hCS0pmNndFcENVclBHZ0lVd2J1eE9rKzhwWVBCUHJiLzdYT1pwcFVMYmlCOUJ5Mld6Zm9GdWcKSjNTbFZxNis0RHQxcnBSMHpaZUFVSm0vd2UwZWpncDFvOWxmeG4xRkJabkM2T0V5cnM3ZEhHcjFyby9VVlNVaApWeUxqd3dsTllObUdmYnBPd0srQjdEemRuOUlPN0pXRzd1dHpWNDg4N0FqbUdGN3Q3TVVEaytVVjNkVTQ1VG9UCjg3dU5XbzB2QWdNQkFBRUNnZ0VBRlBJb2RFNmdTc01VVDlWSlUxUFEwUW9ZWHNvNGVKY0xzeGVKR3RlL3dMdzYKZEJ5d1IyTndybEFCNXdPVmJKaXptcTNNdUQ3blZLUE85ZUw3OW50dURxOHJNSkRUUThWT0FkSldTK0RjWm93aQppbE9lTzJzeVBNZVlaVlp1bS8vZ0czbm5Fd2RXRG54Uy9TMHltMEpYYWdEclE5bmo5clRWOTZqWDZKeE5QUktSCk5xWU9OenJyZk81R2RiMzBNdVZObU5STVBsUytzdXRqQSt3K1ZCOWQrMVV5Z21kWFc3cTlTN2hweW10VTZnTU8KRTNvYVdsRDVZTWhLQjNpYStzL3RPdkNydWJsWTVMekpMOWUyUDJVUDcrS2JCd1pyQVZSZU1TNm5PQ2M1S0wwbQpGcnV4QTBhWFIvOEx0aFpnUjg4c1B3WndNMnNlNEtGaFdZTFBkTXVMNFFLQmdRRFBhNU80ZndjeVZnSnV4c2YzCkZxTnlMbk5KSDZlMUdTbHlFOHF0QkFrU0ZmQ0Z3WVIzRmZsRjBDcUtGMGY1eGRMRVhwajJOWE1WUDF5WnBHNDEKd29JYmtpY00vK28zOXNiMHlOa3ZBbnVRT0RPL3NtQnZuR2FkbnJWM1lqTWNNMU5VdDMwOGxmc2k0VzJhSlFBMwpDYU96WEtUMFN4TGlhQ09hVEdldlJ1c1lrd0tCZ1FEKzVwbS9SZUVlNS8wZTdqbmxPcFhVK2NKeS9YOWdmSHhWCktXRmR4bmcweFBTNURxRmt5ZEpoSjFPMVJNRkJkS3RadzRpdmlkR2xlZ1AraVV2L1JUdSsxdnFvRXd4L1lROVMKVVY3YXN4Wk5NSnFoTDduS3lJK1padzhDQmU5b1EyWW1VUEVhTHVBTVJ1bjZzNmx5cm9udFQ3dkZPVWdQcjUrQQpDTFdPWHNPbWRRS0JnRkNJN25SR0xoOG5NZzZjOCt0R1NQUCtnUmkxUjhLVElIcUFvTU1JdkJUZm0rSHpQMkdWCmtKSEF2Nk9hWW9IaWczRm5ZWERIVkFXOThsQmRmY1UxM3BxaDVyT3ZjZHVFMzc4UGRQUkJ2SVJFcmlNU09VdGMKcUtNdWlqcnVUL1gxSDdmVy9yTlZjSXNjaUJlL29oTzhsR2tCNGJKUXErWm9sTnBHTEVQci8wQXRBb0dCQUsyWApyaTB0RWR0U2NuZVdGYWVlOWx0TW5MaGpHMVJDY3dvc1hEclk1eFJJN2NENXpjQXVFakJIOENJSzZQSUMybzhQCk13OFk5TVdWQ3hOVnZZUGpTb1QxTTA4emFkZDE2bEZOU1NQM3dzQmUwVy9rYU10Sk8wSmxoVHNEZjcrcDV1OUIKUUhGc2F2anhjbmRoMDR4ZWdXQTBaTlF6NW5lSVN6K09ydForZ3cvaEFvR0FDVkJtOWxJelZ0MHNGWU1oRXp3Vgp1Z0VBdDI3Umc5U2RPTGpVMy9jNnZDVnJ1WHdCL0w4Uy9hNi8yTmtIVnlUY2huRWdkcUlwWUNWdlVERmRJeEtYCit4SWhnRFpMK0JDeE1JMzhxbHlHamJqY0xtd3pHL21PNHRhODdqTG5PZWtReUMzMDVMNlpNK3d5WllRV2xLNEsKazl1OXQvYXFWVzQ2Q2I4RUZVaGcvNjA9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURPakNDQWlLZ0F3SUJBZ0lRVk53RkZtVkZTMGVwYkQxckdleTZVakFOQmdrcWhraUc5dzBCQVFzRkFEQWEKTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3SGhjTk1qQXdOVEExTVRZeU1ESXhXaGNOTWpJdwpOVEExTVRZek1ESXhXakFhTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3Z2dFaU1BMEdDU3FHClNJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURPaDVQS0N2WWtZc1VBNzBCNUZlSGpXcGR0QlpHeWI4OFgKTHIxaDl3L0l5eTFCaTgwQ0pXTndCNXZTWjBvMnE5YWkyTERrRElPYVZrK1B0bE9wTUREZUowNkNtQTJIY1UzZQoxcW9lWG9DUldrVmZGcERpVWkzT2tMV3MzUzRPTjlOemVGRnVtMVlqdXo0S3lxS0FlUmZYY1hCS0pmNndFcENVCnJQR2dJVXdidXhPays4cFlQQlByYi83WE9acHBVTGJpQjlCeTJXemZvRnVnSjNTbFZxNis0RHQxcnBSMHpaZUEKVUptL3dlMGVqZ3AxbzlsZnhuMUZCWm5DNk9FeXJzN2RIR3Ixcm8vVVZTVWhWeUxqd3dsTllObUdmYnBPd0srQgo3RHpkbjlJTzdKV0c3dXR6VjQ4ODdBam1HRjd0N01VRGsrVVYzZFU0NVRvVDg3dU5XbzB2QWdNQkFBR2pmREI2Ck1BNEdBMVVkRHdFQi93UUVBd0lGb0RBSkJnTlZIUk1FQWpBQU1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUIKQmdnckJnRUZCUWNEQWpBZkJnTlZIU01FR0RBV2dCUjNlOUFVa2dHaWNNbm5qTjV1dk9MT2ZqUTNBVEFkQmdOVgpIUTRFRmdRVWQzdlFGSklCb25ESjU0emVicnppem40ME53RXdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBQU9rCkZ2TzJMTWtTWExDUkUwOUMvOTI1S2RHTGwrN1RNdkt1bUR1V0pTVXU0Zk02aVkreitBQnhLbVEzSmErNGFHSUcKbDh6Vk0wYnc2QmVHVmdBOFIrNVBFOVBwa04yOHhRRGRVNmFRMnNnUG1YUmRacEFQT2taYTZBV0RuakNxbmtLRQpsTFFpZlBsVFRxZUFuSGordlZEcDNKSnBFaTlYQnd2cFNlY2Z6amJsSVQrMzdYS25aWVA5dXdpTDZvNTlOVFVtCndReXhsd2d4dVE2VU0vbHZ3OWdlcDNZQ0NmdmU3blZoN1BnbXhaaSt5Qjl4dnZid1FQNG1Tejl5ZjFzUmJnc3oKc2UwZTlKb0R5VzVoNHRHVDNndldJNGQvRUJkb1p3UGNpa0lTUHJONzdJL3BCdFp3N0V2UVE5b0wzUDBUR3pXWApmYis3ODNBalpHMnlDNnZuT3I4PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t"}

setup() {
  if [[ -z "${AZURE_CLIENT_ID}" ]] || [[ -z "${AZURE_CLIENT_SECRET}" ]]; then
    echo "Error: Azure service principal is not provided" >&2
    return 1
  fi
}

@test "install driver helm chart" {
  run helm install csi-secrets-store ${SECRETS_STORE_CSI_DRIVER_PATH}/secrets-store-csi-driver/charts/secrets-store-csi-driver --namespace dev
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
  envsubst < $BATS_TESTS_DIR/nginx-pod-secrets-store-inline-volume.yaml | kubectl apply -f -

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline
  assert_success
}

@test "CSI inline volume test - read azure kv secret from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$SECRET_NAME)
  [[ "$result" -eq "test" ]]
}

@test "CSI inline volume test - read azure kv key from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$KEY_NAME)
  [[ "$result" == *"${KEY_VALUE}"* ]]
}

@test "CSI inline volume test - read azure pem cert from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$CERT1_NAME | base64)
  [[ `$result` -eq `${CERT1_VALUE}` ]]
}

@test "CSI inline volume test - read azure pkcs12 cert from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$CERT2_NAME | base64)
  [[ `$result` -eq `${CERT2_VALUE}` ]]
}

@test "CSI inline volume test - read azure kv secret, if alias present, from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/SECRET_1)
  [[ "$result" -eq "test" ]]
}

@test "CSI inline volume test - read azure kv key, if alias present, from pod" {
  KEY_VALUE_CONTAINS=uiPCav0xdIq
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/KEY_1)
  [[ "$result" == *"${KEY_VALUE_CONTAINS}"* ]]
}

@test "secretproviderclasses crd is established" {
  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get crd/secretproviderclasses.secrets-store.csi.x-k8s.io
  assert_success
}

@test "deploy azure secretproviderclass crd" {
  envsubst < $BATS_TESTS_DIR/azure_v1alpha1_secretproviderclass.yaml | kubectl apply -f -

  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  cmd="kubectl get secretproviderclasses.secrets-store.csi.x-k8s.io/azure -o yaml | grep azure"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"
}

@test "CSI inline volume test with pod portability" {
  run kubectl apply -f $BATS_TESTS_DIR/nginx-pod-secrets-store-inline-volume-crd.yaml
  assert_success

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-crd
  assert_success
}

@test "CSI inline volume test with pod portability - read azure kv secret from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/$SECRET_NAME)
  [[ "$result" -eq "test" ]]
}

@test "CSI inline volume test with pod portability - read azure kv key from pod" {
  KEY_VALUE_CONTAINS=uiPCav0xdIq
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/$KEY_NAME)
  [[ "$result" == *"${KEY_VALUE}"* ]]
}

@test "CSI inline volume test with pod portability - read azure pem cert from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$CERT1_NAME | base64)
  [[ `$result` -eq `${CERT1_VALUE}` ]]
}

@test "CSI inline volume test with pod portability - read azure pkcs12 cert from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline cat /mnt/secrets-store/$CERT2_NAME | base64)
  [[ `$result` -eq `${CERT2_VALUE}` ]]
}

@test "CSI inline volume test with pod portability - read azure kv secret, if alias present, from pod" {
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/SECRET_1)
  [[ "$result" -eq "test" ]]
}

@test "CSI inline volume test with pod portability - read azure kv key, if alias present, from pod" {
  KEY_VALUE_CONTAINS=uiPCav0xdIq
  result=$(kubectl exec -it nginx-secrets-store-inline-crd cat /mnt/secrets-store/KEY_1)
  [[ "$result" == *"${KEY_VALUE_CONTAINS}"* ]]
}

@test "CSI inline volume test with user assigned identity" {
  if [ ${CI_KIND_CLUSTER} ]; then
    skip "not running in azure cluster"
  fi

  envsubst < $BATS_TESTS_DIR/user-assigned-identity/azure_v1alpha1_userassignedidentityenabled.yaml | kubectl apply -f -

  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  cmd="kubectl get secretproviderclasses.secrets-store.csi.x-k8s.io/azure-msi -o yaml | grep azure"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl apply -f $BATS_TESTS_DIR/user-assigned-identity/nginx-pod-user-identity-secrets-store-inline-volume-crd.yaml
  assert_success

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd-msi"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-crd-msi
  assert_success

  result=$(kubectl exec -it nginx-secrets-store-inline-crd-msi cat /mnt/secrets-store/secret1)
  [[ "$result" -eq "test" ]]

  KEY_VALUE_CONTAINS=uiPCav0xdIq
  result=$(kubectl exec -it nginx-secrets-store-inline-crd-msi cat /mnt/secrets-store/key1)
  [[ "$result" == *"${KEY_VALUE_CONTAINS}"* ]]
}
