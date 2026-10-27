ARG GO_VERSION=1.27.3
FROM golang:${GO_VERSION}

ARG TERRAFORM_VERSION=1.17.0
ARG GITLEAKS_VERSION=8.30.1
ARG GH_VERSION=2.103.3
ARG GOPLS_VERSION=0.24.0
ARG DELVE_VERSION=1.27.2
ARG GOTOOLS_VERSION=0.52.0
ARG TFPLUGINDOCS_VERSION=0.25.0
ARG TFPROVIDERLINT_VERSION=0.31.0
ARG GOLANGCI_LINT_VERSION=2.15.0
ARG MARKDOWNLINT_CLI2_VERSION=0.24.0

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    jq \
    gnupg \
    unzip \
    sudo \
    less \
    libatomic1 \
    pre-commit \
    && rm -rf /var/lib/apt/lists/*

# Node (for Claude Code), copied from the official image
COPY --from=node:26-slim /usr/local/bin/node /usr/local/bin/node
COPY --from=node:26-slim /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

RUN npm install -g @anthropic-ai/claude-code @fission-ai/openspec markdownlint-cli2@${MARKDOWNLINT_CLI2_VERSION}

RUN useradd -m -s /bin/bash -u 1000 node \
    && echo "node ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/node

# terraform (pre-commit terraform fmt, tests/integration, examples)
RUN curl -fsSL -o /tmp/terraform.zip "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip" \
    && unzip -q /tmp/terraform.zip -d /usr/local/bin terraform \
    && rm /tmp/terraform.zip

# gitleaks
RUN curl -fsSL "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" | tar -xz -C /usr/local/bin gitleaks

# GitHub CLI (push over HTTPS and CI status via GH_TOKEN)
RUN curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" | tar -xz --strip-components=2 -C /usr/local/bin "gh_${GH_VERSION}_linux_amd64/bin/gh"

# Go tooling, installed outside /home/node so the home volume doesn't shadow it
ENV GOBIN=/usr/local/go-tools
ENV PATH=${GOBIN}:${PATH}
RUN mkdir -p "${GOBIN}" \
    && go install golang.org/x/tools/gopls@v${GOPLS_VERSION} \
    && go install github.com/go-delve/delve/cmd/dlv@v${DELVE_VERSION} \
    && go install golang.org/x/tools/cmd/goimports@v${GOTOOLS_VERSION} \
    && go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v${TFPLUGINDOCS_VERSION} \
    # tfproviderlint's own x/tools is too old to read Go 1.27 export data, so build it against a newer one
    && mkdir /tmp/tfpl && cd /tmp/tfpl && go mod init tfpl \
    && go get golang.org/x/tools@v${GOTOOLS_VERSION} github.com/bflad/tfproviderlint/cmd/tfproviderlint@v${TFPROVIDERLINT_VERSION} \
    && go install github.com/bflad/tfproviderlint/cmd/tfproviderlint && rm -rf /tmp/tfpl \
    && rm -rf /go/pkg /root/.cache/go-build

# golangci-lint
RUN curl -fsSL "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz" | tar -xz -C /usr/local/bin --strip-components=1 "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64/golangci-lint"

USER node
ENV GOPATH=/home/node/go
WORKDIR /workspaces
