package main

import (
	"fmt"
	"os"
)

func exactly_arguments(arguments int) {
	if len(os.Args) != arguments {
		usage()
	}
}

func print_list(list []string) {
	for _, item := range list {
		fmt.Println(item)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: decensor <command> [argument]")
	fmt.Fprintln(os.Stderr, "Command: init")
	fmt.Fprintln(os.Stderr, "Command: back_tag_all_assets")
	fmt.Fprintln(os.Stderr, "Command: basedir")
	fmt.Fprintln(os.Stderr, "Command: hash <file to hash>")
	fmt.Fprintln(os.Stderr, "Command: web <port> (Example: :4444)")
	fmt.Fprintln(os.Stderr, "Command: info <asset>")
	fmt.Fprintln(os.Stderr, "Command: assets")
	fmt.Fprintln(os.Stderr, "Command: assets_by_tag <tag>")
	fmt.Fprintln(os.Stderr, "Command: tags_by_asset <asset>")
	fmt.Fprintln(os.Stderr, "Command: tags")
	fmt.Fprintln(os.Stderr, "Command: tag <asset> <tag> <tag> <tag>...")
	fmt.Fprintln(os.Stderr, "Command: metadata_by_asset <asset>")
	fmt.Fprintln(os.Stderr, "Command: validate_assets")
	fmt.Fprintln(os.Stderr, "Command: add <path to file>")
	fmt.Fprintln(os.Stderr, "Command: add_and_tag <path to file> <tag> <tag> <tag>...")
	fmt.Fprintln(os.Stderr, "Command: remove <asset>")
	os.Exit(1)
}

func main() {
	var err error
	if len(os.Args) <= 1 {
		usage()
	}

	switch os.Args[1] {
	case "init":
		exactly_arguments(2)
		fatal_error(init_folders())
	case "basedir":
		exactly_arguments(2)
		fmt.Println(basedir())
	case "web":
		exactly_arguments(3)
		web(os.Args[2])
	case "hash":
		exactly_arguments(3)
		var hash string
		hash, err = get_hash(os.Args[2])
		fatal_error(err)
		fmt.Println(hash)
	case "add":
		exactly_arguments(3)
		var asset_hash string
		asset_hash, err = add(os.Args[2])
		fatal_error(err)
		fmt.Println(asset_hash)
	case "remove":
		exactly_arguments(3)
		fatal_error(remove(os.Args[2]))
	case "validate_assets":
		exactly_arguments(2)
		fatal_error(validate_assets())
	case "back_tag_all_assets":
		exactly_arguments(2)
		fatal_error(back_tag_all_assets())
	case "tags":
		exactly_arguments(2)
		all_tags, err := tags()
		fatal_error(err)
		print_list(all_tags)
	case "tag":
		if len(os.Args) <= 3 {
			usage()
		}
		err = tag(os.Args[2], os.Args[3:])
		fatal_error(err)
	case "add_and_tag":
		if len(os.Args) <= 3 {
			usage()
		}
		var asset_hash string
		asset_hash, err = add(os.Args[2])
		fatal_error(err)
		tag(asset_hash, os.Args[3:])
		fmt.Println(asset_hash)
	case "assets":
		exactly_arguments(2)
		all_assets, err := assets()
		fatal_error(err)
		print_list(all_assets)
	case "assets_by_tag":
		exactly_arguments(3)
		tag_assets, err := assets_by_tag(os.Args[2])
		fatal_error(err)
		print_list(tag_assets)
	case "tags_by_asset":
		exactly_arguments(3)
		var tags []string
		tags = tags_by_asset(os.Args[2])
		print_list(tags)
	case "info":
		exactly_arguments(3)
		var infotext string
		infotext = info(os.Args[2])
		fmt.Println(infotext)
	default:
		usage()
	}

}
