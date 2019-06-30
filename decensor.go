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
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const decensor_path_suffix = "/.decensor"
const bootstrap_css_asset = "60b19e5da6a9234ff9220668a5ec1125c157a268513256188ee80f2d2c8d8d36"

var base = basedir()

var tags_dir = base + "/tags"
var assets_dir = base + "/assets"
var metadata_dir = base + "/metadata"

func IsHex(hexString string) bool {
	for _, character := range hexString {
		if !strings.Contains("abcdef01234567890", string(character)) {
			return false
		}
	}
	return true
}

func has_dot(some_string string) bool {
	for _, character := range some_string {
		if character == '.' {
			return true
		}
	}
	return false
}

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

func asset_list_html(assets []string) string {
	var formatted_assets string
	formatted_assets = head_html(1) + "<table class=\"table\"><thead><th>Asset</th><th>Tags</th></thead><tbody>"
	for _, asset := range assets {
		asset_name := asset
		filename, err := asset_metadata_filename(asset)
		if err == nil {
			asset_name = filename
		}
		formatted_assets += fmt.Sprintf("\n<tr><td><a href=\"../asset/%s\">%s</a></td><td>", asset, asset_name)
		asset_tags, err := tags_by_asset(asset)
		for _, tag := range asset_tags {
			formatted_assets += fmt.Sprintf("<a class=\"btn btn-outline-secondary btn-sm\" role=\"button\" href=\"../tag/%s\">%s</a>", tag, tag)
		}
		formatted_assets = formatted_assets + "</td></tr>"
	}
	formatted_assets = formatted_assets + "</tbody></table>" + footer_html
	return formatted_assets
}

func LinkOffset(negative_offset int) string {
	/* 0 is "" 1 is ../, 2 is "../../" */
	link_offset_string := ""
	for negative_offset != 0 {
		link_offset_string += "../"
		negative_offset -= 1
	}
	return link_offset_string
}

// Firefox (maybe other browsers) won't load CSS unless it has that Content-Type
// set, even if you say type="text/css" in the <link> tag. So we add a .css extension
// to make sure we serve it as CSS.
func head_html(link_negative_offset int) string {
	link_prefix := LinkOffset(link_negative_offset)
	head_html_string := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<link href="%sasset/%s.css" rel="stylesheet" />
<title>Decensor</title>
</head>
<body>
<div class="container">
<div class="jumbotron">
<h1><a href="%s">Decensor</a></h1>
<p>Checksum-based file tracking and tagging</p>
</div>`, link_prefix, bootstrap_css_asset, link_prefix)
	return head_html_string
}

const footer_html = `</div>
</body>
</html>`

var index_html = head_html(0) + `<ul>
<li><a href="tags/">Explore tags</a></li>
<li><a href="assets/">All assets</a></li>
</ul>` + footer_html

func web(port string) {
	http.HandleFunc("/asset/", func(w http.ResponseWriter, r *http.Request) {
		path_parts := strings.Split(r.URL.Path, "/")
		asset := path_parts[len(path_parts)-1]
		split_asset := strings.Split(asset, ".")
		var extension string
		switch len(split_asset) {
		case 2:
			asset = split_asset[0]
			extension = split_asset[1]
		case 1:
		default:
			http.Error(w, "Why on earth do you have multiple .'s???", 400)
			return
		}
		if validate_asset(asset) == false {
			http.Error(w, "asset must be 64 hex characters.", 400)
			return
		} else {
			if extension != "" {
				// TypeByExtension needs the extension to have the leading "."
				mime_type := mime.TypeByExtension("." + extension)
				if mime_type != "" {
					w.Header().Set("Content-Type", mime_type)
				} else {
					log.Print("Unknown mime type??")
				}
			}
			asset_path := assets_dir + "/" + asset
			http.ServeFile(w, r, asset_path)
		}
	})

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		formatted_assets := asset_list_html(assets())
		_, err := io.WriteString(w, formatted_assets)
		if err != nil {
			log.Print(err)
		}
	})

	http.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		var formatted_tags string
		formatted_tags = head_html(1) + "<ul>"
		for _, a_tag := range tags() {
			formatted_tags = formatted_tags + "<li><a href='../tag/" + a_tag + "'>" + a_tag + "</a></li>"
		}
		formatted_tags = formatted_tags + "</ul>" + footer_html
		_, err := io.WriteString(w, formatted_tags)
		if err != nil {
			log.Print(err)
		}
	})

	http.HandleFunc("/tag/", func(w http.ResponseWriter, r *http.Request) {
		path_parts := strings.Split(r.URL.Path, "/")
		tag := path_parts[len(path_parts)-1]
		if has_dot(tag) == true {
			http.Error(w, ".'s not allowed.", 400)
			return
		}
		formatted_assets := asset_list_html(assets_by_tag(tag))
		_, err := io.WriteString(w, formatted_assets)
		if err != nil {
			log.Print(err)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := io.WriteString(w, index_html)
		if err != nil {
			log.Print(err)
		}
	})
	log.Fatal(http.ListenAndServe(port, nil))
}

func get_hash(path string) (string, error) {
	var data []byte
	var hash_string string
	var err error
	data, err = ioutil.ReadFile(path)
	if err != nil {
		return hash_string, err
	}
	hash := sha256.Sum256(data)
	hash_string = hex.EncodeToString(hash[:])
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

func init_metadata(asset string) error {
	directory := metadata_dir + "/" + asset
	err := os.Mkdir(directory, 0755)
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

func asset_metadata_filename(asset string) (string, error) {
	path := asset_filepath_metadata_filename(asset)
	filename_byte, err := ioutil.ReadFile(path)
	filename := strings.Trim(string(filename_byte), "\n")
	return filename, err
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

func list_directory(directory string) []string {
	dir_entries, err := ioutil.ReadDir(directory)
	fatal_error(err)
	var entries []string
	for _, f := range dir_entries {
		entries = append(entries, f.Name())
	}
	return entries
}

func assets() []string {
	return list_directory(assets_dir)
}

func tags() []string {
	return list_directory(tags_dir)
}

func assets_by_tag(tag string) []string {
	return list_directory(tags_dir + "/" + tag)
}

func tags_by_asset(asset string) ([]string, error) {
	var asset_tags []string
	var err error
	if err = error_asset(asset); err != nil {
		return asset_tags, err
	}
	for _, tag := range tags() {
		for _, possible_asset := range assets_by_tag(tag) {
			if asset == possible_asset {
				asset_tags = append(asset_tags, tag)
			}
		}
	}
	return asset_tags, err
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
	}
	return err
}

func validate_assets() error {
	var hash string
	var err error

	assets := assets()
	has_invalid := false
	for _, asset := range assets {
		hash, err = get_hash(assets_dir + "/" + asset)
		if err != nil {
			return err
		}
		if asset != hash {
			has_invalid = true
			log.Printf("%s does not match %s\n", hash, asset)
		}
	}
	if has_invalid == true {
		return errors.New("Not all assets are valid.")
	}
	return err
}

func info(asset string) (string, error) {
	var err error
	var filename string
	filename, err = asset_metadata_filename(asset)
	if err != nil {
		return filename, err
	}
	var asset_tags []string
	asset_tags, err = tags_by_asset(asset)
	if err != nil {
		return filename, err
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
	case "validate_assets":
		exactly_arguments(2)
		fatal_error(validate_assets())
	case "tags":
		exactly_arguments(2)
		print_list(tags())
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
		print_list(assets())
	case "assets_by_tag":
		exactly_arguments(3)
		print_list(assets_by_tag(os.Args[2]))
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
