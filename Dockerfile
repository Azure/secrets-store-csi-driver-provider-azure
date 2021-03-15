FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.5.0
COPY ./_output/secrets-store-csi-driver-provider-azure /bin/
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
RUN clean-install ca-certificates cifs-utils mount

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-azure"]
