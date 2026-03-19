package litepay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_supports_currency(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}
	cases := []struct {
		slice  []string
		value  string
		result bool
	}{
		{slice, "banana", true},
		{slice, "grape", false},
		{[]string{}, "apple", false},
	}

	for _, tt := range cases {
		found := supportsCurrency(tt.slice, tt.value)
		assert.Equal(t, tt.result, found)
	}
}

func Test_sign_message(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	privKey1Bytes := x509.MarshalPKCS1PrivateKey(privKey)
	privKey1Pem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKey1Bytes,
	})
	privKey8Bytes, _ := x509.MarshalPKCS8PrivateKey(privKey)
	privKey8Pem := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKey8Bytes,
	})

	cases := []struct {
		message string
		privKey string
		err     error
	}{
		{"Hello, World!", string(privKey8Pem), nil},
		{"Hello, World!", string(privKey1Pem), errors.New("x509: failed to parse private key (use ParsePKCS1PrivateKey instead for this key format)")},
		{"Hello, World!", "", errors.New("invalid private key")},
	}

	for _, tt := range cases {
		_, err := signMessage(tt.message, tt.privKey)
		assert.Equal(t, tt.err, err)
	}
}

func Test_parse_body(t *testing.T) {
	cases := []struct {
		body     string
		expected map[string]any
		err      error
	}{
		{
			body:     `{"name": "John", "age": 30}`,
			expected: map[string]any{"name": "John", "age": float64(30)},
			err:      nil,
		},
		{
			body:     `{"key": "value"}`,
			expected: map[string]any{"key": "value"},
			err:      nil,
		},
		{
			body:     `{`,
			expected: nil,
			err:      errors.New("error decoding request body"),
		},
	}

	for _, tt := range cases {
		r := strings.NewReader(tt.body)
		data, err := parseBody(r)
		assert.Equal(t, tt.err, err)
		assert.Equal(t, tt.expected, data)
	}
}

func Test_status_payment(t *testing.T) {
	cases := []struct {
		payment  PaymentSystem
		status   string
		expected Status
	}{
		{STRIPE, "pay", PAID},
		{STRIPE, "paid", PAID},
		{STRIPE, "unpaid", UNPAID},
		{STRIPE, "open", PROCESSED},
		{STRIPE, "complete", PAID},
		{STRIPE, "expired", CANCELED},
		{STRIPE, "requires_payment_method", FAILED},
		{STRIPE, "requires_confirmation", FAILED},
		{STRIPE, "requires_action", FAILED},
		{STRIPE, "processing", PROCESSED},
		{STRIPE, "requires_capture", PROCESSED},
		{STRIPE, "canceled", CANCELED},
		{STRIPE, "succeeded", PAID},
		{SPECTROCOIN, "2", PROCESSED},
		{SPECTROCOIN, "3", PAID},
		{SPECTROCOIN, "4", FAILED},
		{SPECTROCOIN, "5", FAILED},
		{SPECTROCOIN, "6", TEST},
		{SPECTROCOIN, "", FAILED},
		{COINBASE, "NEW", UNPAID},
		{COINBASE, "PENDING", PROCESSED},
		{COINBASE, "COMPLETED", PAID},
		{COINBASE, "EXPIRED", CANCELED},
		{COINBASE, "CANCELED", CANCELED},
		{COINBASE, "RESOLVED", PAID},
		{COINBASE, "UNRESOLVED", FAILED},
		{COINBASE, "", FAILED},
		{DUMMY, "paid", PAID},
		{DUMMY, "", FAILED},
	}

	for _, tt := range cases {
		result := StatusPayment(tt.payment, tt.status)
		assert.Equal(t, tt.expected, result)
	}
}
