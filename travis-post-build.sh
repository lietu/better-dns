#!/usr/bin/env bash

ARTIFACTS="$PWD/artifacts"
cd "$ARTIFACTS"
sha256sum ./* >> SHA256SUMS
cd -

git log -1 --pretty=%B > DESCRIPTION
echo >> DESCRIPTION
echo '```' >> DESCRIPTION
cat "$ARTIFACTS/SHA256SUMS" >> DESCRIPTION
echo '```' >> DESCRIPTION
