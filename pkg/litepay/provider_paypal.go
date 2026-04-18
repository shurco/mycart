package litepay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type paypal struct {
	Cfg
	clientID  string
	secretKey string
}

// Paypal initializes a PayPal payment provider.
//
// Parameters:
//   - clientID: Your PayPal REST API client ID
//   - secretKey: Your PayPal REST API secret key
//
// Returns:
//   - LitePay: A configured PayPal payment provider
//
// Supported currencies: EUR, USD, GBP, AUD, CAD, JPY, CNY, SEK
//
// Note: Currently configured for sandbox mode. Change c.api to "https://api.paypal.com" for production.
//
// Example:
//
//	pay := litepay.New(callbackURL, successURL, cancelURL)
//	paypal := pay.Paypal("client_id", "secret_key")
//	payment, err := paypal.Pay(cart)
func (c Cfg) Paypal(clientID, secretKey string) LitePay {
	c.paymentSystem = PAYPAL
	c.api = "https://api.sandbox.paypal.com" // https://api.paypal.com
	c.currency = []string{"EUR", "USD", "GBP", "AUD", "CAD", "JPY", "CNY", "SEK"}
	return &paypal{
		Cfg:       c,
		clientID:  clientID,
		secretKey: secretKey,
	}
}

func (c *paypal) Pay(cart Cart) (*Payment, error) {
	var totalAmount float64

	currency := strings.ToUpper(cart.Currency)
	if !supportsCurrency(c.currency, currency) {
		return nil, errors.New("this currency is not supported")
	}

	accessToken, err := c.paypalAccessToken()
	if err != nil {
		return nil, err
	}

	for _, s := range cart.Items {
		totalAmount += float64(s.PriceData.UnitAmount) / 100 * float64(s.Quantity)
	}

	order := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{
			{
				"amount": map[string]any{
					"currency_code": currency,
					"value":         fmt.Sprintf("%.2f", totalAmount),
				},
			},
		},
		"payment_source": map[string]any{
			"paypal": map[string]any{
				"experience_context": map[string]string{
					"user_action": "PAY_NOW",
					"return_url":  fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.successURL, c.paymentSystem, cart.ID),
					"cancel_url":  fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.cancelURL, c.paymentSystem, cart.ID),
				},
			},
		},
	}

	orderJSON, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	body := bytes.NewReader(orderJSON)

	req, err := http.NewRequest(
		http.MethodPost,
		c.api+"/v2/checkout/orders",
		body,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var data struct {
		ID    string `json:"id"`
		Links []struct {
			Href   string `json:"href"`
			Rel    string `json:"rel"`
			Method string `json:"method"`
		} `json:"links"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %v", err)
	}

	checkout := &Payment{
		AmountTotal:   int(totalAmount * 100),
		Currency:      currency,
		Status:        StatusPayment(PAYPAL, data.Status),
		PaymentSystem: c.paymentSystem,
	}

	for _, link := range data.Links {
		if link.Rel == "payer-action" {
			checkout.URL = link.Href
			break
		}
	}

	return checkout, nil
}

func (c *paypal) Checkout(payment *Payment, token string) (*Payment, error) {
	accessToken, err := c.paypalAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		c.api+"/v2/checkout/orders/"+token+"/capture",
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 422 {
		return nil, errors.New("unprocessable entity")
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, errors.New("the server returned an error")
	}

	var data struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []struct {
					Amount struct {
						CurrencyCode string `json:"currency_code"`
						Value        string `json:"value"`
					} `json:"amount"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	receiveAmount, _ := strconv.ParseFloat(data.PurchaseUnits[0].Payments.Captures[0].Amount.Value, 64)
	payment.AmountTotal = int(receiveAmount * 100)
	payment.Currency = data.PurchaseUnits[0].Payments.Captures[0].Amount.CurrencyCode
	payment.MerchantID = data.ID
	payment.Status = StatusPayment(PAYPAL, data.Status)

	return payment, nil
}

func (c *paypal) paypalAccessToken() (string, error) {
	req, err := http.NewRequest(
		"POST",
		c.api+"/v1/oauth2/token",
		strings.NewReader("grant_type=client_credentials"),
	)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.clientID, c.secretKey)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("the server returned an error")
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}
