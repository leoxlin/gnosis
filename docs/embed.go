// Package docs exposes the canonical documentation bundled with gnosis.
package docs

import "embed"

//go:embed concepts/*.md procedures/*.md
var Content embed.FS
