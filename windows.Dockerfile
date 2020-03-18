FROM mcr.microsoft.com/windows/servercore:1809 as core

FROM mcr.microsoft.com/windows/servercore:ltsc2019
LABEL description="Secrets Store CSI Driver Provider Azure"
ARG IMAGE_VERSION=0.0.4

COPY ./_output/secrets-store-csi-driver-provider-azure.exe /secrets-store-csi-driver-provider-azure.exe
COPY ./install_windows.ps1 /install_windows.ps1

USER ContainerAdministrator
ENTRYPOINT ["powershell", "C:\\install_windows.ps1"]
