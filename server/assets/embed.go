package assets

import (
	"embed"
	"io/fs"
)

//go:embed css/*
var cssFS embed.FS

func CSS() fs.FS {
	sub, _ := fs.Sub(cssFS, "css")
	return sub
}
