package web

import (
	"embed"
	"io/fs"
)

// Files contains the production browser application. A tracked placeholder
// keeps ordinary Go tooling usable before the frontend has been built.
//
//go:embed public
var files embed.FS

func Files() (fs.FS, error) {
	return fs.Sub(files, "public")
}
