#!/bin/bash

set -eu

if [ "$(uname -s)" != "Linux" ]; then
  echo "This script supports Linux only."
  exit 1
fi

if [ ! -f /etc/debian_version ] || ! command -v apt-get >/dev/null 2>&1; then
  echo "This script supports apt-based Linux distributions only."
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this script as root (use sudo)."
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y redis-server

sed -i '/^#\?\s*requirepass/c\requirepass cyphonic' /etc/redis/redis.conf

echo "Redis installation completed."
