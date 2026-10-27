#!/usr/bin/env bash
#
# Renames the provider's identifier across the whole codebase: the Go
# module path, import paths, provider type name (and therefore every
# derived resource/data-source type, e.g. dial_role -> aidial_role),
# Registry source address, binary/artifact names, test variables, and
# every doc/example reference to them.
#
# Operates over `git ls-files` (tracked files only), so untracked/ignored
# paths (dist/, .terraform/, etc.) are never touched. Always skips
# openapicore/ (the vendored upstream OpenAPI spec), openspec/changes/
# (change artifacts describe old and new names side by side), go.sum and
# this script.
#
# Deliberately case-sensitive: ALL CAPS identifiers (DIAL_BASE_URL, DIAL
# Core) are left alone. "ai-dial-core" is never matched, because every
# pattern below is anchored on a non-letter boundary or a full token.
#
# To reuse this for a future rename (e.g. moving the Registry namespace
# from the current namespace to a new one), edit the REPLACEMENTS array: each entry is
# "perl-regex=>replacement" ($1 allowed in the replacement), applied in order, so
# put more specific forms before the general ones they contain.
#
# Usage: scripts/rename-provider-name.sh

set -euo pipefail

REPLACEMENTS=(
  'registry\.terraform\.io/dial/dial=>registry.terraform.io/epam/aidial'
  'github\.com/dial/terraform-provider-dial=>github.com/epam/terraform-provider-aidial'
  'terraform-provider-dialx=>terraform-provider-aidial'
  'terraform-provider-dial(?![a-z])=>terraform-provider-aidial'
  '(?<![A-Za-z0-9-])dial/dial(?![a-z])=>epam/aidial'
  'DialProvider=>AidialProvider'
  '(?<![A-Za-z])dial_=>aidial_'
  'tf-dial-=>tf-aidial-'
  'provider "dial"=>provider "aidial"'
  '(?m)^(\s*)dial = \{=>$1aidial = {'
  'TypeName = "dial"=>TypeName = "aidial"'
  '(?m)^(\s*)dial "github\.com/=>$1aidial "github.com/'
  'dial\.New=>aidial.New'
  '(?m)--provider-name dial$=>--provider-name aidial'
)

EXCLUDE_RE='^(openapicore/|openspec/changes/|go\.sum$|scripts/rename-provider-name\.sh$)'

cd "$(git rev-parse --show-toplevel)"

files=()
while IFS= read -r -d '' f; do
  files+=("$f")
done < <(git ls-files -z | grep -zvE "$EXCLUDE_RE")

for pair in "${REPLACEMENTS[@]}"; do
  PATTERN="${pair%%=>*}" REPLACEMENT="${pair#*=>}" \
    perl -0777 -i -pe 'BEGIN { $p = qr/$ENV{PATTERN}/; $r = $ENV{REPLACEMENT}; } s/$p/eval "qq\x01$r\x01"/ge' "${files[@]}"
done

# Rename tracked files whose names carry the old identifier.
for f in "${files[@]}"; do
  case "$f" in
  *tf_dial_*)
    newf="${f//tf_dial_/tf_aidial_}"
    git mv "$f" "$newf"
    ;;
  esac
done

echo "Done. Review with: git status && git diff --stat"
