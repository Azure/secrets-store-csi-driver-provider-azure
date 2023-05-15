ARG STEP_CLI_VERSION=0.22.0
ARG STEP_CLI_IMAGE=smallstep/step-cli:${STEP_CLI_VERSION}
FROM $STEP_CLI_IMAGE as step-cli

FROM registry.k8s.io/build-image/debian-base:bullseye-v1.4.2@sha256:e6e8c911b1566556ae105d3600df04c600241f43b06d22c73592ebb446d60c49
ARG KUBE_VERSION=v1.25.3
ARG TARGETARCH

# install dependencies
RUN apt-get update -y && apt-get upgrade -y
RUN DEBIAN_FRONTEND=noninteractive TZ=America/Los_Angeles apt-get install python3-pip bash curl apt-transport-https lsb-release jq gnupg -y --allow-change-held-packages && python3 -m pip install junit_xml

# install helm
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

# install azcli
RUN curl -sL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/microsoft.asc.gpg && \
    CLI_REPO=$(lsb_release -cs) && \
    echo "deb [arch=${TARGETARCH}] https://packages.microsoft.com/repos/azure-cli/ ${CLI_REPO} main" \
    > /etc/apt/sources.list.d/azure-cli.list && \
    apt-get update && \
    apt-get install -y azure-cli && \
    rm -rf /var/lib/apt/lists/*

# install kubectl
RUN curl -LO https://dl.k8s.io/release/${KUBE_VERSION}/bin/linux/${TARGETARCH}/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/kubectl

# install step cli to create self signed certificates
COPY --from=step-cli /usr/local/bin/step /usr/local/bin/step

COPY arc/conformance/plugin/arc_conformance.sh /arc/arc_conformance.sh
COPY arc/conformance/plugin/setup_failure_handler.py /arc/setup_failure_handler.py
COPY test/e2e/_output/${TARGETARCH}/e2e /arc/e2e

RUN ["chmod", "+x", "/arc/arc_conformance.sh"]
ENTRYPOINT ["/arc/arc_conformance.sh"]
