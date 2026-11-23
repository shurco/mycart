package litepay

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CallbackSpectrocoin represents the webhook callback data from SpectroCoin.
// This structure is used to parse webhook notifications sent by SpectroCoin
// when a payment status changes.
type CallbackSpectrocoin struct {
	MerchantID      int     `json:"merchantId" form:"merchantId"`           // SpectroCoin merchant ID
	ApiID           int     `json:"apiId" form:"apiId"`                     // API ID
	UserID          string  `json:"userId" form:"userId"`                   // User identifier
	MerchantApiID   string  `json:"merchantApiId" form:"merchantApiId"`     // Merchant API ID
	OrderID         string  `json:"orderId" form:"orderId"`                 // Order/Cart ID
	PayCurrency     string  `json:"payCurrency" form:"payCurrency"`         // Cryptocurrency used for payment
	PayAmount       float64 `json:"payAmount" form:"payAmount"`             // Amount paid in cryptocurrency
	ReceiveCurrency string  `json:"receiveCurrency" form:"receiveCurrency"` // Fiat currency to receive
	ReceiveAmount   float64 `json:"receiveAmount" form:"receiveAmount"`     // Amount to receive in fiat
	ReceivedAmount  int     `json:"receivedAmount" form:"receivedAmount"`   // Actual received amount
	Description     string  `json:"description" form:"description"`         // Order description
	OrderRequestID  int     `json:"orderRequestId" form:"orderRequestId"`   // Request ID
	Status          int     `json:"status" form:"status"`                   // Payment status (1=new, 2=pending, 3=paid, 4=failed, 5=expired, 6=test)
	Sign            string  `json:"sign" form:"sign"`                       // RSA signature for verification
}

type spectrocoin struct {
	Cfg
	merchantID string
	projectID  string
	privateKey string
}

// Spectrocoin initializes a SpectroCoin cryptocurrency payment provider.
//
// Parameters:
//   - merchantID: Your SpectroCoin merchant ID (UUID format)
//   - projectID: Your SpectroCoin project/API ID (UUID format)
//   - privateKey: Your RSA private key in PEM format (PKCS#8) for signing requests
//
// Returns:
//   - LitePay: A configured SpectroCoin payment provider
//
// Supported currencies: EUR, USD, GBP, AUD, CAD, JPY, CNY, SEK
//
// Note: SpectroCoin accepts cryptocurrency payments (BTC, ETH, etc.) and converts to fiat.
//
// Example:
//
//	pay := litepay.New(callbackURL, successURL, cancelURL)
//	spectrocoin := pay.Spectrocoin(merchantID, projectID, privateKey)
//	payment, err := spectrocoin.Pay(cart)
func (c Cfg) Spectrocoin(merchantID, projectID, privateKey string) LitePay {
	c.paymentSystem = SPECTROCOIN
	c.api = "https://spectrocoin.com"
	c.currency = []string{"EUR", "USD", "GBP", "AUD", "CAD", "JPY", "CNY", "SEK"}
	return &spectrocoin{
		Cfg:        c,
		merchantID: merchantID,
		projectID:  projectID,
		privateKey: privateKey,
	}
}

func (c *spectrocoin) Pay(cart Cart) (*Payment, error) {
	var totalAmount float64
	receiveCurrency := strings.ToUpper(cart.Currency)

	if !supportsCurrency(c.currency, receiveCurrency) {
		return nil, errors.New("this currency is not supported")
	}

	for _, s := range cart.Items {
		totalAmount += float64(s.PriceData.UnitAmount) / 100 * float64(s.Quantity)
	}

	_receiveAmount := fmt.Sprintf("%.2f", totalAmount)
	_receiveAmount = strings.ReplaceAll(_receiveAmount, ".00", ".0")

	body := "userId=" + c.merchantID +
		"&merchantApiId=" + c.projectID +
		"&orderId=" + cart.ID +
		"&payCurrency=BTC" +
		"&payAmount=0.0" +
		"&receiveCurrency=" + receiveCurrency +
		"&receiveAmount=" + _receiveAmount +
		"&description=" +
		"&payerEmail=" +
		"&payerName=" +
		"&payerSurname=" +
		"&culture=" +
		"&callbackUrl=" + url.QueryEscape(joinURL(c.callbackURL, fmt.Sprintf("payment_system=%s&cart_id=%s", c.paymentSystem, cart.ID))) +
		"&successUrl=" + url.QueryEscape(joinURL(c.successURL, fmt.Sprintf("payment_system=%s&cart_id=%s", c.paymentSystem, cart.ID))) +
		"&failureUrl=" + url.QueryEscape(joinURL(c.cancelURL, fmt.Sprintf("payment_system=%s&cart_id=%s", c.paymentSystem, cart.ID)))

	signature, err := signMessage(body, c.privateKey)
	if err != nil {
		return nil, err
	}
	body += "&sign=" + url.QueryEscape(signature)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/merchant/1/createOrder", c.api), bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := parseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	amount, err := field[string](data, "receiveAmount")
	if err != nil {
		return nil, err
	}
	currency, err := field[string](data, "receiveCurrency")
	if err != nil {
		return nil, err
	}
	redirectURL, err := field[string](data, "redirectUrl")
	if err != nil {
		return nil, err
	}

	receiveAmount, _ := strconv.ParseFloat(amount, 64)
	checkout := &Payment{
		AmountTotal:   int(receiveAmount * 100),
		Currency:      currency,
		Status:        PROCESSED,
		URL:           redirectURL,
		PaymentSystem: c.paymentSystem,
	}

	return checkout, nil
}

func (c *spectrocoin) Checkout(payment *Payment, session string) (*Payment, error) {
	return nil, nil
}

// spectroCoinPublicKey is the official SpectroCoin Merchant API public key
// (https://spectrocoin.com/files/merchant.public.pem). SpectroCoin signs every
// order callback with its private counterpart, so callback payloads must be
// verified against this key before any field (status, amount, order id) is
// trusted.
const spectroCoinPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA/Z5qppUzgJ/QvTgmo/UO
qb+lxPMF6xdOE6MwwGGJFP9HZX8nESPS/WPD4DRIx8htNjvwrE5mAsU7Y16WRKfR
Huraepi98zJizDWQLDpzYTz4auRdE5RdYnb0ojR+tUJSSt0q4goIIDwCLtgC5JSY
VPMs2rjzXYRjepF8+NNzpTvKZXqhJCb3dw9nyJNy0vYan7maOLLBWHQNoJbLifqt
2A0Q1zphMXiufaoxqUJ3+0ysLp2G/qvv/j9Lg+OHTao0vIhz/dMqhjtk+MDguoCb
aOzFW43seYdxPWpCbv0JwTwDvXf9jP7jYb4f6yGHLVCOBt40rKLZENI29qDZii0p
JwIDAQAB
-----END PUBLIC KEY-----`

// VerifySpectrocoinCallback validates the RSA-SHA1 signature of a SpectroCoin
// order callback as specified by the Merchant API: the signed data is the
// UTF-8 URL-encoded concatenation of callback parameters (in documented
// order, numbers formatted with the "0.0#######" pattern) and the base64
// decoded `sign` field must verify against the embedded SpectroCoin public key.
func VerifySpectrocoinCallback(cb *CallbackSpectrocoin) error {
	return verifySpectrocoinCallback(spectroCoinPublicKey, cb)
}

func verifySpectrocoinCallback(publicKeyPEM string, cb *CallbackSpectrocoin) error {
	if cb == nil || cb.Sign == "" {
		return errors.New("empty callback signature")
	}

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return errors.New("invalid SpectroCoin public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse SpectroCoin public key: %w", err)
	}
	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return errors.New("SpectroCoin key is not RSA")
	}

	signature, err := base64.StdEncoding.DecodeString(cb.Sign)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	hash := sha1.Sum([]byte(callbackSignString(cb)))
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, hash[:], signature)
}

// callbackSignString builds the exact string that SpectroCoin signs for an
// order callback. Field set and order follow the Merchant API specification.
func callbackSignString(cb *CallbackSpectrocoin) string {
	return "merchantId=" + strconv.Itoa(cb.MerchantID) +
		"&apiId=" + strconv.Itoa(cb.ApiID) +
		"&orderId=" + url.QueryEscape(cb.OrderID) +
		"&payCurrency=" + url.QueryEscape(cb.PayCurrency) +
		"&payAmount=" + spectrocoinAmount(cb.PayAmount) +
		"&receiveCurrency=" + url.QueryEscape(cb.ReceiveCurrency) +
		"&receiveAmount=" + spectrocoinAmount(cb.ReceiveAmount) +
		"&description=" + url.QueryEscape(cb.Description) +
		"&orderRequestId=" + strconv.Itoa(cb.OrderRequestID) +
		"&status=" + strconv.Itoa(cb.Status)
}

// spectrocoinAmount formats a number using the "0.0#######" pattern required
// by the signing specification: at least one fractional digit, no trailing zeros.
func spectrocoinAmount(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
