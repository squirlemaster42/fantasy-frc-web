package assets

import (
	"embed"
	"io/fs"
)

//go:embed css/*
var cssFS embed.FS

//go:embed img/*
var imgFS embed.FS

func CSS() fs.FS {
	sub, _ := fs.Sub(cssFS, "css")
	return sub
}

func Img() fs.FS {
	sub, _ := fs.Sub(imgFS, "img")
	return sub
}
