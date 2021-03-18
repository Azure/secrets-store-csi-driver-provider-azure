FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.4.0
COPY ./_output/secrets-store-csi-driver-provider-azure /bin/
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
# upgrading libzstd1 due to CVE-2021-24032
RUN clean-install ca-certificates cifs-utils mount wget libzstd1

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-azure"]
