FROM golang:1.13.10-alpine as builder
WORKDIR /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
ADD . .
ARG TARGETARCH
ARG TARGETOS
ARG LDFLAGS
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "${LDFLAGS}" -o _output/secrets-store-csi-driver-provider-azure ./cmd/

FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:v2.1.0
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/_output/secrets-store-csi-driver-provider-azure /bin/
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/install.sh /bin/install_azure_provider.sh
RUN clean-install bash
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure
RUN chmod +x /bin/install_azure_provider.sh

ENTRYPOINT ["/bin/install_azure_provider.sh"]
