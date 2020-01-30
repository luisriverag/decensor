# Decensor

golang file/asset manager for tagging and helping ensure data is replicated and not censored.

## Installation

Fetch the code, then run `go build`

Or: `go get -v github.com/teran-mckinney/decensor`

## Upgrading

If you used Decensor from commit b8dd5ff51fdb6b391556e4534a84f77adb574451 (July 15th, 2019) or earlier, you'll need to run `decensor back_tag_all_assets` before doing any other operations.

## Usage

 * decensor init
 * decensor add_and_tag objectioablememe.png censoredtopic_1 censoredtopic_2
 * decensor assets
 * decensor tags
 * decensor web :4444 # Browse to localhost:4444

Also see [decensor.service](decensor.service) for a sample Systemd service file.

### Get Bootstrap theme so web mode doesn't look awful

 * `curl -O https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css`
 * `decensor add bootstrap.min.css` (File extension must end in .css when added for it to work in browsers due to the Content-Type.)

### Add LICENSE file so license link on the index works

 * `decensor add LICENSE`

## TODO

### Features

 * Import/export
 * Markdown rendering on permalink
 * Version reporting

### Performance

 * Caching

## Consider

 * Changing hash format to multihash for shorter SHA256SUMs?

## License

Public domain / Unlicense
