# SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
#
# SPDX-License-Identifier: Apache-2.0

VERSION?=0.0.0-dev.0
BOOTSTRAP_RELEASE_VERSION?=v0.1.0
DEV_VERSION?=0.0.0-$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)-$(shell date +%s)
GO_TEST_ARGS ?= -race
TAG ?= latest

# gitea e2e test
GITEA_TOKEN?=
MPAS_MANAGEMENT_REPO_OWNER?=mpas-management
MPAS_MANAGEMENT_REPO_HOSTNAME?=http://127.0.0.1:3000

# Bootstrap component
MPAS_GITHUB_REPOSITORY?=ghcr.io/open-component-model/mpas-bootstrap-component
GITHUB_USERNAME?=mpas

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

#Verbose Tests
GOTESTSUM ?= $(LOCALBIN)/gotestsum

build:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ./bin/mpas ./cmd/mpas

build-dev:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(DEV_VERSION)" -o ./bin/mpas ./cmd/mpas

.PHONY: build-release-bootstrap-component
build-release-bootstrap-component:
# omit debug info wih -s -w
	go build -v -ldflags="-s -w -X github.com/open-component-model/mpas/cmd/release-bootstrap-component/version.Tag=$(BOOTSTRAP_RELEASE_VERSION)" \
	-o ./bin/mpas-rel ./cmd/release-bootstrap-component


.PHONY e2e:
e2e: prime-test-cluster test-summary-tool
	$(GOTESTSUM) --format testname -- -count=1 -timeout=30m -tags=e2e ./e2e

.PHONY: test-summary-tool
test-summary-tool: ## Download gotestsum locally if necessary.
	GOBIN=$(LOCALBIN) go install gotest.tools/gotestsum@${TAG}

.PHONY: e2e-verbose
e2e-verbose: prime-test-cluster test-summary-tool ## Runs e2e tests in verbose.

	$(GOTESTSUM) --format standard-verbose -- -count=1 -timeout=30m -tags=e2e ./e2e

e2e-cli:
	GITEA_TOKEN=$(GITEA_TOKEN) MPAS_MANAGEMENT_REPO_OWNER=$(MPAS_MANAGEMENT_REPO_OWNER) \
	MPAS_MANAGEMENT_REPO_HOSTNAME=$(MPAS_MANAGEMENT_REPO_HOSTNAME) go test ./e2e-cli --tags=e2e -v -count=1 -run TestBootstrap_gitea

release-bootstrap-component:
	./bin/mpas-rel --repository-url $(MPAS_GITHUB_REPOSITORY) --username $(GITHUB_USERNAME)

test:
	go test -v ./internal/... $(GO_TEST_ARGS) -coverprofile cover.out

.PHONY: prime-test-cluster
prime-test-cluster: ## Runs prime-test-cluster in the ocm-controller project. It expects the project to exist.
	./hack/prime_test_cluster.sh
