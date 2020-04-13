// decensor
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
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
	var size int64
	var mimeType string
	// This is a performance optimization, maybe not ideal.
	if activeTag == "permalink" {
		size, err = getAssetSize(asset)
		if err != nil {
			return
		}
		mimeType = getAssetMimeType(asset)
	}
	tmpl, err := template.New("").Parse(assetHTMLTemplate)
	if err != nil {
		return
	}
	var renderedTemplate bytes.Buffer
	templateArgs := assetHTMLTemplateArgs{Asset: asset,
		Filename:  filename,
		Tags:      tags,
		ActiveTag: activeTag,
		Size:      size,
		MimeType:  mimeType}
	if err = tmpl.Execute(&renderedTemplate, templateArgs); err != nil {
		return
	}
	output = renderedTemplate.String()
	return
}

func assetListHTML(assets []string, activeTag string) (formatted_assets string, err error) {
	// Set activeTag to "" if you don't want any tags highlighted.
	var filename string
	var tags []string
	var html string
	formatted_assets, err = headHTML(1)
	if err != nil {
		return
	}
	for _, asset := range assets {
		filename = getAssetFilename(asset)
		tags = tags_by_asset(asset)
		html, err = assetHTML(asset, filename, tags, activeTag)
		if err != nil {
			return
		}
		formatted_assets += html
	}
	formatted_assets += footerHTML
	return
}

func infoHTML(asset string) (output string, err error) {
	filename := getAssetFilename(asset)
	tags := tags_by_asset(asset)
	output, err = headHTML(1)
	if err != nil {
		return
	}

	mimeType := getAssetMimeType(asset)
	if strings.HasPrefix(mimeType, "image/") {
		output += fmt.Sprintf("<img class=\"img-fluid\" src=\"../asset/%s\"/ alt=\"%s\">", asset, filename)
	} else if strings.HasPrefix(mimeType, "video/") {
		output += fmt.Sprintf("<video controls class=\"img-fluid\"><source src=\"../asset/%s\" /></video>", asset)
	} else if strings.HasPrefix(mimeType, "audio/") {
		output += fmt.Sprintf("<audio controls><source src=\"../asset/%s\" /><a target=\"blank\" href=\"../asset/%s\">Download</a></audio>", asset, asset)
	}
	html, err := assetHTML(asset, filename, tags, "permalink")
	if err != err {
		return
	}
	output += html
	output += footerHTML
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
		Footer:       footerHTML,
		LicenseAsset: licenseAsset}
	if err = tmpl.Execute(&renderedTemplate, templateArgs); err != nil {
		return
	}
	output = renderedTemplate.String()
	return
}

func web(port string) {
	var err error

	/* Golang on Linux does not support setUid/setGid: https://github.com/golang/go/issues/1435 */
	/* chroot() without setuid() can be escaped and is mostly useless.                          */
	/* Non-Linux systems like FreeBSD are fine, however.                                        */
	if isUser(Root) {
		log.Print("We are root, using chroot() and setuid(65534) to sandbox decensor.")
		dir := baseDir()
		if err = syscall.Chroot(dir); err != nil {
			log.Fatal("We are root but unable to chroot() to ", dir, ": ", err.Error())
		}
		if err = os.Chdir("/"); err != nil {
			log.Fatal("Unable to chdir() to / after chroot().")
		}
		inChroot = true
		// 65534 is the nobody user on most systems (BSD and Linux).
		err = syscall.Setgid(Nobody)
		if err == syscall.EOPNOTSUPP {
			log.Print("setgid() not supported on Linux. Sandboxing ineffective: ", err.Error())
		} else if err != nil {
			log.Fatal(err.Error())
		}
		err = syscall.Setuid(Nobody)
		if err == syscall.EOPNOTSUPP {
			log.Print("setuid() not supported on Linux. Sandboxing ineffective: ", err.Error())
		} else if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		log.Print("We are not root, unable to chroot().")
	}

	/* Statsd statistics. This works fine with or without. */
	s, err := statsd.New(statsd.Prefix("decensor"))
	if err != nil {
		// This should be non-fatal.
		log.Print("decensor connection to statsd failed (This is not a problem unless you want statsd): ", err.Error())
	} else {
		log.Print("decensor connected to statsd.")
	}
	defer s.Close()

	http.HandleFunc("/asset/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("asset.hit")
		defer s.NewTiming().Send("asset")
		pathParts := strings.Split(r.URL.Path, "/")
		asset := pathParts[len(pathParts)-1]
		err = validateAsset(asset)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mimeType := getAssetMimeType(asset); mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}
		if filename := getAssetFilename(asset); filename != asset {
			w.Header().Set("Content-Disposition", "inline; filename=\""+filename+"\"")
		}
		http.ServeFile(w, r, getAssetPath(asset))
	})

	http.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("assets.hit")
		defer s.NewTiming().Send("assets")
		all_assets, err := assets()
		if err != nil {
			httpHandle500(w, err)
			return
		}

		formatted_assets, err := assetListHTML(all_assets, "")
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
		formatted_tags += footerHTML
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
		formatted_assets, err := assetListHTML(tag_assets, tag)
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

	http.HandleFunc("/mime/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("mime.hit")
		defer s.NewTiming().Send("mime")
		httpMimeType(w, r)
	})

	http.HandleFunc("/mimes/", func(w http.ResponseWriter, r *http.Request) {
		s.Increment("mimes.hit")
		defer s.NewTiming().Send("mimes")
		httpMimeTypes(w, r)
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
		if r.URL.Path != "/" {
			http.Error(w, "Decensor endpoint does not exist.", http.StatusNotFound)
			return
		}
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
