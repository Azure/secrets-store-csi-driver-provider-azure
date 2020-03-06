IMAGE_NAME=secrets-store-csi-driver-provider-azure
REGISTRY_NAME ?= upstreamk8sci
REGISTRY ?= $(REGISTRY_NAME).azurecr.io
DOCKER_IMAGE ?= $(REGISTRY)/public/k8s/csi/secrets-store/provider-azure
IMAGE_VERSION ?= 0.0.4
BUILD_DATE=$$(date +%Y-%m-%d-%H:%M)
GO_FILES=$(shell go list ./...)
ORG_PATH=github.com/Azure
PROJECT_NAME := secrets-store-csi-driver-provider-azure
REPO_PATH="$(ORG_PATH)/$(PROJECT_NAME)"

BUILD_DATE_VAR := $(REPO_PATH)/pkg/version.BuildDate
BUILD_VERSION_VAR := $(REPO_PATH)/pkg/version.BuildVersion

GO111MODULE ?= on
export GO111MODULE

.PHONY: build
build: setup
	@echo "Building..."
	$Q GOOS=linux CGO_ENABLED=0 go build -ldflags "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(IMAGE_VERSION)" -o _output/secrets-store-csi-driver-provider-azure ./cmd/

image:
# build inside docker container
	@echo "Building docker image..."
	$Q docker build --no-cache -t $(DOCKER_IMAGE):$(IMAGE_VERSION) --build-arg IMAGE_VERSION="$(IMAGE_VERSION)" .

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


KIND_VERSION ?= 0.5.1
KUBERNETES_VERSION ?= 1.15.3

.PHONY: e2e-bootstrap
e2e-bootstrap:
	# Download and install kubectl
	curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl && chmod +x ./kubectl && sudo mv kubectl /usr/local/bin/
	# Download and install kind
	curl -L https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-linux-amd64 --output kind && chmod +x kind && sudo mv kind /usr/local/bin/
	# Download and install Helm
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get | bash
	# Create kind cluster
	kind create cluster --config kind-config.yaml --image kindest/node:v${KUBERNETES_VERSION}
	# Build image
	DOCKER_IMAGE="e2e/secrets-store-csi-driver-provider-azure" IMAGE_VERSION=e2e-$$(git rev-parse --short HEAD) make image
	# Load image into kind cluster
	kind load docker-image --name kind e2e/secrets-store-csi-driver-provider-azure:e2e-$$(git rev-parse --short HEAD)
	# Set up tiller
	kubectl --namespace kube-system --output yaml create serviceaccount tiller --dry-run | kubectl --kubeconfig $$(kind get kubeconfig-path)  apply -f -
	kubectl create --output yaml clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller --dry-run | kubectl --kubeconfig $$(kind get kubeconfig-path) apply -f -
	helm init --service-account tiller --upgrade --wait --kubeconfig $$(kind get kubeconfig-path)

.PHONY: e2e-azure
e2e-azure:
	bats -t test/bats/azure.bats

.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags "-X main.BuildDate=$(BUILD_DATE) -X main.BuildVersion=$(IMAGE_VERSION)" -o _output/secrets-store-csi-driver-provider-azure.exe .

.PHONY: build-container-windows
build-container-windows: build-windows
	docker build --no-cache -t $(DOCKER_IMAGE):$(IMAGE_VERSION) --build-arg IMAGE_VERSION="$(IMAGE_VERSION)" -f windows.Dockerfile .=======.PHONY: azure-keyvaultcreate-keyvault:	# Create AAD App	$(eval clientId = $(shell bash -c "az ad app create --display-name "${DEMOAPP}ServicePrincipal" --identifier-uris "https://${DEMOAPP}ServicePrincipal" --query appId -o tsv"))	# Update the application group memebership claims	az ad app update --id $(clientId) --set groupMembershipClaims=All	# Create a service principal for the Azure AD application	az ad sp create --id $(clientId)	# Get the service principal secret	$(eval clientSecret = $(shell bash -c "az ad sp credential reset  --name $(clientId) --credential-description "APClientSecret"  --query password -o tsv"))	# Create Resource Group	az group create --name $(RESOURCEGROUP) --location westus	#create the key vault	az keyvault create --name $(KVNAME) --resource-group $(RESOURCEGROUP) --location westus	#	## Store ClientSecret and ClientId in Keyvault	#	# Set Client Secret and Client ID in KeyVault	az keyvault secret set --vault-name $(KVNAME) --name "clientSecret" --value $(clientSecret)	az keyvault secret set --vault-name $(KVNAME) --name "clientId" --value $(clientId)	# Show key in Keyvault	az keyvault secret show --name "clientSecret" --vault-name $(KVNAME)	az keyvault secret show --name "clientId" --vault-name $(KVNAME)delete-keyvault:	az group delete -n $(RESOURCEGROUP)>>>>>>> caedad5... AZ AD Creds stored inside AZ Keyvault in Make task
