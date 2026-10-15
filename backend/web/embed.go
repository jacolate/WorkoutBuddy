// Package web bettet den Frontend-Build (frontend → backend/web/dist) ins Binary ein.
package web

import (
	"embed"
	"io/fs"
)

// "all:*" statt "all:dist": Das Muster trifft immer mindestens diese Datei, daher
// kompiliert das Paket auch ohne Frontend-Build (z. B. Go-Tests in CI). Ein Platzhalter
// in dist/ funktioniert nicht, weil adapter-static den Ordner bei jedem Build leert.
//
//go:embed all:*
var files embed.FS

// Dist liefert den Inhalt von dist/ als Dateisystem (leer, wenn nicht gebaut).
func Dist() fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
