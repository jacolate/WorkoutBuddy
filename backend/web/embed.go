// Package web bettet den Frontend-Build (frontend → backend/web/dist) ins Binary ein.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist liefert den Inhalt von dist/ als Dateisystem.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
