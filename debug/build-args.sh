#!/bin/bash
#
LAUNCHPATH=/go/src/secrets-store-csi-driver-provider-azure/
export $(cat $LAUNCHPATH/debug/secrets.env | xargs)

SECRETS="{\"clientId\": \"$AZURE_CLIENT_ID\",\"clientSecret\": \"$AZURE_CLIENT_SECRET\"}"

ATTRIBUTES=$(cat $LAUNCHPATH/debug/parameters.yaml \
  | sed -e "s/{{KEYVAULT_NAME}}/$KEYVAULT_NAME/" \
  | sed -e "s/{{TENANT_ID}}/$TENANT_ID/" \
  | yq r - -j \
)


export SECRETS=--secrets="$SECRETS"
export ATTRIBUTES=--attributes="$ATTRIBUTES"

jq -n '{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Application",
      "type": "go",
      "request": "launch",
      "mode": "exec",
      "program": "${workspaceFolder}/_output/secrets-store-csi-driver-provider-azure",
      "env":{},
      "args": [
       env.SECRETS,
       env.ATTRIBUTES,
        "--targetPath=/tmp/secrets",
        "--permission=420",
        "--debug=true"
      ],
      "preLaunchTask": "create-tmp",
    }
  ]
}' > $LAUNCHPATH/.vscode/launch.json

make build
