package queries

import (
	"testing"
	"time"
)

func TestSessionCRUD(t *testing.T) {
	db, ctx := bootstrap(t)

	now := time.Now().Unix()

	if err := db.AddSession(ctx, "sess-1", "value-1", now+3600); err != nil {
		t.Fatalf("AddSession: %v", err)
	}

	got, err := db.GetSession(ctx, "sess-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got != "value-1" {
		t.Errorf("got %q, want value-1", got)
	}

	// Idempotent re-insert is what AddSession promises.
	if err := db.AddSession(ctx, "sess-1", "value-2", now+3600); err != nil {
		t.Fatalf("AddSession replace: %v", err)
	}
	got, _ = db.GetSession(ctx, "sess-1")
	if got != "value-2" {
		t.Errorf("got %q, want value-2 after replace", got)
	}

	// UpdateSession changes the value directly.
	if err := db.UpdateSession(ctx, "sess-1", "value-3", now+7200); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}
	got, _ = db.GetSession(ctx, "sess-1")
	if got != "value-3" {
		t.Errorf("got %q, want value-3", got)
	}

	// Expired sessions must not be returned.
	if err := db.UpdateSession(ctx, "sess-1", "value-3", now-10); err != nil {
		t.Fatalf("expire UpdateSession: %v", err)
	}
	if _, err := db.GetSession(ctx, "sess-1"); err == nil {
		t.Error("expected error for expired session")
	}

	// DeleteSession removes the row so a follow-up GetSession errors.
	if err := db.DeleteSession(ctx, "sess-1"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := db.GetSession(ctx, "sess-1"); err == nil {
		t.Error("expected error after delete")
	}
}
