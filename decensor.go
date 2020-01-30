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

var tagsDir = base + "/tags"
var assetsDir = base + "/assets"
var metadataDir = base + "/metadata"

func isHex(hexString string) bool {
	for _, character := range hexString {
		if !strings.Contains("abcdef01234567890", string(character)) {
			return false
		}
	}
	return true
}

func validate_asset(asset string) bool {
	if len(asset) != 64 {
		return false
	}
	if isHex(asset) == false {
		return false
	}
	return true
}

func validateAsset(asset string) error {
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

func getAssetSize(asset string) (bytes int64, err error) {
	assetPath := getAssetPath(asset)
	stat, err := os.Stat(assetPath)
	if err != nil {
		return
	}
	bytes = stat.Size()
	return
}

func getAssetPath(hash string) string {
	return assetsDir + "/" + hash
}

func add(path string) (hash string, err error) {
	/* This also checks if we can read the source file. */
	hash, err = get_hash(path)
	if err != nil {
		return hash, err
	}
	/* Make sure we don't already have the asset. */
	asset_path := getAssetPath(hash)
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
	filename := filepath.Base(path)
	if err = addFilename(hash, filename); err != nil {
		return
	}
	return
}

func remove(asset string) error {
	var err error
	if err = validateAsset(asset); err != nil {
		return err
	}
	asset_path := assetsDir + "/" + asset
	_, err = os.Stat(asset_path)
	if os.IsNotExist(err) {
		return errors.New("Asset does not exist, cannot remove.")
	}
	filename := getAssetFilename(asset)
	log.Printf("Asset had filename: %s", filename)
	tags_for_asset, err := forward_tags_by_asset(asset)
	if err != nil {
		return err
	}
	for _, tag := range tags_for_asset {
		if err = os.Remove(tagsDir + "/" + tag + "/" + asset); err != nil {
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
	asset_metadata_path := metadataDir + "/" + asset
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
	directory := metadataDir + "/" + asset
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
	directory := getAssetFilePathTags(asset)
	if _, err = os.Stat(directory); err != nil {
		err = os.Mkdir(directory, 0755)
	}
	return err
}

func addFilename(asset string, filename string) error {
	var err error
	var path string
	if err = validateAsset(asset); err != nil {
		return err
	}
	if err = init_metadata(asset); err != nil {
		return err
	}
	path = getAssetFilePathFilename(asset)
	err = ioutil.WriteFile(path, []byte(filename+"\n"), 0644)
	return err
}

func getAssetFilePathFilename(asset string) string {
	return metadataDir + "/" + asset + "/filename"
}

func getAssetFilePathTags(asset string) string {
	return metadataDir + "/" + asset + "/tags/"
}

func getAssetFilename(asset string) (filename string) {
	filenameByte, err := ioutil.ReadFile(getAssetFilePathFilename(asset))
	if err != nil {
		filename = asset
	} else {
		filename = strings.Trim(string(filenameByte), "\n")
	}
	return
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
	return list_directory(assetsDir)
}

func tags() ([]string, error) {
	return list_directory(tagsDir)
}

func assets_by_tag(tag string) ([]string, error) {
	return list_directory(tagsDir + "/" + tag)
}

func tags_by_asset(asset string) (tags []string) {
	// No real issue if an asset doesn't have tags
	tags, _ = back_tags_by_asset(asset)
	return
}

func forward_tags_by_asset(asset string) ([]string, error) {
	var asset_tags []string
	var err error
	if err = validateAsset(asset); err != nil {
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
	return list_directory(getAssetFilePathTags(asset))
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
	path := getAssetFilePathTags(asset) + tag
	err = ioutil.WriteFile(path, []byte(""), 0644)
	return err
}

func tag(asset string, tags []string) error {
	var err error
	if err = validateAsset(asset); err != nil {
		return err
	}
	for _, tag := range tags {
		directory := tagsDir + "/" + tag
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
		hash, err = get_hash(assetsDir + "/" + asset)
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

func info(asset string) (info_string string) {
	var filename string
	filename = getAssetFilename(asset)
	var asset_tags []string
	asset_tags = tags_by_asset(asset)
	info_string = asset + "\nFilename: " + filename + "Tags:\n"
	for _, tag := range asset_tags {
		info_string = info_string + "\n" + tag
	}
	return
}

func init_folders() error {
	var err error
	if err = os.Mkdir(base, 0755); err != nil {
		return err
	}
	if err = os.Mkdir(assetsDir, 0755); err != nil {
		return err
	}
	if err = os.Mkdir(tagsDir, 0755); err != nil {
		return err
	}
	if err = os.Mkdir(metadataDir, 0755); err != nil {
		return err
	}
	return nil
}
