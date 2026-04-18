package mailer

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	mailer "github.com/xhit/go-simple-mail/v2"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/migrations"
)

// bootstrapDB brings up a fresh queries DB in a temp working directory.
// Several functions in this package call queries.DB() directly so tests
// must share the package-level instance.
func bootstrapDB(t *testing.T) *queries.Base {
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
	return queries.DB()
}

func TestEncryptionTypesLookup(t *testing.T) {
	t.Parallel()
	if EncryptionTypes["None"] != mailer.EncryptionNone {
		t.Error("missing None encryption")
	}
	if EncryptionTypes["SSL/TLS"] != mailer.EncryptionSSL {
		t.Error("missing SSL/TLS encryption")
	}
	if EncryptionTypes["STARTTLS"] != mailer.EncryptionTLS {
		t.Error("missing STARTTLS encryption")
	}
}

func TestSendMail_ValidatesSMTPFields(t *testing.T) {
	t.Parallel()

	// Missing host should short-circuit before any network traffic.
	setting := &models.Mail{SenderEmail: "no@reply"}
	err := SendMail(setting, &models.MessageMail{To: "x@example.com"})
	if err == nil || !strings.Contains(err.Error(), "SMTP settings") {
		t.Fatalf("expected SMTP validation error, got %v", err)
	}
}

func TestSendMail_RequiresSenderEmail(t *testing.T) {
	t.Parallel()

	setting := &models.Mail{}
	setting.SMTP.Host = "localhost"
	setting.SMTP.Port = 25
	setting.SMTP.Username = "u"
	setting.SMTP.Password = "p"

	err := SendMail(setting, &models.MessageMail{To: "to@example.com"})
	if err == nil || !strings.Contains(err.Error(), "sender email") {
		t.Fatalf("expected sender email error, got %v", err)
	}
}

func TestSendMail_UnreachableServer(t *testing.T) {
	t.Parallel()

	setting := &models.Mail{
		SenderEmail: "a@b.com",
	}
	setting.SMTP.Host = "127.0.0.1"
	setting.SMTP.Port = 1 // reserved; connections refused immediately
	setting.SMTP.Username = "u"
	setting.SMTP.Password = "p"

	err := SendMail(setting, &models.MessageMail{
		To:     "to@example.com",
		Letter: models.Letter{Subject: "s", Text: "body"},
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestTextTemplate_RendersData(t *testing.T) {
	t.Parallel()

	out, err := textTemplate("Hello {{.Name}}", map[string]string{"Name": "world"})
	if err != nil {
		t.Fatalf("textTemplate: %v", err)
	}
	if string(out) != "Hello world" {
		t.Errorf("got %q, want %q", out, "Hello world")
	}

	if _, err := textTemplate("{{.}", nil); err == nil {
		t.Error("expected parse error for malformed template")
	}
}

func TestEnsureSenderEmail_UsesConfigured(t *testing.T) {
	db := bootstrapDB(t)
	ctx := context.Background()
	m := &models.Mail{SenderEmail: "configured@example.com"}
	if err := ensureSenderEmail(ctx, db, m); err != nil {
		t.Fatalf("ensureSenderEmail: %v", err)
	}
	if m.SenderEmail != "configured@example.com" {
		t.Errorf("sender mutated unexpectedly: %s", m.SenderEmail)
	}
}

func TestEnsureSenderEmail_EmptyFallbackMissing(t *testing.T) {
	db := bootstrapDB(t)
	ctx := context.Background()

	// With a blank `email` setting the function must surface an error rather
	// than silently leaving SenderEmail blank.
	m := &models.Mail{}
	if err := ensureSenderEmail(ctx, db, m); err == nil {
		t.Error("expected error when both sender email and user email are empty")
	}
}

func TestEnsureSenderEmail_FallbackSuccess(t *testing.T) {
	db := bootstrapDB(t)
	ctx := context.Background()

	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "email",
		Value: "owner@example.com",
	}); err != nil {
		t.Fatalf("seed email: %v", err)
	}

	m := &models.Mail{}
	if err := ensureSenderEmail(ctx, db, m); err != nil {
		t.Fatalf("ensureSenderEmail: %v", err)
	}
	if m.SenderEmail != "owner@example.com" {
		t.Errorf("fallback not applied: %+v", m)
	}
}

func TestSendTestLetter_RejectsMissingSMTP(t *testing.T) {
	bootstrapDB(t)
	if err := SendTestLetter("smtp"); err == nil {
		t.Error("expected error when SMTP is unconfigured")
	}
}

func TestSendPrepaymentLetter_MissingTemplate(t *testing.T) {
	bootstrapDB(t)
	// No template seeded — CartLetterPayment will fail to unmarshal an empty string.
	if err := SendPrepaymentLetter("x@y.com", "1 USD", "http://pay"); err == nil {
		t.Error("expected error for missing payment template")
	}
}

func TestSendPrepaymentLetter_MissingSMTP(t *testing.T) {
	db := bootstrapDB(t)
	ctx := context.Background()

	tpl, _ := json.Marshal(models.Letter{Subject: "S", Text: "{{.Payment_URL}}"})
	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "mail_letter_payment",
		Value: string(tpl),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.UpdateSettingByKey(ctx, &models.SettingName{
		Key:   "site_name",
		Value: "Litecart",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// With the template in place the call reaches SendMail which fails on
	// the (intentionally missing) SMTP configuration.
	if err := SendPrepaymentLetter("x@y.com", "1 USD", "http://pay"); err == nil {
		t.Error("expected SMTP validation error")
	}
}

func TestSendCartLetter_CartNotFound(t *testing.T) {
	bootstrapDB(t)
	if err := SendCartLetter("missing-cart"); err == nil {
		t.Error("expected error for missing cart")
	}
}
