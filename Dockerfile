ARG TARGETARCH
ARG TARGETOS

FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.2.0
COPY ./_output/secrets-store-csi-driver-provider-azure /bin/
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
RUN clean-install ca-certificates cifs-utils mount

ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-azure"]
