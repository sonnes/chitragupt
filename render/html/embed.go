package html

import "embed"

//go:embed templates/*.html
var content embed.FS
