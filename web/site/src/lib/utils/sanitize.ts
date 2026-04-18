import DOMPurify from 'isomorphic-dompurify'

/**
 * Sanitize untrusted HTML before rendering via `{@html}`.
 *
 * Rationale: product descriptions and page contents are authored in a
 * WYSIWYG editor and stored as raw HTML. Rendering that HTML without
 * sanitisation exposes every reader to stored XSS if the admin account
 * is ever compromised or if an upstream import pipeline adds content.
 *
 * DOMPurify's default profile already strips `<script>`, event handlers
 * and `javascript:` URLs; we keep the defaults and only allow the
 * formatting tags and attributes that the TipTap editor actually emits.
 */
export function sanitizeHTML(html: string | null | undefined): string {
  if (!html) return ''
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
    ADD_ATTR: ['target', 'rel'],
  })
}
