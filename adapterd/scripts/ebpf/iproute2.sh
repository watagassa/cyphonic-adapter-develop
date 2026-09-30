#!/bin/bash
set -euo pipefail

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "For Linux only." >&2
  exit 1
fi

PM=""
if   command -v apt-get >/dev/null 2>&1; then PM="apt"
elif command -v dnf     >/dev/null 2>&1; then PM="dnf"
elif command -v yum     >/dev/null 2>&1; then PM="yum"
elif command -v apk     >/dev/null 2>&1; then PM="apk"
elif command -v pacman  >/dev/null 2>&1; then PM="pacman"
elif command -v zypper  >/dev/null 2>&1; then PM="zypper"
else
  echo "No corresponding package manager found." >&2
  exit 1
fi

echo "Detected package manager: ${PM}"
case "${PM}" in
  apt)
    sudo apt-get update -y
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y iproute2
    ;;
  dnf)
    sudo dnf install -y iproute-tc || sudo dnf install -y iproute
    ;;
  yum)
    sudo yum install -y iproute-tc || sudo yum install -y iproute
    ;;
  apk)
    sudo apk add --no-cache iproute2
    ;;
  pacman)
    sudo pacman -Sy --noconfirm iproute2
    ;;
  zypper)
    sudo zypper --non-interactive install iproute2
    ;;
esac

if command -v tc >/dev/null 2>&1; then
  tc --version 2>/dev/null || true
else
  exit 1
fi
