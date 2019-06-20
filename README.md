# Decensor

golang file/asset manager for tagging and helping ensure data is replicated and not censored.

I'm still pretty new to Go and this is fairly ugly. May look a bit like Python in places where it shouldn't. Not all of the error handling is consistent. This is probably my biggest Go project by about twice, so that adds to the issues. Hopefully will clean it up in time.

Nothing should be considered stable as of yet. Interfaces, etc. Even maybe the name will change.

## Installation

Fetch the code, then run `go build`

Or: `go get -v github.com/teran-mckinney/decensor`

## Usage

 * decensor init
 * decensor add_and_tag objectioablememe.png censoredtopic_1 censoredtopic_2
 * decensor assets
 * decensor tags
 * decensor web :4444 # Browse to localhost:4444

## TODO

Lots...

 * Import/export?
 * Make web interface pretty.
 * Add a bunch of unit tests and functional tests.

## Consider

 * Changing hash format to multihash for shorter SHA256SUMs?

## License

Public domain / Unlicense
