// Package docs exposes the canonical documentation bundled with gnosis.
package docs

import (
	"embed"
	"io/fs"
)

//go:embed concepts/*.md gnosis/processes/*/*.md
var content embed.FS

// Content returns the documentation included in the gnosis binary.
func Content() fs.FS {
	return content
}
