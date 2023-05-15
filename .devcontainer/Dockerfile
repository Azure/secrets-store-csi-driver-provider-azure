#-------------------------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See https://go.microsoft.com/fwlink/?linkid=2090316 for license information.
#-------------------------------------------------------------------------------------------------------------
FROM golang:1.20@sha256:685a22e459f9516f27d975c5cc6accc11223ee81fdfbbae60e39cc3b87357306

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive
# JQ Version
ENV JQ_VERSION='1.6'
# Helm Version
ARG HELM_VERSION=3.7.0
#Kind Version
ARG KIND_VERSION=0.12.0
# Kubernetes Version for Kubectl
ARG KUBERNETES_VERSION=1.22.2

# This Dockerfile adds a non-root user with sudo access. Use the "remoteUser"
# property in devcontainer.json to use it. On Linux, the container user's GID/UIDs
# will be updated to match your local UID/GID (when using the dockerFile property).
# See https://aka.ms/vscode-remote/containers/non-root-user for details.
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

# Configure apt, install packages and tools
RUN apt-get update \
  && apt-get install -y apt-transport-https \
  && apt-get -y install --no-install-recommends apt-utils gettext-base dnsutils dialog 2>&1 \
  #
  # Verify git, process tools, lsb-release (common in install instructions for CLIs) installed
  && apt-get -y install git openssh-client vim less iproute2 procps lsb-release \
  #
  # Install Go tools w/module support
  && mkdir -p /tmp/gotools \
  && cd /tmp/gotools \
  && go install -v golang.org/x/tools/gopls@latest 2>&1 \
  && go install -v honnef.co/go/tools/...@latest \
  && go install -v golang.org/x/tools/cmd/gorename@latest \
  && go install -v golang.org/x/tools/cmd/goimports@latest \
  && go install -v golang.org/x/tools/cmd/guru@latest \
  && go install -v golang.org/x/lint/golint@latest \
  && go install -v github.com/mdempsky/gocode@latest \
  && go install -v github.com/cweill/gotests/...@latest \
  && go install -v github.com/haya14busa/goplay/cmd/goplay@latest \
  && go install -v github.com/sqs/goreturns@latest \
  && go install -v github.com/josharian/impl@latest \
  && go install -v github.com/davidrjenni/reftools/cmd/fillstruct@latest \
  && go install -v github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest  \
  && go install -v github.com/ramya-rao-a/go-outline@latest  \
  && go install -v github.com/acroca/go-symbols@latest  \
  && go install -v github.com/godoctor/godoctor@latest  \
  && go install -v github.com/rogpeppe/godef@latest  \
  && go install -v github.com/zmb3/gogetdoc@latest \
  && go install -v github.com/fatih/gomodifytags@latest  \
  && go install -v github.com/mgechev/revive@latest  \
  && go install -v github.com/go-delve/delve/cmd/dlv@latest 2>&1 \
  #
  # Install Go tools w/o module support
  && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>&1 \
  #
  # Create a non-root user to use if preferred - see https://aka.ms/vscode-remote/containers/non-root-user.
  && groupadd --gid $USER_GID $USERNAME \
  && useradd -s /bin/bash --uid $USER_UID --gid $USER_GID -m $USERNAME \
  # [Optional] Add sudo support
  && apt-get install -y sudo \
  && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
  && chmod 0440 /etc/sudoers.d/$USERNAME \
  # Add write permission for /go/pkg
  && chmod -R a+w /go/pkg


# Install jq utility
RUN wget --no-check-certificate https://raw.githubusercontent.com/stedolan/jq/master/sig/jq-release.key -O /tmp/jq-release.key && \
  wget --no-check-certificate https://raw.githubusercontent.com/stedolan/jq/master/sig/v${JQ_VERSION}/jq-linux64.asc -O /tmp/jq-linux64.asc && \
  wget --no-check-certificate https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 -O /tmp/jq-linux64 && \
  gpg --import /tmp/jq-release.key && \
  gpg --verify /tmp/jq-linux64.asc /tmp/jq-linux64 && \
  cp /tmp/jq-linux64 /usr/bin/jq && \
  chmod +x /usr/bin/jq && \
  rm -f /tmp/jq-release.key && \
  rm -f /tmp/jq-linux64.asc && \
  rm -f /tmp/jq-linux64

RUN wget https://github.com/mikefarah/yq/releases/download/3.3.0/yq_linux_amd64 -O /tmp/yq_linux_amd64 \
  && cp /tmp/yq_linux_amd64 /usr/bin/yq \
  && chmod +x /usr/bin/yq \
  && rm -rf /tmp/yq_linux_amd64

# Install Docker CE CLI
RUN apt-get install -y apt-transport-https ca-certificates curl gnupg-agent software-properties-common lsb-release \
  && curl -fsSL https://download.docker.com/linux/$(lsb_release -is | tr '[:upper:]' '[:lower:]')/gpg | (OUT=$(apt-key add - 2>&1) || echo $OUT) \
  && add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/$(lsb_release -is | tr '[:upper:]' '[:lower:]') $(lsb_release -cs) stable" \
  && apt-get update \
  && apt-get install -y docker-ce-cli

# Install Azure CLI
RUN apt-get install -y apt-transport-https curl gnupg2 lsb-release \
  && echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" > /etc/apt/sources.list.d/azure-cli.list \
  && curl -sL https://packages.microsoft.com/keys/microsoft.asc | apt-key add - 2>/dev/null \
  && apt-get update \
  && apt-get install -y azure-cli

# Install Helm
RUN mkdir -p /tmp/downloads/helm \
  && curl -sL -o /tmp/downloads/helm.tar.gz https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz \
  && tar -C /tmp/downloads/helm -zxvf /tmp/downloads/helm.tar.gz \
  && mv /tmp/downloads/helm/linux-amd64/helm /usr/local/bin

# Install Kubectl
RUN curl -LO https://dl.k8s.io/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl \
  && chmod +x ./kubectl \
  && mv kubectl /usr/bin/kubectl

# Install Kind
RUN curl -LO https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 \
  && chmod +x ./kind-linux-amd64 \
  && mv ./kind-linux-amd64 /usr/bin/kind

# clean up
RUN apt-get autoremove -y \
  && apt-get clean -y \
  && rm -rf /var/lib/apt/lists/* /go/src /tmp/gotools\
  && rm -rf /tmp/downloads

# Update this to "on" or "off" as appropriate
ENV GO111MODULE=on

# Switch back to dialog for any ad-hoc use of apt-get
ENV DEBIAN_FRONTEND=dialog
