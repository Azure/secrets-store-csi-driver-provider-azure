# Commands for creating secrets, keys and certificates for testing

```bash
export KEYVAULT_NAME=<keyvault-name>
export LOCATION=<location>
export RESOURCE_GROUP=<resource-group>
```

## Create Key Vault

```bash
az keyvault create --resource-group "${RESOURCE_GROUP}" \
   --location "${LOCATION}" \
   --name "${KEYVAULT_NAME}" \
   --sku premium
```

## Create the secrets

```bash
az keyvault secret set --vault-name $KEYVAULT_NAME --name secret1 --value test
```

## Create the keys

### RSA Key

```bash
az keyvault key create --vault-name $KEYVAULT_NAME --name key1 --kty RSA --size 2048
```

### RSA-HSM Key

```bash
az keyvault key create --vault-name $KEYVAULT_NAME --name rsahsmkey1 --kty RSA-HSM --size 2048
```

### EC-HSM Key

```bash
az keyvault key create --vault-name $KEYVAULT_NAME --name echsmkey1 --kty EC-HSM --curve P-256
```

## Create the certificates

The certificates are generated using [step-cli](https://smallstep.com/cli/) so the SAN can be specified.

### PEM and PKCS12 certificates

```bash
step certificate create test.domain.com test.crt test.key --profile self-signed --subtle --san test.domain.com --kty RSA --not-after 86400h --no-password --insecure
# export to pfx so we can import it into Azure Key Vault
openssl pkcs12 -export -in test.crt -inkey test.key -out test.pfx -passout pass:
az keyvault certificate import --vault-name $KEYVAULT_NAME --name pemcert1 --file test.pfx
az keyvault certificate import --vault-name $KEYVAULT_NAME --name pkcs12cert1 --file test.pfx
```

### ECC certificates

```bash
step certificate create test.domain.com testec.crt testec.key --profile self-signed --subtle --san test.domain.com --kty EC --not-after 86400h --no-password --insecure
# export to pfx so we can import it into Azure Key Vault
openssl pkcs12 -export -in testec.crt -inkey testec.key -out testec.pfx -passout pass:
az keyvault certificate import --vault-name $KEYVAULT_NAME --name ecccert1 --file testec.pfx
```
