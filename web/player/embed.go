// Package player embeds the built listener single-page application. The
// embedded dist directory only contains a placeholder until `make web` builds
// the real frontend.
package player

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the embedded player build, rooted at dist.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
