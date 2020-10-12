#!/usr/bin/env bats

load helpers

BATS_TESTS_DIR=test/bats/tests
WAIT_TIME=60
SLEEP_TIME=1
IMAGE_TAG=${IMAGE_TAG:-e2e-$(git rev-parse --short HEAD)}
PROVIDER_TEST_IMAGE=${PROVIDER_TEST_IMAGE:-"upstreamk8sci.azurecr.io/public/k8s/csi/secrets-store/provider-azure"}
AZURE_ENVIRONMENT=${AZURE_ENVIRONMENT:-"AzurePublicCloud"}
AZURE_ENVIRONMENT_FILEPATH=${AZURE_ENVIRONMENT_FILEPATH:-""}
NODE_SELECTOR_OS=linux
BASE64_FLAGS="-w 0"
if [[ "$OSTYPE" == *"darwin"* ]]; then
  BASE64_FLAGS="-b 0"
fi

if [ -z "$AUTO_ROTATE_SECRET_NAME" ]; then
    export AUTO_ROTATE_SECRET_NAME=secret-$(openssl rand -hex 6)
fi

CONTAINER_IMAGE=nginx
EXEC_COMMAND="cat /mnt/secrets-store"

if [ $TEST_WINDOWS ]; then
  CONTAINER_IMAGE=mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
  EXEC_COMMAND="powershell.exe cat /mnt/secrets-store"
  NODE_SELECTOR_OS=windows
fi

export OBJECT1_NAME=${OBJECT1_NAME:-secret1}
export OBJECT1_ALIAS=${OBJECT1_ALIAS:-SECRET_1}
export OBJECT1_TYPE=${OBJECT1_TYPE:-secret}
export OBJECT1_VALUE=${OBJECT1_VALUE:-test}

export OBJECT2_NAME=${OBJECT2_NAME:-key1}
export OBJECT2_ALIAS=${OBJECT2_ALIAS:-KEY_1}
export OBJECT2_TYPE=${OBJECT2_TYPE:-key}
export OBJECT2_VALUE=${OBJECT2_VALUE:-uiPCav0xdIq}

export CERT1_NAME=${CERT1_NAME:-pemcert1}
export CERT2_NAME=${CERT2_NAME:-pkcs12cert1}
export CERT3_NAME=${CERT3_NAME:-ecccert1}

export CERT1_VERSION=${CERT1_VERSION:-""}
export CERT2_VERSION=${CERT2_VERSION:-""}
export CERT3_VERSION=${CERT3_VERSION:-""}

export CERT1_SECRET_VALUE=${CERT1_SECRET_VALUE:-"LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRQ3hEbHkyS0d6RUQ0RjIKdU9BMStGMXdyQ3Bic0hKNmtDdk15SnNSeWk4WXBCUFdBUmR6OHVKK1hsQXl5MTFyajRZWUN2cXo2UWIrcnhSKwovSmM2WTM3UHVMVnhqMUZkYkluTldOZEdLTUdxTVkya1NzV1BxSjZPSGcxSlZib0pENTdkZ1ZtOEVNZlhSc2t2CnFTWDNhWjN6ZU1MSFR6MUwxcTRhZy9HY2VwOWFKZ1U1M3d2c00zWWFaT1lYUzdxZlI1UGV6WmozNGFVSUxLZDIKR2NKaWFYWU13ODliK25oVG9jUVhuZHNaMUVVcnB4T2cvVzJXUGNzVk8wVmVYUVFWdXhFUUhJQlRXTE1ES1BiRgo4bkpvYitZWTdMY3hodFlpMGU3QWVuUmJoQlBYUUVsc0hIUU5ydmpNVS9UN3dPRHcyNWpXRmJTL0ZyYzIreWg4Ck4xR2RjNkdCQWdNQkFBRUNnZ0VBQW1laHE0MjEwMk5abmlBOGJxT1hSSlVCVTQwSUJLaVpvMVV2MUMrMXNLOU8KampYSTh2MklyZXBDMXQ2QlFBSDV6NW1FQiszT2puZ3ZlMzJFVmxGRzBiUkVPa1E0dnJ6SDQwZUlUNHFZRlBxUAptSERrRjhaM3FxbDlNZVFkbVJDNTdrRVZHY3dNWG5RMm9UMDZPVWtaUmVLSFZJMlkxV1pHQktXS3hWWUEyaXVJCjRLRm9TYi9zRU9yVkFXWGhMWm44Q1R6aUs2TjFpaVhmSGtVNExxN0U1aXg2L01GUEkxV0JKMlhmM3dTc2JOWnUKalFXSlJuenBKT24rYUU1eU5HbTBvRk5nNEFEOVFPWW1LRzRFZFl6dExUSTBTVGJ5WGkvMEZNNVRHeHRVU3F5RQpBZ2pRWXR6VnFPTW03M2dTRzJabDVYdDFmTElmZEl0VU4wenhZR2ZXd3dLQmdRRGUzc1p1V3VaeE83eEZJcm1VCkVQTTJXQnN6MlZXUEdKeGRnUjZ1R0FvbWtBVi9VOXhTOTQxYkFOMkZhY3llWVhnYm1PQWpDRXIxbFIvdnplcXEKVjhQTmVjbTJ6Qk9NM0lVSXF6dFMxYTlBQjBUdWIzbUJsb0kxT0hiVkRFVXNRSmtCUm9CWk9kOTE1MDR4TlZIdwpmVFZBMXA2V25RQTBCbWFSMERtZ0lEa0xvd0tCZ1FETFlDYlpTa2hUb0RiWFdYakxTZTBrM3YrKzNvVU52MnF4CnI2YS9xU1kyVUJpVnFyaDFoTWt4eCtCWnQvcHM0QWVXVUltYWY5UTRJWW9WZzR3Rm1HVWRVL0NIUlhNWEprMW0KUlZWd0dBSUdVQ25aS1VqV3h4SlpxVFRRUmVEV2FRVXVtZzhnZ0pzSUY2NDNhMTQ4bnZVdGRIcnVQanBKc0JMbQpOUkE3UjBGd2l3S0JnUUN6OWVEMnhSR2t4MTVyMlBGTzNTejJhY2gxWW4zUzBVV1p2eVE5NFkxNHUveWtadHZXClpxeE9tbkZGUkR3RWU2SFhidWMxZ29HOHNkQ2ErNFFNVGxmOTkrUm9aWHMzMSt6WUppUDk3Q3Zab01VSlh4d1gKQnFoWFB5TzlQbTR3b0d5citmaXprNmFiOXMxTnNNZGNVRTRLOEFJWWplZlhHb0FDSjhnUVExU3N6d0tCZ0NIdQpPczBKM2FOR0dhQTRKelVUY21NeWFVeTQ1MDN4MzZVaGZ4cCs2QWNydWM1T20xUFFBWmt5bGJXaVFqK2o2T0FsCk02LzVIN2oxcjRvRFZuc2dmODR5MFBCZ24rRCszTzd4SmwzN1EyczJPS1VvaENTQk5naUxlR28vSGxIblY1djgKekFWS0w1TmNFQTdpOU9mOFJUOStMWHhPR1g5dHh0bHRoUFcrMzZZZEFvR0FKTGRuRVRIZ0RHSFJuTDJiK1hKTwpaNHZPSFpMQU1STE5nU085b25RWWU3SjJZQWc5ZmozL3M3dTY5c2pzSzJZVXF5YmVTbDJEd1ZySlpwcVlscHpWCnlCSjFoMXhaa3ZYbi9aWWYvbWVJRHBjRmtmSTBSQ2FucFNOR05FV0tFN0xaMkxlNG9meWZTWm1Xbk53bmlTQ0QKNWNUZXd6aS9OdkozakxqTmp2NkhoQzg9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURPakNDQWlLZ0F3SUJBZ0lRVG9sL3VmZTZTSE92amlkUDIxYlhNVEFOQmdrcWhraUc5dzBCQVFzRkFEQWEKTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3SGhjTk1qQXdOVEExTVRZeU1EQTBXaGNOTWpJdwpOVEExTVRZek1EQTBXakFhTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3Z2dFaU1BMEdDU3FHClNJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUN4RGx5MktHekVENEYydU9BMStGMXdyQ3Bic0hKNmtDdk0KeUpzUnlpOFlwQlBXQVJkejh1SitYbEF5eTExcmo0WVlDdnF6NlFiK3J4UisvSmM2WTM3UHVMVnhqMUZkYkluTgpXTmRHS01HcU1ZMmtTc1dQcUo2T0hnMUpWYm9KRDU3ZGdWbThFTWZYUnNrdnFTWDNhWjN6ZU1MSFR6MUwxcTRhCmcvR2NlcDlhSmdVNTN3dnNNM1lhWk9ZWFM3cWZSNVBlelpqMzRhVUlMS2QyR2NKaWFYWU13ODliK25oVG9jUVgKbmRzWjFFVXJweE9nL1cyV1Bjc1ZPMFZlWFFRVnV4RVFISUJUV0xNREtQYkY4bkpvYitZWTdMY3hodFlpMGU3QQplblJiaEJQWFFFbHNISFFOcnZqTVUvVDd3T0R3MjVqV0ZiUy9GcmMyK3loOE4xR2RjNkdCQWdNQkFBR2pmREI2Ck1BNEdBMVVkRHdFQi93UUVBd0lGb0RBSkJnTlZIUk1FQWpBQU1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUIKQmdnckJnRUZCUWNEQWpBZkJnTlZIU01FR0RBV2dCUzJrRVNzcnkzd3NqSnRiRy9wOTM4RUlVUmExekFkQmdOVgpIUTRFRmdRVXRwQkVySzh0OExJeWJXeHY2ZmQvQkNGRVd0Y3dEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBRzhhCnlJNEwzR1d4OTVVTHB2NE5TdE4zWW8vc3J4WTdVN1RTdlJUMTZOak5QNkM1NGkwVVpEUjJWSUxENEZ5dFp4TEkKZ1M5VTA3Z1pRUXpyamd1M1ZtWERtM2o5ZCtUT1lrbFlLLzIrRW5rR2tZanU5NldxbWhja1V1VDI4VVd1LytqbQozRkh4S3pLQTkwRzhsOXhuM0xyTnA5ekUxS3lESU5RN0xzSDd5OEplNXNIZ0t2aTZ3VkFZK01vSUVJajBjMnEvCi9OcndrSnBTK1RSU2JiRUZ2RGpjdThwaG5GTHA4MldCU0o2ZVFQaWo5cDMwMTlnOExZYWxkMXJ1MXlCVWZoRzQKbTF5aGFuQkd2K1g4aVE1UHNvRDNqeVJscEQ3bnJsWkVrekpJYmtYcXMwUy82Sjg5MWN1bFdVV3hocjV3dm1ubgp0ZmQ4U0JoUktVSE1UcDJTTXR3PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="}
export CERT2_SECRET_VALUE=${CERT2_SECRET_VALUE:-"LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2UUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktjd2dnU2pBZ0VBQW9JQkFRRE9oNVBLQ3ZZa1lzVUEKNzBCNUZlSGpXcGR0QlpHeWI4OFhMcjFoOXcvSXl5MUJpODBDSldOd0I1dlNaMG8ycTlhaTJMRGtESU9hVmsrUAp0bE9wTUREZUowNkNtQTJIY1UzZTFxb2VYb0NSV2tWZkZwRGlVaTNPa0xXczNTNE9OOU56ZUZGdW0xWWp1ejRLCnlxS0FlUmZYY1hCS0pmNndFcENVclBHZ0lVd2J1eE9rKzhwWVBCUHJiLzdYT1pwcFVMYmlCOUJ5Mld6Zm9GdWcKSjNTbFZxNis0RHQxcnBSMHpaZUFVSm0vd2UwZWpncDFvOWxmeG4xRkJabkM2T0V5cnM3ZEhHcjFyby9VVlNVaApWeUxqd3dsTllObUdmYnBPd0srQjdEemRuOUlPN0pXRzd1dHpWNDg4N0FqbUdGN3Q3TVVEaytVVjNkVTQ1VG9UCjg3dU5XbzB2QWdNQkFBRUNnZ0VBRlBJb2RFNmdTc01VVDlWSlUxUFEwUW9ZWHNvNGVKY0xzeGVKR3RlL3dMdzYKZEJ5d1IyTndybEFCNXdPVmJKaXptcTNNdUQ3blZLUE85ZUw3OW50dURxOHJNSkRUUThWT0FkSldTK0RjWm93aQppbE9lTzJzeVBNZVlaVlp1bS8vZ0czbm5Fd2RXRG54Uy9TMHltMEpYYWdEclE5bmo5clRWOTZqWDZKeE5QUktSCk5xWU9OenJyZk81R2RiMzBNdVZObU5STVBsUytzdXRqQSt3K1ZCOWQrMVV5Z21kWFc3cTlTN2hweW10VTZnTU8KRTNvYVdsRDVZTWhLQjNpYStzL3RPdkNydWJsWTVMekpMOWUyUDJVUDcrS2JCd1pyQVZSZU1TNm5PQ2M1S0wwbQpGcnV4QTBhWFIvOEx0aFpnUjg4c1B3WndNMnNlNEtGaFdZTFBkTXVMNFFLQmdRRFBhNU80ZndjeVZnSnV4c2YzCkZxTnlMbk5KSDZlMUdTbHlFOHF0QkFrU0ZmQ0Z3WVIzRmZsRjBDcUtGMGY1eGRMRVhwajJOWE1WUDF5WnBHNDEKd29JYmtpY00vK28zOXNiMHlOa3ZBbnVRT0RPL3NtQnZuR2FkbnJWM1lqTWNNMU5VdDMwOGxmc2k0VzJhSlFBMwpDYU96WEtUMFN4TGlhQ09hVEdldlJ1c1lrd0tCZ1FEKzVwbS9SZUVlNS8wZTdqbmxPcFhVK2NKeS9YOWdmSHhWCktXRmR4bmcweFBTNURxRmt5ZEpoSjFPMVJNRkJkS3RadzRpdmlkR2xlZ1AraVV2L1JUdSsxdnFvRXd4L1lROVMKVVY3YXN4Wk5NSnFoTDduS3lJK1padzhDQmU5b1EyWW1VUEVhTHVBTVJ1bjZzNmx5cm9udFQ3dkZPVWdQcjUrQQpDTFdPWHNPbWRRS0JnRkNJN25SR0xoOG5NZzZjOCt0R1NQUCtnUmkxUjhLVElIcUFvTU1JdkJUZm0rSHpQMkdWCmtKSEF2Nk9hWW9IaWczRm5ZWERIVkFXOThsQmRmY1UxM3BxaDVyT3ZjZHVFMzc4UGRQUkJ2SVJFcmlNU09VdGMKcUtNdWlqcnVUL1gxSDdmVy9yTlZjSXNjaUJlL29oTzhsR2tCNGJKUXErWm9sTnBHTEVQci8wQXRBb0dCQUsyWApyaTB0RWR0U2NuZVdGYWVlOWx0TW5MaGpHMVJDY3dvc1hEclk1eFJJN2NENXpjQXVFakJIOENJSzZQSUMybzhQCk13OFk5TVdWQ3hOVnZZUGpTb1QxTTA4emFkZDE2bEZOU1NQM3dzQmUwVy9rYU10Sk8wSmxoVHNEZjcrcDV1OUIKUUhGc2F2anhjbmRoMDR4ZWdXQTBaTlF6NW5lSVN6K09ydForZ3cvaEFvR0FDVkJtOWxJelZ0MHNGWU1oRXp3Vgp1Z0VBdDI3Umc5U2RPTGpVMy9jNnZDVnJ1WHdCL0w4Uy9hNi8yTmtIVnlUY2huRWdkcUlwWUNWdlVERmRJeEtYCit4SWhnRFpMK0JDeE1JMzhxbHlHamJqY0xtd3pHL21PNHRhODdqTG5PZWtReUMzMDVMNlpNK3d5WllRV2xLNEsKazl1OXQvYXFWVzQ2Q2I4RUZVaGcvNjA9Ci0tLS0tRU5EIFBSSVZBVEUgS0VZLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURPakNDQWlLZ0F3SUJBZ0lRVk53RkZtVkZTMGVwYkQxckdleTZVakFOQmdrcWhraUc5dzBCQVFzRkFEQWEKTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3SGhjTk1qQXdOVEExTVRZeU1ESXhXaGNOTWpJdwpOVEExTVRZek1ESXhXakFhTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3Z2dFaU1BMEdDU3FHClNJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURPaDVQS0N2WWtZc1VBNzBCNUZlSGpXcGR0QlpHeWI4OFgKTHIxaDl3L0l5eTFCaTgwQ0pXTndCNXZTWjBvMnE5YWkyTERrRElPYVZrK1B0bE9wTUREZUowNkNtQTJIY1UzZQoxcW9lWG9DUldrVmZGcERpVWkzT2tMV3MzUzRPTjlOemVGRnVtMVlqdXo0S3lxS0FlUmZYY1hCS0pmNndFcENVCnJQR2dJVXdidXhPays4cFlQQlByYi83WE9acHBVTGJpQjlCeTJXemZvRnVnSjNTbFZxNis0RHQxcnBSMHpaZUEKVUptL3dlMGVqZ3AxbzlsZnhuMUZCWm5DNk9FeXJzN2RIR3Ixcm8vVVZTVWhWeUxqd3dsTllObUdmYnBPd0srQgo3RHpkbjlJTzdKV0c3dXR6VjQ4ODdBam1HRjd0N01VRGsrVVYzZFU0NVRvVDg3dU5XbzB2QWdNQkFBR2pmREI2Ck1BNEdBMVVkRHdFQi93UUVBd0lGb0RBSkJnTlZIUk1FQWpBQU1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUIKQmdnckJnRUZCUWNEQWpBZkJnTlZIU01FR0RBV2dCUjNlOUFVa2dHaWNNbm5qTjV1dk9MT2ZqUTNBVEFkQmdOVgpIUTRFRmdRVWQzdlFGSklCb25ESjU0emVicnppem40ME53RXdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBQU9rCkZ2TzJMTWtTWExDUkUwOUMvOTI1S2RHTGwrN1RNdkt1bUR1V0pTVXU0Zk02aVkreitBQnhLbVEzSmErNGFHSUcKbDh6Vk0wYnc2QmVHVmdBOFIrNVBFOVBwa04yOHhRRGRVNmFRMnNnUG1YUmRacEFQT2taYTZBV0RuakNxbmtLRQpsTFFpZlBsVFRxZUFuSGordlZEcDNKSnBFaTlYQnd2cFNlY2Z6amJsSVQrMzdYS25aWVA5dXdpTDZvNTlOVFVtCndReXhsd2d4dVE2VU0vbHZ3OWdlcDNZQ0NmdmU3blZoN1BnbXhaaSt5Qjl4dnZid1FQNG1Tejl5ZjFzUmJnc3oKc2UwZTlKb0R5VzVoNHRHVDNndldJNGQvRUJkb1p3UGNpa0lTUHJONzdJL3BCdFp3N0V2UVE5b0wzUDBUR3pXWApmYis3ODNBalpHMnlDNnZuT3I4PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="}
export CERT3_SECRET_VALUE=${CERT3_SECRET_VALUE:-"LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JR0hBZ0VBTUJNR0J5cUdTTTQ5QWdFR0NDcUdTTTQ5QXdFSEJHMHdhd0lCQVFRZ0toMEZSYjgzb1c1MWtkUTEKY0djOG8wb2QwWjl4Sk9ibGtLZ25kUkUvbFFDaFJBTkNBQVJRQ1pwTVFEbkZMTERRYTdvTmVZcHVwbjMvWWJmOQpaK3Vqczc2ek5GbDNDVnNJSmVzNGFxYWNxUTJpQVJKcm1vMFVnWmdydHREVzEzMjB0Wkg1N3M1UAotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlCSWpDQnlnSUpBTG5WTTBWNitseHlNQW9HQ0NxR1NNNDlCQU1DTUJveEN6QUpCZ05WQkFZVEFsVlRNUXN3CkNRWURWUVFJREFKWFFUQWVGdzB5TURBMU1UTXhOakl6TVRGYUZ3MHlNVEExTVRNeE5qSXpNVEZhTUJveEN6QUoKQmdOVkJBWVRBbFZUTVFzd0NRWURWUVFJREFKWFFUQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQQpCRkFKbWt4QU9jVXNzTkJydWcxNWltNm1mZjlodC8xbjY2T3p2ck0wV1hjSld3Z2w2emhxcHB5cERhSUJFbXVhCmpSU0JtQ3UyME5iWGZiUzFrZm51ems4d0NnWUlLb1pJemowRUF3SURSd0F3UkFJZ0FNd0hUSGVVT1FiNUhEbnUKR1R3NlhEVjhuUWROQlRWMks3NTdRdFVrekpJQ0lETXBZc0MrVG5BNDFpK2p0ZzFpMXV6TEhpUStQWlJGVFE4SQpHcHB4VENtMQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="}
export CERT1_VALUE=${CERT1_VALUE:-"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURPakNDQWlLZ0F3SUJBZ0lRVG9sL3VmZTZTSE92amlkUDIxYlhNVEFOQmdrcWhraUc5dzBCQVFzRkFEQWEKTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3SGhjTk1qQXdOVEExTVRZeU1EQTBXaGNOTWpJdwpOVEExTVRZek1EQTBXakFhTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3Z2dFaU1BMEdDU3FHClNJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUN4RGx5MktHekVENEYydU9BMStGMXdyQ3Bic0hKNmtDdk0KeUpzUnlpOFlwQlBXQVJkejh1SitYbEF5eTExcmo0WVlDdnF6NlFiK3J4UisvSmM2WTM3UHVMVnhqMUZkYkluTgpXTmRHS01HcU1ZMmtTc1dQcUo2T0hnMUpWYm9KRDU3ZGdWbThFTWZYUnNrdnFTWDNhWjN6ZU1MSFR6MUwxcTRhCmcvR2NlcDlhSmdVNTN3dnNNM1lhWk9ZWFM3cWZSNVBlelpqMzRhVUlMS2QyR2NKaWFYWU13ODliK25oVG9jUVgKbmRzWjFFVXJweE9nL1cyV1Bjc1ZPMFZlWFFRVnV4RVFISUJUV0xNREtQYkY4bkpvYitZWTdMY3hodFlpMGU3QQplblJiaEJQWFFFbHNISFFOcnZqTVUvVDd3T0R3MjVqV0ZiUy9GcmMyK3loOE4xR2RjNkdCQWdNQkFBR2pmREI2Ck1BNEdBMVVkRHdFQi93UUVBd0lGb0RBSkJnTlZIUk1FQWpBQU1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUIKQmdnckJnRUZCUWNEQWpBZkJnTlZIU01FR0RBV2dCUzJrRVNzcnkzd3NqSnRiRy9wOTM4RUlVUmExekFkQmdOVgpIUTRFRmdRVXRwQkVySzh0OExJeWJXeHY2ZmQvQkNGRVd0Y3dEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBRzhhCnlJNEwzR1d4OTVVTHB2NE5TdE4zWW8vc3J4WTdVN1RTdlJUMTZOak5QNkM1NGkwVVpEUjJWSUxENEZ5dFp4TEkKZ1M5VTA3Z1pRUXpyamd1M1ZtWERtM2o5ZCtUT1lrbFlLLzIrRW5rR2tZanU5NldxbWhja1V1VDI4VVd1LytqbQozRkh4S3pLQTkwRzhsOXhuM0xyTnA5ekUxS3lESU5RN0xzSDd5OEplNXNIZ0t2aTZ3VkFZK01vSUVJajBjMnEvCi9OcndrSnBTK1RSU2JiRUZ2RGpjdThwaG5GTHA4MldCU0o2ZVFQaWo5cDMwMTlnOExZYWxkMXJ1MXlCVWZoRzQKbTF5aGFuQkd2K1g4aVE1UHNvRDNqeVJscEQ3bnJsWkVrekpJYmtYcXMwUy82Sjg5MWN1bFdVV3hocjV3dm1ubgp0ZmQ4U0JoUktVSE1UcDJTTXR3PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="}
export CERT2_VALUE=${CERT2_VALUE:-"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURPakNDQWlLZ0F3SUJBZ0lRVk53RkZtVkZTMGVwYkQxckdleTZVakFOQmdrcWhraUc5dzBCQVFzRkFEQWEKTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3SGhjTk1qQXdOVEExTVRZeU1ESXhXaGNOTWpJdwpOVEExTVRZek1ESXhXakFhTVJnd0ZnWURWUVFERXc5MFpYTjBMbVJ2YldGcGJpNWpiMjB3Z2dFaU1BMEdDU3FHClNJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUURPaDVQS0N2WWtZc1VBNzBCNUZlSGpXcGR0QlpHeWI4OFgKTHIxaDl3L0l5eTFCaTgwQ0pXTndCNXZTWjBvMnE5YWkyTERrRElPYVZrK1B0bE9wTUREZUowNkNtQTJIY1UzZQoxcW9lWG9DUldrVmZGcERpVWkzT2tMV3MzUzRPTjlOemVGRnVtMVlqdXo0S3lxS0FlUmZYY1hCS0pmNndFcENVCnJQR2dJVXdidXhPays4cFlQQlByYi83WE9acHBVTGJpQjlCeTJXemZvRnVnSjNTbFZxNis0RHQxcnBSMHpaZUEKVUptL3dlMGVqZ3AxbzlsZnhuMUZCWm5DNk9FeXJzN2RIR3Ixcm8vVVZTVWhWeUxqd3dsTllObUdmYnBPd0srQgo3RHpkbjlJTzdKV0c3dXR6VjQ4ODdBam1HRjd0N01VRGsrVVYzZFU0NVRvVDg3dU5XbzB2QWdNQkFBR2pmREI2Ck1BNEdBMVVkRHdFQi93UUVBd0lGb0RBSkJnTlZIUk1FQWpBQU1CMEdBMVVkSlFRV01CUUdDQ3NHQVFVRkJ3TUIKQmdnckJnRUZCUWNEQWpBZkJnTlZIU01FR0RBV2dCUjNlOUFVa2dHaWNNbm5qTjV1dk9MT2ZqUTNBVEFkQmdOVgpIUTRFRmdRVWQzdlFGSklCb25ESjU0emVicnppem40ME53RXdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBQU9rCkZ2TzJMTWtTWExDUkUwOUMvOTI1S2RHTGwrN1RNdkt1bUR1V0pTVXU0Zk02aVkreitBQnhLbVEzSmErNGFHSUcKbDh6Vk0wYnc2QmVHVmdBOFIrNVBFOVBwa04yOHhRRGRVNmFRMnNnUG1YUmRacEFQT2taYTZBV0RuakNxbmtLRQpsTFFpZlBsVFRxZUFuSGordlZEcDNKSnBFaTlYQnd2cFNlY2Z6amJsSVQrMzdYS25aWVA5dXdpTDZvNTlOVFVtCndReXhsd2d4dVE2VU0vbHZ3OWdlcDNZQ0NmdmU3blZoN1BnbXhaaSt5Qjl4dnZid1FQNG1Tejl5ZjFzUmJnc3oKc2UwZTlKb0R5VzVoNHRHVDNndldJNGQvRUJkb1p3UGNpa0lTUHJONzdJL3BCdFp3N0V2UVE5b0wzUDBUR3pXWApmYis3ODNBalpHMnlDNnZuT3I4PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="}
export CERT3_VALUE=${CERT3_VALUE:-"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJJakNCeWdJSkFMblZNMFY2K2x4eU1Bb0dDQ3FHU000OUJBTUNNQm94Q3pBSkJnTlZCQVlUQWxWVE1Rc3cKQ1FZRFZRUUlEQUpYUVRBZUZ3MHlNREExTVRNeE5qSXpNVEZhRncweU1UQTFNVE14TmpJek1URmFNQm94Q3pBSgpCZ05WQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpYUVRCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBCkJGQUpta3hBT2NVc3NOQnJ1ZzE1aW02bWZmOWh0LzFuNjZPenZyTTBXWGNKV3dnbDZ6aHFwcHlwRGFJQkVtdWEKalJTQm1DdTIwTmJYZmJTMWtmbnV6azh3Q2dZSUtvWkl6ajBFQXdJRFJ3QXdSQUlnQU13SFRIZVVPUWI1SERudQpHVHc2WERWOG5RZE5CVFYySzc1N1F0VWt6SklDSURNcFlzQytUbkE0MWkranRnMWkxdXpMSGlRK1BaUkZUUThJCkdwcHhUQ20xCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"}
export CERT1_KEY_VALUE=${CERT1_KEY_VALUE:-"LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFzUTVjdGloc3hBK0JkcmpnTmZoZApjS3dxVzdCeWVwQXJ6TWliRWNvdkdLUVQxZ0VYYy9MaWZsNVFNc3RkYTQrR0dBcjZzK2tHL3E4VWZ2eVhPbU4rCno3aTFjWTlSWFd5SnpWalhSaWpCcWpHTnBFckZqNmllamg0TlNWVzZDUStlM1lGWnZCREgxMGJKTDZrbDkybWQKODNqQ3gwODlTOWF1R29QeG5IcWZXaVlGT2Q4TDdETjJHbVRtRjB1Nm4wZVQzczJZOStHbENDeW5kaG5DWW1sMgpETVBQVy9wNFU2SEVGNTNiR2RSRks2Y1RvUDF0bGozTEZUdEZYbDBFRmJzUkVCeUFVMWl6QXlqMnhmSnlhRy9tCkdPeTNNWWJXSXRIdXdIcDBXNFFUMTBCSmJCeDBEYTc0ekZQMCs4RGc4TnVZMWhXMHZ4YTNOdnNvZkRkUm5YT2gKZ1FJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="}
export CERT2_KEY_VALUE=${CERT2_KEY_VALUE:-"LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF6b2VUeWdyMkpHTEZBTzlBZVJYaAo0MXFYYlFXUnNtL1BGeTY5WWZjUHlNc3RRWXZOQWlWamNBZWIwbWRLTnF2V290aXc1QXlEbWxaUGo3WlRxVEF3CjNpZE9ncGdOaDNGTjN0YXFIbDZBa1ZwRlh4YVE0bEl0enBDMXJOMHVEamZUYzNoUmJwdFdJN3MrQ3NxaWdIa1gKMTNGd1NpWCtzQktRbEt6eG9DRk1HN3NUcFB2S1dEd1Q2Mi8rMXptYWFWQzI0Z2ZRY3RsczM2QmJvQ2QwcFZhdQp2dUE3ZGE2VWRNMlhnRkNadjhIdEhvNEtkYVBaWDhaOVJRV1p3dWpoTXE3TzNSeHE5YTZQMUZVbElWY2k0OE1KClRXRFpobjI2VHNDdmdldzgzWi9TRHV5Vmh1N3JjMWVQUE93STVoaGU3ZXpGQTVQbEZkM1ZPT1U2RS9PN2pWcU4KTHdJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="}
export CERT3_KEY_VALUE=${CERT3_KEY_VALUE:-"LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFVUFtYVRFQTV4U3l3MEd1NkRYbUticVo5LzJHMwovV2ZybzdPK3N6Ulpkd2xiQ0NYck9HcW1uS2tOb2dFU2E1cU5GSUdZSzdiUTF0ZDl0TFdSK2U3T1R3PT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="}

export CONTAINER_IMAGE=$CONTAINER_IMAGE
export NODE_SELECTOR_OS=$NODE_SELECTOR_OS

setup() {
  if [[ -z "${AZURE_CLIENT_ID}" ]] || [[ -z "${AZURE_CLIENT_SECRET}" ]]; then
    echo "Error: Azure service principal is not provided" >&2
    return 1
  fi
}

@test "install driver helm chart" {
  run helm install csi manifest_staging/charts/csi-secrets-store-provider-azure --namespace dev --set windows.enabled=true \
      --set secrets-store-csi-driver.windows.enabled=true \
      --set image.repository=${PROVIDER_TEST_IMAGE} \
      --set image.tag=${IMAGE_TAG} \
      --set image.pullPolicy="IfNotPresent" \
      --set secrets-store-csi-driver.enableSecretRotation=true \
      --set secrets-store-csi-driver.rotationPollInterval=30s \
      --dependency-update

  assert_success
}

@test "create azure k8s secret" {
  run kubectl create secret generic secrets-store-creds --from-literal clientid=${AZURE_CLIENT_ID} --from-literal clientsecret=${AZURE_CLIENT_SECRET}
  assert_success
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
  envsubst < $BATS_TESTS_DIR/nginx-pod-secrets-store-inline-volume-crd.yaml | kubectl apply -f -

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-crd
  assert_success
}

@test "CSI inline volume test with pod portability - read azure kv ${OBJECT1_TYPE} from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd -- $EXEC_COMMAND/$OBJECT1_NAME)
  [[ "${result//$'\r'}" == *"${OBJECT1_VALUE}" ]]
}

@test "CSI inline volume test with pod portability - read azure kv ${OBJECT2_TYPE} from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd -- $EXEC_COMMAND/$OBJECT2_NAME)
  [[ "${result//$'\r'}" == *"${OBJECT2_VALUE}"* ]]
}

@test "CSI inline volume test with pod portability - read azure kv ${OBJECT1_TYPE}, if alias present, from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd -- $EXEC_COMMAND/$OBJECT1_ALIAS)
  [[ "${result//$'\r'}" == *"${OBJECT1_VALUE}" ]]
}

@test "CSI inline volume test with pod portability - read azure kv ${OBJECT2_TYPE}, if alias present, from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd -- $EXEC_COMMAND/$OBJECT2_ALIAS)
  [[ "${result//$'\r'}" == *"${OBJECT2_VALUE}"* ]]
}

@test "CSI inline volume test with certificates" {
  envsubst < $BATS_TESTS_DIR/certificates/azure_v1alpha1_secretproviderclass.yaml | kubectl apply -f -

  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  cmd="kubectl get secretproviderclasses.secrets-store.csi.x-k8s.io/azure-certs -o yaml | grep azure"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  envsubst < $BATS_TESTS_DIR/certificates/nginx-pod-secrets-store-inline-volume-crd.yaml | kubectl apply -f -

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd-certs"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-crd-certs
  assert_success
}

@test "CSI inline volume test with pod portability - read azure pem cert, priv and pub key from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT1_NAME)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == "${CERT1_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT1_NAME-pub-key)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == "${CERT1_KEY_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT1_NAME-secret)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == "${CERT1_SECRET_VALUE}" ]]
}

@test "CSI inline volume test with pod portability - read azure pkcs12 cert, priv and pub key from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT2_NAME)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == "${CERT2_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT2_NAME-pub-key)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == "${CERT2_KEY_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT2_NAME-secret)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == "${CERT2_SECRET_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT2_NAME-secret-pfx)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 -d | openssl pkcs12 -nodes -passin pass:"" | sed -ne '/-BEGIN PRIVATE KEY-/,/-END PRIVATE KEY-/p; /-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' | base64 ${BASE64_FLAGS})
  diff  <(echo "${result_base64_encoded}" ) <(echo "${CERT2_SECRET_VALUE}")
  [[ "${result_base64_encoded}" == "${CERT2_SECRET_VALUE}" ]]

  kubectl cp nginx-secrets-store-inline-crd-certs:/mnt/secrets-store/$CERT2_NAME-secret-pfx-binary $BATS_TMPDIR/$CERT2_NAME-secret-pfx-binary-e2e
  result_base64_encoded=$(openssl pkcs12 -nodes -in $BATS_TMPDIR/$CERT2_NAME-secret-pfx-binary-e2e -passin pass:"" | sed -ne '/-BEGIN PRIVATE KEY-/,/-END PRIVATE KEY-/p; /-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' | base64 ${BASE64_FLAGS})
  diff  <(echo "${result_base64_encoded}" ) <(echo "${CERT2_SECRET_VALUE}")
  [[ "${result_base64_encoded}" == "${CERT2_SECRET_VALUE}" ]]
  rm $BATS_TMPDIR/$CERT2_NAME-secret-pfx-binary-e2e
}

@test "CSI inline volume test with pod portability - read azure ecc cert, priv and pub key from pod" {
  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT3_NAME)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == ${CERT3_VALUE} ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT3_NAME-pub-key)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == ${CERT3_KEY_VALUE} ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT3_NAME-secret)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == ${CERT3_SECRET_VALUE} ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-certs -- $EXEC_COMMAND/$CERT3_NAME-secret-pfx)
  result_base64_encoded=$(echo "${result//$'\r'}" | base64 -d | openssl pkcs12 -nodes -passin pass:"" | sed -ne '/-BEGIN PRIVATE KEY-/,/-END PRIVATE KEY-/p; /-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' | base64 ${BASE64_FLAGS})
  [[ "${result_base64_encoded}" == ${CERT3_SECRET_VALUE} ]]
}

@test "CSI inline volume test with user assigned identity" {
  if [[ "$CI_KIND_CLUSTER" = true ]]; then
    skip "not running in azure cluster"
  fi

  envsubst < $BATS_TESTS_DIR/user-assigned-identity/azure_v1alpha1_userassignedidentityenabled.yaml | kubectl apply -f -

  cmd="kubectl wait --for condition=established --timeout=60s crd/secretproviderclasses.secrets-store.csi.x-k8s.io"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  cmd="kubectl get secretproviderclasses.secrets-store.csi.x-k8s.io/azure-msi -o yaml | grep azure"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  envsubst < $BATS_TESTS_DIR/user-assigned-identity/nginx-pod-user-identity-secrets-store-inline-volume-crd.yaml | kubectl apply -f -

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd-msi"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"
}


@test "CSI inline volume test with user assigned identity - read ${OBJECT1_TYPE}, ${OBJECT2_TYPE} from pod" {
  if [[ "$CI_KIND_CLUSTER" = true ]]; then
    skip "not running in azure cluster"
  fi

  result=$(kubectl exec nginx-secrets-store-inline-crd-msi -- $EXEC_COMMAND/$OBJECT1_NAME)
  [[ "${result//$'\r'}" == *"${OBJECT1_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-msi -- $EXEC_COMMAND/$OBJECT2_NAME)
  [[ "${result//$'\r'}" == *"${OBJECT2_VALUE}"* ]]
}

@test "CSI inline volume test with pod-identity" {
  if [[ "$CI_KIND_CLUSTER" = true ]] || [[ "$TEST_WINDOWS" = true ]]; then
    skip "not running in azure cluster or running on windows cluster"
  fi

  run helm repo add aad-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts
  assert_success
 
  run helm install pi aad-pod-identity/aad-pod-identity --set nmi.probePort=8081
  assert_success

  cmd="kubectl wait pod --for=condition=Ready --timeout=60s -l app.kubernetes.io/component=mic"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  cmd="kubectl wait pod --for=condition=Ready --timeout=60s -l app.kubernetes.io/component=nmi"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  envsubst < $BATS_TESTS_DIR/pod-identity/pi_azure_identity_binding.yaml | kubectl apply -f -
  envsubst < $BATS_TESTS_DIR/pod-identity/azure_v1alpha1_podidentity.yaml | kubectl apply -f -

  cmd="kubectl get secretproviderclasses.secrets-store.csi.x-k8s.io/azure-pod-identity -o yaml | grep azure"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  envsubst < $BATS_TESTS_DIR/pod-identity/nginx-pod-pi-secrets-store-inline-volume-crd.yaml | kubectl apply -f -

  cmd="kubectl wait --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-crd-pi"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"
}


@test "CSI inline volume test with pod-identity - read ${OBJECT1_TYPE}, ${OBJECT2_TYPE} from pod" {
  if [[ "$CI_KIND_CLUSTER" = true ]] || [[ "$TEST_WINDOWS" = true ]]; then
    skip "not running in azure cluster or running on windows cluster"
  fi

  result=$(kubectl exec nginx-secrets-store-inline-crd-pi -- $EXEC_COMMAND/$OBJECT1_NAME)
  [[ "${result//$'\r'}" == *"${OBJECT1_VALUE}" ]]

  result=$(kubectl exec nginx-secrets-store-inline-crd-pi -- $EXEC_COMMAND/$OBJECT2_NAME)
  [[ "${result//$'\r'}" == *"${OBJECT2_VALUE}"* ]]
}

@test "Test auto rotation of mount contents and K8s secrets with Service Principal - Create deployment" {
  if [[ "$CI_KIND_CLUSTER" = true ]]; then
    skip "not running in azure cluster"
  fi

  run kubectl create ns rotation-sp
  assert_success

  run kubectl create secret generic secrets-store-creds --from-literal clientid=${AZURE_CLIENT_ID} --from-literal clientsecret=${AZURE_CLIENT_SECRET} -n rotation-sp
  assert_success

  run az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-sp --value secret
  assert_success

  envsubst < $BATS_TESTS_DIR/rotation/azure_synck8s_v1alpha1_secretproviderclass.yaml | kubectl apply -n rotation-sp -f -
  envsubst < $BATS_TESTS_DIR/rotation/nginx-pod-synck8s-azure-sp.yaml | kubectl apply -n rotation-sp -f -

  cmd="kubectl wait -n rotation-sp --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-rotation"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-rotation -n rotation-sp
  assert_success
}

@test "Test auto rotation of mount contents and K8s secrets with Service Principal" {
  if [[ "$CI_KIND_CLUSTER" = true ]]; then
    skip "not running in azure cluster"
  fi

  run kubectl cp -n rotation-sp nginx-secrets-store-inline-rotation:/mnt/secrets-store/secretalias $BATS_TMPDIR/before_rotation
  assert_success

  result=$(cat $BATS_TMPDIR/before_rotation)
  [[ "${result//$'\r'}" == "secret" ]]
  rm $BATS_TMPDIR/before_rotation

  result=$(kubectl get secret -n rotation-sp rotationsecret -o jsonpath="{.data.username}" | base64 -d)
  [[ "${result//$'\r'}" == "secret" ]]

  run az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-sp --value rotated
  assert_success

  sleep 60

  run kubectl cp -n rotation-sp nginx-secrets-store-inline-rotation:/mnt/secrets-store/secretalias $BATS_TMPDIR/after_rotation
  assert_success

  result=$(cat $BATS_TMPDIR/after_rotation)
  [[ "${result//$'\r'}" == "rotated" ]]
  rm $BATS_TMPDIR/after_rotation

  result=$(kubectl get secret -n rotation-sp rotationsecret -o jsonpath="{.data.username}" | base64 -d)
  [[ "${result//$'\r'}" == "rotated" ]]

  run az keyvault secret delete --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-sp
  assert_success

  run kubectl delete ns rotation-sp
  assert_success
}

@test "Test auto rotation of mount contents and K8s secrets with Managed Identity - Create deployment" {
    if [[ "$CI_KIND_CLUSTER" = true ]]; then
    skip "not running in azure cluster"
  fi

  run kubectl create ns rotation-msi
  assert_success

  run az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-msi --value secret
  assert_success

  envsubst < $BATS_TESTS_DIR/rotation/azure_synck8s_v1alpha1_secretproviderclass_identity.yaml | kubectl apply -n rotation-msi -f -
  envsubst < $BATS_TESTS_DIR/rotation/nginx-pod-synck8s-azure.yaml | kubectl apply -n rotation-msi -f -

  cmd="kubectl wait -n rotation-msi --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-rotation"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-rotation -n rotation-msi
  assert_success
}

@test "Test auto rotation of mount contents and K8s secrets with Managed Identity" {
  if [[ "$CI_KIND_CLUSTER" = true ]]; then
    skip "not running in azure cluster"
  fi

  run kubectl cp -n rotation-msi nginx-secrets-store-inline-rotation:/mnt/secrets-store/secretalias $BATS_TMPDIR/before_rotation
  assert_success

  result=$(cat $BATS_TMPDIR/before_rotation)
  [[ "${result//$'\r'}" == "secret" ]]
  rm $BATS_TMPDIR/before_rotation

  result=$(kubectl get secret -n rotation-msi rotationsecret -o jsonpath="{.data.username}" | base64 -d)
  [[ "${result//$'\r'}" == "secret" ]]

  run az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-msi --value rotated
  assert_success

  sleep 60

  run kubectl cp -n rotation-msi nginx-secrets-store-inline-rotation:/mnt/secrets-store/secretalias $BATS_TMPDIR/after_rotation
  assert_success

  result=$(cat $BATS_TMPDIR/after_rotation)
  [[ "${result//$'\r'}" == "rotated" ]]
  rm $BATS_TMPDIR/after_rotation

  result=$(kubectl get secret -n rotation-msi rotationsecret -o jsonpath="{.data.username}" | base64 -d)
  [[ "${result//$'\r'}" == "rotated" ]]

  run az keyvault secret delete --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-msi
  assert_success

  run kubectl delete ns rotation-msi
  assert_success
}

@test "Test auto rotation of mount contents and K8s secrets with Pod Identity - Create deployment" {
  if [[ "$CI_KIND_CLUSTER" = true ]] || [[ "$TEST_WINDOWS" = true ]]; then
    skip "not running in azure cluster or running on windows cluster"
  fi

  run kubectl create ns rotation-pi
  assert_success

  run az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-pi --value secret
  assert_success

  envsubst < $BATS_TESTS_DIR/pod-identity/pi_azure_identity_binding.yaml | kubectl apply -n rotation-pi -f -
  envsubst < $BATS_TESTS_DIR/rotation/azure_v1alpha1_podidentity.yaml | kubectl apply -n rotation-pi -f -
  envsubst < $BATS_TESTS_DIR/rotation/nginx-pod-synck8s-azure-pi.yaml | kubectl apply -n rotation-pi -f -

  cmd="kubectl wait -n rotation-pi --for=condition=Ready --timeout=60s pod/nginx-secrets-store-inline-rotation"
  wait_for_process $WAIT_TIME $SLEEP_TIME "$cmd"

  run kubectl get pod/nginx-secrets-store-inline-rotation -n rotation-pi
  assert_success
}

@test "Test auto rotation of mount contents and K8s secrets with Pod Identity" {
  if [[ "$CI_KIND_CLUSTER" = true ]] || [[ "$TEST_WINDOWS" = true ]]; then
    skip "not running in azure cluster or running on windows cluster"
  fi

  run kubectl cp -n rotation-pi nginx-secrets-store-inline-rotation:/mnt/secrets-store/secretalias $BATS_TMPDIR/before_rotation
  assert_success

  result=$(cat $BATS_TMPDIR/before_rotation)
  [[ "${result//$'\r'}" == "secret" ]]
  rm $BATS_TMPDIR/before_rotation

  result=$(kubectl get secret -n rotation-pi rotationsecret -o jsonpath="{.data.username}" | base64 -d)
  [[ "${result//$'\r'}" == "secret" ]]

  run az keyvault secret set --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-pi --value rotated
  assert_success

  sleep 60

  run kubectl cp -n rotation-pi nginx-secrets-store-inline-rotation:/mnt/secrets-store/secretalias $BATS_TMPDIR/after_rotation
  assert_success

  result=$(cat $BATS_TMPDIR/after_rotation)
  [[ "${result//$'\r'}" == "rotated" ]]
  rm $BATS_TMPDIR/after_rotation

  result=$(kubectl get secret -n rotation-pi rotationsecret -o jsonpath="{.data.username}" | base64 -d)
  [[ "${result//$'\r'}" == "rotated" ]]

  run az keyvault secret delete --vault-name ${KEYVAULT_NAME} --name ${AUTO_ROTATE_SECRET_NAME}-pi
  assert_success

  run kubectl delete ns rotation-pi
  assert_success
}
