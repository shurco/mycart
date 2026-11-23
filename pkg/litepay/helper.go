package litepay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

func supportsCurrency(supported []string, currency string) bool {
	return slices.Contains(supported, currency)
}

// joinURL appends a raw query string to a base URL using the correct
// separator, so callers may pass base URLs that already contain query
// parameters (e.g. a cancel URL carrying a capability token).
func joinURL(base, query string) string {
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + query
}

func signMessage(message, privKey string) (string, error) {
	block, _ := pem.Decode([]byte(privKey))
	if block == nil {
		return "", errors.New("invalid private key")
	}

	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	privateKey, ok := parsedKey.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("key is not a valid RSA private key")
	}

	hash := sha1.Sum([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func parseBody(r io.Reader) (map[string]any, error) {
	var data map[string]any

	body, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.New("error reading request body")
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, errors.New("error decoding request body")
		}
	}

	return data, nil
}

// field reads a typed value out of a decoded provider response. A provider that
// answers with an error body instead of the expected object is an error for
// the caller, never a panic.
func field[T any](data map[string]any, key string) (T, error) {
	v, ok := data[key].(T)
	if !ok {
		return v, fmt.Errorf("provider response has no %q of the expected type", key)
	}
	return v, nil
}

var (
	stripeStatuses = map[string]Status{
		"pay": PAID, "paid": PAID, "unpaid": UNPAID, "open": PROCESSED,
		"complete": PAID, "expired": CANCELED, "requires_payment_method": FAILED,
		"requires_confirmation": FAILED, "requires_action": FAILED,
		"processing": PROCESSED, "requires_capture": PROCESSED,
		"canceled": CANCELED, "succeeded": PAID,
	}
	paypalStatuses = map[string]Status{
		"CREATED": PROCESSED, "SAVED": PROCESSED, "APPROVED": PROCESSED,
		"VOIDED": CANCELED, "COMPLETED": PAID, "PAYER_ACTION_REQUIRED": PROCESSED,
	}
	spectrocoinStatuses = map[string]Status{
		"1": UNPAID, "2": PROCESSED, "3": PAID, "4": FAILED, "5": FAILED, "6": TEST,
	}
	coinbaseStatuses = map[string]Status{
		"NEW": UNPAID, "PENDING": PROCESSED, "COMPLETED": PAID,
		"EXPIRED": CANCELED, "CANCELED": CANCELED, "RESOLVED": PAID, "UNRESOLVED": FAILED,
	}
	dummyStatuses = map[string]Status{
		"paid": PAID,
	}
)

// StatusPayment maps provider-specific payment statuses to internal Status values.
//
// Parameters:
//   - system: The payment provider (STRIPE, PAYPAL, SPECTROCOIN, COINBASE, DUMMY)
//   - status: The status string from the provider
//
// Returns:
//   - Status: The normalized internal status (defaults to FAILED for unknown statuses)
func StatusPayment(system PaymentSystem, status string) Status {
	var statusBase map[string]Status

	switch system {
	case STRIPE:
		statusBase = stripeStatuses
	case PAYPAL:
		statusBase = paypalStatuses
	case SPECTROCOIN:
		statusBase = spectrocoinStatuses
	case COINBASE:
		statusBase = coinbaseStatuses
	case DUMMY:
		statusBase = dummyStatuses
	}

	if s := statusBase[status]; s != "" {
		return s
	}
	return FAILED
}
