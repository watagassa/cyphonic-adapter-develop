#!/bin/bash

OS_NAME="$(uname | awk '{print tolower($0)}')"
OS_FULL="$(uname -a)"
ARCH_TYPE=""

case $(uname -m) in
i386) ARCH_TYPE="386" ;;
i486) ARCH_TYPE="386" ;;
i586) ARCH_TYPE="386" ;;
i686) ARCH_TYPE="386" ;;
x86_64) ARCH_TYPE="amd64" ;;
aarch64) ARCH_TYPE="arm64" ;;
armv8l) ARCH_TYPE="arm64" ;;
armv7l) ARCH_TYPE="armv7" ;;
armv6l) ARCH_TYPE="armv6" ;;
armv5l) ARCH_TYPE="armv5" ;;
mips) ARCH_TYPE="mips" ;;
mips64) ARCH_TYPE="mips64" ;;
mipsle) ARCH_TYPE="mipsle" ;;
mips64le) ARCH_TYPE="mips64le" ;;
ppc64) ARCH_TYPE="ppc64" ;;
ppc64le) ARCH_TYPE="ppc64le" ;;
s390x) ARCH_TYPE="s390x" ;;
esac

if [ "${OS_NAME}" == "darwin" ]; then
  if [ "${ARCH_TYPE}" == "amd64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*darwin-amd64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "arm64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*darwin-arm64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  fi
elif [ "${OS_NAME}" == "linux" ]; then
  if [ "${ARCH_TYPE}" == "386" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-386" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "amd64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-amd64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "arm64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-arm64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "armv7" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-armv7" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "armv6" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-armv6" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "armv5" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-armv5" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "mips" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-mips" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "mips64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-mips64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "mipsle" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-mipsle" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "mips64le" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-mips64le" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "ppc64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-ppc64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "ppc64le" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-ppc64le" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "s390x" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*linux-s390x" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  else
    echo "Your architecture is not supported."
  fi
elif [ "${OS_NAME}" == "netbsd" ]; then
  if [ "${ARCH_TYPE}" == "386" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*netbsd-386" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  elif [ "${ARCH_TYPE}" == "amd64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*netbsd-amd64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  else
    echo "Your architecture is not supported."
  fi
elif [ "${OS_NAME}" == "openbsd" ]; then
  if [ "${ARCH_TYPE}" == "amd64" ]; then
    curl -Ss# https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep "browser_download_url.*openbsd-amd64" | cut -d '"' -f 4 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin --strip-components=1 --wildcards '*/node_exporter'
  else
    echo "Your architecture is not supported."
  fi
else
  echo "Your platform ($(uname -a)) is not supported."
fi
