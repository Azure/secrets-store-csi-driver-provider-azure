-include secrets.env
export $(shell test -f secrets.env && sed 's/=.*//' secrets.env)

ORG_PATH=github.com/Azure
PROJECT_NAME := secrets-store-csi-driver-provider-azure
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"

REGISTRY_NAME ?= upstreamk8sci
REPO_PREFIX ?= k8s/csi/secrets-store
REGISTRY ?= $(REGISTRY_NAME).azurecr.io/$(REPO_PREFIX)
IMAGE_VERSION ?= 0.0.11
IMAGE_NAME ?= provider-azure
IMAGE_TAG ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)

# build variables
BUILD_DATE=$$(date +%Y-%m-%d-%H:%M)
BUILD_COMMIT := $$(git rev-parse --short HEAD)
BUILD_DATE_VAR := $(REPO_PATH)/pkg/version.BuildDate
BUILD_VERSION_VAR := $(REPO_PATH)/pkg/version.BuildVersion
VCS_VAR := $(REPO_PATH)/pkg/version.Vcs
LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(IMAGE_VERSION) -X $(VCS_VAR)=$(BUILD_COMMIT)"

GO_FILES=$(shell go list ./... | grep -v /test/e2e)
ALL_DOCS := $(shell find . -name '*.md' -type f | sort)
TOOLS_MOD_DIR := ./tools
TOOLS_DIR := $(abspath ./.tools)

GO111MODULE = on
DOCKER_CLI_EXPERIMENTAL = enabled
export GOPATH GOBIN GO111MODULE DOCKER_CLI_EXPERIMENTAL

# Generate all combination of all OS, ARCH, and OSVERSIONS for iteration
ALL_OS = linux windows
ALL_ARCH.linux = amd64
ALL_OS_ARCH.linux = $(foreach arch, ${ALL_ARCH.linux}, linux-$(arch))
ALL_ARCH.windows = amd64
ALL_OSVERSIONS.windows := 1809 1903 1909 2004
ALL_OS_ARCH.windows = $(foreach arch, $(ALL_ARCH.windows), $(foreach osversion, ${ALL_OSVERSIONS.windows}, windows-${osversion}-${arch}))
ALL_OS_ARCH = $(foreach os, $(ALL_OS), ${ALL_OS_ARCH.${os}})

# The current context of image building
# The architecture of the image
ARCH ?= amd64
# OS Version for the Windows images: 1809, 1903, 1909, 2004
OSVERSION ?= 1809
# Output type of docker buildx build
OUTPUT_TYPE ?= registry
BUILDKIT_VERSION ?= v0.8.1

# E2E test variables
KIND_VERSION ?= 0.8.1
KIND_K8S_VERSION ?= 1.18.2

$(TOOLS_DIR)/golangci-lint: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(TOOLS_DIR)/misspell: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/misspell github.com/client9/misspell/cmd/misspell

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint $(TOOLS_DIR)/misspell
	$(TOOLS_DIR)/golangci-lint run --timeout=5m -v
	$(TOOLS_DIR)/misspell $(ALL_DOCS)

.PHONY: unit-test
unit-test:
	CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic $(GO_FILES) -v

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/secrets-store-csi-driver-provider-azure ./cmd/

.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/secrets-store-csi-driver-provider-azure.exe ./cmd/

.PHONY: build-darwin
build-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -a -ldflags ${LDFLAGS} -o _output/secrets-store-csi-driver-provider-azure.exe ./cmd/

.PHONY: container
container: build
	docker build --no-cache -t $(IMAGE_TAG) -f Dockerfile .

.PHONY: container-linux
container-linux: docker-buildx-builder
	docker buildx build --no-cache --output=type=$(OUTPUT_TYPE) --platform="linux/$(ARCH)" \
			-t $(IMAGE_TAG)-linux-$(ARCH) -f Dockerfile .

.PHONY: container-windows
container-windows: docker-buildx-builder
	docker buildx build --no-cache --output=type=$(OUTPUT_TYPE) --platform="windows/amd64" \
	 		-t $(IMAGE_TAG)-windows-$(OSVERSION)-$(ARCH) --build-arg OSVERSION=$(OSVERSION) -f windows.Dockerfile .

.PHONY: docker-buildx-builder
docker-buildx-builder:
	@if ! DOCKER_CLI_EXPERIMENTAL=enabled docker buildx ls | grep -q container-builder; then\
		DOCKER_CLI_EXPERIMENTAL=enabled docker buildx create --name container-builder --use --driver-opt image=moby/buildkit:$(BUILDKIT_VERSION);\
	fi

.PHONY: container-all
container-all: build build-windows
	$(MAKE) container-linux
	for osversion in $(ALL_OSVERSIONS.windows); do \
		OSVERSION=$${osversion} $(MAKE) container-windows; \
	done

.PHONY: push-manifest
push-manifest:
	docker manifest create --amend $(IMAGE_TAG) $(foreach osarch, $(ALL_OS_ARCH), $(IMAGE_TAG)-${osarch})
	# add "os.version" field to windows images (based on https://github.com/kubernetes/kubernetes/blob/master/build/pause/Makefile)
	set -x; \
	registry_prefix=$(shell (echo ${REGISTRY} | grep -Eq ".*\/.*") && echo "" || echo "docker.io/"); \
	manifest_image_folder=`echo "$${registry_prefix}${IMAGE_TAG}" | sed "s|/|_|g" | sed "s/:/-/"`; \
	for arch in $(ALL_ARCH.windows); do \
		for osversion in $(ALL_OSVERSIONS.windows); do \
			BASEIMAGE=mcr.microsoft.com/windows/nanoserver:$${osversion}; \
			full_version=`docker manifest inspect $${BASEIMAGE} | jq -r '.manifests[0].platform["os.version"]'`; \
			sed -i -r "s/(\"os\"\:\"windows\")/\0,\"os.version\":\"$${full_version}\"/" "${HOME}/.docker/manifests/$${manifest_image_folder}/$${manifest_image_folder}-windows-$${osversion}-$${arch}"; \
		done; \
	done
	docker manifest push --purge $(IMAGE_TAG)
	docker manifest inspect $(IMAGE_TAG)

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf _output

.PHONY: mod
mod:
	@go mod tidy

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
ifdef CI_KIND_CLUSTER
		curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KIND_K8S_VERSION}/bin/linux/amd64/kubectl && chmod +x ./kubectl && sudo mv kubectl /usr/local/bin/
		make setup-kind
endif
	docker pull $(IMAGE_TAG) || make e2e-container

.PHONY: e2e-container
e2e-container:
ifdef CI_KIND_CLUSTER
	$(MAKE) build container
	kind load docker-image --name kind $(IMAGE_TAG)
else
	$(MAKE) container-all push-manifest
endif

.PHONY: e2e-test
e2e-test:
	bats -t test/bats/azure.bats

.PHONY: setup-kind
setup-kind:
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	# Check for existing kind cluster
	if [ $$(kind get clusters) ]; then kind delete cluster; fi
	# using kind config to create cluster for testing custom cloud environments
	TERM=dumb kind create cluster --image kindest/node:v${KIND_K8S_VERSION} --config test/kind-config.yaml

.PHONY: install-helm
install-helm:
	helm version --short | grep -q v3 || (curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)

.PHONY: e2e-local-bootstrap
e2e-local-bootstrap: build
	kind create cluster --image kindest/node:v${KIND_K8S_VERSION} --config test/kind-config.yaml
	make image
	kind load docker-image --name kind $(IMAGE_TAG)

.PHONY: e2e-kind-cleanup
e2e-kind-cleanup:
	kind delete cluster --name kind

.PHONY: helm-lint
helm-lint:
	# Download and install Helm
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
	# install driver dep as helm 3.4.0 requires dependencies for helm lint
	helm dep update charts/csi-secrets-store-provider-azure
	helm dep update manifest_staging/charts/csi-secrets-store-provider-azure
	# run lint on helm charts
	helm lint --strict charts/csi-secrets-store-provider-azure
	helm lint --strict manifest_staging/charts/csi-secrets-store-provider-azure
