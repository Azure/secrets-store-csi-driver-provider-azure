FROM golang:1.13.4-alpine as builder
RUN apk add --update make
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go
COPY . /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
WORKDIR /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure
ARG IMAGE_VERSION=0.0.5
RUN make build

FROM alpine:3.11.5
RUN apk add --no-cache bash
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/_output/secrets-store-csi-driver-provider-azure /bin/
COPY --from=builder /go/src/github.com/Azure/secrets-store-csi-driver-provider-azure/install.sh /bin/install_azure_provider.sh
RUN chmod a+x /bin/secrets-store-csi-driver-provider-azure

ENTRYPOINT ["/bin/install_azure_provider.sh"]
