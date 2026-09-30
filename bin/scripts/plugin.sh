#!/bin/bash -e

tools="$(
  cd $(dirname $0)
  pwd
)/../../.tool-versions"

echo "🏃 Installing asdf tools in progress ..."
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
awk '{ print $1; }' ${tools} | while read tool; do
  asdf plugin add ${tool}
done
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
asdf install

plugins=(
  https://github.com/databus23/helm-diff
)

echo "🏃 Installing helm plugins in progress ..."
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
for plugin in "${plugins[@]}"; do
  helm plugin install ${plugin}
done
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
helm plugin list

rakes=(
  debase
  debride
  faster
  reek
  rubocop
  ruby-debug-ide
  ruby-lint
)

echo "🏃 Installing gem rakes in progress ..."
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
for rake in "${rakes[@]}"; do
  gem install ${rake}
done
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@"
gem list --local
