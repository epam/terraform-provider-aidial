#!/usr/bin/env bash
# Run the tests/integration-jwt flow against a throwaway DIAL Core, authenticating only
# with an OIDC ID token: build the provider, start Core (scripts/core.sh), then validate,
# apply and destroy. Core is stopped on exit. Used by the CI integration-jwt job.
#
# Env: AIDIAL_ID_TOKEN (required: a JWT Core trusts, see tests/core/docker-compose.yml),
# plus the scripts/core.sh variables (CORE_VERSION, CORE_HOST, CORE_PORT, CORE_BIND, DOCKER_HOST).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
run_id="${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-$$}"

token="${AIDIAL_ID_TOKEN:-}"
if [ -z "$token" ]; then
  echo "AIDIAL_ID_TOKEN is required" >&2
  exit 1
fi
# The token goes to Terraform only through -var-file; no ambient credential may reach the provider.
unset AIDIAL_ID_TOKEN AIDIAL_API_KEY

work="$(mktemp -d)"
cleanup() {
  rm -f "$root/tests/integration-jwt/terraform.tfvars.json"
  rm -rf "$work"
  "$root/scripts/core.sh" down >/dev/null 2>&1 || true
}
trap cleanup EXIT

"$root/scripts/core.sh" up
base_url="$("$root/scripts/core.sh" url)"

mkdir -p "$work/provider"
(cd "$root" && CGO_ENABLED=0 go build -trimpath -o "$work/provider/terraform-provider-aidial" .)
cat > "$work/terraformrc" <<TFRC
provider_installation {
  dev_overrides {
    "epam/aidial" = "$work/provider"
  }
  direct {}
}
TFRC
export TF_CLI_CONFIG_FILE="$work/terraformrc" TF_IN_AUTOMATION=true TF_INPUT=false

cd "$root/tests/integration-jwt"
# -var-file outranks TF_VAR_*, so the environment cannot redirect the run to another Core.
umask 077
cat > terraform.tfvars.json <<J
{
  "aidial_base_url": "${base_url}",
  "aidial_id_token": "${token}",
  "aidial_role_bucket": "platform",
  "aidial_role_path": "tf-aidial-jwt-${run_id}"
}
J
terraform validate
terraform apply -auto-approve -var-file=terraform.tfvars.json
terraform destroy -auto-approve -var-file=terraform.tfvars.json
