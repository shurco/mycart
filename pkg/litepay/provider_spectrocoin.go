package litepay

import (
	"bytes"
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
		"&callbackUrl=" + url.QueryEscape(fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.callbackURL, c.paymentSystem, cart.ID)) +
		"&successUrl=" + url.QueryEscape(fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.successURL, c.paymentSystem, cart.ID)) +
		"&failureUrl=" + url.QueryEscape(fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.cancelURL, c.paymentSystem, cart.ID))

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

	receiveAmount, _ := strconv.ParseFloat(data["receiveAmount"].(string), 64)
	checkout := &Payment{
		AmountTotal:   int(receiveAmount * 100),
		Currency:      data["receiveCurrency"].(string),
		Status:        PROCESSED,
		URL:           data["redirectUrl"].(string),
		PaymentSystem: c.paymentSystem,
	}

	return checkout, nil
}

func (c *spectrocoin) Checkout(payment *Payment, session string) (*Payment, error) {
	return nil, nil
}
