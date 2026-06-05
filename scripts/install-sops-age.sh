#!/bin/sh
set -eu

AGE_VERSION="${AGE_VERSION:-v1.2.1}"
SOPS_VERSION="${SOPS_VERSION:-v3.11.0}"

if ! command -v curl >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq curl
fi

if ! command -v age-keygen >/dev/null 2>&1; then
  curl -sSL "https://github.com/FiloSottile/age/releases/download/${AGE_VERSION}/age-${AGE_VERSION}-linux-amd64.tar.gz" | tar xz -C /tmp
  install /tmp/age/age /tmp/age/age-keygen /usr/local/bin/
fi

if ! command -v sops >/dev/null 2>&1; then
  curl -sSL -o /usr/local/bin/sops "https://github.com/getsops/sops/releases/download/${SOPS_VERSION}/sops-${SOPS_VERSION}.linux.amd64"
  chmod +x /usr/local/bin/sops
fi

age --version
sops --version | head -1
