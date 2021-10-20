FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base:bullseye-v1.0.0
ARG ARCH
COPY ./_output/${ARCH}/secrets-store-csi-driver-provider-azure /bin/
RUN mkdir -p /scripts
COPY /scripts/entrypoint.sh /scripts/entrypoint.sh
RUN chmod +x /scripts/entrypoint.sh

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["/scripts/entrypoint.sh"]
