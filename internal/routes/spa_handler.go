package routes

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	indexHTML       = "index.html"
	contentTypeHTML = "text/html"
)

func setupSPAHandler(embedFS fs.FS, skipPaths func(string) bool, stripPrefix string) fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Path()

		if skipPaths(path) {
			return c.Next()
		}

		if stripPrefix != "" {
			path = strings.TrimPrefix(path, stripPrefix)
		}

		normalizedPath := normalizePath(path)

		file, err := embedFS.Open(normalizedPath)
		if err == nil {
			defer file.Close()

			stat, err := file.Stat()
			if err == nil && stat.IsDir() {
				file.Close()
				file, err = embedFS.Open(indexHTML)
				if err != nil {
					return c.Next()
				}
				defer file.Close()
				normalizedPath = indexHTML
			}

			ext := filepath.Ext(normalizedPath)
			contentType := getContentType(ext)
			c.Set("Content-Type", contentType)
			return c.SendStream(file)
		}

		if isStaticAsset(normalizedPath) {
			return c.Status(fiber.StatusNotFound).SendString("Not Found")
		}

		indexFile, err := embedFS.Open(indexHTML)
		if err != nil {
			return c.Next()
		}
		defer indexFile.Close()
		c.Set("Content-Type", contentTypeHTML)
		return c.SendStream(indexFile)
	}
}

func normalizePath(path string) string {
	if path == "" || path == "/" {
		return indexHTML
	}
	return strings.TrimPrefix(path, "/")
}

func isStaticAsset(path string) bool {
	staticExts := map[string]bool{
		".js": true, ".css": true, ".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".svg": true, ".ico": true, ".woff": true, ".woff2": true,
		".ttf": true, ".eot": true, ".json": true,
	}
	ext := strings.ToLower(filepath.Ext(path))
	return staticExts[ext]
}

func getContentType(ext string) string {
	ext = strings.ToLower(ext)
	contentTypes := map[string]string{
		".html":  "text/html",
		".js":    "application/javascript",
		".css":   "text/css",
		".json":  "application/json",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".eot":   "application/vnd.ms-fontobject",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
