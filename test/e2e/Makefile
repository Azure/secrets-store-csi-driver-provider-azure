REPO_ROOT := $(shell git rev-parse --show-toplevel)
TEST_E2E_DIR := $(REPO_ROOT)/test/e2e
GINKGO_FOCUS ?=
GINKGO_SKIP ?=
GINKGO_FAIL_FAST ?= true
GINKGO_DRY_RUN ?= false

# NOTE: junit report can be simply created by executing your tests with the new --junit-report flags instead.
JUNIT_OUTPUT_FILEPATH ?= $(PWD)/_artifacts
GINKGO_JUNIT_REPORT := $(shell echo $(JUNIT_OUTPUT_FILEPATH) | sed 's/\/$$//')/junit.xml

.PHONY: run
run:
	mkdir -p $(JUNIT_OUTPUT_FILEPATH)
	cd $(TEST_E2E_DIR); go test -tags=e2e -timeout=90m -v -ginkgo.v \
		-ginkgo.focus=$(GINKGO_FOCUS) \
		-ginkgo.skip=$(GINKGO_SKIP) \
		-ginkgo.fail-fast=$(GINKGO_FAIL_FAST) \
		-ginkgo.dry-run=$(GINKGO_DRY_RUN) \
		-ginkgo.junit-report=$(GINKGO_JUNIT_REPORT)

.PHONY: build
build:
	cd $(TEST_E2E_DIR); CGO_ENABLED=0 GOARCH=${ARCH} GOOS=linux go test -c -tags=e2e -timeout=90m -v -o _output/${ARCH}/e2e
