package webassets

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embedded embed.FS

func Dist() (fs.FS, error) {
	return fs.Sub(embedded, "dist")
}

func ReadIndex() ([]byte, error) {
	return fs.ReadFile(embedded, "dist/index.html")
}
