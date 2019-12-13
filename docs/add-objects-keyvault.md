# Adding Objects to Key Vault

## Adding Secrets

1. Create Secret in Key Vault

```bash
  SECRET_VALUE=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 40 | head -n 1)
  SECRET_NAME=<secretNameHere>
  az keyvault secret set --vault-name $KEYVAULT_NAME --name $SECRET_NAME --value $SECRET_VALUE
```

2. Add Secret to `secrets.env` file.

```bash
# Example
OBJECT[1/2]_NAME=secret1
OBJECT[1/2]_VALUE=a1b2c3
OBJECT[1/2]_TYPE=secret
OBJECT[1/2]_ALIAS=SECRET_1
OBJECT[1/2]_VERSION=""
```

## Adding Keys

1. Import Key to KeyVault

```bash
  KEY_NAME=<keyNameHere>
  az keyvault key create --vault-name $KEYVAULT_NAME --name $KEY_NAME --pem-file my_key_file
  # Let's take the value of the key and add it to secrets.env

  cat my_rsa_key | base64 -b 0
  # => XXX-XXX
```

2. Add Key to `secrets.env` file.

```bash
  OBJECT[1/2]_NAME=key1
  OBJECT[1/2]_VALUE=XXX-XXX
  OBJECT[1/2]_TYPE=key
  OBJECT[1/2]_ALIAS=KEY_1
  OBJECT[1/2]_VERSION=""
```

## Adding Certificates

**PEM Certificate**

1. Create PEM Cert and Upload to Key Vault

```bash
  # Generate RSA based Certificate
  CERT_NAME=pemCert
  openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 -keyout "${CERT_NAME}.pem" -out "${CERT_NAME}.pem"
  # import certificate to Key Vault
  az keyvault certificate import -n $CERT_NAME --vault-name $KEYVAULT_NAME -f "${CERT_NAME}.pem"
```

2. Add PEM Cert to `secrets.env` file.

```bash
  # Abstract Certificate Value from PEM Cert
  awk '{print >out}; /-----END PRIVATE KEY-----/{out="cert1.cert.pem"}' out=/dev/null "${CERT_NAME}.pem"
  # show Certificate Value from Cert  <CERT1_VALUE>
  cat cert1.cert.pem | base64 -w 0 # base64 -b 0  
  # Get Public Key from PEM Cert <CERT1_KEY_VALUE>
  openssl x509 -pubkey -noout -in "${CERT_NAME}.pem"  | base64 -w 0 # base64 -b 0
  # Get Secret Value <CERT1_SECRET_VALUE>
  cat "${CERT_NAME}.pem" | base64 -w 0 # base64 -b 0

  CERT1_NAME=<CERT_NAME>
  CERT1_VERSION=""
  CERT1_KEY_VALUE=XXX
  CERT1_VALUE=XXX-XXX
  CERT1_SECRET_VALUE=XXX-XXX-XXX
```

**PKCS12 Cert**

1. Create PKCS12 Cert  and add to Key Vault.

```bash
  PKCS_NAME=pksCert
  # Convert previously created PEM cert to PKCS12 Format
  openssl pkcs12 -export  -in "${CERT_NAME}.pem" -out "${PKCS_NAME}.pfx" -passout pass:
  # import to keyvault
  az keyvault certificate import --file "${PKCS_NAME}.pfx" --name $PKCS_NAME --vault-name $KEYVAULT_NAME
```

2. Decode PkCS12 Cert to PEM and store properties in `secrets.env`

```bash
  # Get Cert From PKCS12
  openssl pkcs12 -in "${PKCS_NAME}.pfx" -clcerts -nokeys | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > "${PKCS_NAME}_decoded.pem"
  # Get Private Key for new PEM cert (decoded from PKCS12 Cert)
  openssl pkcs12 -in "${PKCS_NAME}.pfx" -nocerts -nodes | sed -ne '/-BEGIN PRIVATE KEY-/,/-END PRIVATE KEY-/p' > "${PKCS_NAME}.key"
  # Combine Private Key and Cert to make  PEM Cert.
  cat "${PKCS_NAME}.key" "${PKCS_NAME}_decoded.pem" > "${PKCS_NAME}.pem"  
  # Read out Cert Value <CERT2_VALUE>
  cat "${PKCS_NAME}_decoded".pem | base64 -w 0
  # Get Secret from Cert(PKCS12 Decoded to PEM) <CERT2_SECRET_VALUE>
  cat "${PKCS_NAME}.pem" | base64 -w 0
  # Get Public Key from Cert <CERT2_KEY_VALUE>
  openssl x509 -pubkey -noout -in "${PKCS_NAME}.pem"  | base64 -w 0 # base64 -b 0

  CERT2_NAME=<PKCS_NAME>
  CERT2_VERSION=""
  CERT2_KEY_VALUE=XXX
  CERT2_VALUE=XXX-XXX
  CERT2_SECRET_VALUE=XXX-XXX-XXX
```

**ECC Cert**

1. Generate ECC Cert and store in Key Vault in pfx format.

```bash
  ECC_NAME=eccCert
  # generate EC param for Cert
  openssl ecparam -name prime256v1 -out "${ECC_NAME}.prime256v1.param.pem"
  # Check ECC with: openssl ecparam -in "${ECC_NAME}.prime256v1.param.pem" -text -noout
  
  # Using EC Create Cert and Private Key
  openssl req -new -x509 -sha256 -newkey ec:"${ECC_NAME}.prime256v1.param.pem" -nodes -keyout "${ECC_NAME}.prime256v1.key.pem" -days 365 -out "${ECC_NAME}.prime256v1.cert.pem"
  # Check EC Cert : openssl x509 -in "${ECC_NAME}.prime256v1.cert.pem" -text -noout
  
  # Convert ECC Cert to PFX format.
  openssl pkcs12 -export -keysig -out "${ECC_NAME}.prime256v1.cert.pfx" -inkey "${ECC_NAME}.prime256v1.key.pem" -in "${ECC_NAME}.prime256v1.cert.pem"
  # store ECC Cert in Key Vault
  az keyvault certificate import --file "${ECC_NAME}.prime256v1.cert.pfx" --name $ECC_NAME --vault-name $KEYVAULT_NAME
```

2. Store ECC Cert Properties in `secrets.env` file.
```bash
  # Combine Private Key and Cert for a complete PEM Cert
  cat "${ECC_NAME}.prime256v1.key.pem" "${ECC_NAME}.prime256v1.cert.pem" > "${ECC_NAME}.pem"
  # Get Cert Value <CERT3_VALUE>
  cat "${ECC_NAME}.prime256v1.cert.pem" | base64 -w 0
  # Get Secret Val from PEM cert. <CERT3_SECRET_VALUE>
  cat "${ECC_NAME}.pem" | base64 -w 0
  # Get Public Key from Cert. <CERT3_KEY_VALUE>
  openssl x509 -pubkey -noout -in "${ECC_NAME}.pem" | base64 -w 0   # base64 -b 0

  CERT3_NAME=<ECC_NAME>
  CERT3_VERSION=""
  CERT3_KEY_VALUE=XXX
  CERT3_VALUE=XXX-XXX
  CERT3_SECRET_VALUE=XXX-XXX-XXX
```
