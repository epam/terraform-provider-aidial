# Tasks

## 1. Refresh tooling

- [x] 1.1 Add `scripts/update-openapi.sh <tag>` (POSIX sh, `set -eu`, usage header like `scripts/install-provider.sh`): download `docs/open_api_core.yaml` from the tag to a temp file, check it starts with `openapi:`, then move it to `openapicore/open_api_core.yaml` and write `openapicore/SOURCE.md` (tag, URL, upstream `info.version`). Verify: a bogus tag (`scripts/update-openapi.sh 9.9.9`) exits non-zero and `git status` shows no changes; no argument prints usage and exits non-zero.
- [x] 1.2 Add `make openapi TAG=<tag>` to the `Makefile` (add to `.PHONY`), failing with usage when `TAG` is empty. Verify: `make openapi` errors without changes; `make openapi TAG=0.49.3` runs the script.
- [x] 1.3 Document the refresh in `README.md` (the `openapicore/` tree line and a short "Updating the Core OpenAPI contract" note). Verify: `make lint-md` passes.

## 2. Refresh the contract

- [x] 2.1 Run `make openapi TAG=0.49.3`. Verify: `openapicore/open_api_core.yaml` is identical to `curl -fsSL https://raw.githubusercontent.com/epam/ai-dial-core/0.49.3/docs/open_api_core.yaml`, and `openapicore/SOURCE.md` names tag `0.49.3`.
- [x] 2.2 Diff old vs new contract and reconcile notes in `openspec/{model,role,route}-{schema,client}/spec.md` that cite it (`CostLimit` typing, ETag/`If-Match` gap, `Model`/`Route`/`Role` field gaps). Update only notes whose claims changed; add newly found gaps as notes, not requirements. Verify: `grep -rn open_api_core openspec/specs openspec/*/spec.md` shows each cited schema name still exists in the new file (`grep -n "^    <Schema>:" openapicore/open_api_core.yaml`).

## 3. Integration check

- [x] 3.1 Run `go build ./... && go vet ./... && go test ./... -count=1 -race` and `openspec validate sync-core-openapi-spec`. Verify: all pass and no Go files changed (`git diff --stat -- '*.go'` is empty).

## Workflow follow-up

- Archive the change after review so `core-openapi-contract` becomes a main spec.
