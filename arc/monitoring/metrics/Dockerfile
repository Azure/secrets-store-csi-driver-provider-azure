FROM gcr.io/distroless/static
ARG TARGETARCH
COPY arc/monitoring/metrics/_output/${TARGETARCH}/provider-azure-arc-metrics-agent /bin/

LABEL maintainers="nilekh"
LABEL description="Geneva metrics integration for AKVSecretsProvider Arc extension"

ENTRYPOINT ["provider-azure-arc-metrics-agent"]
