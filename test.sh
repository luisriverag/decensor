#!/usr/bin/env bash

# No real need to use stderr here since it's just tests.

TEST_SCRAP_DIR=test_scrap_dir
TEST_DECENSOR_DIR=test_decensor_dir

set -eE

shellcheck "$0"

# Before we build...
go fmt
go doc
go test

go build

cleanup() {
    echo "Cleaning up."
    rm -r "$TEST_DECENSOR_DIR" || true
    rm -r "$TEST_SCRAP_DIR" || true
}

trap fail $(seq 1 64)

fail() {
    echo "FAIL: $1"
    cleanup
    exit 1
}

export DECENSOR_DIR=$TEST_DECENSOR_DIR

[ -d "$TEST_SCRAP_DIR" ] && fail "$TEST_SCRAP_DIR should not exist."
[ -d "$TEST_DECENSOR_DIR" ] && fail "$TEST_DECENSOR_DIR should not exist."

./decensor add decensor.go && fail "Should not be able to add a file without decensor init"
./decensor init || fail "Unable to init"
./decensor add decensor.go || fail "Unable to add decensor.go"
./decensor add noneexistentfile && fail "Should not be able to add non-existent file."

mkdir "$TEST_SCRAP_DIR"
echo Hello\ World > "$TEST_SCRAP_DIR"/hello

./decensor add "$TEST_SCRAP_DIR"/hello || fail "Unable to add Hello World"

[ -f "$TEST_DECENSOR_DIR"/assets/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26 ] || fail "Hello World hash not found."

./decensor add "$TEST_SCRAP_DIR"/hello && fail "Should not be able to add same file twice."

echo Hello\ World\ 2 > "$TEST_SCRAP_DIR"/hello2

./decensor add_and_tag "$TEST_SCRAP_DIR"/hello2 stuff things || fail "Failed to add hello2"

echo Hello\ World\ 3 > "$TEST_SCRAP_DIR"/hello3

./decensor add_and_tag "$TEST_SCRAP_DIR"/hello3 stuff morethings || fail "Failed to add hello3"

./decensor tags | grep stuff || fail "stuff tag not found."

./decensor validate_assets || fail "Assets should be valid"

echo A >> "$TEST_DECENSOR_DIR"/assets/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26

./decensor validate_assets && fail "Assets should be invalid"

cleanup

echo Success
