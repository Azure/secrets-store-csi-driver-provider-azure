FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:buster-v1.3.0
COPY ./_output/secrets-store-csi-driver-provider-azure /bin/
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
RUN clean-install ca-certificates cifs-utils mount wget
RUN GRPC_HEALTH_PROBE_VERSION=v0.3.1 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-azure"]
