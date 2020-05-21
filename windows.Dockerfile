FROM --platform=$BUILDPLATFORM golang:1.13.10-alpine3.10 as builder
WORKDIR /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
ADD . .
ARG TARGETARCH
ARG TARGETOS
ARG LDFLAGS
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "${LDFLAGS}" -o _output/secrets-store-csi-driver-provider-azure.exe ./cmd/

FROM mcr.microsoft.com/powershell:lts-nanoserver-1809
LABEL description="Secrets Store CSI Driver Provider Azure"

COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/_output/secrets-store-csi-driver-provider-azure.exe /secrets-store-csi-driver-provider-azure.exe

USER ContainerAdministrator
CMD ["\"C:\\Program Files\\PowerShell\\pwsh.exe\"","-Command", "$global:TargetDir = $env:TARGET_DIR; \
    if ([string]::IsNullOrEmpty($TargetDir)) { throw 'target dir is not set. please set TARGET_DIR env var' }; \
    $azureProviderDir = Join-Path -Path $TargetDir -ChildPath \\azure; \
    if (!(Test-Path $azureProviderDir)) { New-Item -path $azureProviderDir -type Directory }; \
    write-host \"Copying file to $azureProviderDir\\provider-azure.exe\"; \
    Copy-Item C:\\secrets-store-csi-driver-provider-azure.exe -Destination $azureProviderDir/provider-azure.exe; \
    while ($true) { write-host 'install done, daemonset sleeping'; Start-Sleep -Seconds 60 }"]
