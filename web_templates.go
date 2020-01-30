package main

const headHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
<link href="{{.LinkPrefix}}asset/{{.CSSAsset}}" rel="stylesheet" />
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Decensor</title>
</head>
<body>
<div class="container">
<header>
<div class="mt-2 mb-2">
<h1><a href="{{.LinkPrefix}}">Decensor</a></h1>
<p>Checksum-based file tracking and tagging</p>
<a class="btn btn-outline-primary" href="{{.LinkPrefix}}assets/">All Assets <span class="badge badge-dark">{{.AssetCount}}</span></a>
<a class="btn btn-outline-primary" href="{{.LinkPrefix}}tags/">All Tags <span class="badge badge-dark">{{.TagCount}}</span></a>
<a class="btn btn-outline-primary" href="{{.LinkPrefix}}mimes/">By File Type</a>
</div>
</header>
<article>
`

type headHTMLTemplateArgs struct {
	LinkPrefix string
	CSSAsset   string
	AssetCount int
	TagCount   int
}

const footerHTML = `</article></div>
</body>
</html>`

const indexHTMLTemplate = `
{{.Head}}
<p>
Decensor is written in <a target="_blank" href="https://golang.org/">Golang</a> and released into the <a target="_blank" href="asset/{{.LicenseAsset}}">public domain</a>. Source code is available on <a target="_blank" href="https://github.com/teran-mckinney/decensor">Github</a>.
</p>
{{.Footer}}
`

type indexHTMLTemplateArgs struct {
	Head         string
	Footer       string
	LicenseAsset string
}

const assetHTMLTemplate = `
<div class="card card-body"><h5><a href="../asset/{{.Asset}}">{{.Filename}}</a></h5><div class="mb-2">
{{range $tag := .Tags}}
<a class="btn btn-outline-secondary btn-sm{{if eq $.ActiveTag $tag}} active{{end}}" href="../tag/{{$tag}}">{{$tag}}</a>
{{end}}
<a class="btn btn-outline-danger btn-sm{{if eq .ActiveTag "permalink"}} active{{end}}" href="../info/{{.Asset}}">Permalink</a>
</div>
{{if eq .ActiveTag "permalink"}}
<div class="small">Size: <code>{{.Size}}</code> bytes</div><div class="small">SHA256: <code>{{.Asset}}</code></div><div class="small">Mime Type: <code>{{.MimeType}}</code></div>
{{end}}
</div>
`

type assetHTMLTemplateArgs struct {
	Asset     string
	Filename  string
	Tags      []string
	ActiveTag string
	Size      int64
	MimeType  string
}
