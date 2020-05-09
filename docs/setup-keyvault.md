# Set Up an Azure Key Vault

The [Azure Key Vault](https://docs.microsoft.com/azure/key-vault/) will be used to store your secrets. Run the following to create a Resource Group, and a uniquely named Azure Key Vault:

```bash
  KEYVAULT_RESOURCE_GROUP=<keyvault-resource-group>
  KEYVAULT_LOCATION=<keyvault-location>
  KEYVAULT_NAME=secret-store-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)

  az group create -n $KEYVAULT_RESOURCE_GROUP --location $KEYVAULT_LOCATION
  az keyvault create -n $KEYVAULT_NAME -g $KEYVAULT_RESOURCE_GROUP --location $KEYVAULT_LOCATION
```
 If you were to check in your Azure Subscription, you will have a new Resource Group and Key Vault created. You can learn more [here](https://docs.microsoft.com/azure/key-vault/about-keys-secrets-and-certificates#objects-identifiers-and-versioning) about Azure Key Vault naming, objects, types, and versioning.

Below are examples of how to quickly add a few objects to your Key Vault (make sure to add your own object name):

💡 **Please add 2 Objects (Keys and/or Secrets) and 2 Certificates (1 PEM, and 1 PKCS12) to your Key Vault.**

```bash
az keyvault secret set --vault-name $KEYVAULT_NAME --name <secretNameHere> --value $(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 40 | head -n 1)

az keyvault key create --vault-name $KEYVAULT_NAME --name <keyNameHere>

az keyvault certificate create --vault-name $KEYVAULT_NAME --name <certNameHere> -p "$(az keyvault certificate get-default-policy)"
```

You can retrieve the value of a Key Vault secret with the following script:

```bash
# to view a given secret's value
az keyvault secret show --vault-name $KEYVAULT_NAME --name <secretNameHere> --query value -o tsv
# to view a given key's value
az keyvault key show --vault-name $KEYVAULT_NAME --name <yourKeyNameHere> --query "key.n" -o tsv
# to view a given certificate's value
az keyvault certificate show --vault-name $KEYVAULT_NAME --name <yourCertificateNameHere> --query cer -o tsv
```
