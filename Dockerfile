FROM mcr.microsoft.com/cbl-mariner/distroless/minimal:2.0
ARG TARGETARCH
COPY ./_output/${TARGETARCH}/secrets-store-csi-driver-provider-azure /bin/

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["secrets-store-csi-driver-provider-azure"]
