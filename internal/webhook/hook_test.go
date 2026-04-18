package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/migrations"
	"github.com/shurco/mycart/pkg/litepay"
)

// setupTestDB migrates a fresh database inside a temp working directory so
// queries.DB() returns a usable instance for SendPaymentHook.
func setupTestDB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	_ = os.MkdirAll("lc_base", 0o775)
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if err := queries.New(migrations.Embed()); err != nil {
		t.Fatalf("queries.New: %v", err)
	}
}

func TestSendPaymentHook_EmptyURLIsNoop(t *testing.T) {
	setupTestDB(t)
	// Default webhook_url is empty — the hook must short-circuit with no error.
	err := SendPaymentHook(&Payment{Event: PAYMENT_INITIATION})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendPaymentHook_DeliversToConfiguredURL(t *testing.T) {
	setupTestDB(t)

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	db := queries.DB()
	if err := db.UpdateSettingByGroup(ctx, &models.Webhook{Url: srv.URL}); err != nil {
		t.Fatalf("seed webhook url: %v", err)
	}

	payload := &Payment{
		Event:     PAYMENT_SUCCESS,
		TimeStamp: time.Now().Unix(),
		Data: Data{
			CartID:        "cart-1",
			PaymentSystem: litepay.STRIPE,
			PaymentStatus: litepay.PAID,
			TotalAmount:   100,
			Currency:      "USD",
		},
	}
	if err := SendPaymentHook(payload); err != nil {
		t.Fatalf("SendPaymentHook: %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("expected 1 hit, got %d", hits.Load())
	}
}

func TestSendPaymentHook_Non2xxLogsAndReturnsNil(t *testing.T) {
	setupTestDB(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := queries.DB().UpdateSettingByGroup(ctx, &models.Webhook{Url: srv.URL}); err != nil {
		t.Fatalf("seed webhook url: %v", err)
	}

	// The function deliberately swallows errors to avoid blocking the
	// payment pipeline; the important contract is "no error bubbles up".
	if err := SendPaymentHook(&Payment{Event: PAYMENT_ERROR}); err != nil {
		t.Fatalf("SendPaymentHook: %v", err)
	}
}

func TestSendPaymentHook_TransportFailureSwallowed(t *testing.T) {
	setupTestDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := queries.DB().UpdateSettingByGroup(ctx, &models.Webhook{
		Url: "http://127.0.0.1:1/unreachable",
	}); err != nil {
		t.Fatalf("seed webhook url: %v", err)
	}
	if err := SendPaymentHook(&Payment{Event: PAYMENT_CANCEL}); err != nil {
		t.Fatalf("SendPaymentHook should not surface transport errors: %v", err)
	}
}
