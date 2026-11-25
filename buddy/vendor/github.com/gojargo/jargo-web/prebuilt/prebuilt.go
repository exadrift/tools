// Package prebuilt serves a ready-made browser client for jargo bots: the
// pipecat-ai-prebuilt client, a development and debugging UI that connects to a
// bot over WebRTC and the other transports the jargo runner serves.
//
// The client is upstream's published build, embedded unmodified. Version names
// the release it comes from; scripts/update-prebuilt.sh replaces it with
// another one.
//
// The jargo runner serves it at /client:
//
//	runner.Frontend = prebuilt.Handler()
package prebuilt

import (
	"embed"
	"io/fs"
	"net/http"
)

// Version is the pipecat-ai-prebuilt release embedded here.
const Version = "1.3.0"

//go:embed all:dist
var dist embed.FS

// Handler serves the client's files, with index.html for a directory, so it
// can be mounted under a prefix with http.StripPrefix.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		// The embedded tree always has dist at its root.
		panic(err)
	}
	return http.FileServerFS(sub)
}
