package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// TestFiber_RegistersAllMiddleware ensures Fiber() wires CORS, helmet,
// compression, request logging and the recoverer without panicking and that
// a handler registered after the middleware chain still runs.
func TestFiber_RegistersAllMiddleware(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	log := zerolog.Nop()
	Fiber(app, &log)

	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	// Helmet sets a couple of well-known security headers — probe one to
	// confirm the middleware actually executed, without over-specifying
	// which exact hardening we rely on.
	if resp.Header.Get("X-Content-Type-Options") == "" {
		t.Error("helmet middleware did not set X-Content-Type-Options")
	}
}

func TestFiber_RecoversFromPanic(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	log := zerolog.Nop()
	Fiber(app, &log)

	app.Get("/boom", func(c fiber.Ctx) error {
		panic("synthetic panic")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	// Without the recoverer the test would never return; assert that the
	// panic was caught and mapped to a 5xx instead of crashing the process.
	if resp.StatusCode < 500 {
		t.Errorf("expected 5xx from recovered panic, got %d", resp.StatusCode)
	}
}
