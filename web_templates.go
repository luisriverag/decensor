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

const footer_html = `</article></div>
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
