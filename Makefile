-include secrets.env
export $(shell test -f secrets.env && sed 's/=.*//' secrets.env)

ORG_PATH=github.com/Azure
PROJECT_NAME := secrets-store-csi-driver-provider-azure
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"

REGISTRY_NAME ?= upstreamk8sci
REPO_PREFIX ?= k8s/csi/secrets-store
REGISTRY ?= $(REGISTRY_NAME).azurecr.io/$(REPO_PREFIX)
IMAGE_VERSION ?= v1.0.1
IMAGE_NAME ?= provider-azure
CONFORMANCE_IMAGE_NAME ?= provider-azure-arc-conformance
IMAGE_TAG := $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
CONFORMANCE_IMAGE_TAG := $(REGISTRY)/$(CONFORMANCE_IMAGE_NAME):$(IMAGE_VERSION)

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
TOOLS_BIN_DIR := $(abspath $(TOOLS_DIR)/bin)

GO111MODULE = on
DOCKER_CLI_EXPERIMENTAL = enabled
export GOPATH GOBIN GO111MODULE DOCKER_CLI_EXPERIMENTAL

# Generate all combination of all OS, ARCH, and OSVERSIONS for iteration
ALL_OS = linux windows
ALL_ARCH.linux = amd64 arm64
ALL_OS_ARCH.linux = $(foreach arch, ${ALL_ARCH.linux}, linux-$(arch))
ALL_ARCH.windows = amd64
ALL_OSVERSIONS.windows := 1809 1903 1909 2004 ltsc2022
ALL_OS_ARCH.windows = $(foreach arch, $(ALL_ARCH.windows), $(foreach osversion, ${ALL_OSVERSIONS.windows}, windows-${osversion}-${arch}))
ALL_OS_ARCH = $(foreach os, $(ALL_OS), ${ALL_OS_ARCH.${os}})

# The current context of image building
# The architecture of the image
ARCH ?= amd64
CONFORMANCE_ARCH ?= linux/amd64,linux/arm64
# OS Version for the Windows images: 1809, 1903, 1909, 2004, ltsc2022
OSVERSION ?= 1809
# Output type of docker buildx build
OUTPUT_TYPE ?= registry
BUILDX_BUILDER_NAME ?= img-builder

# step cli version
STEP_CLI_VERSION=0.18.0

# E2E test variables
KIND_VERSION ?= 0.11.0
KIND_K8S_VERSION ?= v1.22.4
SHELLCHECK_VER ?= v0.7.2

$(TOOLS_DIR)/golangci-lint: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

$(TOOLS_DIR)/misspell: $(TOOLS_MOD_DIR)/go.mod $(TOOLS_MOD_DIR)/go.sum $(TOOLS_MOD_DIR)/tools.go
	cd $(TOOLS_MOD_DIR) && \
	go build -o $(TOOLS_DIR)/misspell github.com/client9/misspell/cmd/misspell

SHELLCHECK := $(TOOLS_BIN_DIR)/shellcheck-$(SHELLCHECK_VER)
$(SHELLCHECK): OS := $(shell uname | tr '[:upper:]' '[:lower:]')
$(SHELLCHECK): ARCH := $(shell uname -m)
$(SHELLCHECK):
	mkdir -p $(TOOLS_BIN_DIR)
	rm -rf "$(SHELLCHECK)*"
	curl -sfOL "https://github.com/koalaman/shellcheck/releases/download/$(SHELLCHECK_VER)/shellcheck-$(SHELLCHECK_VER).$(OS).$(ARCH).tar.xz"
	tar xf shellcheck-$(SHELLCHECK_VER).$(OS).$(ARCH).tar.xz
	cp "shellcheck-$(SHELLCHECK_VER)/shellcheck" "$(SHELLCHECK)"
	ln -sf "$(SHELLCHECK)" "$(TOOLS_BIN_DIR)/shellcheck"
	chmod +x "$(TOOLS_BIN_DIR)/shellcheck" "$(SHELLCHECK)"
	rm -rf shellcheck*

.PHONY: lint
lint: $(TOOLS_DIR)/golangci-lint $(TOOLS_DIR)/misspell
	$(TOOLS_DIR)/golangci-lint run --timeout=5m -v
	$(TOOLS_DIR)/misspell $(ALL_DOCS)

.PHONY: shellcheck
shellcheck: $(SHELLCHECK)
	find . -name '*.sh' | xargs $(SHELLCHECK)

.PHONY: unit-test
unit-test:
	CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic $(GO_FILES) -v

.PHONY: build
build:
	CGO_ENABLED=0 GOARCH=${ARCH} GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/${ARCH}/secrets-store-csi-driver-provider-azure ./cmd/

.PHONY: build-e2e-test
build-e2e-test:
	ARCH=${ARCH} make -C test/e2e/ build

.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOARCH=${ARCH} GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/${ARCH}/secrets-store-csi-driver-provider-azure.exe ./cmd/

.PHONY: build-darwin
build-darwin:
	CGO_ENABLED=0 GOARCH=${ARCH} GOOS=darwin go build -a -ldflags ${LDFLAGS} -o _output/${ARCH}/secrets-store-csi-driver-provider-azure ./cmd/

.PHONY: container
container: build
	docker buildx build --platform="linux/$(ARCH)" --no-cache -t $(IMAGE_TAG) -f Dockerfile --progress=plain .

.PHONY: arc-conformance-container
arc-conformance-container: docker-buildx-builder build-e2e-test
	docker buildx build \
	--no-cache \
	--platform="linux/$(ARCH)" \
	--output=type=$(OUTPUT_TYPE) \
	--build-arg STEP_CLI_VERSION=$(STEP_CLI_VERSION) \
	-t $(CONFORMANCE_IMAGE_TAG)-linux-$(ARCH) -f arc/conformance/plugin/Dockerfile .

.PHONY: container-linux
container-linux: docker-buildx-builder
	docker buildx build \
			--no-cache \
			--output=type=$(OUTPUT_TYPE) \
			--platform="linux/$(ARCH)" \
			-t $(IMAGE_TAG)-linux-$(ARCH) -f Dockerfile .

.PHONY: container-windows
container-windows: docker-buildx-builder
	docker buildx build \
			--no-cache \
			--output=type=$(OUTPUT_TYPE) \
			--platform="windows/amd64" \
			--build-arg OSVERSION=$(OSVERSION) \
	 		-t $(IMAGE_TAG)-windows-$(OSVERSION)-$(ARCH) -f windows.Dockerfile .

.PHONY: docker-buildx-builder
docker-buildx-builder:
	@if ! docker buildx ls | grep $(BUILDX_BUILDER_NAME); then \
		docker buildx create --name $(BUILDX_BUILDER_NAME) --use; \
		docker buildx inspect $(BUILDX_BUILDER_NAME) --bootstrap; \
	fi

.PHONY: container-all
container-all: build-windows
	for arch in $(ALL_ARCH.linux); do \
		ARCH=$${arch} $(MAKE) build; \
		ARCH=$${arch} $(MAKE) container-linux; \
	done
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

.PHONY: push-conformance-manifest
push-conformance-manifest: docker-buildx-builder build-e2e-test
	docker buildx build \
	--no-cache \
	--push \
	--platform=${CONFORMANCE_ARCH} \
	--build-arg STEP_CLI_VERSION=$(STEP_CLI_VERSION) \
	-t $(CONFORMANCE_IMAGE_TAG) -f arc/conformance/plugin/Dockerfile .

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf _output

.PHONY: mod
mod:
	@go mod tidy

.PHONY: install-kubectl
install-kubectl:
	curl -LO https://storage.googleapis.com/kubernetes-release/release/${KIND_K8S_VERSION}/bin/linux/amd64/kubectl && chmod +x ./kubectl && sudo mv kubectl /usr/local/bin/

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
ifdef CI_KIND_CLUSTER
		make install-kubectl setup-kind
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
	make -C test/e2e/ run

.PHONY: setup-kind
setup-kind:
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	# Check for existing kind cluster
	if [ $$(kind get clusters) ]; then kind delete cluster; fi
	# using kind config to create cluster for testing custom cloud environments
	TERM=dumb kind create cluster --image kindest/node:${KIND_K8S_VERSION} --config test/kind-config.yaml

.PHONY: install-helm
install-helm:
	helm version --short | grep -q v3 || (curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash)

.PHONY: e2e-local-bootstrap
e2e-local-bootstrap: build
	kind create cluster --image kindest/node:${KIND_K8S_VERSION} --config test/kind-config.yaml
	$(MAKE) container-all push-manifest
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

## --------------------------------------
## Release
## --------------------------------------

.PHONY: promote-staging-manifest
promote-staging-manifest: #promote staging manifests to release dir
	@rm -rf deployment
	@cp -r manifest_staging/deployment .
	@rm -rf charts/csi-secrets-store-provider-azure
	@cp -r manifest_staging/charts/csi-secrets-store-provider-azure ./charts
