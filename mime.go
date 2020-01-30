package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

func getAssetMimeType(asset string) (mimeType string) {
	// Try to get the mime type from the filename if we have a filename.
	// Not all files, like CSS, can get the mime type from magic bytes.
	// This does not return a mime type from magic bytes if we don't have
	// a filename or can't detect it from the extension alone.
	filename := getAssetFilename(asset)
	extension := filepath.Ext(filename)
	if extension == ".md" {
		// Return Markdown as text/plain so the browser previews it
		// rather than prompting for a download. This may change in
		// the future.
		mimeType = "text/plain"
	} else {
		mimeType = mime.TypeByExtension(filepath.Ext(filename))
	}
	return
}

func getMimeMajor(mime string) string {
	return strings.Split(mime, "/")[0]
}

func getAssetsByMimeTypeMajor(mimeType string) (assetsOutput []string, err error) {
	// Returns asset by major type from mimetype (so video, text, not video/mpeg or text/markdown)
	assets, err := assets()
	if err != nil {
		return
	}
	for _, asset := range assets {
		assetMimeType := getAssetMimeType(asset)
		mimeMajor := getMimeMajor(assetMimeType)
		if mimeMajor != "" {
			if mimeMajor == mimeType {
				assetsOutput = append(assetsOutput, asset)
			}
		}
	}
	return
}

func getMimeTypes() (mimeTypes map[string]uint64, err error) {
	mimeTypes = make(map[string]uint64)
	// Returns major mime types.
	assets, err := assets()
	if err != nil {
		return
	}
	for _, asset := range assets {
		assetMimeType := getAssetMimeType(asset)
		mimeMajor := getMimeMajor(assetMimeType)
		if mimeMajor != "" {
			mimeTypes[mimeMajor] += 1
		}
	}
	return
}

func httpMimeType(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	mimeType := pathParts[len(pathParts)-1]
	mimeAssets, err := getAssetsByMimeTypeMajor(mimeType)
	if err != nil {
		httpHandle500(w, err)
		return
	}
	if len(mimeAssets) == 0 {
		log.Print(err)
		http.Error(w, "No such assets under that mime type found.", http.StatusNotFound)
		return
	}
	formatted_assets, err := assetListHTML(mimeAssets, "")
	if err != nil {
		httpHandle500(w, err)
		return
	}
	_, err = io.WriteString(w, formatted_assets)
	if err != nil {
		log.Print(err)
		return
	}
}

func httpMimeTypes(w http.ResponseWriter, r *http.Request) {
	output, err := headHTML(1)
	if err != nil {
		httpHandle500(w, err)
		return
	}
	allMimes, err := getMimeTypes()
	if err != nil {
		httpHandle500(w, err)
		return
	}
	// If we don't sort this, output is very unstable in terms of order.
	var keys []string
	for key := range allMimes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		output += fmt.Sprintf("<div><a class=\"btn btn-outline-secondary\" href=\"../mime/%s\">%s/* <span class=\"badge badge-dark\">%d</span></a></div>\n", key, key, allMimes[key])
	}
	output += footerHTML
	_, err = io.WriteString(w, output)
	if err != nil {
		log.Print(err)
		return
	}
}
