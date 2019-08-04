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

	"gopkg.in/alexcesaro/statsd.v2"
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
		filename := asset_metadata_filename(asset)
		if filename != "" {
			asset_name = filename
		}
		formatted_assets += fmt.Sprintf("\n<tr><td><a href=\"../asset/%s\">%s</a></td><td>", asset, asset_name)
		asset_tags, err := tags_by_asset(asset)
		if err != nil {
			log.Print(err)
		}
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

func head_html(link_negative_offset int) string {
	link_prefix := LinkOffset(link_negative_offset)
	head_html_string := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<link href="%sasset/%s" rel="stylesheet" />
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1">
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

func asset_mime_type(asset string) string {
	// Try to get the mime type from the filename if we have a filename.
	// Not all files, like CSS, can get the mime type from magic bytes.
	// This does not return a mime type from magic bytes if we don't have
	// a filename or can't detect it from the extension alone.
	filename := asset_metadata_filename(asset)
	extension := filepath.Ext(filename)
	if extension == ".md" {
		// Return Markdown as text/plain so the browser previews it
		// rather than prompting for a download. This may change in
		// the future.
		return "text/plain"
	}
	mime_type := mime.TypeByExtension(filepath.Ext(filename))
	return mime_type
}

func web(port string) {
	/* Statsd statistics. This works fine with or without. */
	s, err := statsd.New(statsd.Prefix("decensor"))
	if err != nil {
		log.Print("Decensor connection to statsd failed. This is not a problem unless you want statsd.")
		// This should be non-fatal.
		log.Print(err)
	}
	defer s.Close()

	http.HandleFunc("/asset/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("asset.hit")
		path_parts := strings.Split(r.URL.Path, "/")
		asset := path_parts[len(path_parts)-1]
		err = error_asset(asset)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), 400)
			return
		}
		mime_type := asset_mime_type(asset)
		if mime_type != "" {
			w.Header().Set("Content-Type", mime_type)
		} else {
			log.Printf("Unknown mime type for %s", asset)
		}
		asset_path := assets_dir + "/" + asset
		http.ServeFile(w, r, asset_path)
	})

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("assets.hit")
		assets_time := s.NewTiming()
		all_assets, err := assets()
		if err != nil {
			log.Print(err)
			http.Error(w, "Cannot return assets, please contact us.", 500)
			return
		}
		assets_time.Send("assets.tiime")

		asset_list_html_time := s.NewTiming()
		formatted_assets := asset_list_html(all_assets)
		asset_list_html_time.Send("asset_list_html.time")
		_, err = io.WriteString(w, formatted_assets)
		if err != nil {
			// We don't need to http.Error because this means the connection was broken.
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("tags.hit")
		var formatted_tags string
		formatted_tags = head_html(1) + "<ul>"
		all_tags, err := tags()
		if err != nil {
			log.Print(err)
			http.Error(w, "Cannot return tags, please contact us.", 500)
			return
		}
		for _, a_tag := range all_tags {
			formatted_tags = formatted_tags + "<li><a href='../tag/" + a_tag + "'>" + a_tag + "</a></li>"
		}
		formatted_tags = formatted_tags + "</ul>" + footer_html
		_, err = io.WriteString(w, formatted_tags)
		if err != nil {
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/tag/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("tag.hit")
		path_parts := strings.Split(r.URL.Path, "/")
		tag := path_parts[len(path_parts)-1]
		if has_dot(tag) == true {
			http.Error(w, ".'s not allowed.", 400)
			return
		}
		tag_assets, err := assets_by_tag(tag)
		if err != nil {
			log.Print(err)
			http.Error(w, "No such tag found.", 404)
			return
		}
		formatted_assets := asset_list_html(tag_assets)
		_, err = io.WriteString(w, formatted_assets)
		if err != nil {
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("index.hit")
		_, err := io.WriteString(w, index_html)
		if err != nil {
			log.Print(err)
			return
		}
	})
	log.Fatal(http.ListenAndServe(port, nil))
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
