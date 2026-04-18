import DOMPurify from 'isomorphic-dompurify'

/**
 * Sanitize untrusted HTML before rendering via `{@html}`.
 * Mirrors site/src/lib/utils/sanitize.ts — duplicated because the two apps
 * ship as independent SvelteKit builds with no shared package.
 */
export function sanitizeHTML(html: string | null | undefined): string {
  if (!html) return ''
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target', 'rel'],
  })
}
