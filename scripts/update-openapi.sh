#!/bin/sh
# Refreshes openapicore/open_api_core.yaml from docs/open_api_core.yaml of a
# tagged epam/ai-dial-core release and records the source in
# openapicore/SOURCE.md. Nothing is changed if the download fails.
#
# Usage: scripts/update-openapi.sh <tag>
#   tag   upstream release tag without a "v" prefix, e.g. 0.49.3
#
# Env:
#   CORE_REPO   default: epam/ai-dial-core
set -eu

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "usage: $0 <tag>   (e.g. 0.49.3)" >&2
  exit 2
fi

tag=$1
repo=${CORE_REPO:-epam/ai-dial-core}
url="https://raw.githubusercontent.com/$repo/$tag/docs/open_api_core.yaml"
dir=$(cd "$(dirname "$0")/.." && pwd)/openapicore

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

if ! curl -fsSL -o "$tmp" "$url"; then
  echo "failed to download $url" >&2
  exit 1
fi
if ! head -n 1 "$tmp" | grep -q '^openapi:'; then
  echo "downloaded file is not an OpenAPI document: $url" >&2
  exit 1
fi

info_version=$(awk '/^info:/{i=1;next} i&&/^[^ ]/{exit} i&&/^  version:/{print $2;exit}' "$tmp")

mv "$tmp" "$dir/open_api_core.yaml"
chmod 644 "$dir/open_api_core.yaml"
cat > "$dir/SOURCE.md" <<EOT
# Source of open_api_core.yaml

\`open_api_core.yaml\` is an unmodified copy of \`docs/open_api_core.yaml\` from a tagged DIAL Core release.

- Repository: <https://github.com/$repo>
- Tag: $tag
- URL: <$url>
- Upstream \`info.version\`: ${info_version:-unknown}

Refresh with \`make openapi TAG=<tag>\` (see \`scripts/update-openapi.sh\`). Do not edit the YAML by hand.
EOT
echo "updated $dir/open_api_core.yaml to $tag"
