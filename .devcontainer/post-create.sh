#!/usr/bin/env bash
set -euo pipefail

git config --global user.name "$GIT_USER_NAME"
git config --global user.email "$GIT_USER_EMAIL"

# GitHub access: gh reads GH_TOKEN from ~/devcontainer.env; the credential helper lets git push over HTTPS with it.
if [ -n "${GH_TOKEN:-}" ]; then
  gh auth setup-git
else
  echo "GH_TOKEN is not set; add it to ~/devcontainer.env on the host to push to GitHub and read CI status."
fi
git remote get-url github >/dev/null 2>&1 \
  || git remote add github https://github.com/epam/terraform-provider-aidial.git

# Terraform dev_overrides for local provider builds (`make install-dev`).
# The home dir is a persistent volume, so never overwrite an existing file.
if [ -e "$HOME/.terraformrc" ]; then
  echo "~/.terraformrc already exists; not touching it. dev_overrides for epam/aidial should point at $PWD/dist/linux_amd64"
else
  cat > "$HOME/.terraformrc" <<TFRC
provider_installation {
  dev_overrides {
    "epam/aidial" = "$PWD/dist/linux_amd64"
  }
  direct {}
}
TFRC
fi

go mod download
pre-commit install
