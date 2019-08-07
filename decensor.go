// decensor
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const decensor_path_suffix = "/.decensor"

var base = basedir()

var tags_dir = base + "/tags"
var assets_dir = base + "/assets"
var metadata_dir = base + "/metadata"

func validate_asset(asset string) bool {
	if len(asset) != 64 {
		return false
	}
	if IsHex(asset) == false {
		return false
	}
	return true
}

func error_asset(asset string) error {
	if validate_asset(asset) == false {
		return errors.New("Assets must be 64 hex characters.")
	} else {
		return nil
	}
}

func get_hash(path string) (string, error) {
	// If we do this all in one chunk, we can easily run out of memory on big files.
	// Instead, we use io.Copy and hash as we go.
	var hash_string string

	fd, err := os.Open(path)
	if err != nil {
		return hash_string, err
	}
	defer fd.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, fd)
	if err != nil {
		return hash_string, err
	}
	hash_sum := hash.Sum(nil)
	hash_string = hex.EncodeToString(hash_sum[:])
	return hash_string, err
}

func copy_file(source, destination string) error {
	source_fp, err := os.Open(source)
	if err != nil {
		return err
	}
	defer source_fp.Close()

	destination_fp, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destination_fp.Close()

	_, err = io.Copy(destination_fp, source_fp)

	return err
}

func add(path string) (string, error) {
	var hash string
	var err error

	/* This also checks if we can read the source file. */
	hash, err = get_hash(path)
	if err != nil {
		return hash, err
	}
	/* Make sure we don't already have the asset. */
	asset_path := assets_dir + "/" + hash
	if _, err = os.Stat(asset_path); err == nil {
		return hash, errors.New("Asset already exists.")
	}
	/* Use hard links to save space when the file is on the same device. If it's not, copy it. */
	if err = os.Link(path, asset_path); err != nil {
		log.Println("Hard link failed, attempting copy.")
		err = copy_file(path, asset_path)
		if err != nil {
			return hash, err
		}
	}
	// In case someone is adding /dir/foo.jpg and not foo.jpg
	path = filepath.Base(path)
	add_filename(hash, path)
	return hash, err
}

func remove(asset string) error {
	var err error
	if err = error_asset(asset); err != nil {
		return err
	}
	asset_path := assets_dir + "/" + asset
	_, err = os.Stat(asset_path)
	if os.IsNotExist(err) {
		return errors.New("Asset does not exist, cannot remove.")
	}
	filename := asset_metadata_filename(asset)
	log.Printf("Asset had filename: %s", filename)
	tags_for_asset, err := forward_tags_by_asset(asset)
	if err != nil {
		return err
	}
	for _, tag := range tags_for_asset {
		if err = os.Remove(tags_dir + "/" + tag + "/" + asset); err != nil {
			return err
		} else {
			log.Printf("Removed from tag %s", tag)
			tag_assets, err := assets_by_tag(tag)
			if err != nil {
				return err
			}
			if len(tag_assets) == 0 {
				log.Printf("Tag %s is now empty, consider deleting.", tag)
			}
		}
	}
	asset_metadata_path := metadata_dir + "/" + asset
	_, err = os.Stat(asset_metadata_path)
	log.Print(asset_metadata_path)
	if err == nil {
		if err = os.RemoveAll(asset_metadata_path); err != nil {
			return err
		}
	} else {
		log.Print("No metadata for asset found.")
	}
	err = os.Remove(asset_path)
	// We'll return nil if os.Remove was fine.
	return err
}

func init_metadata(asset string) error {
	var err error
	directory := metadata_dir + "/" + asset
	if _, err = os.Stat(directory); err != nil {
		err = os.Mkdir(directory, 0755)
	}
	return err
}

func init_back_tags(asset string) error {
	var err error
	if err = init_metadata(asset); err != nil {
		return err
	}
	// Make tags directory if it doesn't exist already.
	directory := asset_filepath_metadata_tags(asset)
	if _, err = os.Stat(directory); err != nil {
		err = os.Mkdir(directory, 0755)
	}
	return err
}

func add_filename(asset string, filename string) error {
	var err error
	var path string
	if err = error_asset(asset); err != nil {
		return err
	}
	if err = init_metadata(asset); err != nil {
		return err
	}
	path = asset_filepath_metadata_filename(asset)
	err = ioutil.WriteFile(path, []byte(filename+"\n"), 0644)
	return err
}

func asset_filepath_metadata_filename(asset string) string {
	return metadata_dir + "/" + asset + "/filename"
}

func asset_filepath_metadata_tags(asset string) string {
	return metadata_dir + "/" + asset + "/tags/"
}

func asset_metadata_filename(asset string) string {
	path := asset_filepath_metadata_filename(asset)
	filename_byte, err := ioutil.ReadFile(path)
	if err != nil {
		log.Print(err.Error())
		return ""
	}
	filename := strings.Trim(string(filename_byte), "\n")
	return filename
}

func basedir() string {
	environment_path := os.Getenv("DECENSOR_DIR")
	if environment_path == "" {
		home, err := os.UserHomeDir()
		fatal_error(err)
		return home + decensor_path_suffix
	} else {
		return environment_path
	}
}

func fatal_error(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func list_directory(directory string) ([]string, error) {
	return list_directory_sorted(directory)
}

func list_directory_sorted(directory string) ([]string, error) {
	// This should be slower because it sorts the output.
	// The sorting actually does not appear to be very slow at all.
	dir_entries, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var entries []string
	for _, f := range dir_entries {
		entries = append(entries, f.Name())
	}
	return entries, nil
}

func list_directory_unsorted(directory string) ([]string, error) {
	// This should be faster because there's no sorting.
	fd, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	dir_entries, err := fd.Readdir(-1)
	fd.Close()
	if err != nil {
		return nil, err
	}
	var entries []string
	for _, f := range dir_entries {
		entries = append(entries, f.Name())
	}
	return entries, nil
}

func assets() ([]string, error) {
	return list_directory(assets_dir)
}

func tags() ([]string, error) {
	return list_directory(tags_dir)
}

func assets_by_tag(tag string) ([]string, error) {
	return list_directory(tags_dir + "/" + tag)
}

func tags_by_asset(asset string) ([]string, error) {
	return back_tags_by_asset(asset)
}

func forward_tags_by_asset(asset string) ([]string, error) {
	var asset_tags []string
	var err error
	if err = error_asset(asset); err != nil {
		return asset_tags, err
	}
	all_tags, err := tags()
	if err != nil {
		return nil, err
	}
	for _, tag := range all_tags {
		tag_assets, err := assets_by_tag(tag)
		if err != nil {
			return nil, err
		}
		for _, possible_asset := range tag_assets {
			if asset == possible_asset {
				asset_tags = append(asset_tags, tag)
				break
			}
		}
	}
	return asset_tags, err
}

func back_tags_by_asset(asset string) ([]string, error) {
	return list_directory(asset_filepath_metadata_tags(asset))
}

func validate_asset_tags_forward_and_back(asset string) error {
	forward_tags, err := forward_tags_by_asset(asset)
	if err != nil {
		return err
	}
	back_tags, err := back_tags_by_asset(asset)
	if err != nil {
		log.Print(err.Error())
	}
	if len(forward_tags) != len(back_tags) {
		goto return_error
	}
	for index, _ := range forward_tags {
		if forward_tags[index] != back_tags[index] {
			goto return_error
		}
	}
	return nil

return_error:
	return errors.New(fmt.Sprintf("%s has tags that do not match", asset))
}

func back_tag(asset string, tag string) error {
	var err error
	if err = init_back_tags(asset); err != nil {
		return err
	}
	path := asset_filepath_metadata_tags(asset) + tag
	err = ioutil.WriteFile(path, []byte(""), 0644)
	return err
}

func tag(asset string, tags []string) error {
	var err error
	if err = error_asset(asset); err != nil {
		return err
	}
	for _, tag := range tags {
		directory := tags_dir + "/" + tag
		_, err = os.Stat(directory)
		/* Make the tag if it doesn't exist already */
		if os.IsNotExist(err) {
			log.Printf("Tag %s does not exist, creating.", tag)
			err = os.Mkdir(directory, 0755)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		// Check if asset already has this tag.
		tag_path := directory + "/" + asset
		_, err = os.Stat(tag_path)
		if os.IsNotExist(err) {
			err = ioutil.WriteFile(directory+"/"+asset, []byte(""), 0644)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			return errors.New("Asset already has this tag.")
		}
		// Add back tag to keep things fast.
		if err = back_tag(asset, tag); err != nil {
			return err
		}
	}
	return err
}

func validate_assets() error {
	var hash string
	var err error

	assets, err := assets()
	if err != nil {
		return err
	}
	for _, asset := range assets {
		hash, err = get_hash(assets_dir + "/" + asset)
		if err != nil {
			return err
		}
		if asset != hash {
			return errors.New(fmt.Sprintf("%s does not match %s", hash, asset))
		}
		if err = validate_asset_tags_forward_and_back(asset); err != nil {
			return err
		}
	}
	return err
}

func back_tag_all_assets() error {
	// This must be ran before adding any new assets! It's to port legacy systems over.
	// 2019-07-15 and prior
	var err error
	all_assets, err := assets()
	if err != nil {
		return err
	}
	for _, asset := range all_assets {
		asset_tags, err := forward_tags_by_asset(asset)
		if err != nil {
			log.Printf("Failure in back_tag_all_assets with asset: %s", asset)
			return err
		}
		for _, tag := range asset_tags {
			err := back_tag(asset, tag)
			if err != nil {
				log.Printf("Failure in back_tag_all_assets with asset: %s", asset)
				return err
			}
		}
	}
	return err
}

func info(asset string) (string, error) {
	var err error
	var filename string
	filename = asset_metadata_filename(asset)
	var asset_tags []string
	asset_tags, err = tags_by_asset(asset)
	if err != nil {
		return "", err
	}
	var info_string string
	info_string = asset + "\nFilename: " + filename + "Tags:\n"
	for _, tag := range asset_tags {
		info_string = info_string + "\n" + tag
	}
	return info_string, nil
}

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

func init_folders() error {
	var err error
	if err = os.Mkdir(base, 0755); err != nil {
		return err
	}
	if err = os.Mkdir(assets_dir, 0755); err != nil {
		return err
	}
	if err = os.Mkdir(tags_dir, 0755); err != nil {
		return err
	}
	if err = os.Mkdir(metadata_dir, 0755); err != nil {
		return err
	}
	return nil
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
		tags, err = tags_by_asset(os.Args[2])
		fatal_error(err)
		print_list(tags)
	case "info":
		exactly_arguments(3)
		var infotext string
		infotext, err = info(os.Args[2])
		fatal_error(err)
		fmt.Println(infotext)
	default:
		usage()
	}

}
