#!/bin/sh
# Second-apply check for tf_aidial_integration_model_null_vs_empty.tf.
#
# Edits the model directly through the DIAL API (as the DIAL UI would), then
# verifies that Terraform:
#   1. detects the drift between null and ""/[] in `terraform plan`,
#   2. reverts it with a second `terraform apply`,
#   3. reports no changes afterwards.
#
# Run from tests/integration after `terraform apply`. Needs curl and jq.
# Env: DIAL_BASE_URL, DIAL_API_KEY; TF_VAR_FILE (default terraform.tfvars.json).
set -eu

: "${DIAL_BASE_URL:?DIAL_BASE_URL is not set}"
: "${DIAL_API_KEY:?DIAL_API_KEY is not set}"
VAR_FILE="${TF_VAR_FILE:-terraform.tfvars.json}"
TARGET="aidial_model.tf_aidial_integration_model_null_vs_empty"
URL="${DIAL_BASE_URL%/}/v1/models/platform/tf-aidial-ci-test-model-null-vs-empty"

dial() {
  curl --fail-with-body --silent --show-error -H "Api-Key: ${DIAL_API_KEY}" "$@"
}

plan_exit_code() {
  set +e
  terraform plan -var-file="$VAR_FILE" -target="$TARGET" -detailed-exitcode -no-color > plan.out 2>&1
  code=$?
  set -e
  echo "$code"
}

echo "== 1. Edit the model outside Terraform:"
echo "     description: null -> \"\", user_roles: [] -> null, icon_url: \"\" -> null"
dial "$URL" \
  | jq 'del(._metadata, .status) | .description = "" | .userRoles = null | .iconUrl = null' \
  > drifted-model.json
dial -X PUT -H "Content-Type: application/json" --data @drifted-model.json "$URL" > /dev/null
rm -f drifted-model.json

echo "== 2. terraform plan must detect the drift"
code=$(plan_exit_code)
sed -n '/Terraform will perform/,/^Plan:/p' plan.out
if [ "$code" != "2" ]; then
  cat plan.out
  echo "FAIL: expected plan exit code 2 (changes), got $code"
  exit 1
fi
for attribute in description user_roles icon_url; do
  if ! grep -Eq "^[[:space:]]+[-+~] ${attribute}[[:space:]]" plan.out; then
    echo "FAIL: plan does not show a change for ${attribute}"
    exit 1
  fi
done

echo "== 3. Second apply reverts the drift"
terraform apply -auto-approve -var-file="$VAR_FILE" -target="$TARGET" -no-color

echo "== 4. DIAL Core holds the configured values again"
dial "$URL" | jq -e '.description == null and .userRoles == [] and .iconUrl == ""' > /dev/null \
  || { echo "FAIL: DIAL Core does not hold description=null, userRoles=[], iconUrl=\"\""; dial "$URL"; exit 1; }

echo "== 5. Next plan shows no changes"
code=$(plan_exit_code)
if [ "$code" != "0" ]; then
  cat plan.out
  echo "FAIL: expected no changes after the second apply, got plan exit code $code"
  exit 1
fi
rm -f plan.out
echo "OK: null vs empty drift detected and reverted"
