FROM k8s.gcr.io/build-image/debian-base:bullseye-v1.1.0
ARG KUBE_VERSION=v1.21.2

# install dependencies
RUN apt-get update -y && apt-get upgrade -y
RUN DEBIAN_FRONTEND=noninteractive TZ=America/Los_Angeles apt-get install python3-pip bash -y
RUN python3 -m pip install junit_xml

# install azcli
RUN apt-get update && apt-get -y --allow-change-held-packages upgrade && \
    apt-get -f -y --allow-change-held-packages install curl apt-transport-https lsb-release gnupg python3-pip && \
    curl -sL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > /etc/apt/trusted.gpg.d/microsoft.asc.gpg && \
    CLI_REPO=$(lsb_release -cs) && \
    echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ ${CLI_REPO} main" \
    > /etc/apt/sources.list.d/azure-cli.list && \
    apt-get update && \
    apt-get install -y azure-cli && \
    rm -rf /var/lib/apt/lists/*

# install kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/kubectl

# install step cli to create self signed certificates
COPY --from=smallstep/step-cli:0.18.0 /usr/local/bin/step /usr/local/bin/step

COPY arc/conformance/plugin/arc_conformance.sh /arc/arc_conformance.sh
COPY arc/conformance/plugin/setup_failure_handler.py /arc/setup_failure_handler.py
COPY test/e2e/_output/amd64/provider-e2e-test /arc/provider-e2e-test

RUN ["chmod", "+x", "/arc/arc_conformance.sh"]
ENTRYPOINT ["/arc/arc_conformance.sh"]
