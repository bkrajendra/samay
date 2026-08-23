// Package web embeds the built frontend assets into the server binary. The
// Docker build replaces web/static with the real Vite build output before
// compiling; outside that build, web/static/index.html is a placeholder so
// `go build` still succeeds locally.
package web

import "embed"

//go:embed static
var StaticFS embed.FS
