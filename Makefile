VERSION?="0.0.0-dev.0"
BOOTSTRAP_RELEASE_VERSION?="v0.0.1"
DEV_VERSION?=0.0.0-$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)-$(shell date +%s)

# gitea e2e test
GITEA_TOKEN ?=
MPAS_MANAGEMENT_REPO_OWNER ?= mpas-management
MPAS_MANAGEMENT_REPO_HOSTNAME ?= http://127.0.0.1:3000

# Bootstrap component
FLUX_VERSION ?= 1.21.0
OCM_CONTROLLER_VERSION ?= 0.0.1
MPAS_GITHUB_REPOSITORY ?= ghcr.io/souleb/mpas-bootstrap-component

# Github
GITHUB_USERNAME ?=

build:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ./bin/mpas ./cmd/mpas

build-dev:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(DEV_VERSION)" -o ./bin/mpas ./cmd/mpas

build-release-bootstrap-component:
# omit debug info wih -s -w
	go build -ldflags="-s -w -X main.Version=$(BOOTSTRAP_RELEASE_VERSION)" -o ./bin/mpas ./cmd/release-bootstrap-component

e2e:
	go test -v ./e2e/...

e2e-cli:
	GITEA_TOKEN=$(GITEA_TOKEN) MPAS_MANAGEMENT_REPO_OWNER=$(MPAS_MANAGEMENT_REPO_OWNER) \
	MPAS_MANAGEMENT_REPO_HOSTNAME=$(MPAS_MANAGEMENT_REPO_HOSTNAME) go test -v ./e2e/... -run TestCli

release-bootstrap-component:
	go run ./cmd/release-bootstrap-component/main.go --flux-version $(FLUX_VERSION) OCM_CONTROLLER_VERSION=$(OCM_CONTROLLER_VERSION) \
	--repository-url $(MPAS_GITHUB_REPOSITORY) \
	--username $(GITHUB_USERNAME)
