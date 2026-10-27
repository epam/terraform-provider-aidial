# Design

## Context

The provider is a Plugin Framework provider (`terraform-plugin-framework` v1.9.0, so protocol 6) in `internal/provider` and `internal/client`. The type name is set once in `provider.go` (`resp.TypeName = "dial"`); resource names derive from it. `main.go` hard-codes `registry.terraform.io/dial/dial`. CI exists only in `.gitlab-ci.yml` (lint, test, build, darwin validate, stub-core apply, ephemeral-Core integration, schema export, generic-package release). `docs/` is generated but not committed, there is no LICENSE, and `scripts/rename-provider-name.sh` is a one-off `dialx=dial` rename. See proposal.md for motivation.

The repository root has two spec trees: `openspec/specs/` (CLI-recognized, 3 capabilities) and per-resource specs directly under `openspec/<capability>/` (not recognized by `openspec list --specs`). Deltas can only target the former.

## Goals / Non-Goals

**Goals:**
- A repository that can be pushed to `github.com/alexey-ban/terraform-provider-aidial` and ingested by the Registry with only manual account steps remaining.
- One mechanical, reviewable rename with no behavior change.

**Non-Goals:**
- Wiping git history and changing `origin` (done manually afterwards).
- Provider schema or API behavior changes, deprecated `dial_*` aliases, acceptance-test framework adoption.
- Publishing under the `epam` namespace now.

## Decisions

1. **Provider type `aidial`, address `alexey-ban/aidial`.** The Registry derives the type from the repository name (`terraform-provider-aidial`), so type and repo must match. Alternative: keep `dial` type with an `aidial` repo; rejected, the Registry would mismatch names and docs.
2. **Clean rename, no aliases.** The provider was never published, so there is no state to migrate. Aliases would double the schema surface and docs.
3. **Rename the Go type `DialProvider` to `AidialProvider`.** Not required, but leaving it would leave `dial` as a false legacy name for the grep-based "no legacy names" check. Cheap and mechanical with gopls/sed.
4. **Rename the Terraform test variables to `aidial_*`** (`aidial_base_url`, `aidial_api_key`, ...) in `tests/integration`, `stub/models` and examples, to keep the new naming consistent and the legacy-name search meaningful. CI-injected secrets named `DIALX_*` no longer exist after the GitLab removal.
5. **Rename implemented as a reviewed script**, replacing the obsolete `scripts/rename-provider-name.sh` replacement list, applied over `git ls-files` excluding `openapicore/` and `openspec/changes/archive/`. Ordered replacements: module path and old address first, then `DialProvider`, then `dial_` type prefixes, then `tf_dial_` file names (via `git mv`). Alternative: manual editing, too error-prone over ~100 files. Words such as `epam/ai-dial-core`, `DIAL Core` and `dial_` inside API field names (if any) must be preserved, so the script is run and then diffed by a human; the legacy-name check uses an explicit pattern list instead of bare `dial`.
6. **GoReleaser with GitHub Actions.** The Registry mandates zip naming, `SHA256SUMS`, a detached GPG signature and a manifest, which is exactly the HashiCorp scaffolding layout. Alternative: reuse the GitLab generic-package approach; rejected, the Registry reads GitHub releases only. Signing key via secrets `GPG_PRIVATE_KEY` and `PASSPHRASE` (imported with `crazy-max/ghaction-import-gpg`); the workflow fails without them.
7. **Platform matrix**: linux, darwin, windows, freebsd on amd64, arm64 plus linux/windows/freebsd 386 and arm, ignoring darwin 386/arm and windows/arm, matching HashiCorp's scaffolding.
8. **CI port**: one `ci.yml` (lint, test, build, validate on ubuntu and macOS, docs drift, integration with service containers for `epam/ai-dial-core` and Valkey). The macOS `validate` job now runs real since public repos get hosted macOS runners, so the `DARWIN_RUNNER_AVAILABLE` toggle is dropped. The stub-core apply job moves behind `workflow_dispatch` (it is opt-in today via `STUB_CORE_ENABLED`). The schema-export build job is not ported; the schema is reproducible via `terraform providers schema -json` and is not a Registry requirement. Pin actions to commit SHAs where practical; Dependabot (gomod, github-actions) keeps them current.
9. **Docs**: adopt the tfplugindocs layout (`examples/provider/provider.tf`, `examples/resources/aidial_*/{resource.tf,import.sh}`, `examples/data-sources/aidial_*/data-source.tf`), commit generated `docs/`, and enforce no drift in CI. The existing flat `examples/*.tf` content is split into those files, not copied wholesale.
10. **Namespace is parameterized by convention.** The namespace `alexey-ban` appears only in `go.mod`, imports, `main.go`, README, docs and examples, `.devcontainer`, scripts and the Makefile. A later move to `epam` is a repeat of the same script with a new replacement list.
11. **OpenSpec**: deltas cover the CLI-recognized capabilities. The per-resource specs outside `openspec/specs/` get their `dial_*` names renamed by the same script as a task. Archived changes stay untouched as history.

## Risks / Trade-offs

- [Rename script corrupts unrelated text such as `ai-dial-core` or API field names] → ordered, anchored replacements; human review of the full diff; `go build`, tests and the integration run catch breakage.
- [Registry ingestion fails on first tag (key not uploaded, wrong zip names)] → `goreleaser release --snapshot` dry run, documented manual checklist, first tag `v0.1.0` can be deleted and re-cut before anyone depends on it.
- [Personal namespace becomes permanent] → clean-rename keeps a later `epam` migration to a mechanical change; the README states the namespace. Users of `alexey-ban/aidial` would have to change `source` once.
- [Secrets or internal references leak at publication] → audit task with gitleaks plus pattern search runs before the history wipe; the wipe itself removes old history that may contain the same items.
- [Go version mismatch: `go.mod` says 1.21, CI uses 1.24] → keep the `go` directive aligned with what the code needs and let CI use a pinned newer toolchain via `go-version-file` or an explicit version; decided during implementation, not behavior-affecting.
- [macOS runner minutes or flaky integration in GitHub] → integration stays Linux-only; macOS only builds and validates.

## Migration Plan

1. Land all changes in this repository while it is still the GitLab project (tests green locally).
2. Owner creates the public GitHub repo, generates the GPG key, adds `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets, and adds the public key under the Registry namespace settings.
3. Owner wipes history, sets `origin` to GitHub and pushes.
4. Owner pushes tag `v0.1.0`, confirms the release assets, and publishes the provider on registry.terraform.io through GitHub sign-in.

Rollback: before step 4, nothing is public to users; after, delete the release and tag (Registry versions can be deleted from the provider page) and re-cut.

## Audit findings (task group 1)

- Internal host `gitlab.deltixhub.com`, `glab`, GitLab token wiring and the GitLab VS Code extension in `.devcontainer/`: removed.
- `scripts/install-provider.sh` (GitLab Generic Package installer, internal project path): removed; the Registry replaces it.
- README GitLab package/token/release sections and `.gitlab-ci.yml`: removed in task groups 3 and 5.
- gitleaks hits: `openapicore/` curl examples (upstream doc placeholders) and the local stub Core demo key in `stub/` (`.env.example`, `docker-compose.yml`, `core/config.json`; only valid for the disposable local Core). Allow-listed in `.gitleaks.toml`; no real secrets found.
- `example.com` in README GPG examples and `tests/integration/variables.tf` defaults: placeholders, kept.
- `stub/models/**` (34 files) contained real-looking organization identifiers: AWS account IDs and IAM role names in Bedrock ARNs, Azure OpenAI resource hostnames and GCP project IDs. Replaced with placeholders (`arn:aws:iam::10000000000N:role/example-bedrock-adapter-N`, `example-{a,b,c,d}-<region>-N.openai.azure.com`, `example-project-{1,2}`) keeping distinct values distinct. A rescan with the default gitleaks rules (no allowlist) finds only the demo Core key.
