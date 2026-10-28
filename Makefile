BINARY  := terraform-provider-aidial
DEV_DIR := dist/linux_amd64

.PHONY: build install-dev test vet fmt lint lint-md docs openapi core-up core-down test-integration testacc

build:
	go build -o $(BINARY)

# Build into the directory referenced by dev_overrides in ~/.terraformrc.
install-dev:
	mkdir -p $(DEV_DIR)
	go build -o $(DEV_DIR)/$(BINARY)

test:
	go test ./... -count=1 -race

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .
	terraform fmt -recursive

lint:
	golangci-lint run
	tfproviderlint ./...

lint-md:
	markdownlint-cli2

docs:
	tfplugindocs generate --provider-name aidial

# Refresh openapicore/open_api_core.yaml from a DIAL Core release tag.
openapi:
	@test -n "$(TAG)" || { echo "usage: make openapi TAG=<tag>   (e.g. 0.49.3)" >&2; exit 2; }
	scripts/update-openapi.sh "$(TAG)"

# Throwaway DIAL Core (tests/core). CORE_VERSION selects the image tag (default latest);
# DOCKER_HOST selects the daemon. See README "Testing against a local Core".
core-up:
	scripts/core.sh up

core-down:
	scripts/core.sh down

# Same flow as the CI integration job: apply, drift check, destroy against a fresh Core.
test-integration:
	scripts/test-integration.sh

# Go acceptance tests against a fresh Core, stopped afterwards.
testacc:
	@scripts/core.sh up && \
	  { DIAL_BASE_URL="$$(scripts/core.sh url)" DIAL_API_KEY=ci-check-key TF_ACC=1 \
	      go test ./internal/provider -run TestAcc -count=1 -v; rc=$$?; \
	    scripts/core.sh down; exit $$rc; }
