package webutil

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// runResponseHandler wires a minimal Fiber app that invokes the provided
// handler and returns the decoded body + status.
func runResponseHandler(t *testing.T, h fiber.Handler) (int, []byte) {
	t.Helper()
	app := fiber.New()
	app.Get("/", h)

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func TestResponse_WithMessageProducesEnvelope(t *testing.T) {
	t.Parallel()

	status, body := runResponseHandler(t, func(c fiber.Ctx) error {
		return Response(c, fiber.StatusOK, "hello", map[string]string{"k": "v"})
	})

	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	var got HTTPResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.Success || got.Message != "hello" {
		t.Errorf("envelope fields wrong: %+v", got)
	}
}

func TestResponse_EmptyMessageReturnsRawData(t *testing.T) {
	t.Parallel()

	_, body := runResponseHandler(t, func(c fiber.Ctx) error {
		return Response(c, fiber.StatusOK, "", map[string]string{"raw": "yes"})
	})
	// When message is empty the handler returns the plain `data` payload,
	// not the envelope. That's used by a few endpoints that stream models
	// directly.
	var got map[string]string
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if got["raw"] != "yes" {
		t.Errorf("raw response lost field: %+v", got)
	}
}

func TestResponse_NonOKSetsSuccessFalse(t *testing.T) {
	t.Parallel()

	status, body := runResponseHandler(t, func(c fiber.Ctx) error {
		return Response(c, fiber.StatusInternalServerError, "nope", nil)
	})
	if status != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want 500", status)
	}
	var got HTTPResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Success {
		t.Error("success must be false for non-200 status")
	}
}

func TestStatusHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    fiber.Handler
		wantStatus int
		wantMsg    string
	}{
		{
			"StatusOK",
			func(c fiber.Ctx) error { return StatusOK(c, "ok", nil) },
			fiber.StatusOK,
			"ok",
		},
		{
			"StatusNotFound",
			StatusNotFound,
			fiber.StatusNotFound,
			"Not Found",
		},
		{
			"StatusInternalServerError",
			StatusInternalServerError,
			fiber.StatusInternalServerError,
			"Internal Server Error",
		},
		{
			"StatusBadRequest",
			func(c fiber.Ctx) error { return StatusBadRequest(c, "bad input") },
			fiber.StatusBadRequest,
			"Bad Request",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, body := runResponseHandler(t, tc.handler)
			if status != tc.wantStatus {
				t.Errorf("status = %d, want %d", status, tc.wantStatus)
			}
			var got HTTPResponse
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Message != tc.wantMsg {
				t.Errorf("message = %q, want %q", got.Message, tc.wantMsg)
			}
		})
	}
}
