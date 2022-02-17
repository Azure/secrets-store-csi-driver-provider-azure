REPO_ROOT := $(shell git rev-parse --show-toplevel)
TEST_E2E_DIR := $(REPO_ROOT)/test/e2e

# Ginkgo configurations
GINKGO_FOCUS ?=
GINKGO_SKIP ?=
GINKGO_NODES ?= 1
GINKGO_FAIL_FAST ?= true
GINKGO_TIMEOUT ?= 15m
GINKGO_DRY_RUN ?= false

.PHONY: run
run:
	cd $(TEST_E2E_DIR); go test -tags=e2e -timeout=90m -v -ginkgo.v \
		-ginkgo.focus=$(GINKGO_FOCUS) \
		-ginkgo.skip=$(GINKGO_SKIP) \
		-ginkgo.nodes=$(GINKGO_NODES) \
		-ginkgo.failFast=$(GINKGO_FAIL_FAST) \
		-ginkgo.timeout=$(GINKGO_TIMEOUT) \
		-ginkgo.dryRun=$(GINKGO_DRY_RUN)

.PHONY: build
build:
	cd $(TEST_E2E_DIR); CGO_ENABLED=0 GOARCH=${ARCH} GOOS=linux go test -c -tags=e2e -timeout=90m -v -o _output/${ARCH}/e2e
