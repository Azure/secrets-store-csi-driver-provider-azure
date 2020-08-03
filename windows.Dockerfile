FROM mcr.microsoft.com/windows/servercore:1809 as core

ARG TARGETARCH
ARG TARGETOS

FROM mcr.microsoft.com/powershell:lts-nanoserver-1809
LABEL description="Secrets Store CSI Driver Provider Azure"

COPY ./_output/secrets-store-csi-driver-provider-azure.exe /secrets-store-csi-driver-provider-azure.exe

COPY --from=core /Windows/System32/netapi32.dll /Windows/System32/netapi32.dll
USER ContainerAdministrator

ENTRYPOINT ["/secrets-store-csi-driver-provider-azure.exe"]
