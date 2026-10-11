#!/usr/bin/env bash
# Runs the tests against the runner's tmux, then again against the latest tmux release built from
# source, since tmux's command parsing and output escaping differ between versions.
set -euo pipefail

TMUX_PACKAGES=(./internal/tmuxctl/... ./internal/popup/... ./internal/picker/... ./internal/preview/...)
SUDO=$([ "$(id -u)" = 0 ] || echo sudo)

echo "::group::tests ($(tmux -V))"
go test -v ./...
echo "::endgroup::"

echo "::group::build latest tmux"
$SUDO apt-get update -qq
$SUDO apt-get install -y -qq build-essential libevent-dev libncurses-dev bison pkg-config curl >/dev/null
tag=$(git ls-remote --tags --refs https://github.com/tmux/tmux | sed 's#.*refs/tags/##' |
  grep -E '^[0-9]+\.[0-9]+[a-z]?$' | sort -V | tail -1)
src=$(mktemp -d)
curl -fsSL "https://github.com/tmux/tmux/releases/download/${tag}/tmux-${tag}.tar.gz" | tar -xz -C "$src" --strip-components 1
prefix="$HOME/.local/tmux-${tag}"
(cd "$src" && ./configure --prefix="$prefix" >/dev/null && make -j"$(nproc)" >/dev/null && make install >/dev/null)
echo "::endgroup::"

export PATH="$prefix/bin:$PATH"
echo "::group::tests ($(tmux -V))"
go test -v -count=1 "${TMUX_PACKAGES[@]}"
echo "::endgroup::"
