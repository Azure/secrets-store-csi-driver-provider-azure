IMAGE_NAME=secrets-store-csi-driver-provider-azure
REGISTRY_NAME=aramase
IMAGE_VERSION=v0.0.5
IMAGE_TAG=$(REGISTRY_NAME)/$(IMAGE_NAME):$(IMAGE_VERSION)
IMAGE_TAG_LATEST=$(REGISTRY_NAME)/$(IMAGE_NAME):latest

GO111MODULE ?= on
export GO111MODULE

.PHONY: build
build: setup
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=0 go build . 

image: build
	@echo "Building docker image..."
	$Q docker build --no-cache -t $(IMAGE_TAG) .

push: image
	docker push $(IMAGE_TAG)

setup:
	@echo "Setup..."
	$Q go env

.PHONY: mod
mod:
	@go mod tidy
