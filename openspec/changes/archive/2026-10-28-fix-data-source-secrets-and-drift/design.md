# Design

## Context

See proposal.md, "Why". Current state relevant to the approach:

- **URL builders.** In `internal/client/client.go`, the five `bucket`/`path` builders (`roleURL`, `keyURL`, `modelURL`, `routeURL`, `catalogSchemaURL`) each call `strings.ReplaceAll(x, "/", "%2F")`, and `fileModelURL` concatenates `name` raw. Requests are built from a URL string by `http.NewRequestWithContext`, whose parser keeps an already-escaped path in `RawPath`, so `%2F` reaches Core intact.
- **API contract.** `openapicore/open_api_core.yaml` (0.49.2) declares `{bucket}`, `{path}` and `{name}` as plain `string` path parameters, so each is one segment.
- **Data source upstreams.** `model_data_source.go` and `file_model_data_source.go` declare their own `upstreams` nested schema, separate from the resource's schema, which already marks `key` and `secret_extra_data` `Sensitive`.
- **Key state.** `setKeyState` assigns `types.StringValue(key.Project)` and `types.StringValue(key.Role)`, so an omitted field becomes `""`. The client struct uses `omitempty`, so `""` is never sent to Core either.
- **Release pipeline.** GoReleaser runs as `version: "~> v2"` with the changelog disabled, and there is no `--release-notes`. The devcontainer pins GoReleaser 2.18.3 and syft 1.54.1, with checksums.
- **Docs generation.** CI runs tfplugindocs v0.22.0, and the devcontainer has 0.25.0. v0.22.0 already renders `examples/resources/<type>/import-by-string-id.tf` as an import-block example.

## Goals / Non-Goals

**Goals:**

- Each fix is the smallest one that satisfies its spec, so v0.2.1 is a safe patch.
- The release pipeline fails before publishing when it can, and loudly afterwards when it cannot.

**Non-Goals:**

- Sharing one upstream schema between resources and data sources. The umbrella change's "data sources mirror resources" rule covers that.
- Any retry, User-Agent or logging work in the client (umbrella change).
- Upgrading tfplugindocs or other CI tool versions (umbrella change).

## Decisions

### Escape every path parameter with `url.PathEscape`

One unexported helper, used by all six URL builders, applies `url.PathEscape`, which encodes the value as one path segment. For inputs that contain no reserved characters this gives the same bytes as today, including `/` → `%2F`. It also escapes `?`, `#`, `%`, space and `;`, each of which currently corrupts the request.

- **Alternative: keep `ReplaceAll` and add more replacements.** Rejected: that re-implements RFC 3986 by hand.
- **Alternative: leave `fileModelURL` unescaped, as the current test asserts.** Rejected: the contract declares `{name}` a single segment, and the test's own comment already describes the escaped behavior.

### Mark the two data source attributes sensitive in place

Add `Sensitive: true` to `key` and `secret_extra_data` in both data source schemas. Nothing else changes in those schemas.

### Key `project`/`role`: null unless the plan or prior state is `""`

A small helper maps the API value: a non-empty value is stored as is. An empty value keeps the planned value (Create/Update) or prior state value (Read) only if that value is exactly `""`; otherwise it stores null.

Keeping only `""` matters. Keeping any prior value would hide out-of-band removals on refresh. Never keeping `""` would make `project = ""` in config produce a "Provider produced inconsistent result" error, and a diff on every plan.

Existing state that already holds `""` therefore stays `""`, with no diff.

### Release notes extracted from `CHANGELOG.md` in the workflow

A shell step extracts the section from the `## <version>` heading (the tag without its `v`) up to the next `## ` heading, and writes it to a file under `$RUNNER_TEMP`. It fails if the section is missing or empty. GoReleaser then gets `--release-notes <file>`.

- **Alternative: GoReleaser's git-commit changelog.** Rejected: the curated CHANGELOG is where `BREAKING CHANGES` migration notes live.

### Pin the release tools to the devcontainer's versions

- The GoReleaser action uses `version: v2.18.3`, and `download-syft` uses `syft-version: v1.54.1`. These match the checksummed versions in `.devcontainer/Dockerfile`, so a local snapshot and CI build use the same tools.
- CI's `lint` job runs the same SHA-pinned `goreleaser-action` with the same version and `args: check`.
- The version string is repeated in `ci.yml` and `release.yml`, and a comment in each points to the other.

### Verify the published release, not `dist/`

After the attestation step, the job:

1. downloads the release's assets with `gh release download "$TAG"` into a fresh directory
2. runs `gpg --verify` on `*_SHA256SUMS.sig` against `*_SHA256SUMS`, using the key already imported for signing
3. runs `sha256sum -c` over every listed file, with no `--ignore-missing`
4. runs `gh attestation verify` on the `linux_amd64` zip with `--repo "$GITHUB_REPOSITORY"`

This checks exactly what the Registry will ingest, including the renamed `_manifest.json`, which `dist/` may not hold under its published name.

### Bound the release job

- `timeout-minutes: 30` on the job.
- A workflow-level `concurrency: { group: release-${{ github.ref }}, cancel-in-progress: false }`.

### Docs regenerated with CI's tfplugindocs version

Run `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.22.0 generate --provider-name aidial`, not the devcontainer's 0.25.0 binary. That keeps the CI drift check green.

Each resource gets an `import-by-string-id.tf`:

```hcl
import {
  to = aidial_role.example
  id = "platform/my-role"
}
```

The provider example pins `version = "~> 0.2"`, matching the README.

### SECURITY.md branch protection wording

SECURITY.md currently asks for required code owner review. With one maintainer and one code owner, GitHub would block every merge, because authors cannot approve their own pull requests.

- The settings list becomes: pull requests required, and the `CI` and `Security` checks required. Code owner review is noted as something to enable once there is a second maintainer.
- Private vulnerability reporting, the read-only workflow token that cannot approve pull requests, and required SHA pinning are added. This matches the modified "Documented GitHub settings" requirement.

## Risks / Trade-offs

- **[Core may route `%2F` in file model names differently from the old raw `/`]** → Names containing reserved characters were already broken or mis-routed. Plain names produce identical URLs. The integration suite exercises plain names only, and that is unchanged.
- **[Root module outputs that expose data source upstream secrets start failing with "Output refers to sensitive values"]** → Intended. The CHANGELOG entry tells users to add `sensitive = true` to such outputs.
- **[Post-publish verification cannot un-publish a bad release]** → GoReleaser OSS publishes in one step. A failing check turns the run red right away, so the maintainer can delete the GitHub release before the Registry picks it up, or ship a fixed patch.
- **[The GoReleaser version is pinned in two workflows]** → Cross-referencing comments. The umbrella change's single-source-of-truth work can consolidate it.
- **[The extraction step depends on the `## X.Y.Z - date` heading format]** → It fails before publishing, so a wrong format cannot produce an empty release.

## Migration Plan

1. Merge the change, then move the CHANGELOG `Unreleased` entries into `## 0.2.1 - <date>`.
2. The maintainer creates the `release` environment and moves `GPG_PRIVATE_KEY`/`PASSPHRASE` into it, following the settings in SECURITY.md.
3. Create and push the annotated tag `v0.2.1`.
4. Approve the environment deployment.
5. Confirm that the verification steps pass and that the Registry lists 0.2.1. This also closes the "confirm the release path on the next tag" follow-up from the archived #10 change.

Rollback: there are no state or schema migrations. A faulty release is superseded by v0.2.2.
