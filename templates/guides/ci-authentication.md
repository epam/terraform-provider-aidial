---
page_title: "Authenticating from CI/CD"
subcategory: ""
description: |-
  Use an OIDC ID token from GitLab CI or GitHub Actions instead of a long-lived API key, and configure DIAL Core to trust it.
---

# Authenticating from CI/CD

The provider can authenticate with an OIDC ID token (`id_token` or `AIDIAL_ID_TOKEN`) instead of an API key. The credential precedence rules are in the provider documentation and the project README. This guide covers the CI recipes and the DIAL Core side.

## GitLab CI

Request a token with the audience that DIAL Core expects, and let the provider read it from the environment:

```yaml
terraform-apply:
  id_tokens:
    AIDIAL_ID_TOKEN:
      aud: https://dial-core.example.com
  script:
    - terraform apply -auto-approve
```

## GitHub Actions

Grant the job `id-token: write`, request a token for your audience, mask it and export it to the step that runs Terraform:

```yaml
jobs:
  apply:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v7
      - name: Terraform apply
        env:
          OIDC_AUDIENCE: https://dial-core.example.com
        run: |
          token="$(curl -sSf -H "Authorization: Bearer ${ACTIONS_ID_TOKEN_REQUEST_TOKEN}" \
            "${ACTIONS_ID_TOKEN_REQUEST_URL}&audience=${OIDC_AUDIENCE}" | jq -r .value)"
          echo "::add-mask::${token}"
          AIDIAL_ID_TOKEN="${token}" terraform apply -auto-approve
```

Pull requests from forks and from Dependabot receive no OIDC token, so run the job only for events that do.

## DIAL Core setup

Core must trust the CI system as an identity provider and map the token to a role that may call the admin API. In the Core static settings (`aidial.settings.json`), this example trusts GitHub Actions tokens issued for one repository:

```json
{
  "identityProviders": {
    "github": {
      "jwksUrl": "https://token.actions.githubusercontent.com/.well-known/jwks",
      "issuerPattern": "^https://token\\.actions\\.githubusercontent\\.com$",
      "rolePath": "repository",
      "rolesDelimiter": " ",
      "audience": "https://dial-core.example.com"
    }
  },
  "access": {
    "admin": {
      "rules": [
        { "function": "EQUAL", "source": "roles", "targets": ["my-org/my-infra-repo"] }
      ]
    }
  }
}
```

GitHub tokens have no roles claim and you cannot add custom claims, so the example uses the `repository` claim as the role and grants admin to that exact repository. For GitLab, point `jwksUrl` at `https://gitlab.example.com/oauth/discovery/keys`, `issuerPattern` at your GitLab URL and `rolePath` at a claim you control (for example `project_path`). Pick a claim that a pipeline author cannot choose freely, and always set `audience`. See the [DIAL identity provider guides](https://docs.dialx.ai/tutorials/devops/auth-and-access-control/configure-idps/keycloak) for the full list of settings.

The repository's own CI uses this setup against a throwaway Core (`tests/core/docker-compose.yml`, job `integration-jwt`).

## Limitations

- The provider reads the token once, when Terraform configures it, and never refreshes it. The token must stay valid for the whole `terraform apply`, so set a token lifetime longer than the longest run where your CI system allows it.
- Providers are configured at the start of a run: a token minted after that point is not seen.
