BINARY  := terraform-provider-aidial
DEV_DIR := dist/linux_amd64

.PHONY: build install-dev test vet fmt lint lint-md docs openapi

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
