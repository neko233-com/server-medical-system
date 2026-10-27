// Package web embeds the compiled single-page application (SPA) into the Go
// binary so that a release artefact is a **single self-contained executable**.
//
// Why this exists
// ---------------
// Production deployments install only the server binary (see install.sh /
// install.ps1). If the SPA were served from `./web/dist` on disk, a real
// deployment would boot successfully but render a blank page — the classic
// "API works, UI is 404" failure. Embedding removes the runtime dependency
// on a sibling directory entirely.
//
// Build contract
// --------------
// The frontend must be built **before** the Go binary:
//
//	cd web && npm ci && npm run build   # produces web/dist
//	go build ./cmd/server                 # embeds web/dist
//
// A committed placeholder (web/dist/.gitkeep) guarantees that `go build`
// still succeeds on a fresh clone where the frontend has not been built yet.
// In that case SPA() reports "not available" and the server falls back to
// serving ./web/dist from disk if it happens to exist.
package web

import (
	"embed"
	"io/fs"
)

// embedded holds the built SPA. The `all:` prefix keeps dot-prefixed files
// (e.g. .vite manifest) that some asset pipelines emit.
//
//go:embed all:dist
var embedded embed.FS

// SPA returns the embedded single-page application rooted at the dist folder.
//
// The second return value reports whether a usable SPA is present: it is
// false when the binary was compiled without a frontend build (fresh clone,
// API-only deployment). Callers should then fall back to on-disk serving or
// simply expose the API alone.
func SPA() (fs.FS, bool) {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, false
	}
	// index.html is the SPA entry point; without it there is nothing to serve.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
