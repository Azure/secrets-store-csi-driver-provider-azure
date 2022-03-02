REPO_ROOT := $(shell git rev-parse --show-toplevel)
METRICS_DIR := $(REPO_ROOT)/arc/monitoring/metrics

.PHONY: build
build:
	cd $(METRICS_DIR); CGO_ENABLED=0 GOARCH=${ARCH} GOOS=linux go build -a -o _output/${ARCH}/provider-azure-arc-metrics-agent .
