package litepay

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type stripe struct {
	Cfg
	apiToken   string
	successURL string
	cancelURL  string
}

// Stripe initializes a Stripe payment provider.
//
// Parameters:
//   - apiToken: Your Stripe secret API key (starts with "sk_test_" for test mode or "sk_live_" for production)
//
// Returns:
//   - LitePay: A configured Stripe payment provider
//
// Supported currencies: EUR, USD, GBP, AUD, CAD, JPY, CNY, SEK
//
// Example:
//
//	pay := litepay.New(callbackURL, successURL, cancelURL)
//	stripe := pay.Stripe("sk_test_...")
//	payment, err := stripe.Pay(cart)
func (c Cfg) Stripe(apiToken string) LitePay {
	c.paymentSystem = STRIPE
	c.api = "https://api.stripe.com"
	c.currency = []string{"EUR", "USD", "GBP", "AUD", "CAD", "JPY", "CNY", "SEK"}
	return &stripe{
		Cfg:        c,
		apiToken:   apiToken,
		successURL: c.successURL,
		cancelURL:  c.cancelURL,
	}
}

func (c *stripe) Pay(cart Cart) (*Payment, error) {
	currency := strings.ToUpper(cart.Currency)
	if !supportsCurrency(c.currency, currency) {
		return nil, errors.New("this currency is not supported")
	}

	params := url.Values{}
	for i, s := range cart.Items {
		iString := strconv.Itoa(i)
		params.Add("line_items["+iString+"][price_data][unit_amount]", strconv.Itoa(s.PriceData.UnitAmount))
		params.Add("line_items["+iString+"][price_data][currency]", currency)
		params.Add("line_items["+iString+"][price_data][product_data][name]", s.PriceData.Product.Name)
		for ii, img := range s.PriceData.Product.Images {
			params.Add("line_items["+iString+"][price_data][product_data][images]["+strconv.Itoa(ii)+"]", img)
		}
		params.Add("line_items["+iString+"][quantity]", strconv.Itoa(s.Quantity))
	}
	params.Add("success_url", joinURL(c.successURL, fmt.Sprintf("payment_system=%s&cart_id=%s&session={CHECKOUT_SESSION_ID}", c.paymentSystem, cart.ID)))
	params.Add("cancel_url", joinURL(c.cancelURL, fmt.Sprintf("payment_system=%s&cart_id=%s", c.paymentSystem, cart.ID)))
	params.Add("mode", `payment`)
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(
		http.MethodPost,
		c.api+"/v1/checkout/sessions",
		body,
	)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.apiToken, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("the server returned an error")
	}

	data, err := parseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	id, err := field[string](data, "id")
	if err != nil {
		return nil, err
	}
	amount, err := field[float64](data, "amount_total")
	if err != nil {
		return nil, err
	}
	paidCurrency, err := field[string](data, "currency")
	if err != nil {
		return nil, err
	}
	status, err := field[string](data, "payment_status")
	if err != nil {
		return nil, err
	}
	redirectURL, err := field[string](data, "url")
	if err != nil {
		return nil, err
	}

	checkout := &Payment{
		MerchantID:    id,
		AmountTotal:   int(amount),
		Currency:      strings.ToUpper(paidCurrency),
		Status:        StatusPayment(STRIPE, status),
		URL:           redirectURL,
		PaymentSystem: c.paymentSystem,
	}

	return checkout, nil
}

func (c *stripe) Checkout(payment *Payment, session string) (*Payment, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		c.api+"/v1/checkout/sessions/"+session,
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.apiToken, "")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, errors.New("the server returned an error")
	}

	data, err := parseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	intent, err := field[string](data, "payment_intent")
	if err != nil {
		return nil, err
	}
	amount, err := field[float64](data, "amount_total")
	if err != nil {
		return nil, err
	}
	currency, err := field[string](data, "currency")
	if err != nil {
		return nil, err
	}
	status, err := field[string](data, "payment_status")
	if err != nil {
		return nil, err
	}

	payment.MerchantID = intent
	payment.AmountTotal = int(amount)
	payment.Currency = strings.ToUpper(currency)
	payment.Status = StatusPayment(STRIPE, status)

	return payment, nil
}
