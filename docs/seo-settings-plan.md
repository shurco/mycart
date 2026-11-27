# Plan: SEO settings (sitemap, llms.txt, ai.txt, robots.txt, OpenAPI, CDN rules)

Branch: `feature/seo_settings` (based on upstream `main`, plain query layer, no sqlc twins).

## 0. Goal

Admin → **Settings → Main** gains four sections. Each section opens right-side
drawers, the same way **Payment → Display Settings → Number Formatting** does
(`web/admin/src/routes/settings/payment/+page.svelte` + `NumberFormatForm.svelte`).

| Section | Drawers (chips) | Public URLs served |
|---|---|---|
| Sitemap | URL rules (include/exclude), Preview | `/sitemap.xml` |
| Text files | llms.txt, ai.txt, robots.txt (template + preview) | `/llms.txt`, `/ai.txt`, `/.well-known/ai.txt`, `/robots.txt` |
| OpenAPI | Include/exclude paths | `/openapi.json`, `/swagger.json`, `/.well-known/openapi.json`, `/.well-known/swagger.json` |
| CDN | Cache rules | `Cache-Control` header on matching GET responses |

## 1. Decisions

1. **One setting group `seo`.** It holds only `string` and `bool` fields. Two
   things follow from that. `serializeSettingValue` / `parseSettingValue`
   already handle both types. `UpdateSettingByGroup` upserts, so missing rows
   are created on first save. Result: **no migration**, no new JSON types in
   `setting.go`.
2. **Empty template means built-in default.** Defaults live in Go
   (`//go:embed seo_defaults/*`). The admin GET for `seo` fills empty fields
   from them, so the drawer always shows a working template. "Reset to
   default" fetches them too.
3. **Template syntax `{{ product.name }}`** uses Liquid,
   `github.com/osteele/liquid`. It is the only new runtime dependency except
   gofiber-swagger. Go `text/template` would need `{{ .Product.Name }}`, which
   is not the syntax asked for. Liquid has no function calls into Go, so an
   admin-authored template cannot reach the process.
4. **Include/exclude lists are two multiline textareas**, one rule per line.
   The repo has no dual-list control, and the request allows textareas. A
   "Generate rules" button fills the include box from current products and
   pages.
5. **Rule syntax is a glob (`path.Match`)** plus a trailing `*` prefix form:
   `/products/*` matches every product, `/about` matches one page. Exclude
   wins over include.
6. **Previews render the unsaved drawer state.** `POST /api/_/seo/preview/:file`
   takes the `seo` body and returns the rendered text, so the admin sees the
   result before saving.
7. **CDN rules are cached in memory** (`atomic.Pointer`). The middleware runs on
   every request, so it must not hit the DB. The cache reloads on startup and
   after `PATCH /api/_/settings/seo`.
8. **Storefront pages live at `/{slug}`, not `/pages/{slug}`.** See
   `web/site/src/routes/[slug]`. The example rule `/pages/|10m` would match
   nothing. The defaults and help text use `/*`-style rules that match real
   routes.

Fixes to the pasted snippets:
- Fiber here is **v3.7.0**: handlers take `fiber.Ctx`, not `*fiber.Ctx`.
- Sitemap namespace must be `http://www.sitemaps.org/schemas/sitemap/0.9`, not `http://sitemaps.org`.
- The `robots.txt` `Sitemap:` line needs the full `…/sitemap.xml` URL.
- `time.ParseDuration` has no `d` unit, so `10d` needs its own parse step (task 6.1).

## 2. Data model

`internal/models/setting_seo.go`

```go
package models

import (
	"fmt"
	"path"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// maxSEOText caps each template and rule list; they are rendered per request.
const maxSEOText = 64 << 10

// SEO is the "seo" setting group: crawler-facing files and CDN cache rules.
// Every field is always sent, empty strings included (see Branding).
type SEO struct {
	SitemapEnabled bool   `json:"sitemap_enabled"`
	SitemapInclude string `json:"sitemap_include"` // one rule per line
	SitemapExclude string `json:"sitemap_exclude"`

	LlmsEnabled    bool   `json:"llms_enabled"`
	LlmsTemplate   string `json:"llms_template"`
	AiEnabled      bool   `json:"ai_enabled"`
	AiTemplate     string `json:"ai_template"`
	RobotsEnabled  bool   `json:"robots_enabled"`
	RobotsTemplate string `json:"robots_template"`

	OpenAPIEnabled bool   `json:"openapi_enabled"`
	OpenAPIInclude string `json:"openapi_include"`
	OpenAPIExclude string `json:"openapi_exclude"`

	CDNEnabled bool   `json:"cdn_enabled"`
	CDNRules   string `json:"cdn_rules"` // "/products/*|24h" per line or comma
}

func (v SEO) Validate() error {
	text := validation.Length(0, maxSEOText)
	return validation.ValidateStruct(&v,
		validation.Field(&v.SitemapInclude, text, validation.By(validGlobs)),
		validation.Field(&v.SitemapExclude, text, validation.By(validGlobs)),
		validation.Field(&v.LlmsTemplate, text),
		validation.Field(&v.AiTemplate, text),
		validation.Field(&v.RobotsTemplate, text),
		validation.Field(&v.OpenAPIInclude, text, validation.By(validGlobs)),
		validation.Field(&v.OpenAPIExclude, text, validation.By(validGlobs)),
		validation.Field(&v.CDNRules, text, validation.By(validCDNRules)),
	)
}

// Lines splits a rule list on newlines and commas, dropping blanks and # comments.
func Lines(s string) []string {
	var out []string
	for _, l := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == ',' }) {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

func validGlobs(value any) error {
	for _, g := range Lines(value.(string)) {
		if !strings.HasPrefix(g, "/") {
			return fmt.Errorf("rule %q must start with /", g)
		}
		if _, err := path.Match(g, "/"); err != nil {
			return fmt.Errorf("rule %q: %w", g, err)
		}
	}
	return nil
}
```

`validCDNRules` calls the CDN parser from task 6.1. Keep the parser in `models`
(next to `Lines`) so `internal/seo` imports `models`, not the reverse. That
avoids an import cycle.

`internal/queries/setting_fields.go`: add the case and the map:

```go
case *models.SEO:
	return seoFields(s)

// seoFields maps the keys of the SEO setting group to its fields.
func seoFields(s *models.SEO) map[string]any {
	return map[string]any{
		"seo_sitemap_enabled": &s.SitemapEnabled,
		"seo_sitemap_include": &s.SitemapInclude,
		"seo_sitemap_exclude": &s.SitemapExclude,
		"seo_llms_enabled":    &s.LlmsEnabled,
		"seo_llms_template":   &s.LlmsTemplate,
		"seo_ai_enabled":      &s.AiEnabled,
		"seo_ai_template":     &s.AiTemplate,
		"seo_robots_enabled":  &s.RobotsEnabled,
		"seo_robots_template": &s.RobotsTemplate,
		"seo_openapi_enabled": &s.OpenAPIEnabled,
		"seo_openapi_include": &s.OpenAPIInclude,
		"seo_openapi_exclude": &s.OpenAPIExclude,
		"seo_cdn_enabled":     &s.CDNEnabled,
		"seo_cdn_rules":       &s.CDNRules,
	}
}
```

`internal/handlers/private/setting_registry.go`: add one line:

```go
"seo": func() any { return &models.SEO{} },
```

Upgrade note: on an existing shop no `seo_*` row exists, so every `*_enabled`
reads `false`. Nothing changes until the admin turns a file on. The one
exception is robots.txt, covered in task 4.4.

## 3. Shared package `internal/seo`

One package owns rendering, so the public handlers and the admin preview share
one code path.

```
internal/seo/
  defaults.go        //go:embed seo_defaults/*.liquid; Default(file) string
  seo_defaults/llms.txt.liquid  ai.txt.liquid  robots.txt.liquid
  data.go            Data struct + Load(ctx, db, baseURL) (bindings for Liquid)
  match.go           Matcher{Include, Exclude}.Allow(path) bool
  sitemap.go         BuildSitemap(data, seo) ([]byte, error)
  text.go            Render(tpl, data) (string, error)
  openapi.go         FilterSpec(spec []byte, m Matcher) ([]byte, error)
  cdn.go             atomic rule store + Middleware
  *_test.go
```

### 3.1 Bindings (`data.go`)

| Variable | Source |
|---|---|
| `site.name`, `site.email`, `site.tagline` | `main_site_name`, `main_email`, `branding_tagline` |
| `site.url` | `https://` + `main.domain` if set, else `c.BaseURL()` |
| `products[]`: `name`, `slug`, `url`, `brief`, `description`, `updated` (YYYY-MM-DD) | active products |
| `pages[]`: `name`, `slug`, `url`, `updated` | active pages |
| `sitemap_url`, `now` | derived |

New query, kept portable: unix seconds through `Dialect`, booleans as `TRUE`.
Read `internal/database/AGENTS.md` before writing it.

```go
// internal/queries/seo.go
type SEOEntry struct {
	Name, Slug, Brief, Description string
	Updated                        int64 // unix seconds
}

// ActiveProductsForSEO lists what crawlers may see; capped so one request stays bounded.
func (q *SEOQueries) ActiveProductsForSEO(ctx context.Context, limit int) ([]SEOEntry, error) {
	query := fmt.Sprintf(`SELECT name, slug, brief, description,
		%s FROM product WHERE active = TRUE ORDER BY name LIMIT ?`,
		q.DB.Dialect().UnixEpoch("COALESCE(updated, created)")) // verify helper name in Dialect
	rows, err := q.DB.QueryContext(ctx, query, limit)
	// ... scan into []SEOEntry, wrap errors with fmt.Errorf("seo products: %w", err)
}
```

Verify that `product.description` exists (`brief` was added in
`20231129131044_brief.sql`). Write `ActivePagesForSEO` the same way, without
the description columns.

shortcut: limit 50,000 entries (one sitemap file's maximum). Upgrade to a sitemap index when a shop exceeds it.

### 3.2 Matcher (`match.go`)

```go
type Matcher struct{ Include, Exclude []string }

func NewMatcher(include, exclude string) Matcher {
	return Matcher{models.Lines(include), models.Lines(exclude)}
}

// Allow reports whether p is included and not excluded. An empty include list includes all.
func (m Matcher) Allow(p string) bool {
	return (len(m.Include) == 0 || anyMatch(m.Include, p)) && !anyMatch(m.Exclude, p)
}

func anyMatch(rules []string, p string) bool {
	for _, r := range rules {
		if strings.HasSuffix(r, "*") && !strings.ContainsAny(r[:len(r)-1], "*?[") {
			if strings.HasPrefix(p, r[:len(r)-1]) { // "/products/*" covers nested paths too
				return true
			}
			continue
		}
		if ok, _ := path.Match(r, p); ok {
			return true
		}
	}
	return false
}
```

### 3.3 Text render (`text.go`)

```go
var engine = liquid.NewEngine()

func Render(tpl string, data Data) (string, error) {
	out, err := engine.ParseAndRenderString(tpl, data.Bindings())
	if err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return out, nil
}
```

`Data.Bindings()` returns `map[string]any` with lowercase keys
(`site`, `products`, …). Each product is a `map[string]any`, so `{{ product.name }}` resolves.
Before coding, confirm the `osteele/liquid` API through ctx7.

## 4. Phase A: public endpoints

### Task 4.1 Routes

`internal/routes/site_routes.go`: register before the SPA `c.Use("/")`
(registration order decides precedence). Then add the paths to `skipPaths`.

```go
c.Get("/sitemap.xml", handlers.Sitemap)
c.Get("/robots.txt", handlers.SEOText("robots"))
c.Get("/llms.txt", handlers.SEOText("llms"))
c.Get("/ai.txt", handlers.SEOText("ai"))
c.Get("/.well-known/ai.txt", handlers.SEOText("ai"))
for _, p := range []string{"/openapi.json", "/swagger.json", "/.well-known/openapi.json", "/.well-known/swagger.json"} {
	c.Get(p, handlers.OpenAPISpec)
}
```

### Task 4.2 Sitemap handler (`internal/handlers/public/seo.go`)

```go
// internal/seo/sitemap.go
type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func BuildSitemap(d Data, s *models.SEO) ([]byte, error) {
	m := NewMatcher(s.SitemapInclude, s.SitemapExclude)
	set := urlSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	add := func(p, lastmod string) {
		if m.Allow(p) {
			set.URLs = append(set.URLs, sitemapURL{Loc: d.Site.URL + p, LastMod: lastmod})
		}
	}
	add("/", "")
	for _, p := range d.Products {
		add("/products/"+p.Slug, p.Updated)
	}
	for _, p := range d.Pages {
		add("/"+p.Slug, p.Updated)
	}
	body, err := xml.MarshalIndent(set, "", "  ") // escapes &, < in slugs
	if err != nil {
		return nil, fmt.Errorf("marshal sitemap: %w", err)
	}
	return append([]byte(xml.Header), body...), nil
}

// internal/handlers/public/seo.go
func Sitemap(c fiber.Ctx) error {
	s, d, err := loadSEO(c) // GetSettingByGroup(&models.SEO{}) + seo.Load
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if !s.SitemapEnabled {
		return c.SendStatus(fiber.StatusNotFound)
	}
	body, err := seo.BuildSitemap(d, s)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	c.Set(fiber.HeaderContentType, "application/xml; charset=utf-8")
	return c.Send(body)
}
```

Changefreq and priority are skipped: Google ignores both. Add them only if asked.

### Task 4.3 Text handler

```go
// seoTextFiles is the registry for the three template-driven files (AGENTS.md: registry, not switch).
var seoTextFiles = map[string]struct {
	mime     string
	enabled  func(*models.SEO) bool
	template func(*models.SEO) string
}{
	"llms":   {"text/markdown; charset=utf-8", func(s *models.SEO) bool { return s.LlmsEnabled }, func(s *models.SEO) string { return s.LlmsTemplate }},
	"ai":     {"text/plain; charset=utf-8", func(s *models.SEO) bool { return s.AiEnabled }, func(s *models.SEO) string { return s.AiTemplate }},
	"robots": {"text/plain; charset=utf-8", func(s *models.SEO) bool { return s.RobotsEnabled }, func(s *models.SEO) string { return s.RobotsTemplate }},
}

func SEOText(file string) fiber.Handler {
	f := seoTextFiles[file]
	return func(c fiber.Ctx) error {
		s, d, err := loadSEO(c)
		if err != nil {
			log.ErrorStack(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		tpl := f.template(s)
		if !f.enabled(s) {
			if file != "robots" {
				return c.SendStatus(fiber.StatusNotFound)
			}
			tpl = "" // robots always answers: see 4.4
		}
		if tpl == "" {
			tpl = seo.Default(file)
		}
		out, err := seo.Render(tpl, d)
		if err != nil {
			log.ErrorStack(err) // never echo template errors publicly
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		c.Set(fiber.HeaderContentType, f.mime)
		c.Set("X-Content-Type-Options", "nosniff")
		return c.SendString(out)
	}
}
```

### Task 4.4 robots.txt conflict

`web/site/static/robots.txt` (`User-agent: *` / `Disallow: /_`) is embedded in
the storefront build. The Go route wins once registered, but delete the static
file so there is one source of truth. When `robots_enabled` is false, serve the
built-in default. That default already contains `Disallow: /_`, so existing
shops keep their current protection.

### Task 4.5 Default templates (`internal/seo/seo_defaults/`)

`robots.txt.liquid`
```
User-agent: *
Disallow: /_
Disallow: /api/
Disallow: /cart/
Disallow: /account/

User-agent: GPTBot
Disallow: /

Sitemap: {{ site.url }}/sitemap.xml
```

`llms.txt.liquid`
```
# {{ site.name }}

> {{ site.tagline }}

## Products
{% for product in products %}
- [{{ product.name }}]({{ product.url }}): {{ product.brief }}
{% endfor %}

## Pages
{% for page in pages %}
- [{{ page.name }}]({{ page.url }})
{% endfor %}

## Contact

{{ site.email }}

## Last Updated

{{ now }}
```

`ai.txt.liquid`
```
# ai.txt for {{ site.name }}
# Policy on how AI systems may use this site's content.

User-Agent: *
Allow: /products/
Disallow: /_
Disallow: /api/
Disallow-Training: /

Contact: {{ site.email }}
```

ai.txt has no single standard. The template is a starting point the admin
edits, not a guarantee any crawler obeys it.

## 5. Phase B: admin API

### Task 5.1 Preview endpoint

`internal/routes/api_private_routes.go`, inside the JWT-protected group:

```go
private.Post("/seo/preview/:file", handlers.SEOPreview) // file ∈ sitemap.xml|llms.txt|ai.txt|robots.txt|openapi.json
```

```go
func SEOPreview(c fiber.Ctx) error {
	s := new(models.SEO)
	if err := c.Bind().Body(s); err != nil {
		return webutil.StatusBadRequest(c, err.Error())
	}
	if err := s.Validate(); err != nil {
		return webutil.StatusBadRequest(c, err)
	}
	// load Data, dispatch on c.Params("file") via the same seoTextFiles map plus sitemap/openapi.
	// The admin may see the Liquid error text (line + message); public endpoints never do.
	return webutil.Response(c, fiber.StatusOK, "preview", fiber.Map{"content": out})
}
```

### Task 5.2 Defaults endpoint

`GET /api/_/seo/defaults` returns `{llms, ai, robots, sitemap_include, openapi_exclude}`.
The drawers' "Reset to default" and "Generate rules" buttons call it.
`sitemap_include` is generated from live data: `/`, `/products/*`, plus one
`/{slug}` line per active page.

### Task 5.3 Reload CDN cache on save

`internal/handlers/private/setting.go` `UpdateSetting`, after a successful save:

```go
if settingKey == "seo" {
	seo.StoreCDN(model.(*models.SEO)) // reparse; rules were validated already
}
```

## 6. Phase C: CDN rules

### Task 6.1 Parser (`internal/models/setting_seo.go`)

```go
type CDNRule struct {
	Prefix string
	TTL    time.Duration
}

// ParseCDNRules reads "/products/|24h, /sitemap.xml|10d"; the longest prefix wins at lookup.
func ParseCDNRules(s string) ([]CDNRule, error) {
	var rules []CDNRule
	for _, l := range Lines(s) {
		prefix, dur, ok := strings.Cut(l, "|")
		prefix = strings.TrimSpace(prefix)
		if !ok || !strings.HasPrefix(prefix, "/") {
			return nil, fmt.Errorf("cdn rule %q: want /path|duration", l)
		}
		ttl, err := parseTTL(strings.TrimSpace(dur))
		if err != nil {
			return nil, fmt.Errorf("cdn rule %q: %w", l, err)
		}
		rules = append(rules, CDNRule{strings.TrimSuffix(prefix, "*"), ttl})
	}
	sort.Slice(rules, func(i, j int) bool { return len(rules[i].Prefix) > len(rules[j].Prefix) })
	return rules, nil
}

func validCDNRules(value any) error {
	_, err := ParseCDNRules(value.(string))
	return err
}

// parseTTL adds a "d" (day) unit on top of time.ParseDuration.
func parseTTL(s string) (time.Duration, error) {
	if n, ok := strings.CutSuffix(s, "d"); ok {
		days, err := strconv.Atoi(n)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("bad days %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("bad duration %q", s)
	}
	return d, nil
}
```

### Task 6.2 Middleware (`internal/seo/cdn.go`)

```go
// neverCache: per-user or admin responses; a CDN rule must not leak them across visitors.
var neverCache = []string{"/_", "/api/_", "/api/customer", "/api/cart", "/cart"}

var cdnRules atomic.Pointer[[]models.CDNRule]

// StoreCDN swaps the live rules; nil (or cdn disabled) turns the middleware off.
func StoreCDN(s *models.SEO) {
	if s == nil || !s.CDNEnabled {
		cdnRules.Store(nil)
		return
	}
	rules, err := models.ParseCDNRules(s.CDNRules)
	if err != nil {
		cdnRules.Store(nil) // validated on save, so only a hand-edited DB lands here
		return
	}
	cdnRules.Store(&rules)
}

func CDNMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()
		rules := cdnRules.Load()
		if err != nil || rules == nil || (c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead) ||
			c.Response().StatusCode() != fiber.StatusOK || hasAnyPrefix(c.Path(), neverCache) ||
			len(c.Response().Header.Peek(fiber.HeaderSetCookie)) > 0 {
			return err
		}
		for _, r := range *rules {
			if strings.HasPrefix(c.Path(), r.Prefix) {
				s := int(r.TTL.Seconds())
				// Edge keeps it for the rule's TTL; browsers revalidate after 60s at most.
				c.Set(fiber.HeaderCacheControl, fmt.Sprintf("public, max-age=%d, s-maxage=%d", min(s, 60), s))
				break
			}
		}
		return nil
	}
}
```

Register in `internal/app.go` `setupRoutes` before the route groups:
`app.Use(seo.CDNMiddleware())`. Load the rules once at startup from the DB
(`seo.StoreCDN(loadedSEO)`).

Skipped: per-CDN trusted proxy ranges (`CF-Connecting-IP`) and `Vary` tuning.
`TrustProxy` with loopback/private is already set in `setupFiberApp`. The
storefront API sends no CORS headers. Add both only when a real CDN deployment
needs them.

## 7. Phase D: OpenAPI (gofiber-swagger)

Facts about `github.com/TDiblik/gofiber-swagger` v1.0.7:
- It targets Fiber v3.
- Routes are documented through its `SwaggerRouter` wrapper:
  `router.Get(path string, docs *RouteInfo, handler any, handlers ...any) fiber.Router`.
- `Register(app *fiber.App, config Config) error` serves UI + spec at
  `Config.SwaggerPath` (default `/swagger`).
- There is **no getter for the spec**. With `CreateSwaggerFiles: true` it writes
  `swagger.json` to `SwaggerFilesPath`.
- It is unconfirmed whether routes added with plain `app.Get` appear in the spec.

### Task 7.0 Spike (do first, timebox it)

Write a test that registers one plain `app.Get` route and one wrapped route,
calls `Register`, and reads the generated `swagger.json`. The result decides
how much of 7.1 is needed.

### Task 7.1 Wrap routes

Change `ApiPublicRoutes` / `ApiPrivateRoutes` to build their groups from
`gofiberswagger.NewRouter(app)` / `NewRouterFromRouter(group)`. Pass `nil` docs
at first. Path parameters are detected automatically, and request/response
types can be added later route by route. `SwaggerRouter` has no `Use`, so
middleware groups (`middleware.AccountEnabled()`) stay on the underlying
`fiber.Router` (`router.Router.Use(...)`).

```go
cfg := gofiberswagger.DefaultConfig
cfg.SwaggerPath = "/_/swagger"                           // under the admin prefix
cfg.SwaggerFilesPath = filepath.Join(baseDir, "swagger") // lc_base/swagger, writable
cfg.Swagger.Info.Title = "myCart API"
app.Use("/_/swagger", middleware.JWTProtected())         // raw unfiltered spec is admin-only
if err := gofiberswagger.Register(app, cfg); err != nil {
	return fmt.Errorf("register openapi: %w", err)
}
```

`/_/` is the admin SPA prefix. Check that `AdminRoutes` / `InstallCheck` do not
swallow `/_/swagger` before the swagger handler runs. If they do, use `/api/_/swagger`.

### Task 7.2 Filtered public spec

```go
// internal/seo/openapi.go
func FilterSpec(spec []byte, m Matcher) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi: %w", err)
	}
	paths, _ := doc["paths"].(map[string]any)
	for p := range paths {
		if !m.Allow(p) {
			delete(paths, p)
		}
	}
	return json.Marshal(doc)
}
```

`OpenAPISpec` handler: 404 if disabled. Otherwise read
`lc_base/swagger/swagger.json` once into memory (`sync.Once`; the spec is
fixed per binary), filter it with the saved include/exclude rules, and serve it
as `application/json`. Default exclude: `/api/_/*`, `/api/install*`,
`/api/customer/*`, `/cart/*`.

Retire `docs/swagger` (swaggo, OpenAPI 2.0, dev-only) once the new spec covers
the same routes. Until then both exist. Note this in `docs/swagger/swagger-setup.md`.

**Fallback if the spike fails:** filter and serve the existing embedded
`docs/swagger/swagger.json` with the same `FilterSpec`. That needs no new
dependency and no route rewrite. Recommend it if 7.0 shows that plain routes
are invisible, because wrapping every route is the biggest diff in this plan.

## 8. Phase E: admin UI

Files in `web/admin/src/routes/settings/main/`:

| File | Purpose |
|---|---|
| `+page.svelte` | adds the 4 `Section`s, `Drawer` with `DrawerMode`, loads/saves `seo` group |
| `SitemapRulesForm.svelte` | toggle + include/exclude textareas + "Generate rules" |
| `PreviewPane.svelte` | readonly `<pre>` that POSTs to `/api/_/seo/preview/:file` (debounced) |
| `TextFileForm.svelte` | one component for llms/ai/robots: toggle, template textarea, variables help, Reset, preview |
| `SwaggerRulesForm.svelte` | toggle + include/exclude + preview of resulting paths |
| `CdnRulesForm.svelte` | toggle + rules textarea + parsed-rules table |
| `seo.ts` | `SEOSettings` type, `file → field` map |

### Task 8.1 Page wiring (mirrors payment page)

```svelte
<script lang="ts">
  import { Chip, ChipGroup, Drawer, Section } from '$lib/components'
  import { createDelayedReset } from '$lib/utils/delayedReset'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import type { SEOSettings, SEOTextFile } from './seo'

  type DrawerMode = 'sitemap-rules' | 'sitemap-preview' | SEOTextFile | 'swagger' | 'cdn'

  let seo = $state<SEOSettings | null>(null)
  let drawerOpen = $state(false)
  let drawerMode = $state<DrawerMode | null>(null)
  const resetDrawer = createDelayedReset(DRAWER_CLOSE_DELAY_MS)

  onMount(async () => {
    seo = await loadSettings<SEOSettings>('seo', emptySEO())
  })

  async function saveSEO() {
    await saveSettings('seo', seo, t('settings.seo.saved'))
  }
  function openDrawer(m: DrawerMode) { drawerMode = m; drawerOpen = true }
  function closeDrawer() { drawerOpen = false; resetDrawer(() => (drawerMode = null)) }
</script>

<Section title={t('settings.seo.sitemap')}>
  <ChipGroup>
    <Chip active={seo?.sitemap_enabled} onclick={() => openDrawer('sitemap-rules')}>{t('settings.seo.urlRules')}</Chip>
    <Chip onclick={() => openDrawer('sitemap-preview')}>{t('settings.preview')}</Chip>
  </ChipGroup>
</Section>
<Section title={t('settings.seo.textFiles')}>
  <ChipGroup>
    <Chip active={seo?.llms_enabled} onclick={() => openDrawer('llms')}>llms.txt</Chip>
    <Chip active={seo?.ai_enabled} onclick={() => openDrawer('ai')}>ai.txt</Chip>
    <Chip active={seo?.robots_enabled} onclick={() => openDrawer('robots')}>robots.txt</Chip>
  </ChipGroup>
</Section>
<!-- OpenAPI and CDN sections: same pattern -->

{#if drawerOpen && seo}
  <Drawer isOpen={drawerOpen} onclose={closeDrawer} maxWidth="710px">
    {#if drawerMode === 'sitemap-rules'}
      <SitemapRulesForm bind:seo onsubmit={saveSEO} onclose={closeDrawer} />
    {:else if drawerMode === 'llms' || drawerMode === 'ai' || drawerMode === 'robots'}
      <TextFileForm file={drawerMode} bind:seo onsubmit={saveSEO} onclose={closeDrawer} />
    {/if}
    <!-- … -->
  </Drawer>
{/if}
```

### Task 8.2 TextFileForm (core of 3 drawers)

```svelte
<script lang="ts">
  import { DrawerFooter, DrawerHeader, FormTextarea, FormToggle } from '$lib/components'
  import { loadData } from '$lib/utils/apiHelpers'
  import PreviewPane from './PreviewPane.svelte'
  import { TEXT_FIELDS, type SEOSettings, type SEOTextFile } from './seo'

  let { file, seo = $bindable(), onsubmit, onclose }:
    { file: SEOTextFile; seo: SEOSettings; onsubmit: () => void; onclose: () => void } = $props()

  const f = $derived(TEXT_FIELDS[file]) // { enabled: 'llms_enabled', template: 'llms_template', path: 'llms.txt' }

  async function resetDefault() {
    const d = await loadData('/api/_/seo/defaults')
    seo[f.template] = d[file]
  }
</script>

<DrawerHeader title={f.path} />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }}>
  <FormToggle id="{file}-enabled" bind:value={seo[f.enabled]} />
  <FormTextarea id="{file}-template" rows={16} bind:value={seo[f.template]} class="font-mono" />
  <p class="text-xs text-gray-500">
    {'{{ site.name }} {{ site.url }} {% for product in products %}{{ product.name }}{% endfor %}'}
  </p>
  <button type="button" onclick={resetDefault}>{t('settings.seo.resetDefault')}</button>
  <PreviewPane file={f.path} {seo} />
  <DrawerFooter {onclose} />
</form>
```

Check the exported names (`FormTextarea`, `FormToggle`) in
`$lib/components/index.ts`. The underlying files are `form/Textarea.svelte` and
`form/Toggle.svelte`.

`PreviewPane` shows the result as text inside `<pre>`, **never `{@html}`**. The
output is crawler text, so it needs no `sanitizeHTML()`. It debounces with
`SEARCH_DEBOUNCE_MS` and clears its timer in `onDestroy`.

### Task 8.3 i18n

Add `settings.seo.*` keys to every locale file under
`web/admin/src/lib/i18n/`: section titles, toggles, help text, saved message,
reset, generate rules, and the rule-syntax hint.

## 9. Ordered task list

| # | Task | Files | Done when |
|---|---|---|---|
| 1 | `models.SEO` + `Validate` + `Lines` + `ParseCDNRules` | `internal/models/setting_seo.go` | table tests: lines, globs, `24h`/`10d`/`10m`/bad input |
| 2 | Field map + registry entry | `setting_fields.go`, `setting_registry.go`, `setting_test.go` (add to known types) | GET/PATCH `/api/_/settings/seo` round-trip test |
| 3 | `SEOQueries` active products/pages | `internal/queries/seo.go` | test on SQLite and PostgreSQL |
| 4 | `internal/seo`: Data, Matcher, Render, defaults | `internal/seo/*` | table tests: glob, prefix, exclude-wins, Liquid vars |
| 5 | Sitemap builder + handler | `seo/sitemap.go`, `handlers/public/seo.go` | inactive product absent, excluded slug absent, `&` escaped, 404 when disabled |
| 6 | Text handlers + routes + drop static robots.txt | `site_routes.go`, `web/site/static/robots.txt` | content types correct, robots fallback, template error returns 500 without leaking it |
| 7 | Preview + defaults endpoints | `handlers/private/seo.go`, `api_private_routes.go` | 401 without JWT, 400 on bad rules |
| 8 | CDN middleware + startup load + reload hook | `seo/cdn.go`, `app.go`, `setting.go` | `/api/_` never gets `public`; Set-Cookie responses skipped; reload after PATCH |
| 9 | OpenAPI spike | test only | decision recorded |
| 10 | OpenAPI wrap + register + filtered aliases (or fallback) | routes, `app.go`, `seo/openapi.go` | 4 aliases serve the filtered spec; raw spec needs JWT |
| 11 | Admin UI: page + 5 drawer components | `web/admin/src/routes/settings/main/*` | `bun run build` + `bun run check` clean |
| 12 | i18n keys | `web/admin/src/lib/i18n/*` | no missing-key warnings |
| 13 | e2e: enable sitemap, add exclude, fetch `/sitemap.xml` | `e2e/` (`useAdminSession`, `startIsolatedServer`) | passes |
| 14 | Docs | `docs/` | each setting and the template variables documented |

Each task should be one commit. Tasks 1–8 deliver value without OpenAPI. Phase D can ship separately.

## 10. Verification

```bash
go vet ./... && go test ./... -count=1 -race
docker compose -f docker/docker-compose.yml -f docker/docker-compose_dev.yml up -d pgtestdb
TEST_DB_DRIVER=postgres \
TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5433/postgres?sslmode=disable' \
go test ./... -count=1 -race
cd web/admin && bun run build
go run ./cmd serve &
curl -s localhost:8080/sitemap.xml | xmllint --noout -   # well-formed
curl -sI localhost:8080/robots.txt | grep -i content-type
curl -sI localhost:8080/products/some-slug | grep -i cache-control
```

## 11. Risks

| Risk | Mitigation |
|---|---|
| A CDN rule caches a per-user response | hard `neverCache` list + skip responses with `Set-Cookie`; test |
| Template loops over many products slow every request | 50k cap + CDN rule on `/sitemap.xml`/`/llms.txt`; later an in-memory cache keyed on settings version |
| gofiber-swagger needs every route rewrapped, and its spec only reaches disk | spike first; fallback to the existing swaggo spec |
| Raw spec exposes admin endpoints | raw path behind JWT; public aliases filtered, `/api/_/*` excluded by default |
| robots.txt regression on upgrade | default template keeps `Disallow: /_` and is served even when disabled |
| Liquid error leaks on the public site | public handlers log + 500; only the admin preview shows the message |

## 12. Open questions

1. Should robots.txt stay always-on (current plan), or return 404 when disabled?
2. Should llms.txt also be served at `/.well-known/llms.txt`?
3. OpenAPI: accept the full route wrap, or use the swaggo fallback if the spike is ambiguous?
