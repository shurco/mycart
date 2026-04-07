# LitePay - Payment Provider Abstraction Layer

The `litepay` package provides a unified interface for working with various payment providers (Stripe, PayPal, SpectroCoin, etc.).

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Usage](#usage)
- [Adding a New Provider](#adding-a-new-provider)
- [API Reference](#api-reference)
- [Examples](#examples)

## Features

- 🔌 **Unified interface** for all payment providers
- 🔄 **Status mapping** from provider format to internal format
- 🔒 **Security**: signature verification support (for SpectroCoin)
- 💰 **Multiple currency support**: EUR, USD, GBP, AUD, CAD, JPY, CNY, SEK
- 🎯 **Callback/Webhook** system for event handling
- 📦 **Modularity**: easy to add new providers

## Architecture

### Main Interface

```go
type LitePay interface {
    Pay(cart Cart) (*Payment, error)      // Create payment session
    Checkout(payment *Payment, session string) (*Payment, error) // Check payment status
}
```

### Data Structures

#### Cart
```go
type Cart struct {
    ID       string // Unique cart identifier
    Currency string // Currency code (EUR, USD, etc.)
    Items    []Item // Items in the cart
}

type Item struct {
    PriceData Price // Price information
    Quantity  int   // Quantity
}

type Price struct {
    UnitAmount int     // Price in smallest currency unit (cents/kopeks)
    Product    Product // Product information
}

type Product struct {
    Name        string   // Product name
    Description string   // Description (optional)
    Images      []string // Image URLs
}
```

#### Payment
```go
type Payment struct {
    PaymentSystem PaymentSystem // Provider (stripe, paypal, etc.)
    MerchantID    string        // Transaction ID from provider
    CartID        string        // Cart ID
    AmountTotal   int           // Total amount in smallest currency unit
    Currency      string        // Currency
    Status        Status        // Payment status
    URL           string        // Redirect URL for customer
    Coin          *Coin         // Cryptocurrency payment info (optional)
}
```

#### Status
```go
const (
    NEW       Status = "new"       // Created
    UNPAID    Status = "unpaid"    // Awaiting payment
    PAID      Status = "paid"      // Successfully completed (final)
    CANCELED  Status = "canceled"  // Canceled (final)
    FAILED    Status = "failed"    // Failed (final)
    PROCESSED Status = "processed" // Being processed
    TEST      Status = "test"      // Test payment
)
```

### Supported Providers

```go
const (
    STRIPE      PaymentSystem = "stripe"      // Stripe payment provider
    PAYPAL      PaymentSystem = "paypal"      // PayPal payment provider
    SPECTROCOIN PaymentSystem = "spectrocoin" // SpectroCoin cryptocurrency provider
    DUMMY       PaymentSystem = "dummy"       // For testing/free items
)
```

## Usage

### 1. Initialization

```go
import "github.com/shurco/mycart/pkg/litepay"

// Create base configuration
pay := litepay.New(
    "https://example.com/cart/payment/callback", // Callback URL for webhook
    "https://example.com/cart/payment/success",  // Success URL
    "https://example.com/cart/payment/cancel",   // Cancel URL
)
```

### 2. Creating a Cart

```go
cart := litepay.Cart{
    ID:       "ABC123XYZ456789", // 15 characters
    Currency: "USD",
    Items: []litepay.Item{
        {
            PriceData: litepay.Price{
                UnitAmount: 1999, // $19.99 in cents
                Product: litepay.Product{
                    Name:        "Premium Plan",
                    Description: "Monthly subscription",
                    Images:      []string{"https://example.com/image.jpg"},
                },
            },
            Quantity: 1,
        },
    },
}
```

### 3. Using Providers

#### Stripe

```go
// Initialize provider
stripe := pay.Stripe("sk_test_your_secret_key")

// Create payment session
payment, err := stripe.Pay(cart)
if err != nil {
    log.Fatal(err)
}

// Redirect user to payment.URL
fmt.Println("Redirect to:", payment.URL)

// Check status (in success callback)
updatedPayment, err := stripe.Checkout(payment, sessionID)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Status:", updatedPayment.Status)
```

#### PayPal

```go
// Initialize provider
paypal := pay.Paypal("client_id", "secret_key")

// Create payment session
payment, err := paypal.Pay(cart)
if err != nil {
    log.Fatal(err)
}

// Redirect user
fmt.Println("Redirect to:", payment.URL)

// Check status
updatedPayment, err := paypal.Checkout(payment, token)
if err != nil {
    log.Fatal(err)
}
```

#### SpectroCoin

```go
// Initialize provider
spectrocoin := pay.Spectrocoin("merchant_id", "project_id", "private_key")

// Create payment session
payment, err := spectrocoin.Pay(cart)
if err != nil {
    log.Fatal(err)
}

// Redirect user
fmt.Println("Redirect to:", payment.URL)

// Handle callback (webhook from SpectroCoin)
// payment.Status is updated via StatusPayment()
```

#### Dummy (for testing)

```go
// Initialize provider (no parameters)
dummy := pay.Dummy()

// Create "payment" (always succeeds)
payment, err := dummy.Pay(cart)
if err != nil {
    log.Fatal(err)
}

// Status is immediately PAID
fmt.Println("Status:", payment.Status) // "paid"
```

### 4. Status Mapping

The `StatusPayment()` function converts provider statuses to internal format:

```go
// Example for Stripe
status := litepay.StatusPayment(litepay.STRIPE, "succeeded")
// status == litepay.PAID

// Example for PayPal
status = litepay.StatusPayment(litepay.PAYPAL, "COMPLETED")
// status == litepay.PAID

// Example for SpectroCoin
status = litepay.StatusPayment(litepay.SPECTROCOIN, "3")
// status == litepay.PAID
```

## Adding a New Provider

### Step 1: Add Constant

File: `provider.go`

```go
const (
    // ... existing
    NEWPROVIDER PaymentSystem = "newprovider"
)
```

### Step 2: Create provider_newprovider.go File

```go
package litepay

import (
    "encoding/json"
    "net/http"
    "strings"
)

// Provider structure
type newprovider struct {
    Cfg
    apiKey    string
    secretKey string
}

// Constructor
func (c Cfg) NewProvider(apiKey, secretKey string) LitePay {
    c.paymentSystem = NEWPROVIDER
    c.api = "https://api.newprovider.com"
    c.currency = []string{"EUR", "USD", "GBP"}
    return &newprovider{
        Cfg:       c,
        apiKey:    apiKey,
        secretKey: secretKey,
    }
}

// Implement Pay method
func (n *newprovider) Pay(cart Cart) (*Payment, error) {
    // 1. Currency validation
    if !findInSlice(n.currency, strings.ToUpper(cart.Currency)) {
        return nil, errors.New("currency not supported")
    }

    // 2. Calculate total
    var totalAmount int
    for _, item := range cart.Items {
        totalAmount += item.PriceData.UnitAmount * item.Quantity
    }

    // 3. Create request to provider API
    payload := map[string]any{
        "amount":   totalAmount,
        "currency": cart.Currency,
        "success_url": n.successURL + "?cart_id=" + cart.ID,
        "cancel_url":  n.cancelURL + "?cart_id=" + cart.ID,
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", n.api+"/checkout", strings.NewReader(string(body)))
    req.Header.Set("Authorization", "Bearer "+n.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 4. Parse response
    var result struct {
        ID  string `json:"id"`
        URL string `json:"checkout_url"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    // 5. Return Payment
    return &Payment{
        MerchantID:    result.ID,
        AmountTotal:   totalAmount,
        Currency:      cart.Currency,
        Status:        PROCESSED,
        URL:           result.URL,
        PaymentSystem: n.paymentSystem,
    }, nil
}

// Implement Checkout method
func (n *newprovider) Checkout(payment *Payment, sessionID string) (*Payment, error) {
    // 1. Request status from provider API
    req, _ := http.NewRequest("GET", n.api+"/checkout/"+sessionID, nil)
    req.Header.Set("Authorization", "Bearer "+n.apiKey)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 2. Parse response
    var result struct {
        Status string `json:"status"`
        Amount int    `json:"amount"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    // 3. Update payment
    payment.Status = StatusPayment(NEWPROVIDER, result.Status)
    payment.AmountTotal = result.Amount

    return payment, nil
}
```

### Step 3: Add Status Mapping

File: `helper.go`, function `StatusPayment()`

```go
case NEWPROVIDER:
    statusBase = map[string]Status{
        "pending":   PROCESSED,
        "completed": PAID,
        "failed":    FAILED,
        "canceled":  CANCELED,
    }
```

### Step 4: Tests

File: `provider_newprovider_test.go`

```go
package litepay

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestNewProvider_Pay(t *testing.T) {
    pay := New("http://callback.url", "http://success.url", "http://cancel.url")
    provider := pay.NewProvider("test_key", "test_secret")

    cart := Cart{
        ID:       "TEST12345678900",
        Currency: "USD",
        Items: []Item{
            {
                PriceData: Price{
                    UnitAmount: 1000,
                    Product: Product{
                        Name: "Test Product",
                    },
                },
                Quantity: 1,
            },
        },
    }

    payment, err := provider.Pay(cart)
    assert.NoError(t, err)
    assert.NotNil(t, payment)
    assert.Equal(t, 1000, payment.AmountTotal)
    assert.Equal(t, "USD", payment.Currency)
}

func TestNewProvider_Checkout(t *testing.T) {
    // Test status check
}
```

## API Reference

### Functions

#### New
```go
func New(callbackURL, successURL, cancelURL string) Cfg
```
Creates base configuration for all providers.

#### StatusPayment
```go
func StatusPayment(system PaymentSystem, status string) Status
```
Converts provider status to internal format.

### Methods (on Cfg)

#### Stripe
```go
func (c Cfg) Stripe(apiToken string) LitePay
```

#### Paypal
```go
func (c Cfg) Paypal(clientID, secretKey string) LitePay
```

#### Spectrocoin
```go
func (c Cfg) Spectrocoin(merchantID, projectID, privateKey string) LitePay
```

#### Dummy
```go
func (c Cfg) Dummy() LitePay
```

### Interface Methods

#### Pay
```go
Pay(cart Cart) (*Payment, error)
```
Creates a payment session with the provider.

**Parameters:**
- `cart` - cart with items

**Returns:**
- `*Payment` - payment information with redirect URL
- `error` - error if something went wrong

#### Checkout
```go
Checkout(payment *Payment, session string) (*Payment, error)
```
Verifies and updates payment status.

**Parameters:**
- `payment` - existing payment
- `session` - session ID from provider

**Returns:**
- `*Payment` - updated payment
- `error` - error if something went wrong

### Validation

#### Payment.Validate
```go
func (v Payment) Validate() error
```
Validates Payment structure (CartID must be 15 characters).

## Examples

### Example 1: Simple Payment via Stripe

```go
package main

import (
    "fmt"
    "github.com/shurco/mycart/pkg/litepay"
)

func main() {
    // Configuration
    pay := litepay.New(
        "https://myshop.com/payment/callback",
        "https://myshop.com/payment/success",
        "https://myshop.com/payment/cancel",
    )

    // Cart
    cart := litepay.Cart{
        ID:       "ORDER1234567890",
        Currency: "USD",
        Items: []litepay.Item{
            {
                PriceData: litepay.Price{
                    UnitAmount: 4999, // $49.99
                    Product: litepay.Product{
                        Name:        "E-book: Go Programming",
                        Description: "Complete guide to Go",
                        Images:      []string{"https://myshop.com/ebook.jpg"},
                    },
                },
                Quantity: 1,
            },
        },
    }

    // Create payment via Stripe
    stripe := pay.Stripe("sk_live_your_secret_key")
    payment, err := stripe.Pay(cart)
    if err != nil {
        panic(err)
    }

    // Redirect user
    fmt.Printf("Redirect user to: %s\n", payment.URL)
    fmt.Printf("Payment ID: %s\n", payment.MerchantID)
    fmt.Printf("Status: %s\n", payment.Status)
}
```

### Example 2: Handling Successful Payment

```go
// Handler for success URL
func PaymentSuccessHandler(w http.ResponseWriter, r *http.Request) {
    cartID := r.URL.Query().Get("cart_id")
    sessionID := r.URL.Query().Get("session") // From Stripe
    
    // Load payment information from DB
    payment := loadPaymentFromDB(cartID)
    
    // Check status with provider
    pay := litepay.New("", "", "")
    stripe := pay.Stripe("sk_live_your_secret_key")
    
    updatedPayment, err := stripe.Checkout(payment, sessionID)
    if err != nil {
        http.Error(w, "Payment verification failed", 500)
        return
    }
    
    // Check status
    if updatedPayment.Status == litepay.PAID {
        // Fulfill order
        fulfillOrder(cartID)
        fmt.Fprintf(w, "Thank you for your purchase!")
    } else {
        fmt.Fprintf(w, "Payment status: %s", updatedPayment.Status)
    }
}
```

### Example 3: Handling Webhook from SpectroCoin

```go
// Handler for callback URL
func PaymentCallbackHandler(w http.ResponseWriter, r *http.Request) {
    var callback litepay.CallbackSpectrocoin
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid request", 400)
        return
    }
    
    // Bind form data
    callback.OrderID = r.FormValue("orderId")
    callback.Status, _ = strconv.Atoi(r.FormValue("status"))
    callback.PayAmount, _ = strconv.ParseFloat(r.FormValue("payAmount"), 64)
    // ... other fields
    
    // Convert status
    status := litepay.StatusPayment(
        litepay.SPECTROCOIN, 
        strconv.Itoa(callback.Status),
    )
    
    // Update in DB
    updatePaymentStatus(callback.OrderID, status)
    
    // If paid - fulfill order
    if status == litepay.PAID {
        fulfillOrder(callback.OrderID)
    }
    
    // Respond to provider
    w.WriteHeader(200)
}
```

### Example 4: Multiple Providers

```go
func CreatePayment(provider string, cart litepay.Cart) (*litepay.Payment, error) {
    pay := litepay.New(
        "https://myshop.com/callback",
        "https://myshop.com/success",
        "https://myshop.com/cancel",
    )
    
    var session litepay.LitePay
    
    switch litepay.PaymentSystem(provider) {
    case litepay.STRIPE:
        session = pay.Stripe(os.Getenv("STRIPE_SECRET_KEY"))
    case litepay.PAYPAL:
        session = pay.Paypal(
            os.Getenv("PAYPAL_CLIENT_ID"),
            os.Getenv("PAYPAL_SECRET_KEY"),
        )
    case litepay.SPECTROCOIN:
        session = pay.Spectrocoin(
            os.Getenv("SPECTROCOIN_MERCHANT_ID"),
            os.Getenv("SPECTROCOIN_PROJECT_ID"),
            os.Getenv("SPECTROCOIN_PRIVATE_KEY"),
        )
    case litepay.DUMMY:
        session = pay.Dummy()
    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
    
    return session.Pay(cart)
}
```

## Security

### Currency Validation
```go
// Each provider checks supported currencies
if !findInSlice(c.currency, strings.ToUpper(currency)) {
    return nil, errors.New("this currency is not supported")
}
```

### Signature Verification (SpectroCoin)
```go
// SpectroCoin uses RSA signature
signature, err := signMessage(body, privateKey)
if err != nil {
    return nil, err
}
```

### HTTPS
All requests to provider APIs use HTTPS.

### Key Storage
⚠️ **Important**: Never commit API keys to the repository. Use environment variables:

```go
stripe := pay.Stripe(os.Getenv("STRIPE_SECRET_KEY"))
```

## Testing

### Unit Tests
```bash
go test ./pkg/litepay/...
```

### Provider Test Modes
- **Stripe**: use `sk_test_` keys
- **PayPal**: sandbox API (`https://api.sandbox.paypal.com`)
- **Dummy**: always returns success

## Supported Currencies

All providers support:
- EUR (Euro)
- USD (US Dollar)
- GBP (British Pound)
- AUD (Australian Dollar)
- CAD (Canadian Dollar)
- JPY (Japanese Yen)
- CNY (Chinese Yuan)
- SEK (Swedish Krona)

## License

See LICENSE file in the project root.

## Additional Documentation

- [Payment Interface Customization](../../docs/payment-customization.md)

## Support

For questions and suggestions:
- GitHub Issues: https://github.com/shurco/mycart/issues
- Documentation: https://github.com/shurco/mycart
