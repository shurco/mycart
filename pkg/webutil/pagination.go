package webutil

import "github.com/gofiber/fiber/v3"

// Pagination describes resolved list-endpoint parameters after clamping to
// the safe ranges documented below. It is intentionally a plain value type;
// handlers pass it by value to storage layers.
type Pagination struct {
	Page   int // 1-based page number, always >= 1
	Limit  int // items per page, clamped to [1, MaxLimit]
	Offset int // (Page - 1) * Limit
}

const (
	defaultPage  = 1
	defaultLimit = 20
	// MaxLimit caps the number of rows a single list request can ask for.
	// Chosen to keep query latencies bounded; bulk exports should use a
	// dedicated endpoint if/when that ever becomes a requirement.
	MaxLimit = 100
)

// ParsePagination extracts `page` and `limit` from the request query string
// and returns a fully-clamped Pagination value. Invalid or missing inputs
// fall back to the defaults instead of returning errors — callers are
// list-handlers that should never 400 just because a client passed
// `?page=0` or `?limit=5000`.
func ParsePagination(c fiber.Ctx) Pagination {
	page := fiber.Query[int](c, "page", defaultPage)
	limit := fiber.Query[int](c, "limit", defaultLimit)

	if page < 1 {
		page = defaultPage
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return Pagination{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}
