// Package web embeds the browser client.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// Static is the client, rooted so index.html is served at /.
var Static, _ = fs.Sub(files, "static")
