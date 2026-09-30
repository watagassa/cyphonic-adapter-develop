#!/bin/bash

OS_NAME="$(uname | awk '{print tolower($0)}')"
OS_FULL="$(uname -a)"
OS_TYPE=
ARCH_TYPE=""

case $(uname -m) in
i386) ARCH_TYPE="386" ;;
i686) ARCH_TYPE="386" ;;
x86_64) ARCH_TYPE="amd64" ;;
aarch64) ARCH_TYPE="arm64" ;;
arm) dpkg --print-architecture | grep -q "arm64" && ARCH_TYPE="arm64" || ARCH_TYPE="arm" ;;
esac

if [ "${OS_NAME}" == "linux" ]; then
  if [ $(echo "${OS_FULL}" | grep -c "amzn1") -gt 0 ]; then
    OS_TYPE="yum"
  elif [ $(echo "${OS_FULL}" | grep -c "amzn2") -gt 0 ]; then
    OS_TYPE="yum"
  elif [ $(echo "${OS_FULL}" | grep -c "el6") -gt 0 ]; then
    OS_TYPE="yum"
  elif [ $(echo "${OS_FULL}" | grep -c "el7") -gt 0 ]; then
    OS_TYPE="yum"
  elif [ $(echo "${OS_FULL}" | grep -c "Ubuntu") -gt 0 ]; then
    OS_TYPE="apt"
  elif [ $(echo "${OS_FULL}" | grep -c "coreos") -gt 0 ]; then
    OS_TYPE="apt"
  fi
elif [ "${OS_NAME}" == "darwin" ]; then
  OS_TYPE="brew"
fi

if [ "${OS_NAME}" == "darwin" ]; then
  curl -Ss# https://api.github.com/repos/coredns/coredns/releases/latest | grep "browser_download_url.*_darwin_amd64.tgz" | grep "coredns" | grep -v "sha" | cut -d : -f 2,3 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin
  OS='Mac'
elif [ "${OS_NAME}" == "linux" ]; then
  if [ "${ARCH_TYPE}" == "amd64" ]; then
    curl -Ss# https://api.github.com/repos/coredns/coredns/releases/latest | grep "browser_download_url.*_linux_amd64.tgz" | grep "coredns" | grep -v "sha" | cut -d : -f 2,3 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin
  elif [ "${ARCH_TYPE}" == "arm64" ]; then
    curl -Ss# https://api.github.com/repos/coredns/coredns/releases/latest | grep "browser_download_url.*_linux_arm64.tgz" | grep "coredns" | grep -v "sha" | cut -d : -f 2,3 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin
  elif [ "${ARCH_TYPE}" == "arm" ]; then
    curl -Ss# https://api.github.com/repos/coredns/coredns/releases/latest | grep "browser_download_url.*_linux_arm.tgz" | grep "coredns" | grep -v "sha" | cut -d : -f 2,3 | tr -d \" | xargs curl -L | tar -xvzf - -C /usr/local/bin
  fi
  OS='Linux'
elif [ "$(expr substr $(uname -s) 1 10)" == 'MINGW32_NT' ]; then
  OS='Cygwin'
else
  echo "Your platform ($(uname -a)) is not supported."
fi
