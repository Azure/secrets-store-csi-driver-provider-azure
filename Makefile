-include secrets.env
export $(shell test -f secrets.env && sed 's/=.*//' secrets.env)

REGISTRY_NAME ?= upstreamk8sci
REGISTRY ?= $(REGISTRY_NAME).azurecr.io
DOCKER_IMAGE ?= $(REGISTRY)/public/k8s/csi/secrets-store/provider-azure
IMAGE_VERSION ?= 0.0.9
IMAGE_NAME ?= secrets-store-csi-driver-provider-azure

BUILD_DATE=$$(date +%Y-%m-%d-%H:%M)
GO_FILES=$(shell go list ./...)
ORG_PATH=github.com/Azure
REPO_PATH="$(ORG_PATH)/$(IMAGE_NAME)"
E2E_IMAGE_TAG=$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)

BUILD_DATE_VAR := $(REPO_PATH)/pkg/version.BuildDate
BUILD_VERSION_VAR := $(REPO_PATH)/pkg/version.BuildVersion

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(IMAGE_VERSION)"

GO111MODULE ?= on
export GO111MODULE
DOCKER_CLI_EXPERIMENTAL = enabled
export GOPATH GOBIN GO111MODULE DOCKER_CLI_EXPERIMENTAL

.PHONY: build
build: setup
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=0 go build -ldflags ${LDFLAGS} -o _output/secrets-store-csi-driver-provider-azure ./cmd/

.PHONY: build-windows
build-windows:
	@echo "Building windows binary..."
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/secrets-store-csi-driver-provider-azure.exe ./cmd/

image:
	@echo "Building docker image..."
	docker buildx build --no-cache -t $(DOCKER_IMAGE):$(IMAGE_VERSION) -f Dockerfile --platform="linux/amd64" --output "type=docker,push=false" .

.PHONY: build-container-windows
build-container-windows:
	@echo "Building windows docker image..."
	docker buildx build --no-cache -t $(DOCKER_IMAGE):$(IMAGE_VERSION) -f windows.Dockerfile --platform="windows/amd64" --output "type=docker,push=false" .

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
KIND_K8S_VERSION ?= 1.18.2
KIND_K8S_SHA=@sha256:7b27a6d0f2517ff88ba444025beae41491b016bc6af573ba467b70c5e8e0d85f

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
ifdef CI_KIND_CLUSTER
		curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KIND_K8S_VERSION}/bin/linux/amd64/kubectl && chmod +x ./kubectl && sudo mv kubectl /usr/local/bin/
		make setup-kind
endif
	docker pull $(E2E_IMAGE_TAG) || make e2e-container

.PHONY: e2e-container
e2e-container: build build-windows
	docker buildx rm container-builder || true
	docker buildx create --use --name=container-builder
ifdef CI_KIND_CLUSTER
		DOCKER_IMAGE=$(REGISTRY)/$(IMAGE_NAME) make image
		kind load docker-image --name kind $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
else
		az acr login --name $(REGISTRY_NAME)
		docker buildx build --no-cache -t $(E2E_IMAGE_TAG)-linux-amd64 -f Dockerfile --platform="linux/amd64" --push .
		docker buildx build --no-cache -t $(E2E_IMAGE_TAG)-windows-1809-amd64 -f windows.Dockerfile --platform="windows/amd64" --push .
		docker manifest create $(E2E_IMAGE_TAG) $(E2E_IMAGE_TAG)-linux-amd64 $(E2E_IMAGE_TAG)-windows-1809-amd64
		docker manifest inspect $(E2E_IMAGE_TAG)
		docker manifest push --purge $(E2E_IMAGE_TAG)
endif

.PHONY: e2e-test
e2e-test:
	bats -t test/bats/azure.bats

.PHONY: setup-kind
setup-kind:
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	kind create cluster --image kindest/node:v${KIND_K8S_VERSION} --config test/kind-config.yaml

.PHONY: install-helm
install-helm:
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: e2e-local-bootstrap
e2e-local-bootstrap:
	kind create cluster --image kindest/node:v${KIND_K8S_VERSION}${KIND_K8S_SHA}
	make image
	kind load --name kind docker-image $(DOCKER_IMAGE):$(IMAGE_VERSION)
	# Create Dev namespace for local e2e-testing
	kubectl create ns dev

.PHONY: e2e-kind-cleanup
e2e-kind-cleanup:
	kind delete cluster --name kind

.PHONY: setup-debug-launchjson
setup-debug-launchjson:
	chmod +x debug/build-args.sh && ./debug/build-args.sh

.PHONY: setup-keyvault
setup-keyvault:
	chmod +x debug/set_up_keyvault.sh  && ./debug/set_up_keyvault.sh $(rg_name) $(azure_location) $(docker_user)

.PHONY: e2e-kind-cleanup
e2e-kind-cleanup:
	kind delete cluster --name kind

.PHONY: helm-lint
helm-lint:
	# Download and install Helm
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
	# run lint on helm charts
	helm lint --strict charts/csi-secrets-store-provider-azure
