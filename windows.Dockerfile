ARG TARGETARCH
ARG TARGETOS

FROM mcr.microsoft.com/powershell:lts-nanoserver-1809
LABEL description="Secrets Store CSI Driver Provider Azure"

COPY ./_output/secrets-store-csi-driver-provider-azure.exe /secrets-store-csi-driver-provider-azure.exe

USER ContainerAdministrator
CMD ["\"C:\\Program Files\\PowerShell\\pwsh.exe\"","-Command", "$global:TargetDir = $env:TARGET_DIR; \
    if ([string]::IsNullOrEmpty($TargetDir)) { throw 'target dir is not set. please set TARGET_DIR env var' }; \
    $azureProviderDir = Join-Path -Path $TargetDir -ChildPath \\azure; \
    if (!(Test-Path $azureProviderDir)) { New-Item -path $azureProviderDir -type Directory }; \
    write-host \"Copying file to $azureProviderDir\\provider-azure.exe\"; \
    Copy-Item C:\\secrets-store-csi-driver-provider-azure.exe -Destination $azureProviderDir/provider-azure.exe; \
    while ($true) { Start-Sleep -Seconds 60 }"]
