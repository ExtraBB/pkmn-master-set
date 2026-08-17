// Package web holds the embedded templates and static assets.
package web

import "embed"

//go:embed templates/*.html
var Templates embed.FS

//go:embed static
var Static embed.FS
