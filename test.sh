#!/usr/bin/env bash

# No real need to use stderr here since it's just tests.

set -eE

shellcheck "$0"

# Before we build...
go fmt
go doc
go test

go build
