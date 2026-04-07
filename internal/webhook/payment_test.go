package webhook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/shurco/mycart/pkg/litepay"
)

func Test_send_payment_hook(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var p Payment
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			w.WriteHeader(400)
			return
		}
		got.Store(p)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := &Payment{
		Event: PAYMENT_INITIATION,
		Data:  Data{PaymentSystem: litepay.STRIPE, PaymentStatus: litepay.NEW},
	}
	resp, err := Send(srv.URL, mustJSON(p))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}

	v := got.Load()
	if v == nil {
		t.Fatalf("no payload captured")
	}
	pay := v.(Payment)
	if pay.Event != PAYMENT_INITIATION {
		t.Fatalf("unexpected event")
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
