package seo

import (
	"path"
	"strings"

	"github.com/shurco/mycart/internal/models"
)

// Matcher decides which paths to include in sitemaps/OpenAPI specs.
type Matcher struct {
	Include []string
	Exclude []string
}

// NewMatcher parses include/exclude rule lists.
func NewMatcher(include, exclude string) Matcher {
	return Matcher{
		Include: models.Lines(include),
		Exclude: models.Lines(exclude),
	}
}

// Allow reports whether p is included and not excluded.
// An empty include list includes all paths.
func (m Matcher) Allow(p string) bool {
	return (len(m.Include) == 0 || anyMatch(m.Include, p)) && !anyMatch(m.Exclude, p)
}

func anyMatch(rules []string, p string) bool {
	for _, r := range rules {
		// Fast prefix match for trailing-* rules.
		if strings.HasSuffix(r, "*") && !strings.ContainsAny(r[:len(r)-1], "*?[") {
			if strings.HasPrefix(p, r[:len(r)-1]) {
				return true
			}
			continue
		}
		// Full glob match.
		if ok, _ := path.Match(r, p); ok {
			return true
		}
	}
	return false
}
