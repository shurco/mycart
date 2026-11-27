# SEO Settings

Configure sitemap.xml, robots.txt, llms.txt, ai.txt, OpenAPI specs, and CDN cache rules.

## Overview

SEO settings control crawler-facing files and cache behavior:

- **Sitemap**: `/sitemap.xml` lists active products and pages
- **Text files**: `/robots.txt`, `/llms.txt`, `/ai.txt`, `/.well-known/{llms,ai}.txt`
- **OpenAPI**: `/openapi.json`, `/swagger.json`, `/.well-known/{openapi,swagger}.json`
- **CDN**: `Cache-Control` headers on matching GET responses

Admin: **Settings → Main → SEO**

## Sitemap

**Enabled**: Serves sitemap.xml with active products/pages.

**Include/Exclude Rules**: One glob per line. Prefix form (`/products/*`) matches all nested paths. Exclude wins.

Example:
```
Include:
/
/products/*
/about

Exclude:
/products/hidden
```

## Text Files (robots.txt, llms.txt, ai.txt)

**Template**: Liquid syntax with these variables:

| Variable | Value |
|----------|-------|
| `site.name` | Site name (main settings) |
| `site.url` | `https://` + domain |
| `site.email` | Contact email |
| `products[]` | Active products: `name`, `slug`, `url`, `brief`, `description`, `updated` (YYYY-MM-DD) |
| `pages[]` | Active pages: `name`, `slug`, `url`, `updated` |

Example:
```liquid
# {{ site.name }}

{% for product in products %}
- {{ product.name }}: {{ product.url }}
{% endfor %}
```

**Defaults**: Embedded in binary. "Reset to default" button fetches them.

**Disabled**: Returns 404 (not served).

## OpenAPI

**Enabled**: Serves filtered OpenAPI spec.

**Include/Exclude**: Path rules like sitemap. Filters spec paths.

Default exclude:
```
/api/_/*
/api/install*
/api/customer/*
/cart/*
```

**Endpoints**:
- `/openapi.json`, `/swagger.json`, `/.well-known/openapi.json`, `/.well-known/swagger.json`: Filtered public spec
- `/_/swagger`: Unfiltered admin UI

## CDN Cache

**Enabled**: Applies `Cache-Control` headers.

**Rules**: One rule per line, format: `/path|duration`

Duration units: `s` (seconds), `m` (minutes), `h` (hours), `d` (days)

Example:
```
/products/*|24h
/sitemap.xml|10d
/robots.txt|7d
```

Longest prefix wins. Never cached: `POST/PUT/DELETE`, errors, cart/checkout paths.

## Notes

- Changes apply immediately (CDN rules reload on save)
- Empty template = use default
- Preview shows rendered output before saving
- Glob patterns must start with `/`
