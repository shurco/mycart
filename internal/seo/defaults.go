package seo

import (
	"embed"
	"strings"
)

//go:embed seo_defaults/*.liquid
var defaultsFS embed.FS

// Default returns the built-in template for file (robots, llms, ai).
func Default(file string) string {
	data, err := defaultsFS.ReadFile("seo_defaults/" + file + ".txt.liquid")
	if err != nil {
		return "" // file not found; caller falls back to empty
	}
	return strings.TrimSpace(string(data))
}
