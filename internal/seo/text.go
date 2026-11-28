package seo

import (
	"fmt"

	"github.com/osteele/liquid"
)

var engine = liquid.NewEngine()

// Render executes a Liquid template with the given data bindings.
func Render(tpl string, data Data) (string, error) {
	out, err := engine.ParseAndRenderString(tpl, data.Bindings())
	if err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return out, nil
}
