#!/usr/bin/env bash
# Install sandbox dependencies for Tier-1 verification (Cursor cloud VM / local dev).
set -euo pipefail

echo "install-sandbox: python conformance deps"
python3 -m pip install --user -r "$(dirname "$0")/requirements.txt"

if command -v apt-get >/dev/null 2>&1; then
  echo "install-sandbox: apt packages (vault, postgres client, kubeconform)"
  sudo apt-get update -qq
  sudo apt-get install -y -qq curl gnupg lsb-release postgresql-client python3-pip

  if ! command -v vault >/dev/null 2>&1; then
    curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
    echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" \
      | sudo tee /etc/apt/sources.list.d/hashicorp.list >/dev/null
    sudo apt-get update -qq && sudo apt-get install -y -qq vault
  fi

  if ! command -v kubeconform >/dev/null 2>&1; then
    KUBECONFORM_VERSION="${KUBECONFORM_VERSION:-0.6.7}"
    curl -fsSL "https://github.com/yannh/kubeconform/releases/download/v${KUBECONFORM_VERSION}/kubeconform-linux-amd64.tar.gz" \
      | sudo tar xz -C /usr/local/bin kubeconform
  fi
fi

echo "install-sandbox: done"
