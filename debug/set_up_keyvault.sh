# Check if User is authenticated with azure login

if [ "$#" -eq 0 ]; then
  echo "USAGE: ./set_up_keyvault.sh [\$RG_NAME] [\$AZURE_LOCATION] [\$DOCKER_USERNAME]"
  exit 0
fi

# tr: Illegal byte sequence
export LC_ALL=C
export RG_NAME=$1
export AZURE_LOCATION=$2
export DOCKER_USER=$3
export KEYVAULT_NAME=kv$RG_NAME-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)
export BASE64_FLAGS="-w 0"

echo "\n Creating Resource Group \n"
az group create \
  --name $RG_NAME \
  --location $AZURE_LOCATION

echo "\n Creating Key Vault \n"
export KEYVAULT_ID=$(az keyvault create --name $KEYVAULT_NAME --resource-group $RG_NAME --location $AZURE_LOCATION --query id -o tsv)

echo "\n Setting up Folder for Certs"
export CERT_PATH=debug/certs
mkdir -p $CERT_PATH
echo "\n Setting Up PEM Cert \n"
export CERT1_NAME=pemCert
openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 -keyout "${CERT1_NAME}.pem" -out "${CERT1_NAME}.pem"
az keyvault certificate import -n $CERT1_NAME --vault-name $KEYVAULT_NAME -f "${CERT1_NAME}.pem"
# Add PEM Cert to Secrets.env
awk '{print >out}; /-----END PRIVATE KEY-----/{out="debug/certs/cert1.cert.pem"}' out=/dev/null "${CERT1_NAME}.pem"
export CERT1_VALUE=$(cat debug/certs/cert1.cert.pem | base64 -w 0)
export CERT1_KEY_VALUE=$(openssl x509 -pubkey -noout -in "${CERT1_NAME}.pem"  | base64 -w 0)
export CERT1_SECRET_VALUE=$(cat "${CERT1_NAME}.pem" | base64 -w 0)

echo "\n Setting up PKCS Cert \n"
export CERT2_NAME=pksCert
openssl pkcs12 -export  -in "${CERT1_NAME}.pem" -out "$CERT_PATH/${CERT2_NAME}.pfx" -passout pass:
az keyvault certificate import --file "$CERT_PATH/${CERT2_NAME}.pfx" --name $CERT2_NAME --vault-name $KEYVAULT_NAME
# Add PKCS Cert to secrets.env
openssl pkcs12 -in "$CERT_PATH/${CERT2_NAME}.pfx" -clcerts -nokeys | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > "$CERT_PATH/${CERT2_NAME}_decoded.pem"
openssl pkcs12 -in "$CERT_PATH/${CERT2_NAME}.pfx" -nocerts -nodes | sed -ne '/-BEGIN PRIVATE KEY-/,/-END PRIVATE KEY-/p' > "$CERT_PATH/${CERT2_NAME}.key"
cat "$CERT_PATH/${CERT2_NAME}.key" "$CERT_PATH/${CERT2_NAME}_decoded.pem" > "$CERT_PATH/${CERT2_NAME}.pem"
export CERT2_VALUE=$(cat "$CERT_PATH/${CERT2_NAME}_decoded".pem | base64 -w 0)
export CERT2_SECRET_VALUE=$(cat "$CERT_PATH/${CERT2_NAME}.pem" | base64 -w 0)
export CERT2_KEY_VALUE=$(openssl x509 -pubkey -noout -in "$CERT_PATH/${CERT2_NAME}.pem"  | base64 -w 0)


echo "\n Setting up ECC Cert \n"
export CERT3_NAME=eccCert
openssl ecparam -name prime256v1 -out "$CERT_PATH/${CERT3_NAME}.prime256v1.param.pem"
openssl req -new -x509 -sha256 -newkey ec:"$CERT_PATH/${CERT3_NAME}.prime256v1.param.pem" -nodes -keyout "$CERT_PATH/${CERT3_NAME}.prime256v1.key.pem" -days 365 -out "$CERT_PATH/${CERT3_NAME}.prime256v1.cert.pem"
openssl pkcs12 -export -keysig -out "$CERT_PATH/${CERT3_NAME}.prime256v1.cert.pfx" -inkey "$CERT_PATH/${CERT3_NAME}.prime256v1.key.pem" -in "$CERT_PATH/${CERT3_NAME}.prime256v1.cert.pem"
az keyvault certificate import --file "$CERT_PATH/${CERT3_NAME}.prime256v1.cert.pfx" --name $CERT3_NAME --vault-name $KEYVAULT_NAME
# Add ECCCert to secrets.env
cat "$CERT_PATH/${CERT3_NAME}.prime256v1.key.pem" "$CERT_PATH/${CERT3_NAME}.prime256v1.cert.pem" > "$CERT_PATH/${CERT3_NAME}.pem"
export CERT3_VALUE=$(cat "$CERT_PATH/${CERT3_NAME}.prime256v1.cert.pem" | base64 -w 0)
export CERT3_SECRET_VALUE=$(cat "$CERT_PATH/${CERT3_NAME}.pem" | base64 -w 0)
export CERT3_KEY_VALUE=$(openssl x509 -pubkey -noout -in "$CERT_PATH/${CERT3_NAME}.pem" | base64 -w 0)

echo "\n Setting Object 1 in Key Vault \n"
export OBJECT1_NAME=secret1
export OBJECT1_VALUE=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)
export OBJECT1_TYPE=secret
export OBJECT1_ALIAS=SECRET_1
az keyvault secret set --vault-name $KEYVAULT_NAME --name $OBJECT1_NAME --value $OBJECT1_VALUE

echo "\n Setting Object 2 in Key Vault \n"
export OBJECT2_NAME=secret2
export OBJECT2_VALUE=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)
export OBJECT2_TYPE=secret
export OBJECT2_ALIAS=SECRET_2
az keyvault secret set --vault-name $KEYVAULT_NAME --name $OBJECT2_NAME --value $OBJECT2_VALUE

echo "\n Creating Service Principal and Assigning Roles \n"
# Create Service Principal
export AZURE_CLIENT_ID=$(az ad sp create-for-rbac --name "$RG_NAME-sp" --role="Reader" --scopes $KEYVAULT_ID --query "{appId: appId}" -o tsv)
export AZURE_CLIENT_SECRET=$(az ad sp credential reset --name $RG_NAME-sp --credential-description $RG_NAME --query password -o tsv)

# assign roles and policies
az keyvault set-policy -n $KEYVAULT_NAME --secret-permissions get --spn $AZURE_CLIENT_ID
az keyvault set-policy -n $KEYVAULT_NAME --certificate-permissions get --spn $AZURE_CLIENT_ID
az keyvault set-policy -n $KEYVAULT_NAME --key-permissions get --spn $AZURE_CLIENT_ID

export TENANT_ID=$(az account show --query tenantId -o tsv)

# echo "\n Creating secrets.env file with env vars"
cat secrets.env.sample \
  | sed -e "s/{{OBJECT1_NAME}}/$OBJECT1_NAME/" \
  | sed -e "s/{{OBJECT1_VALUE}}/$OBJECT1_VALUE/" \
  | sed -e "s/{{OBJECT1_TYPE}}/$OBJECT1_TYPE/" \
  | sed -e "s/{{OBJECT1_ALIAS}}/$OBJECT1_ALIAS/" \
  | sed -e "s/{{OBJECT1_VERSION}}/$OBJECT1_VERSION/" \
  | sed -e "s/{{OBJECT2_NAME}}/$OBJECT2_NAME/" \
  | sed -e "s/{{OBJECT2_VALUE}}/$OBJECT2_VALUE/" \
  | sed -e "s/{{OBJECT2_TYPE}}/$OBJECT2_TYPE/" \
  | sed -e "s/{{OBJECT2_ALIAS}}/$OBJECT2_ALIAS/" \
  | sed -e "s/{{OBJECT2_VERSION}}/$OBJECT2_VERSION/" \
  | sed -e "s/{{CERT1_NAME}}/$CERT1_NAME/" \
  | sed -e "s/{{CERT1_VALUE}}/$CERT1_VALUE/" \
  | sed -e "s/{{CERT1_VERSION}}/$CERT1_VERSION/" \
  | sed -e "s/{{CERT1_KEY_VALUE}}/$CERT1_KEY_VALUE/" \
  | sed -e "s/{{CERT1_SECRET_VALUE}}/$CERT1_SECRET_VALUE/" \
  | sed -e "s/{{CERT2_NAME}}/$CERT2_NAME/" \
  | sed -e "s/{{CERT2_VALUE}}/$CERT2_VALUE/" \
  | sed -e "s/{{CERT2_VERSION}}/$CERT2_VERSION/" \
  | sed -e "s/{{CERT2_KEY_VALUE}}/$CERT2_KEY_VALUE/" \
  | sed -e "s/{{CERT2_SECRET_VALUE}}/$CERT2_SECRET_VALUE/" \
  | sed -e "s/{{CERT3_NAME}}/$CERT3_NAME/" \
  | sed -e "s/{{CERT3_VALUE}}/$CERT3_VALUE/" \
  | sed -e "s/{{CERT3_VERSION}}/$CERT3_VERSION/" \
  | sed -e "s/{{CERT3_KEY_VALUE}}/$CERT3_KEY_VALUE/" \
  | sed -e "s/{{CERT3_SECRET_VALUE}}/$CERT3_SECRET_VALUE/" \
  | sed -e "s/{{KEYVAULT_NAME}}/$KEYVAULT_NAME/" \
  | sed -e "s/{{AZURE_CLIENT_ID}}/$AZURE_CLIENT_ID/" \
  | sed -e "s/{{AZURE_CLIENT_SECRET}}/$AZURE_CLIENT_SECRET/" \
  | sed -e "s@{{DOCKER_IMAGE}}@$DOCKER_USER/provider-azure@gi" \
  | sed -e "s/{{AZURE_CLIENT_SECRET}}/$AZURE_CLIENT_SECRET/" \
  | sed -e "s/{{TENANT_ID}}/$TENANT_ID/" \
> secrets.env

echo " --- Good to go! ---"
