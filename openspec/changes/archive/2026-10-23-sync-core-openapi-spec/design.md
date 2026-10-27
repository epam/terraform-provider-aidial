# Design

## Context

`openapicore/open_api_core.yaml` is a hand copy (`info.version: 0.0.0`). Upstream publishes the same document at `docs/open_api_core.yaml` in `epam/ai-dial-core`; at tag `0.49.3` it reads `info.version: 0.49.2`, so that field is not a reliable release marker. Tags are bare versions (`0.49.3`, no `v`). Scripts live in `scripts/` as POSIX `sh` with `set -eu` and a usage header (see `scripts/install-provider.sh`). `openspec/{model,role,route}-{schema,client}` cite the file for known-gap notes. See proposal.md for motivation, specs for requirements.

## Goals / Non-Goals

**Goals:**
- One command refreshes the contract from a release tag and records that tag.
- The refresh fails safely and leaves existing files untouched.

**Non-Goals:**
- Pinning the CI Core image (`epam/ai-dial-core:latest` stays).
- Changing provider code, schemas or Terraform behavior.
- Automatic or scheduled refresh, or CI fetching from the network.

## Decisions

- **Fetch from `raw.githubusercontent.com/epam/ai-dial-core/<tag>/docs/open_api_core.yaml`** with `curl -fsSL`. Alternative: `git clone`/sparse checkout (heavier) or the GitHub API (rate limits, needs JSON decoding). Raw URLs need no auth for a public repo.
- **Tags, not branches.** Branch heads can contain unreleased endpoints the tested Core image lacks.
- **Download to a temp file, then move into place.** `curl -f` plus a sanity check (first line `openapi:`) avoids committing an HTML error page; nothing is overwritten on failure.
- **Provenance in `openapicore/SOURCE.md`** (tag, URL, and the upstream `info.version` for reference). A sidecar file keeps the YAML byte-identical to upstream so a diff against upstream stays trivial. Alternative: edit `info.version` in the YAML (loses byte equality).
- **`scripts/update-openapi.sh <tag>` plus `make openapi TAG=<tag>`**, matching the existing script and Makefile style. The Makefile target fails with usage when `TAG` is empty.
- **Reconcile existing spec notes manually** after the refresh by diffing old vs new (`Projects` tag, `/openai/responses`, and anything touching `Model`, `Role`, `Route`, `CostLimit`, ETag headers). Automating this is not worth it for a handful of notes.

## Risks / Trade-offs

- [Upstream moves or renames `docs/open_api_core.yaml`] → script fails loudly on 404; update the path then.
- [Newer contract exposes fields the provider does not model] → record as known gaps in the affected specs; do not expand scope here.
- [CI image `latest` may not match the pinned spec tag] → accepted; note in `SOURCE.md`, revisit in a separate change.

## Migration Plan

Run the script once with `0.49.3`, review the diff, update affected spec notes, commit. Rollback is `git revert`.
