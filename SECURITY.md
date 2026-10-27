# Security Policy

## Supported versions

Security fixes are released for the latest minor version.

## Reporting a vulnerability

Please do not open a public issue for security problems. Report them privately through GitHub's [private vulnerability reporting](https://github.com/epam/terraform-provider-aidial/security/advisories/new) for this repository, with a description, reproduction steps and the affected version. You should receive an acknowledgement within a few working days.

## Handling secrets

The provider sends the configured `api_key` in the `Api-Key` header and stores the `aidial_key` secret in Terraform state. Mark variables that hold them `sensitive = true` and protect your state backend accordingly.
