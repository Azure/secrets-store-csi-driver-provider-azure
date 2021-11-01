ARG OSVERSION
FROM --platform=linux/amd64 gcr.io/k8s-staging-e2e-test-images/windows-servercore-cache:1.0-linux-amd64-${OSVERSION} as core

FROM mcr.microsoft.com/windows/nanoserver:${OSVERSION}
LABEL maintainers="aramase"
LABEL description="Secrets Store CSI Driver Provider Azure"

ARG TARGETARCH

COPY ./_output/${TARGETARCH}/secrets-store-csi-driver-provider-azure.exe /secrets-store-csi-driver-provider-azure.exe
COPY --from=core /Windows/System32/netapi32.dll /Windows/System32/netapi32.dll
USER ContainerAdministrator

ENTRYPOINT ["/secrets-store-csi-driver-provider-azure.exe"]
