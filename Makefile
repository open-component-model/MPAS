VERSION?="0.0.0-dev.0"
BOOTSTRAP_RELEASE_VERSION?="v0.0.1"
DEV_VERSION?=0.0.0-$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)-$(shell date +%s)
GO_TEST_ARGS ?= -race

# gitea e2e test
GITEA_TOKEN ?=
MPAS_MANAGEMENT_REPO_OWNER ?= mpas-management
MPAS_MANAGEMENT_REPO_HOSTNAME ?= http://127.0.0.1:3000

# Bootstrap component
FLUX_VERSION ?= "v2.0.0-rc.5"
OCM_CONTROLLER_VERSION ?= "v0.0.1"
MPAS_GITHUB_REPOSITORY ?= ghcr.io/open-component-model/mpas-bootstrap-component

# Github
GITHUB_USERNAME ?= mpas

build:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ./bin/mpas ./cmd/mpas

build-dev:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(DEV_VERSION)" -o ./bin/mpas ./cmd/mpas

build-release-bootstrap-component:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(BOOTSTRAP_RELEASE_VERSION)" -o ./bin/mpas-rel ./cmd/release-bootstrap-component

e2e:
	go test -v ./e2e/...

e2e-cli:
	GITEA_TOKEN=$(GITEA_TOKEN) MPAS_MANAGEMENT_REPO_OWNER=$(MPAS_MANAGEMENT_REPO_OWNER) \
	MPAS_MANAGEMENT_REPO_HOSTNAME=$(MPAS_MANAGEMENT_REPO_HOSTNAME) go test -v ./e2e/... -run TestCli

release-bootstrap-component:
	./bin/mpas-rel --repository-url $(MPAS_GITHUB_REPOSITORY) --username $(GITHUB_USERNAME)

test:
	go test -v ./pkg/... $(GO_TEST_ARGS) -coverprofile cover.out
