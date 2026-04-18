# pkg/AGENTS.md

Library-style packages with stable(ish) APIs. Keep them import-cycle-free
and free of `internal/` dependencies.

## Packages

| Package | Responsibility |
|---------|----------------|
| `archive` | Zip/tar helpers for digital-product delivery. |
| `errors` | Wrapped error + stacktrace printer (`ErrorStack`). |
| `fsutil` | File/path helpers (`ExtName`, existence checks). |
| `httpclient` | **Shared HTTP client factory.** Use for any outbound call. |
| `jwtutil` | Parse/issue JWTs; enforces HMAC-only signing. |
| `litepay` | Payment-gateway adapters (Stripe, PayPal, Coinbase, SpectroCoin, Dummy). |
| `logging` | Structured logger built on stdlib `slog`. |
| `security` | Password hashing (bcrypt.DefaultCost) + token generation (bcrypt + SHA-256). |
| `strutil` | String helpers (slugs, case conversions). |
| `update` | GitHub Releases polling + 24 h cached version info. |
| `webutil` | Fiber response helpers + `ParsePagination`. |

## Rules

1. **Never** reach into `internal/` from `pkg/`. `pkg` packages must
   remain reusable.
2. **Outbound HTTP.** New integrations add a single shared client via
   `httpclient.New()` (see `pkg/litepay/client.go`). Do not create
   per-call `http.Client{}`.
3. **JWT.** `jwtutil` and any consumer must assert
   `*jwt.SigningMethodHMAC` inside `Keyfunc`. Rejecting "alg confusion"
   is non-negotiable.
4. **Crypto.** MD5 is banned for anything security-related. Passwords use
   `bcrypt.DefaultCost` (cost 10). Token generation is
   `bcrypt → sha256 → hex`.
5. **No silent timeouts.** `httpclient.DefaultTimeout` is 30 s because CI
   runners occasionally see slow TLS handshakes with payment sandboxes;
   override with `NewWithTimeout` if a shorter ceiling fits the use
   case.
6. **Examples vs tests.** `pkg/litepay` contains `Example*` functions
   that perform real network calls against public sandbox endpoints.
   They should fail gracefully on timeout rather than flake.
