#!/bin/bash

set -e

CGO_ENABLED=0 gox -ldflags "-s -w ${LDFLAGS}" -output="build/packctl_{{.OS}}_{{.Arch}}" --osarch="darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm linux/arm64"

cd build

rhash -r -a . -o checksums

rhash -r -a --bsd . -o checksums-bsd

rhash --list-hashes > checksums_hashes_order