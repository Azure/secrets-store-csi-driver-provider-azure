FROM --platform=$BUILDPLATFORM golang:1.13.10-alpine3.10 as builder
WORKDIR /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
ADD . .
ARG TARGETARCH
ARG TARGETOS
ARG LDFLAGS
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "${LDFLAGS}" -o _output/secrets-store-csi-driver-provider-azure.exe ./cmd/

FROM mcr.microsoft.com/windows/servercore:ltsc2019
LABEL description="Secrets Store CSI Driver Provider Azure"

COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/_output/secrets-store-csi-driver-provider-azure.exe /secrets-store-csi-driver-provider-azure.exe
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/install_windows.ps1 /install_windows.ps1

USER ContainerAdministrator
ENTRYPOINT ["powershell", "C:\\install_windows.ps1"]