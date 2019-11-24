#!/usr/bin/env bash

OSES="darwin linux windows"
# In future maybe: dragonfly freebsd netbsd openbsd solaris

declare -A ARCHITECTURES
ARCHITECTURES["darwin"]="amd64"
ARCHITECTURES["linux"]="amd64 386 arm arm64"
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
  local text=$(capitalize "$1")

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
  local cmd="$@"
  echo "$cmd"
  $cmd
}

# ----- END FUNCTIONS ----- #

set -e

GO111MODULE=on
CGO_ENABLED=0
ARTIFACTS="$PWD/artifacts"
mkdir -p "$ARTIFACTS"

for os in $OSES; do
  big_label "$os Better DNS"
  first=1

  for arch in ${ARCHITECTURES[${os}]}; do
    export GOOS="$os"
    export GOARCH="$arch"

    if [[ "$first" == "1" ]]; then
      cmd go vet ./client ./server ./shared ./cmd/better-dns
      cmd staticcheck ./client ./server ./shared ./cmd/better-dns
      cmd errcheck ./client ./server ./shared ./cmd/better-dns
      cmd golangci-lint run ./client ./server ./shared ./cmd/better-dns
      first=0
    fi

    cd cmd/better-dns

    target="better-dns-$os-$arch"
    if [[ "$os" == "windows" ]] && [[ ! -f "better-dns_windows.syso" ]]; then
      target="$target.exe"
      cmd rsrc -manifest better-dns.exe.manifest -o better-dns_windows.syso
    fi

    label "Building $target for $os $arch"

    cmd go build -o "$target" better-dns.go
    cmd mv "$target" "$ARTIFACTS"

    cd -
  done
done

cp cmd/better-dns/*.manifest "$ARTIFACTS"
