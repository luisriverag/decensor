// decensor
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/alexcesaro/statsd.v2"
)

const bootstrapCSSAsset = "60b19e5da6a9234ff9220668a5ec1125c157a268513256188ee80f2d2c8d8d36"
const licenseAsset = "88d9b4eb60579c191ec391ca04c16130572d7eedc4a86daa58bf28c6e14c9bcd"

func has_dot(some_string string) bool {
	for _, character := range some_string {
		if character == '.' {
			return true
		}
	}
	return false
}

func assetHTML(asset string, filename string, tags []string, activeTag string) (output string, err error) {
	var assetSize int64
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
		assetSize, err = getAssetSize(asset)
		if err != nil {
			return
		}
		output += fmt.Sprintf("</div><div class=\"small\">Size: <code>%d</code> bytes</div><div class=\"small\">SHA256: <code>%s</code></div></div>\n", assetSize, asset)
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

func asset_list_html(assets []string, active_tag string) (formatted_assets string, err error) {
	// Set active_tag to "" if you don't want any tags highlighted.
	var filename string
	var tags []string
	var html string
	formatted_assets, err = headHTML(1)
	if err != nil {
		return
	}
	for _, asset := range assets {
		filename, tags = getAsset(asset)
		html, err = assetHTML(asset, filename, tags, active_tag)
		if err != nil {
			return
		}
		formatted_assets += html
	}
	formatted_assets += footer_html
	return
}

func infoHTML(asset string) (output string, err error) {
	filename, tags := getAsset(asset)
	output, err = headHTML(1)
	if err != nil {
		return
	}

	mimeType := asset_mime_type(asset)
	if strings.HasPrefix(mimeType, "image/") {
		output += fmt.Sprintf("<img class=\"img-fluid\" src=\"../asset/%s\"/ alt=\"%s\">", asset, filename)
	} else if strings.HasPrefix(mimeType, "video/") {
		output += fmt.Sprintf("<video controls class=\"img-fluid\"><source src=\"../asset/%s\" /></video>", asset)
	} else if strings.HasPrefix(mimeType, "audio/") {
		output += fmt.Sprintf("<audio controls><source src=\"../asset/%s\" /></audio>", asset)
	}
	html, err := assetHTML(asset, filename, tags, "permalink")
	if err != err {
		return
	}
	output += html
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

func countTags() (count int, err error) {
	tags, err := tags()
	if err != nil {
		return
	}
	count = len(tags)
	return
}

func countAssets() (count int, err error) {
	assets, err := assets()
	if err != nil {
		return
	}
	count = len(assets)
	return
}

func headHTML(link_negative_offset int) (headHTML string, err error) {
	linkPrefix := linkOffset(link_negative_offset)
	tmpl, err := template.New("").Parse(headHTMLTemplate)
	if err != nil {
		return
	}
	tagCount, err := countTags()
	if err != nil {
		return
	}
	assetCount, err := countAssets()
	if err != nil {
		return
	}
	var renderedTemplate bytes.Buffer
	templateArgs := headHTMLTemplateArgs{LinkPrefix: linkPrefix,
		CSSAsset:   bootstrapCSSAsset,
		AssetCount: assetCount,
		TagCount:   tagCount}
	if err = tmpl.Execute(&renderedTemplate, templateArgs); err != nil {
		return
	}
	headHTML = renderedTemplate.String()
	return
}

func indexHTML() (output string, err error) {
	head, err := headHTML(0)
	if err != nil {
		return
	}
	tmpl, err := template.New("").Parse(indexHTMLTemplate)
	if err != nil {
		return
	}
	var renderedTemplate bytes.Buffer
	templateArgs := indexHTMLTemplateArgs{Head: head,
		Footer:       footer_html,
		LicenseAsset: licenseAsset}
	if err = tmpl.Execute(&renderedTemplate, templateArgs); err != nil {
		return
	}
	output = renderedTemplate.String()
	return
}

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
		err = validateAsset(asset)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
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
			httpHandle500(w, err)
			return
		}

		formatted_assets, err := asset_list_html(all_assets, "")
		if err != nil {
			httpHandle500(w, err)
			return
		}
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
		formatted_tags, err := headHTML(1)
		if err != nil {
			httpHandle500(w, err)
			return
		}
		all_tags, err := tags()
		if err != nil {
			httpHandle500(w, err)
			return
		}
		for _, tag := range all_tags {
			assets, err := assets_by_tag(tag)
			if err != nil {
				httpHandle500(w, err)
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
			http.Error(w, ".'s not allowed.", http.StatusBadRequest)
			return
		}
		tag_assets, err := assets_by_tag(tag)
		if err != nil {
			log.Print(err)
			http.Error(w, "No such tag found.", http.StatusNotFound)
			return
		}
		formatted_assets, err := asset_list_html(tag_assets, tag)
		if err != nil {
			httpHandle500(w, err)
			return
		}
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
		err = validateAsset(asset)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		info_html, err := infoHTML(asset)
		if err != nil {
			httpHandle500(w, err)
			return
		}
		_, err = io.WriteString(w, info_html)
		if err != nil {
			log.Print(err)
			return
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("index.hit")
		defer s.NewTiming().Send("index")
		index_html, err := indexHTML()
		if err != nil {
			httpHandle500(w, err)
			return
		}
		_, err = io.WriteString(w, index_html)
		if err != nil {
			log.Print(err)
			return
		}
	})

	go statsdLoop(s)

	log.Fatal(http.ListenAndServe(port, nil))
}

func httpHandle400(w http.ResponseWriter, err error) {
	log.Print(err.Error())
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func httpHandle500(w http.ResponseWriter, err error) {
	log.Print(err.Error())
	http.Error(w, "Something broke in Decensor. Please try again.", http.StatusInternalServerError)
}
