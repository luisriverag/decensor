// decensor
package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"gopkg.in/alexcesaro/statsd.v2"
)

const bootstrap_css_asset = "60b19e5da6a9234ff9220668a5ec1125c157a268513256188ee80f2d2c8d8d36"

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
		log.Print("decensor connection to statsd failed. This is not a problem unless you want statsd.")
		// This should be non-fatal.
		log.Print(err)
	} else {
		log.Print("decensor connected to statsd.")
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
