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

type coinbase struct {
	Cfg
	apiKey string
}

// Coinbase initializes a Coinbase Commerce payment provider.
//
// Parameters:
//   - apiKey: Your Coinbase Commerce API key
//
// Returns:
//   - LitePay: A configured Coinbase Commerce payment provider
//
// Supported currencies: USD, EUR, GBP, BTC, ETH, USDC
//
// Example:
//
//	pay := litepay.New(callbackURL, successURL, cancelURL)
//	coinbase := pay.Coinbase("api_key")
//	payment, err := coinbase.Pay(cart)
func (c Cfg) Coinbase(apiKey string) LitePay {
	c.paymentSystem = COINBASE
	c.api = "https://api.commerce.coinbase.com"
	c.currency = []string{"USD", "EUR", "GBP", "BTC", "ETH", "USDC"}
	return &coinbase{
		Cfg:    c,
		apiKey: apiKey,
	}
}

func (c *coinbase) Pay(cart Cart) (*Payment, error) {
	currency := strings.ToUpper(cart.Currency)
	if !supportsCurrency(c.currency, currency) {
		return nil, errors.New("this currency is not supported")
	}

	var totalAmount float64
	for _, item := range cart.Items {
		totalAmount += float64(item.PriceData.UnitAmount) / 100 * float64(item.Quantity)
	}

	charge := map[string]any{
		"name":         "Cart " + cart.ID,
		"description":  "Payment for cart items",
		"pricing_type": "fixed_price",
		"local_price": map[string]any{
			"amount":   fmt.Sprintf("%.2f", totalAmount),
			"currency": currency,
		},
		"redirect_url": fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.successURL, c.paymentSystem, cart.ID),
		"cancel_url":   fmt.Sprintf("%s/?payment_system=%s&cart_id=%s", c.cancelURL, c.paymentSystem, cart.ID),
	}

	body, err := json.Marshal(charge)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.api+"/charges", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-CC-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return nil, errors.New("the server returned an error")
	}

	var result struct {
		Data struct {
			ID        string `json:"id"`
			HostedURL string `json:"hosted_url"`
			Pricing   struct {
				Local struct {
					Amount   string `json:"amount"`
					Currency string `json:"currency"`
				} `json:"local"`
			} `json:"pricing"`
			Timeline []struct {
				Status string `json:"status"`
			} `json:"timeline"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Get initial status from timeline
	status := "NEW"
	if len(result.Data.Timeline) > 0 {
		status = result.Data.Timeline[len(result.Data.Timeline)-1].Status
	}

	amountTotal, _ := strconv.ParseFloat(result.Data.Pricing.Local.Amount, 64)

	return &Payment{
		MerchantID:    result.Data.ID,
		AmountTotal:   int(amountTotal * 100),
		Currency:      result.Data.Pricing.Local.Currency,
		Status:        StatusPayment(COINBASE, status),
		URL:           result.Data.HostedURL,
		PaymentSystem: c.paymentSystem,
	}, nil
}

func (c *coinbase) Checkout(payment *Payment, chargeID string) (*Payment, error) {
	req, err := http.NewRequest(http.MethodGet, c.api+"/charges/"+chargeID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-CC-Api-Key", c.apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, errors.New("the server returned an error")
	}

	var result struct {
		Data struct {
			Timeline []struct {
				Status string `json:"status"`
			} `json:"timeline"`
			Pricing struct {
				Local struct {
					Amount   string `json:"amount"`
					Currency string `json:"currency"`
				} `json:"local"`
			} `json:"pricing"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Get last status from timeline
	lastStatus := "NEW"
	if len(result.Data.Timeline) > 0 {
		lastStatus = result.Data.Timeline[len(result.Data.Timeline)-1].Status
	}

	amountTotal, _ := strconv.ParseFloat(result.Data.Pricing.Local.Amount, 64)

	payment.Status = StatusPayment(COINBASE, lastStatus)
	payment.AmountTotal = int(amountTotal * 100)
	payment.Currency = result.Data.Pricing.Local.Currency

	return payment, nil
}
