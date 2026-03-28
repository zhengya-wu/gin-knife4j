package ginknife4j

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var embeddedStaticFiles embed.FS

func defaultStaticFS() fs.FS {
	sub, err := fs.Sub(embeddedStaticFiles, "static")
	if err != nil {
		return nil
	}
	return sub
}
