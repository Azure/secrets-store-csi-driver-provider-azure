FROM alpine:3.10

WORKDIR /bin

RUN apk add --no-cache bash
ADD ./secrets-store-csi-driver-provider-azure /bin/secrets-store-csi-driver-provider-azure
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
ADD ./install.sh /bin/install_azure_provider.sh

ENTRYPOINT ["/bin/install_azure_provider.sh"]
