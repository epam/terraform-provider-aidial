#!/usr/bin/env bash
# Run the tests/integration flow against a throwaway DIAL Core: build the provider,
# start Core (scripts/core.sh), then validate, plan, apply, null-vs-empty drift check
# and destroy. Core is stopped on exit. Used locally (make test-integration) and by CI.
#
# Env: see scripts/core.sh (CORE_VERSION, CORE_HOST, CORE_PORT, CORE_BIND, DOCKER_HOST).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
api_key="ci-check-key" # must match tests/core/docker-compose.yml
run_id="${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-$$}"

work="$(mktemp -d)"
cleanup() {
  rm -f "$root/tests/integration/terraform.tfvars.json"
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

cd "$root/tests/integration"
# -var-file outranks TF_VAR_*, so the environment cannot redirect the run to another Core.
cat > terraform.tfvars.json <<J
{
  "aidial_base_url": "${base_url}",
  "aidial_api_key": "${api_key}",
  "aidial_role_bucket": "platform",
  "aidial_role_path": "tf-aidial-ci-${run_id}",
  "aidial_service_key": "ci-dummy"
}
J
terraform validate
terraform plan -var-file=terraform.tfvars.json
terraform apply -auto-approve -var-file=terraform.tfvars.json
# Change the model outside Terraform and show that null vs "" / [] drift is detected and reverted.
DIAL_BASE_URL="$base_url" DIAL_API_KEY="$api_key" sh ./drift_check_null_vs_empty.sh
terraform destroy -auto-approve -var-file=terraform.tfvars.json
