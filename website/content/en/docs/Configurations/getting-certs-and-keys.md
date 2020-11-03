---
title: "Getting Certificates and Keys using Azure Key Vault Provider"
linkTitle: "Getting Certificates and Keys using Azure Key Vault Provider"
weight: 6
description: >
  
---

> Note: This behavior was introduced in 0.0.6 release of Azure Key Vault Provider for Secrets Store CSI Driver. This is backward incompatible with the prior releases. 

The Azure Key Vault Provider for Secrets Store CSI Driver has been designed to closely align with the current behavior of  [az keyvault certificate/secret/key download](https://docs.microsoft.com/en-us/cli/azure/keyvault?view=azure-cli-latest).

[Azure Key Vault](https://docs.microsoft.com/azure/key-vault/) design makes sharp distinctions between Keys, Secrets and Certificates. The KeyVault service's Certificates features were designed making use of it's Keys and Secrets capabilities.

> When a Key Vault certificate is created, an addressable key and secret are also created with the same name. The Key Vault key allows key operations and the Key Vault secret allows retrieval of the certificate value as a secret. A Key Vault certificate also contains public x509 certificate metadata.

The KeyVault service stores both the public and the private parts of your certificate in a KeyVault secret, along with any other secret you might have created in that same KeyVault instance.

## How to obtain the certificate

Knowing that the certificate is stored in a Key Vault certificate, we can retrieve it by using object type `cert`.

> Note: For chain of certificates, using object type `cert` only returns the Server certificate and not the entire chain.

```yaml
        array:
          - |
            objectName: certName
            objectType: cert
            objectVersion: ""
```

The contents of the file will be the certificate in PEM format.

## How to obtain the public key

Knowing that the public key is stored in a Key Vault key, we can retrieve it by using object type `key`

```yaml
        array:
          - |
            objectName: certName
            objectType: key
            objectVersion: ""
```

The contents of the file will be the public key in PEM format.

## How to obtain the private key and certificate

Knowing that the private key is stored in a Key Vault secret with the public certificate included, we can retrieve it by using object type `secret`

```yaml
        array:
          - |
            objectName: certName
            objectType: secret
            objectVersion: ""
```

The contents of the file will be the private key and certificate in PEM format.

> Note: For chain of certificates, using object type `secret` returns entire certificate chain along with the private key.
