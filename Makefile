-include secrets.env
export $(shell test -f secrets.env && sed 's/=.*//' secrets.env)

REGISTRY_NAME ?= upstreamk8sci
REGISTRY ?= $(REGISTRY_NAME).azurecr.io
DOCKER_IMAGE ?= $(REGISTRY)/public/k8s/csi/secrets-store/provider-azure
IMAGE_VERSION ?= 0.0.5
PROJECT_NAME ?= secrets-store-csi-driver-provider-azure

# Use a custom version for E2E tests if we are testing in CI
ifdef CI
override IMAGE_VERSION := e2e-$$(git rev-parse --short HEAD)
endif


BUILD_DATE=$$(date +%Y-%m-%d-%H:%M)
GO_FILES=$(shell go list ./...)
ORG_PATH=github.com/Azure
REPO_PATH="$(ORG_PATH)/$(IMAGE_NAME)"
E2E_IMAGE_TAG=$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)

BUILD_DATE_VAR := $(REPO_PATH)/pkg/version.BuildDate
BUILD_VERSION_VAR := $(REPO_PATH)/pkg/version.BuildVersion

GO111MODULE ?= on
export GO111MODULE
DOCKER_CLI_EXPERIMENTAL = enabled
export GOPATH GOBIN GO111MODULE DOCKER_CLI_EXPERIMENTAL

.PHONY: build
build: setup
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=0 go build -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(IMAGE_VERSION)" -o _output/secrets-store-csi-driver-provider-azure ./cmd/

.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags "-X main.BuildDate=$(BUILD_DATE) -X main.BuildVersion=$(IMAGE_VERSION)" -o _output/secrets-store-csi-driver-provider-azure.exe ./cmd/

image:
# build inside docker container
	@echo "Building docker image..."
	$Q docker build --no-cache -t $(DOCKER_IMAGE):$(IMAGE_VERSION) --build-arg IMAGE_VERSION="$(IMAGE_VERSION)" .

.PHONY: build-container-windows
build-container-windows: build-windows
	docker build --no-cache -t $(DOCKER_IMAGE):$(IMAGE_VERSION) --build-arg IMAGE_VERSION="$(IMAGE_VERSION)" -f windows.Dockerfile .

push: image
	docker push $(DOCKER_IMAGE):$(IMAGE_VERSION)

setup: clean
	@echo "Setup..."
	$Q go env

clean:
	-rm -rf _output

.PHONY: mod
mod:
	@go mod tidy

.PHONY: unit-test
unit-test:
	go test $(GO_FILES) -v


KIND_VERSION ?= 0.6.0
KIND_K8S_VERSION ?= 1.16.3

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
		DOCKER_IMAGE=$(REGISTRY)/$(IMAGE_NAME) make image
		kind load docker-image --name kind $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
else
		az acr login --name $(REGISTRY_NAME)
		make build build-windows
		az acr build --registry $(REGISTRY_NAME) -t $(E2E_IMAGE_TAG)-linux-amd64 -f Dockerfile --platform linux .
		az acr build --registry $(REGISTRY_NAME) -t $(E2E_IMAGE_TAG)-windows-1809-amd64 -f windows.Dockerfile --platform windows .
		docker manifest create $(E2E_IMAGE_TAG) $(E2E_IMAGE_TAG)-linux-amd64 $(E2E_IMAGE_TAG)-windows-1809-amd64
		docker manifest inspect $(E2E_IMAGE_TAG)
		docker manifest push --purge $(E2E_IMAGE_TAG)
endif

.PHONY: e2e-container-cleanup
e2e-container-cleanup:
ifndef CI_KIND_CLUSTER
	az acr login --name $(REGISTRY_NAME)
	az acr repository delete --name $(REGISTRY_NAME) --image $(IMAGE_NAME):$(IMAGE_VERSION)-linux-amd64 -y
	az acr repository delete --name $(REGISTRY_NAME) --image $(IMAGE_NAME):$(IMAGE_VERSION)-windows-1809-amd64 -y
	az acr repository delete --name $(REGISTRY_NAME) --image $(IMAGE_NAME):$(IMAGE_VERSION) -y
endif

.PHONY: e2e-test
e2e-test:
	bats -t test/bats/azure.bats

.PHONY: setup-kind
setup-kind:
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	kind create cluster --image kindest/node:v${KIND_K8S_VERSION}

.PHONY: install-helm
install-helm:
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: e2e-local-bootstrap
e2e-local-bootstrap:
	kind create cluster --image kindest/node:v${KIND_K8S_VERSION}
	make image
	kind load docker-image --name kind $(DOCKER_IMAGE):$(IMAGE_VERSION)
