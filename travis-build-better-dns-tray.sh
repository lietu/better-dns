#!/usr/bin/env bash

OSES="windows linux"  # TODO: Why darwin no workee?
# In future maybe: dragonfly freebsd netbsd openbsd solaris

declare -A ARCHITECTURES
ARCHITECTURES["darwin"]="amd64"
ARCHITECTURES["linux"]="amd64"
ARCHITECTURES["windows"]="amd64 386"

# ----- END CONFIGURATION ----- #

function capitalize() {
  local text="$1"
  echo -n "${text:0:1}" | tr a-z A-Z; echo -n ${text:1:999}
}

function big_label() {
  local len
  local zero
  local fill
  local text

  text=$(capitalize "$1")
  len=$(echo -n "$text" | wc -c)
  zero=$(printf %${len}s)
  fill=$(echo -n "$zero" | tr " " "-")

  echo
  echo
  echo
  echo  "/----$fill----\\"
  echo  "|    $zero    |"
  echo  "|    $text    |"
  echo  "|    $zero    |"
  echo "\\----$fill----/"
  echo
}

function label() {
  local len
  local zero
  local fill
  local text="$1"

  len=$(echo -n "$text" | wc -c)
  zero=$(printf %${len}s)
  fill=$(echo -n "$zero" | tr " " "-")

  echo
  echo  "/-$fill-\\"
  echo  "| $text |"
  echo "\\-$fill-/"
  echo
}

# Poor man's set -x (less spammy)
function cmd() {
  echo "$@"
  "$@"
}

# ----- END FUNCTIONS ----- #

set -e

GOFLAGS=-mod=vendor
ARTIFACTS="$PWD/artifacts"
mkdir -p "$ARTIFACTS"

for os in $OSES; do
  big_label "$os Better DNS manager"
  first=1

  for arch in ${ARCHITECTURES[${os}]}; do
    export GOOS="$os"
    export GOARCH="$arch"

    if [[ "$first" == "1" ]]; then
      cmd go vet ./cmd/better-dns-tray
      cmd staticcheck ./cmd/better-dns-tray
      cmd errcheck ./cmd/better-dns-tray
      cmd golangci-lint run ./cmd/better-dns-tray
      first=0
    fi

    cd cmd/better-dns-tray

    target="better-dns-tray-$os-$arch"
    if [[ "$os" == "windows" ]]; then
      target="$target.exe"
      cmd rsrc -manifest better-dns-tray.exe.manifest -o better-dns-tray_windows.syso
    fi

    label "Building $target for $os $arch"

    if [[ "$os" == "windows" ]]; then
      cmd go build -ldflags "-H windowsgui" -o "$target" better-dns-tray.go
    else
      cmd go build -o "$target" better-dns-tray.go
    fi

    cmd mv "$target" "$ARTIFACTS"
    cd -
  done
done

cp cmd/better-dns-tray/*.manifest "$ARTIFACTS"
