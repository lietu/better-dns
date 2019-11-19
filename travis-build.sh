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

GO111MODULE=on
CGO_ENABLED=0
ARTIFACTS="$PWD/artifacts"
mkdir -p "$ARTIFACTS"

for os in $OSES; do
  big_label "$os"

  export GOOS="$os"

  cmd go vet . ./client ./server ./shared
  cmd staticcheck . ./client ./server ./shared
  cmd errcheck . ./client ./server ./shared
  cmd golangci-lint run . ./client ./server ./shared

  for arch in ${ARCHITECTURES[${os}]}; do
    label "Building for $os $arch target"

    export GOARCH="$arch"

    target="better-dns-$os-$arch"

    if [[ "$os" == "windows" ]]; then
      target="$target.exe"
    fi

    cmd go build -o "$target" better-dns.go
    cmd mv "$target" "$ARTIFACTS"
  done
done
