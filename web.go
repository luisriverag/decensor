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

func has_dot(some_string string) bool {
	for _, character := range some_string {
		if character == '.' {
			return true
		}
	}
	return false
}

func assetHTML(asset string, filename string, tags []string, activeTag string) (output string) {
	// If activeTag is permalink, behavior is slightly altered.
	output = fmt.Sprintf("<div class=\"card card-body\"><h5><a href=\"../asset/%s\">%s</a></h5><div class=\"mb-2\">", asset, filename)
	for _, tag := range tags {
		output += "<a class=\"btn btn-outline-secondary btn-sm"
		if tag == activeTag {
			output += " active"
		}
		output += fmt.Sprintf("\" href=\"../tag/%s\">%s</a>", tag, tag)
	}
	output += "<a class=\"btn btn-outline-danger btn-sm"
	if activeTag == "permalink" {
		output += " active"
	}
	output += fmt.Sprintf("\" href=\"../info/%s\">Permalink</a>", asset)
	if activeTag == "permalink" {
		output += fmt.Sprintf("</div><div class=\"small\">SHA256: <code>%s</code></div></div>\n", asset)
	} else {
		output += "</div></div>\n"
	}
	return
}

func getAsset(asset string) (filename string, tags []string) {
	if filename = asset_metadata_filename(asset); filename == "" {
		filename = asset
	}
	tags = tags_by_asset(asset)
	return
}

func asset_list_html(assets []string, active_tag string) (formatted_assets string) {
	// Set active_tag to "" if you don't want any tags highlighted.
	var filename string
	var tags []string
	formatted_assets = head_html(1)
	for _, asset := range assets {
		filename, tags = getAsset(asset)
		formatted_assets += assetHTML(asset, filename, tags, active_tag)
	}
	formatted_assets += footer_html
	return
}

func infoHTML(asset string) (output string) {
	filename, tags := getAsset(asset)
	output = head_html(1)

	mimeType := asset_mime_type(asset)
	if strings.HasPrefix(mimeType, "image/") {
		output += fmt.Sprintf("<img class=\"img-fluid\" src=\"../asset/%s\"/ alt=\"%s\">", asset, filename)
	} else if strings.HasPrefix(mimeType, "video/") {
		output += fmt.Sprintf("<video controls class=\"img-fluid\"><source src=\"../asset/%s\" /></video>", asset)
	} else if strings.HasPrefix(mimeType, "audio/") {
		output += fmt.Sprintf("<audio controls><source src=\"../asset/%s\" /></audio>", asset)
	}
	output += assetHTML(asset, filename, tags, "permalink")
	output += footer_html
	return
}

func linkOffset(negative_offset int) string {
	/* 0 is "" 1 is ../, 2 is "../../" */
	link_offset_string := ""
	for negative_offset != 0 {
		link_offset_string += "../"
		negative_offset -= 1
	}
	return link_offset_string
}

func head_html(link_negative_offset int) string {
	link_prefix := linkOffset(link_negative_offset)
	head_html_string := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<link href="%sasset/%s" rel="stylesheet" />
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Decensor</title>
</head>
<body>
<div class="container">
<header>
<div class="mt-2 mb-2">
<h1><a href="%s">Decensor</a></h1>
<p>Checksum-based file tracking and tagging</p>
<a class="btn btn-outline-primary" href="%sassets/">All Assets</a>
<a class="btn btn-outline-primary" href="%stags/">All Tags</a>
</div>
</header>
<article>
`, link_prefix, bootstrap_css_asset, link_prefix, link_prefix, link_prefix)
	return head_html_string
}

const footer_html = `</article></div>
</body>
</html>`

var index_html = head_html(0) + `
<p>
Decensor is written in <a target="_blank" href="https://golang.org/">Golang</a> and released into the <a target="_blank" href="https://unlicense.org/">public domain</a>. Source code is available on <a target="_blank" href="https://github.com/teran-mckinney/decensor">Github</a>.
</p>
` + footer_html

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
		defer s.NewTiming().Send("asset")
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
		if filename := asset_metadata_filename(asset); filename != "" {
			w.Header().Set("Content-Disposition", "inline; filename=\""+filename+"\"")
		}
		asset_path := assets_dir + "/" + asset
		http.ServeFile(w, r, asset_path)
	})

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("assets.hit")
		defer s.NewTiming().Send("assets")
		all_assets, err := assets()
		if err != nil {
			log.Print(err)
			http.Error(w, "Cannot return assets, please contact us.", 500)
			return
		}

		formatted_assets := asset_list_html(all_assets, "")
		_, err = io.WriteString(w, formatted_assets)
		if err != nil {
			// We don't need to http.Error because this means the connection was broken.
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("tags.hit")
		defer s.NewTiming().Send("tags")
		var formatted_tags string
		formatted_tags = head_html(1)
		all_tags, err := tags()
		if err != nil {
			log.Print(err)
			http.Error(w, "Cannot return tags, please contact us.", 500)
			return
		}
		for _, tag := range all_tags {
			assets, err := assets_by_tag(tag)
			if err != nil {
				http.Error(w, "Cannot return tags, please contact us.", 500)
				return
			}
			tag_asset_count := len(assets)
			formatted_tags += fmt.Sprintf("<div><a class=\"btn btn-outline-secondary\" href=\"../tag/%s\">%s <span class=\"badge badge-dark\">%d</span></a></div>\n", tag, tag, tag_asset_count)
		}
		formatted_tags += footer_html
		_, err = io.WriteString(w, formatted_tags)
		if err != nil {
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/tag/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("tag.hit")
		defer s.NewTiming().Send("tag")
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
		formatted_assets := asset_list_html(tag_assets, tag)
		_, err = io.WriteString(w, formatted_assets)
		if err != nil {
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/info/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("info.hit")
		defer s.NewTiming().Send("info")
		path_parts := strings.Split(r.URL.Path, "/")
		asset := path_parts[len(path_parts)-1]
		err = error_asset(asset)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), 400)
			return
		}

		info_html := infoHTML(asset)
		_, err = io.WriteString(w, info_html)
		if err != nil {
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("index.hit")
		defer s.NewTiming().Send("index")
		_, err := io.WriteString(w, index_html)
		if err != nil {
			log.Print(err)
			return
		}
	})
	log.Fatal(http.ListenAndServe(port, nil))
}
