FROM golang:1.13.4-alpine as builder
RUN apk add --update make
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go
COPY . /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
WORKDIR /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
RUN make build

FROM alpine:3.10.3
RUN apk add --no-cache bash
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/secrets-store-csi-driver-provider-azure /bin/
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/install.sh /bin/install_azure_provider.sh
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure

ENTRYPOINT ["/bin/install_azure_provider.sh"]
