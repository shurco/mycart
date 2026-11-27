package seo

import (
	"encoding/json"
	"fmt"
)

// FilterSpec removes paths from an OpenAPI spec that don't match the matcher.
func FilterSpec(spec []byte, m Matcher) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi: %w", err)
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		// No paths object; return as-is.
		return spec, nil
	}

	for p := range paths {
		if !m.Allow(p) {
			delete(paths, p)
		}
	}

	filtered, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal filtered spec: %w", err)
	}
	return filtered, nil
}
