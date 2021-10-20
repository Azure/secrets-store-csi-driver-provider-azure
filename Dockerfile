FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base:bullseye-v1.0.0
ARG ARCH
COPY ./_output/${ARCH}/secrets-store-csi-driver-provider-azure /bin/
RUN mkdir -p /scripts
COPY /scripts/entrypoint.sh /scripts/entrypoint.sh
RUN chmod +x /scripts/entrypoint.sh

# upgrading libssl1.1 due to CVE-2021-3711 & CVE-2021-3712
# upgrading libgssapi-krb5-2 and libk5crypto3 due to CVE-2021-37750
RUN clean-install ca-certificates libssl1.1 libgssapi-krb5-2 libk5crypto3

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["/scripts/entrypoint.sh"]
