#!/usr/bin/env bash

# No real need to use stderr here since it's just tests.

TEST_SCRAP_DIR=test_scrap_dir
TEST_DECENSOR_DIR=test_decensor_dir

shellcheck "$0"

# Before we build...
go fmt
go doc
go test

go build

strip -s decensor

cleanup() {
    echo "Cleaning up."
    if [ -n "$PID" ]; then
        kill "$PID" || true
    fi
    rm -r "$TEST_DECENSOR_DIR" || true
    rm -r "$TEST_SCRAP_DIR" || true
}

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

./decensor tag d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26 justanewtag || fail "Failed to add a tag"

./decensor tag d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26 justanewtag && fail "Should fail to add a tag twice."

./decensor validate_assets || fail "Assets should be valid"

## Remove back tag

rm "$TEST_DECENSOR_DIR"/metadata/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26/tags/justanewtag

./decensor validate_assets && fail "Assets should be invalid"

touch "$TEST_DECENSOR_DIR"/metadata/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26/tags/justanewtag

./decensor validate_assets || fail "Assets should be valid"

##

## Remove forward tag

rm "$TEST_DECENSOR_DIR"/tags/justanewtag/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26

./decensor validate_assets && fail "Assets should be invalid"

touch "$TEST_DECENSOR_DIR"/tags/justanewtag/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26

./decensor validate_assets || fail "Assets should be valid"

##

echo A >> "$TEST_DECENSOR_DIR"/assets/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26

./decensor validate_assets && fail "Assets should be invalid"

cleanup

mkdir "$TEST_SCRAP_DIR"
echo Hello\ World > "$TEST_SCRAP_DIR"/hello

./decensor init || fail "Second init failed"

./decensor add_and_tag "$TEST_SCRAP_DIR"/hello sametag ragtag || fail "Unable to add Hello World"

./decensor add_and_tag decensor.go sometag sametag || fail "Should be able to add a file after decensor init"

./decensor remove d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26 || fail "Should be able to remove Hello World"

./decensor remove d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26 && fail "Should already be removed."

find "$DECENSOR_DIR"

[ "$(find "$DECENSOR_DIR" | wc -l)" -eq 15 ] || fail "Found more files than expected after remove."

./decensor add "$TEST_SCRAP_DIR"/hello || fail "Unable to add Hello World"

# Test without a metadata dir.
rm -r "$DECENSOR_DIR"/metadata/d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26/

./decensor remove d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26 || fail "Should be able to remove Hello World, even without a metadata dir"

# Web tests.

./decensor web :4999 &
PID=$!

curl -so /dev/null --show-error --fail "http://localhost:4999/assets/" || fail "404 for assets?"

curl -so /dev/null --show-error --fail "http://localhost:4999/tag/no_tag" && fail "No 404 for no tag?"

curl -so /dev/null --show-error --fail "http://localhost:4999/tags/whatever" || fail "tags/whatever should return /tags/, at least for now"

curl -so /dev/null --show-error --fail "http://localhost:4999/tags/" || fail "404 for tags?"

## Make sure CSS is returned as CSS.

echo 'a:' > "$TEST_SCRAP_DIR/foo.css"

./decensor add "$TEST_SCRAP_DIR/foo.css"

curl -I -s --show-error --fail "http://localhost:4999/asset/81a039d5debf48b9eccf2bbd53aa6140627b3354e95c74a81a5d6317c81581f6" | grep text/css || fail "Invalid content type for CSS"

##

## Make sure Markdown is returned as text so browsers show and don't download (for now)

echo '# I am Markdown' > "$TEST_SCRAP_DIR/foo.md"

./decensor add "$TEST_SCRAP_DIR/foo.md"

curl -I -s --show-error --fail "http://localhost:4999/asset/c8deb6b237964318040fe890deb2d8f6129cc3f3a6311e95d8553ef88791ccf3" | grep text/plain || fail "Invalid content type for Markdown"

./decensor tag c8deb6b237964318040fe890deb2d8f6129cc3f3a6311e95d8553ef88791ccf3 foo || fail "Should be able to tag"

./decensor validate_assets || fail "assets should be valid"

rm -r "$DECENSOR_DIR/metadata/c8deb6b237964318040fe890deb2d8f6129cc3f3a6311e95d8553ef88791ccf3/tags"

./decensor validate_assets && fail "assets should be invalid"

./decensor back_tag_all_assets || fail "should be able to back tag all assets"

./decensor validate_assets || fail "assets should be valid"

##

# All done

cleanup

echo Success
