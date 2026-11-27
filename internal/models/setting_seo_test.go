package models

import (
	"testing"
	"time"
)

func TestLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single line", "/products", []string{"/products"}},
		{"multiple lines", "/a\n/b\n/c", []string{"/a", "/b", "/c"}},
		{"comma separated", "/a,/b,/c", []string{"/a", "/b", "/c"}},
		{"mixed", "/a\n/b,/c", []string{"/a", "/b", "/c"}},
		{"with blanks", "/a\n\n/b\n  \n/c", []string{"/a", "/b", "/c"}},
		{"with comments", "/a\n# comment\n/b", []string{"/a", "/b"}},
		{"comment first", "# header\n/a\n/b", []string{"/a", "/b"}},
		{"spaces trimmed", "  /a  \n  /b  ", []string{"/a", "/b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lines(tt.in)
			if len(got) != len(tt.want) {
				t.Errorf("Lines() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Lines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidGlobs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty", "", false},
		{"single valid", "/products", false},
		{"wildcard", "/products/*", false},
		{"multiple", "/a\n/b/*\n/c", false},
		{"no leading slash", "products", true},
		{"bad glob", "/products/[", true},
		{"mixed valid/invalid", "/a\nproducts", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validGlobs(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("validGlobs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTTL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", "30s", 30 * time.Second, false},
		{"minutes", "10m", 10 * time.Minute, false},
		{"hours", "24h", 24 * time.Hour, false},
		{"days", "10d", 10 * 24 * time.Hour, false},
		{"one day", "1d", 24 * time.Hour, false},
		{"zero", "0s", 0, false},
		{"negative", "-10s", 0, true},
		{"negative days", "-1d", 0, true},
		{"bad format", "10x", 0, true},
		{"empty", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTTL(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTTL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseTTL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCDNRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    []CDNRule
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"single", "/products/*|24h", []CDNRule{{"/products/", 24 * time.Hour}}, false},
		{"multiple", "/a|1h\n/b|2h", []CDNRule{{"/a", 1 * time.Hour}, {"/b", 2 * time.Hour}}, false},
		{"days", "/sitemap.xml|10d", []CDNRule{{"/sitemap.xml", 10 * 24 * time.Hour}}, false},
		{"sorted longest first", "/a|1h\n/a/b/c|2h\n/a/b|3h", []CDNRule{{"/a/b/c", 2 * time.Hour}, {"/a/b", 3 * time.Hour}, {"/a", 1 * time.Hour}}, false},
		{"wildcard trimmed", "/products/*|1h", []CDNRule{{"/products/", 1 * time.Hour}}, false},
		{"no slash", "products|1h", nil, true},
		{"no pipe", "/products", nil, true},
		{"bad duration", "/products|bad", nil, true},
		{"negative duration", "/products|-1h", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCDNRules(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCDNRules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseCDNRules() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i].Prefix != tt.want[i].Prefix || got[i].TTL != tt.want[i].TTL {
					t.Errorf("ParseCDNRules()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSEOValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		seo     SEO
		wantErr bool
	}{
		{"empty valid", SEO{}, false},
		{"valid sitemap rules", SEO{SitemapInclude: "/", SitemapExclude: "/admin"}, false},
		{"valid cdn", SEO{CDNRules: "/products/*|24h"}, false},
		{"invalid sitemap include", SEO{SitemapInclude: "no-slash"}, true},
		{"invalid cdn rule", SEO{CDNRules: "/products"}, true},
		{"template too large", SEO{LlmsTemplate: string(make([]byte, maxSEOText+1))}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.seo.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SEO.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
