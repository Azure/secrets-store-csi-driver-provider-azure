FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.2.0
COPY ./_output/secrets-store-csi-driver-provider-azure /bin/
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
# upgrading apt &libapt-pkg5.0 due to CVE-2020-27350
# upgrading libp11-kit0 due to CVE-2020-29362, CVE-2020-29363 and CVE-2020-29361
RUN apt-mark unhold apt && \
    clean-install ca-certificates cifs-utils mount apt libapt-pkg5.0 libp11-kit0 wget
RUN GRPC_HEALTH_PROBE_VERSION=v0.3.1 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-azure"]
