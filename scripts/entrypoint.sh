#!/bin/sh
# ca certs needs to be updaed in case arc extension is running with proxy configuration.
if [ -f "/usr/local/share/ca-certificates/proxy-cert.crt" ]
then
    echo "Running update-ca-certificates"
    update-ca-certificates
fi
echo "starting secret store csi driver azure provider"
secrets-store-csi-driver-provider-azure
