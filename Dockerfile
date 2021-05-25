FROM gcr.io/distroless/static:nonroot
COPY ./_output/secrets-store-csi-driver-provider-azure /bin/

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["secrets-store-csi-driver-provider-azure"]
