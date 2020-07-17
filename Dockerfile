FROM golang:1.13.10-alpine as builder
WORKDIR /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
ADD . .
ARG TARGETARCH
ARG TARGETOS
ARG LDFLAGS
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags "${LDFLAGS}" -o _output/secrets-store-csi-driver-provider-azure ./cmd/

FROM us.gcr.io/k8s-artifacts-prod/build-image/debian-base-amd64:v2.1.0
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/_output/secrets-store-csi-driver-provider-azure /bin/
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure

CMD ["sh","-c", "if [ -z \"${TARGET_DIR}\" ]; then echo 'target dir is not set. please set TARGET_DIR env var'; exit 1; fi; \
    mkdir -p ${TARGET_DIR}/azure || exit 1; \
    cp /bin/secrets-store-csi-driver-provider-azure ${TARGET_DIR}/azure/provider-azure || exit 1; \
    echo \"install done at ${TARGET_DIR}/azure, daemonset sleeping\"; \
    while true; do sleep 60; done"]
