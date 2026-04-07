# LitePay Quick Reference

Quick reference guide for working with the litepay package.

## Quick Start

```go
import "github.com/shurco/mycart/pkg/litepay"

// 1. Create configuration
pay := litepay.New(callbackURL, successURL, cancelURL)

// 2. Choose provider
stripe := pay.Stripe("sk_test_...")
// or
paypal := pay.Paypal("client_id", "secret")
// or
spectrocoin := pay.Spectrocoin("merchant_id", "project_id", "private_key")
// or
dummy := pay.Dummy()

// 3. Create cart
cart := litepay.Cart{
    ID:       "ABC123XYZ456789",
    Currency: "USD",
    Items: []litepay.Item{
        {
            PriceData: litepay.Price{
                UnitAmount: 1999, // $19.99 in cents
                Product: litepay.Product{
                    Name: "Product Name",
                },
            },
            Quantity: 1,
        },
    },
}

// 4. Create payment
payment, err := stripe.Pay(cart)

// 5. Redirect to payment.URL
```

## Statuses

| Constant | Description | Final |
|----------|-------------|-------|
| `NEW` | Created | ❌ |
| `UNPAID` | Unpaid | ❌ |
| `PAID` | Paid | ✅ |
| `CANCELED` | Canceled | ✅ |
| `FAILED` | Failed | ✅ |
| `PROCESSED` | Processing | ❌ |
| `TEST` | Test | ❌ |

## Providers

### Stripe

```go
stripe := pay.Stripe("sk_test_...")
payment, err := stripe.Pay(cart)
// Redirect to: payment.URL

// In success callback:
payment, err := stripe.Checkout(payment, sessionID)
```

**Keys:**
- Test: `sk_test_...`
- Live: `sk_live_...`

### PayPal

```go
paypal := pay.Paypal("client_id", "secret_key")
payment, err := paypal.Pay(cart)
// Redirect to: payment.URL

// In success callback:
payment, err := paypal.Checkout(payment, token)
```

**API:**
- Sandbox: `https://api.sandbox.paypal.com`
- Live: `https://api.paypal.com`

### SpectroCoin

```go
spectrocoin := pay.Spectrocoin(
    "merchant_id",  // UUID
    "project_id",   // UUID
    privateKey,     // PEM PKCS#8
)
payment, err := spectrocoin.Pay(cart)
// Redirect to: payment.URL

// In webhook callback:
var cb litepay.CallbackSpectrocoin
// ... parse form data
status := litepay.StatusPayment(litepay.SPECTROCOIN, strconv.Itoa(cb.Status))
```

**Statuses:**
- `1` = NEW (new)
- `2` = PROCESSED (processing)
- `3` = PAID (paid)
- `4` = FAILED (error)
- `5` = FAILED (expired)
- `6` = TEST (test)

### Dummy

```go
dummy := pay.Dummy()
payment, err := dummy.Pay(cart)
// Immediately status = PAID
```

⚠️ **For testing or free items only!**

## Status Mapping

```go
status := litepay.StatusPayment(provider, providerStatus)
```

**Examples:**

```go
litepay.StatusPayment(litepay.STRIPE, "succeeded")      // → PAID
litepay.StatusPayment(litepay.STRIPE, "canceled")       // → CANCELED
litepay.StatusPayment(litepay.PAYPAL, "COMPLETED")      // → PAID
litepay.StatusPayment(litepay.SPECTROCOIN, "3")         // → PAID
litepay.StatusPayment(litepay.DUMMY, "paid")            // → PAID
```

## Currencies

All providers:
```
EUR, USD, GBP, AUD, CAD, JPY, CNY, SEK
```

## Cart Structure

```go
litepay.Cart{
    ID:       "ABC123XYZ456789", // 15 characters (required)
    Currency: "USD",              // ISO currency code
    Items: []litepay.Item{
        {
            PriceData: litepay.Price{
                UnitAmount: 1999,  // Price in smallest units (cents)
                Product: litepay.Product{
                    Name:        "Product Name",
                    Description: "Optional",
                    Images:      []string{"https://..."},
                },
            },
            Quantity: 2,           // Quantity
        },
    },
}
```

## Payment Structure

```go
payment := &litepay.Payment{
    PaymentSystem: litepay.STRIPE,     // Provider
    MerchantID:    "ch_...",           // ID from provider
    CartID:        "ABC123XYZ456789",  // Cart ID
    AmountTotal:   3998,               // Amount in cents
    Currency:      "USD",              // Currency
    Status:        litepay.PAID,       // Status
    URL:           "https://...",      // Redirect URL
    Coin:          nil,                // Optional for crypto
}
```

## Usage Patterns

### 1. Creating Payment

```go
func CreatePayment(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req struct {
        Provider string   `json:"provider"`
        Products []string `json:"products"`
        Email    string   `json:"email"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Create cart
    cart := buildCart(req.Products)
    
    // Choose provider
    pay := litepay.New(callbackURL, successURL, cancelURL)
    var session litepay.LitePay
    
    switch req.Provider {
    case "stripe":
        session = pay.Stripe(os.Getenv("STRIPE_KEY"))
    case "paypal":
        session = pay.Paypal(os.Getenv("PAYPAL_CLIENT"), os.Getenv("PAYPAL_SECRET"))
    }
    
    // Create payment
    payment, err := session.Pay(cart)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    
    // Save to DB
    savePayment(payment)
    
    // Return URL
    json.NewEncoder(w).Encode(map[string]string{
        "url": payment.URL,
    })
}
```

### 2. Success Callback

```go
func PaymentSuccess(w http.ResponseWriter, r *http.Request) {
    cartID := r.URL.Query().Get("cart_id")
    sessionID := r.URL.Query().Get("session")
    
    // Load from DB
    payment := loadPayment(cartID)
    
    // Verify with provider
    pay := litepay.New("", "", "")
    var session litepay.LitePay
    
    switch payment.PaymentSystem {
    case litepay.STRIPE:
        session = pay.Stripe(os.Getenv("STRIPE_KEY"))
    case litepay.PAYPAL:
        session = pay.Paypal(os.Getenv("PAYPAL_CLIENT"), os.Getenv("PAYPAL_SECRET"))
    }
    
    updatedPayment, err := session.Checkout(payment, sessionID)
    if err != nil {
        http.Error(w, "Verification failed", 500)
        return
    }
    
    // Update in DB
    updatePayment(updatedPayment)
    
    // If paid - fulfill order
    if updatedPayment.Status == litepay.PAID {
        fulfillOrder(cartID)
    }
    
    // Show success page
    renderSuccessPage(w, updatedPayment)
}
```

### 3. Webhook (SpectroCoin)

```go
func PaymentCallback(w http.ResponseWriter, r *http.Request) {
    var callback litepay.CallbackSpectrocoin
    
    // Parse form
    r.ParseForm()
    callback.OrderID = r.FormValue("orderId")
    callback.Status, _ = strconv.Atoi(r.FormValue("status"))
    // ... other fields
    
    // Map status
    status := litepay.StatusPayment(
        litepay.SPECTROCOIN,
        strconv.Itoa(callback.Status),
    )
    
    // Update in DB
    updatePaymentStatus(callback.OrderID, status)
    
    // Fulfill order
    if status == litepay.PAID {
        fulfillOrder(callback.OrderID)
    }
    
    // Respond to provider
    w.WriteHeader(200)
}
```

## Testing

```go
// Unit tests
go test ./pkg/litepay/...

// Coverage
go test -cover ./pkg/litepay/...

// Verbose
go test -v ./pkg/litepay/...
```

## Environment Variables

```bash
# Stripe
STRIPE_SECRET_KEY=sk_test_...

# PayPal
PAYPAL_CLIENT_ID=...
PAYPAL_SECRET_KEY=...

# SpectroCoin
SPECTROCOIN_MERCHANT_ID=...
SPECTROCOIN_PROJECT_ID=...
SPECTROCOIN_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----..."
```

## Status Check

```go
switch payment.Status {
case litepay.PAID:
    // Fulfill order
    fulfillOrder()
case litepay.FAILED, litepay.CANCELED:
    // Handle failure
    handleFailure()
case litepay.PROCESSED:
    // Wait
    waitForCompletion()
}
```

## Validation

```go
// Cart ID must be 15 characters
err := payment.Validate()
if err != nil {
    // CartID invalid
}

// Currency check
if !isSupportedCurrency(cart.Currency) {
    return errors.New("currency not supported")
}

// Amount check
if totalAmount < 50 {
    return errors.New("minimum amount is $0.50")
}
```

## Common Mistakes

### ❌ Invalid Currency
```go
cart.Currency = "RUB" // Not supported
```

### ❌ Invalid CartID Length
```go
cart.ID = "SHORT" // Must be 15 characters
```

### ❌ Price in Dollars Instead of Cents
```go
UnitAmount: 19.99 // ❌ Wrong
UnitAmount: 1999  // ✅ Correct
```

### ❌ Using Dummy for Paid Items
```go
// ❌ Unsafe in production
if cart.AmountTotal > 0 {
    session = pay.Dummy() // Allows bypassing payment!
}

// ✅ Correct
if cart.AmountTotal == 0 {
    session = pay.Dummy()
} else {
    session = pay.Stripe(apiKey)
}
```

## Additional Resources

- [Full Documentation](README.md)
- [UI Customization](../../docs/payment-customization.md)
- [Tests](helper_test.go)
