# PortOne Payment Gateway Integration Design

**Date:** 2026-07-14  
**Status:** Approved  
**Implementation Approach:** Browser SDK with Backend Verification

## Overview

This design adds PortOne as a payment gateway option to mycart. Unlike existing providers (Stripe, PayPal) that use server-initiated flows through the `pkg/litepay` abstraction, PortOne uses a browser-first architecture where the frontend SDK handles payment UI and flows directly, with the backend providing verification and webhook handling.

## Architecture

### Three-Layer Integration

1. **Browser Layer** (Svelte site frontend)
   - `@portone/browser-sdk` NPM package handles all payment UI and flows
   - User clicks "Pay" → SDK opens payment window → completes in browser
   - No backend redirect needed - payment happens entirely client-side
   - Frontend passes payment result to backend for verification

2. **Verification Layer** (Go backend API)
   - `POST /api/payment/portone/complete` - Verifies payment after browser completes
   - `POST /api/payment/portone/webhook` - Handles PortOne async notifications
   - Direct HTTP calls to PortOne REST API (`https://api.portone.io`)
   - Validates payment amount, currency, items match cart before marking as paid
   - No `pkg/litepay` provider - bypasses abstraction due to different paradigm

3. **Configuration Layer** (Settings + Admin UI)
   - Database stores: `portone_store_id`, `portone_channel_key`, `portone_api_secret`, `portone_active`
   - Admin UI component matches PayPal/Stripe pattern
   - Settings API endpoint `/api/private/setting/portone`

### Key Architectural Difference

**Existing Providers (Stripe/PayPal):**
```
User → Backend calls litepay.Pay() → Redirect to payment page → 
Backend handles callback → Update cart
```

**PortOne:**
```
User → Frontend calls PortOne SDK → Browser completes payment → 
Backend verifies result → Update cart
```

PortOne bypasses `pkg/litepay` because the browser drives the flow, not the server.

## Data Model

### Settings Model

**File:** `internal/models/setting.go`

```go
// Portone payment gateway settings
type Portone struct {
    StoreID    string `json:"store_id"`
    ChannelKey string `json:"channel_key"`
    ApiSecret  string `json:"api_secret"`
    Active     bool   `json:"active"`
}

// Validate portone settings
func (v Portone) Validate() error {
    return validation.ValidateStruct(&v,
        validation.Field(&v.StoreID, validation.Length(24, 50)),
        validation.Field(&v.ChannelKey, validation.Length(20, 100)),
        validation.Field(&v.ApiSecret, validation.Length(30, 200)),
    )
}
```

**Update PaymentSystem struct:**
```go
type PaymentSystem struct {
    Active      []string
    Stripe      Stripe
    Paypal      Paypal
    Spectrocoin Spectrocoin
    Coinbase    Coinbase
    Portone     Portone  // New
    Dummy       Dummy
}

func (v PaymentSystem) Validate() error {
    return validation.ValidateStruct(&v,
        validation.Field(&v.Stripe),
        validation.Field(&v.Paypal),
        validation.Field(&v.Spectrocoin),
        validation.Field(&v.Coinbase),
        validation.Field(&v.Portone),  // New
    )
}
```

### Database Schema

**Migration:** `migrations/YYYYMMDDHHMMSS_portone.sql`

```sql
-- PortOne payment gateway settings
INSERT INTO setting VALUES ('portone_001', 'portone_active', 'false');
INSERT INTO setting VALUES ('portone_002', 'portone_store_id', '');
INSERT INTO setting VALUES ('portone_003', 'portone_channel_key', '');
INSERT INTO setting VALUES ('portone_004', 'portone_api_secret', '');
```

**Query Layer Updates:**

**File:** `internal/queries/setting.go`

Add to `GroupFieldMap()`:
```go
case *models.Portone:
    return map[string]any{
        "portone_store_id":    &s.StoreID,
        "portone_channel_key": &s.ChannelKey,
        "portone_api_secret":  &s.ApiSecret,
        "portone_active":      &s.Active,
    }
```

**File:** `internal/queries/cart.go`

Update payment methods query to include `portone_active`.

## Backend API Implementation

### 1. Settings Registry

**File:** `internal/handlers/private/setting_registry.go`

```go
var settingRegistry = map[string]func() any{
    // ... existing providers ...
    "portone": func() any { return &models.Portone{} },
}
```

Enables `GET/PUT /api/private/setting/portone` automatically.

### 2. Payment Verification Endpoint

**File:** `internal/handlers/public/payment_portone.go`

**Endpoint:** `POST /api/payment/portone/complete`

**Request Body:**
```json
{
  "payment_id": "payment-xxxxx",
  "cart_id": "abc123"
}
```

**Implementation Flow:**

1. Parse request to get `payment_id` and `cart_id`
2. Load cart from database using `cart_id`
3. Load PortOne settings (store_id, api_secret) from database
4. Call PortOne API: `GET https://api.portone.io/payments/{payment_id}`
   - Add header: `Authorization: PortOne {api_secret}`
5. Verify payment details:
   - Payment status is "PAID" or "VIRTUAL_ACCOUNT_ISSUED"
   - Amount matches cart total
   - Currency matches cart currency
   - Custom data contains matching `cart_id`
6. If valid: Update cart status to paid, return success
7. If invalid: Log security warning, return 400 error

**Response:**
```json
{
  "success": true,
  "status": "PAID"
}
```

**Helper Functions:**

```go
// getPortoneSettings loads PortOne credentials from database
func getPortoneSettings(db *sql.DB) (*models.Portone, error)

// verifyPortonePayment calls PortOne API and validates payment
func verifyPortonePayment(paymentID, cartID string, cart *models.Cart, settings *models.Portone) error

// callPortoneAPI makes authenticated HTTP request to PortOne
func callPortoneAPI(endpoint, apiSecret string) (*http.Response, error)
```

### 3. Webhook Endpoint

**Endpoint:** `POST /api/payment/portone/webhook`

**Purpose:** Handle async payment notifications (virtual account deposits, delayed confirmations)

**Implementation Flow:**

1. Read raw request body
2. Verify webhook signature using PortOne webhook secret
   - Extract signature from `PortOne-Signature` header
   - Compute HMAC-SHA256 of raw body using `api_secret`
   - Compare computed signature with header value (constant-time comparison)
3. Parse webhook JSON payload
4. Extract `payment_id` from payload
5. Call shared `verifyPortonePayment()` logic
6. Return `200 OK` to acknowledge receipt

**Signature Verification:**
```go
func verifyWebhookSignature(body []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expectedSignature := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
```

**Note:** Actual webhook signature format will be verified against PortOne documentation during implementation. If different from HMAC-SHA256, update this function accordingly.

### PortOne API Reference

**Base URL:** `https://api.portone.io`

**Authentication:** 
- Header: `Authorization: PortOne {api_secret}`

**Get Payment Details:**
```
GET /payments/{paymentId}
```

**Response:**
```json
{
  "id": "payment-xxxxx",
  "status": "PAID",
  "orderName": "Order #123",
  "amount": {
    "total": 50000,
    "currency": "KRW"
  },
  "customData": "{\"cart_id\":\"abc123\"}",
  "channel": {
    "type": "LIVE"
  }
}
```

## Frontend Implementation

### Admin Panel

**Component:** `web/admin/src/lib/components/payment/Portone.svelte`

Structure mirrors `Paypal.svelte`:

```svelte
<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import FormInput from '../form/Input.svelte'
  import FormToggle from '../form/Toggle.svelte'
  import { loadPaymentSettings, savePaymentSettings, togglePaymentActive } from '$lib/composables/usePaymentSettings'
  import { systemStore } from '$lib/stores/system'
  import type { PortoneSettings } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  let t = $derived($translate)
  let { onclose }: Props = $props()

  let settings = $state<PortoneSettings>({
    active: false,
    store_id: '',
    channel_key: '',
    api_secret: ''
  })

  onMount(async () => {
    settings = await loadPaymentSettings<PortoneSettings>('portone', settings)
    // Subscribe to active state changes
  })

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    // Validate fields
    await savePaymentSettings('portone', settings)
  }

  async function handleToggleActive() {
    await togglePaymentActive('portone', settings.active)
  }
</script>

<div>
  <h1>PortOne</h1>
  <FormToggle bind:value={settings.active} onchange={handleToggleActive} />
  
  <form onsubmit={handleSubmit}>
    <FormInput id="store_id" title="Store ID" bind:value={settings.store_id} />
    <FormInput id="channel_key" title="Channel Key" bind:value={settings.channel_key} />
    <FormInput id="api_secret" type="password" title="API Secret" bind:value={settings.api_secret} />
    
    <FormButton type="submit" name={t('common.save')} />
    <FormButton type="button" name={t('common.close')} onclick={close} />
  </form>
</div>
```

**Integration Points:**

**File:** `web/admin/src/routes/settings/payment/+page.svelte`

1. Add drawer mode: `type DrawerMode = 'stripe' | 'paypal' | 'spectrocoin' | 'coinbase' | 'portone' | null`
2. Add PortOne card to payment provider list
3. Add drawer component: `{:else if drawerMode === 'portone'} <Portone {onclose} />`

**Validation Constants:** `web/admin/src/lib/constants/validation.ts`
```typescript
export const MIN_PORTONE_STORE_ID_LENGTH = 24
export const MIN_PORTONE_CHANNEL_KEY_LENGTH = 20
export const MIN_PORTONE_API_SECRET_LENGTH = 30

export const ERROR_MESSAGES = {
  // ... existing ...
  PORTONE_STORE_ID_TOO_SHORT: 'Store ID must be at least 24 characters',
  PORTONE_CHANNEL_KEY_TOO_SHORT: 'Channel Key must be at least 20 characters',
  PORTONE_API_SECRET_TOO_SHORT: 'API Secret must be at least 30 characters',
}
```

**Type Definitions:** `web/admin/src/lib/types/models.ts`
```typescript
export interface PortoneSettings {
  active: boolean
  store_id: string
  channel_key: string
  api_secret: string
}
```

**Translations:** Add to `web/admin/src/lib/i18n/locales/en.json` and `zh.json`:
```json
{
  "payment": {
    "portone": "PortOne",
    "storeId": "Store ID",
    "channelKey": "Channel Key"
  }
}
```

### Storefront

**Package Installation:**
```bash
cd web/site
npm install @portone/browser-sdk
```

**Cart Page Updates:** `web/site/src/routes/cart/+page.svelte`

**1. Import SDK:**
```typescript
import * as PortOne from '@portone/browser-sdk/v2'
```

**2. Load Settings:**
```typescript
let portoneStoreId = $state('')
let portoneChannelKey = $state('')

onMount(async () => {
  // Load PortOne public settings (store_id, channel_key)
  // These are safe to expose to frontend
  const res = await apiGet('/api/cart/portone-config')
  if (res.success) {
    portoneStoreId = res.result.store_id
    portoneChannelKey = res.result.channel_key
  }
})
```

**Note:** Need new endpoint `GET /api/cart/portone-config` that returns public settings (store_id, channel_key) but NOT api_secret.

**3. Add Payment Option UI:**
```svelte
{#if payments.portone}
  <input type="radio" bind:group={provider} value="portone" id="portone" class="peer hidden" />
  <label for="portone" class="...">
    <img src="/assets/img/payments/portone.svg" alt="PortOne" />
    <p class="text-xl font-black">{t('cart.portone')}</p>
    <p class="text-lg">{t('cart.portoneDescription')}</p>
  </label>
{/if}
```

**4. Payment Handler Logic:**
```typescript
async function checkOut(e: Event) {
  e.preventDefault()
  
  if (provider === 'portone') {
    showOverlay = true
    
    try {
      // Generate unique payment ID
      const paymentId = `payment-${crypto.randomUUID()}`
      
      // Call PortOne SDK
      const response = await PortOne.requestPayment({
        storeId: portoneStoreId,
        channelKey: portoneChannelKey,
        paymentId: paymentId,
        orderName: `Order ${cartId}`,
        totalAmount: cartTotal,
        currency: currency.toUpperCase(),
        customData: JSON.stringify({ cart_id: cartId })
      })
      
      // Check for payment errors
      if (response.code != null) {
        error = response.message
        showOverlay = true
        return
      }
      
      // Verify payment with backend
      const verifyRes = await apiPost('/api/payment/portone/complete', {
        payment_id: response.paymentId,
        cart_id: cartId
      })
      
      if (verifyRes.success) {
        // Clear cart and redirect to success page
        cartStore.set([])
        goto('/success')
      } else {
        error = 'Payment verification failed'
        showOverlay = true
      }
    } catch (err) {
      error = 'Payment failed. Please try again.'
      showOverlay = true
    }
  }
  
  // ... existing provider logic ...
}
```

**Type Definitions:** `web/site/src/lib/types/models.ts`
```typescript
export interface PaymentMethods {
  stripe?: boolean
  paypal?: boolean
  spectrocoin?: boolean
  coinbase?: boolean
  portone?: boolean  // New
}
```

**Translations:** `web/site/src/lib/i18n/locales/en.json` and `zh.json`
```json
{
  "cart": {
    "portone": "PortOne",
    "portoneDescription": "Credit card, virtual account, and mobile payment"
  }
}
```

**Payment Utilities:** `web/site/src/lib/utils/payment.ts`
```typescript
const PROVIDER_KEYS = ['stripe', 'paypal', 'spectrocoin', 'coinbase', 'portone'] as const
```

## Error Handling

### Payment Verification Failures

| Error Scenario | HTTP Code | Response | Action |
|----------------|-----------|----------|--------|
| Amount mismatch | 400 | `{"success": false, "message": "Amount mismatch"}` | Log security warning, prevent cart completion |
| Invalid payment_id | 404 | `{"success": false, "message": "Payment not found"}` | Log as potential fraud attempt |
| PortOne API timeout | 503 | `{"success": false, "message": "Service unavailable"}` | Allow retry |
| Payment pending | 400 | `{"success": false, "message": "Payment not completed"}` | User should retry payment |
| Webhook signature invalid | 401 | Empty response | Log security event |
| Currency mismatch | 400 | `{"success": false, "message": "Currency mismatch"}` | Log security warning |

### Frontend Error States

```typescript
// SDK returned error code
if (response.code != null) {
  error = response.message  // Show overlay with PortOne error message
}

// Backend verification failed
if (!verifyRes.success) {
  error = 'Payment verification failed. Contact support.'
}

// Network error
catch (err) {
  error = 'Connection error. Please try again.'
}
```

### Security Validations

**Critical Security Rules:**

1. **Always verify server-side** - Never trust frontend payment status
2. **Validate cart_id** - Check `customData.cart_id` matches request
3. **Amount/currency validation** - Must match cart before marking paid
4. **Webhook signature** - Always verify signature on webhook requests
5. **Audit logging** - Log all payment verification attempts with:
   - Timestamp
   - Payment ID
   - Cart ID
   - Verification result
   - IP address

**Example Verification Logic:**
```go
// Verify payment details match cart
if payment.Amount.Total != cart.Total {
    log.SecurityWarning("Amount mismatch", paymentID, cartID)
    return errors.New("amount mismatch")
}

if payment.Currency != cart.Currency {
    log.SecurityWarning("Currency mismatch", paymentID, cartID)
    return errors.New("currency mismatch")
}

customData := parseJSON(payment.CustomData)
if customData.CartID != cart.ID {
    log.SecurityWarning("Cart ID mismatch", paymentID, cartID)
    return errors.New("cart_id mismatch")
}
```

## Testing Strategy

### Unit Tests

**1. Models Test:** `internal/models/setting_test.go`
```go
func TestPortone_Validate(t *testing.T) {
    // Valid settings pass
    valid := models.Portone{
        StoreID: "store-12345678-1234-1234-1234-123456789012",
        ChannelKey: "channel-key-example",
        ApiSecret: "api-secret-example-long-enough",
    }
    if err := valid.Validate(); err != nil {
        t.Errorf("valid portone rejected: %v", err)
    }
    
    // Invalid settings fail
    invalid := models.Portone{
        StoreID: "short",  // Too short
        ChannelKey: "key",
        ApiSecret: "secret",
    }
    if err := invalid.Validate(); err == nil {
        t.Error("invalid portone accepted")
    }
}
```

**2. Handler Registry Test:** `internal/handlers/private/setting_registry_test.go`
```go
func TestSettingRegistry_Portone(t *testing.T) {
    factory, exists := settingRegistry["portone"]
    if !exists {
        t.Fatal("portone not in registry")
    }
    
    instance := factory()
    if _, ok := instance.(*models.Portone); !ok {
        t.Errorf("wrong type: got %T", instance)
    }
}
```

**3. Payment Verification Test:** `internal/handlers/public/payment_portone_test.go`
```go
func TestVerifyPortonePayment_Success(t *testing.T) {
    // Mock PortOne API server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/payments/test-payment-id" {
            json.NewEncoder(w).Encode(map[string]any{
                "id": "test-payment-id",
                "status": "PAID",
                "amount": map[string]any{
                    "total": 5000,
                    "currency": "USD",
                },
                "customData": `{"cart_id":"test-cart"}`,
            })
        }
    }))
    defer server.Close()
    
    // Test verification logic
    // ...
}

func TestVerifyPortonePayment_AmountMismatch(t *testing.T) {
    // Test that amount mismatch is detected
    // ...
}

func TestPortoneWebhook_ValidSignature(t *testing.T) {
    // Test webhook signature verification
    // ...
}
```

### Integration Tests

**Use PortOne Test/Sandbox Mode:**

1. **Test Credentials:** Obtain sandbox store_id, channel_key, api_secret from PortOne
2. **Full Payment Flow:**
   - Configure PortOne in admin panel (sandbox mode)
   - Add items to cart
   - Select PortOne payment
   - Complete test payment
   - Verify cart marked as paid
3. **Webhook Testing:**
   - Configure webhook URL in PortOne dashboard
   - Trigger payment events
   - Verify webhooks received and processed
4. **Virtual Account Flow:**
   - Test virtual account issuance
   - Simulate deposit notification
   - Verify payment completion
5. **Error Scenarios:**
   - Test payment cancellation
   - Test expired payment
   - Test insufficient funds

### Manual Testing Checklist

- [ ] Admin: Save PortOne settings with valid credentials
- [ ] Admin: Save fails with invalid credentials (too short)
- [ ] Admin: Toggle PortOne active/inactive
- [ ] Admin: PortOne toggle updates immediately in UI
- [ ] Site: PortOne appears in payment options when active
- [ ] Site: PortOne hidden when inactive
- [ ] Site: Complete test payment successfully
- [ ] Site: Payment amount matches cart total
- [ ] Site: Handle payment cancellation gracefully
- [ ] Site: Error overlay shows for failed payments
- [ ] Site: Free cart doesn't show PortOne option
- [ ] Backend: Webhook properly updates payment status
- [ ] Backend: Invalid webhook signature rejected
- [ ] Backend: Amount mismatch prevents payment completion
- [ ] Security: API secret never exposed to frontend
- [ ] Security: Payment verification logs created

## Documentation Updates

### 1. API Documentation

**File:** `docs/api.md`

**Add to Payment Providers Table:**
| Provider | Code | Description |
|----------|------|-------------|
| PortOne | `portone` | Korean payment gateway (cards, virtual accounts, mobile) |

**Add Endpoints:**

#### Get PortOne Public Config
```
GET /api/cart/portone-config
```

**Response:**
```json
{
  "success": true,
  "result": {
    "store_id": "store-xxxxx",
    "channel_key": "channel-xxxxx"
  }
}
```

#### Complete PortOne Payment
```
POST /api/payment/portone/complete

Request:
{
  "payment_id": "payment-xxxxx",
  "cart_id": "abc123"
}

Response:
{
  "success": true,
  "status": "PAID"
}
```

#### PortOne Webhook
```
POST /api/payment/portone/webhook

Headers:
  PortOne-Signature: <webhook_signature>

Body: (PortOne webhook payload)

Response: 200 OK
```

**Add to Settings Endpoint:**
```
GET/PUT /api/private/setting/portone

Response:
{
  "store_id": "store-xxxxx",
  "channel_key": "channel-xxxxx",
  "api_secret": "secret-xxxxx",
  "active": false
}
```

### 2. Payment Customization Guide

**File:** `docs/payment-customization.md`

**Add Icon Requirement:**
```markdown
### Payment Provider Icons

Required icons in `web/site/static/assets/img/payments/`:
- `stripe.svg`
- `paypal.svg`
- `spectrocoin.svg`
- `coinbase.svg`
- `portone.svg` ← **New**
```

**Update Provider Order:**
```typescript
export const PAYMENT_PROVIDER_ORDER = [
  'stripe',
  'paypal',
  'portone',  // New
  'spectrocoin',
  'coinbase'
] as const
```

**Add Configuration Section:**
```markdown
## PortOne Configuration

1. Sign up at [PortOne Developer](https://portone.io)
2. Create a store and channel
3. Copy credentials:
   - Store ID (format: `store-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
   - Channel Key
   - API Secret (V2 API)
4. Configure webhook URL: `https://yourdomain.com/api/payment/portone/webhook`
5. Enter credentials in mycart admin panel: Settings → Payment → PortOne

**Test Mode:**
Use PortOne's test channel for development. Test payments use sandbox credentials.

**Supported Payment Methods:**
- Credit/Debit Cards
- Virtual Accounts
- Mobile Payments (Samsung Pay, Apple Pay)
- Korean Bank Transfers
```

### 3. README

**File:** `README.md`

**Update Payment Providers List:**
```markdown
## Payment Providers

- [x] Payment Stripe
- [x] Payment PayPal
- [x] Payment SpectroCoin
- [x] Payment Coinbase
- [x] Payment PortOne ← **New**
```

**Add Setup Link:**
```markdown
For PortOne setup instructions, see [Payment Customization Guide](docs/payment-customization.md#portone-configuration).
```

## Implementation Checklist

### Backend
- [ ] Create `internal/models/setting.go` - Add `Portone` struct and validation
- [ ] Update `internal/models/setting.go` - Add `Portone` to `PaymentSystem`
- [ ] Create migration `migrations/YYYYMMDDHHMMSS_portone.sql`
- [ ] Update `internal/queries/setting.go` - Add `Portone` case to `GroupFieldMap()`
- [ ] Update `internal/queries/cart.go` - Include `portone_active` in payment methods query
- [ ] Update `internal/handlers/private/setting_registry.go` - Add `portone` to registry
- [ ] Create `internal/handlers/public/payment_portone.go`:
  - [ ] `GET /api/cart/portone-config` - Return public settings
  - [ ] `POST /api/payment/portone/complete` - Verify payment
  - [ ] `POST /api/payment/portone/webhook` - Handle webhooks
  - [ ] Helper: `getPortoneSettings()`
  - [ ] Helper: `verifyPortonePayment()`
  - [ ] Helper: `callPortoneAPI()`
  - [ ] Helper: `verifyWebhookSignature()`
- [ ] Create tests `internal/models/setting_test.go` - Test Portone validation
- [ ] Create tests `internal/handlers/private/setting_registry_test.go` - Test registry
- [ ] Create tests `internal/handlers/public/payment_portone_test.go` - Test verification

### Frontend - Admin
- [ ] Create `web/admin/src/lib/components/payment/Portone.svelte`
- [ ] Update `web/admin/src/routes/settings/payment/+page.svelte` - Add drawer mode
- [ ] Update `web/admin/src/lib/constants/validation.ts` - Add PortOne constants
- [ ] Update `web/admin/src/lib/types/models.ts` - Add `PortoneSettings` interface
- [ ] Update `web/admin/src/lib/i18n/locales/en.json` - Add translations
- [ ] Update `web/admin/src/lib/i18n/locales/zh.json` - Add translations
- [ ] Add icon `web/admin/public/assets/img/payments/portone.svg`

### Frontend - Site
- [ ] Run `npm install @portone/browser-sdk` in `web/site/`
- [ ] Update `web/site/src/routes/cart/+page.svelte`:
  - [ ] Import PortOne SDK
  - [ ] Load PortOne config on mount
  - [ ] Add PortOne payment option UI
  - [ ] Implement PortOne payment handler in `checkOut()`
  - [ ] Add error handling for PortOne payments
- [ ] Update `web/site/src/lib/types/models.ts` - Add `portone` to `PaymentMethods`
- [ ] Update `web/site/src/lib/utils/payment.ts` - Add `portone` to `PROVIDER_KEYS`
- [ ] Update `web/site/src/lib/i18n/locales/en.json` - Add translations
- [ ] Update `web/site/src/lib/i18n/locales/zh.json` - Add translations
- [ ] Add icon `web/site/static/assets/img/payments/portone.svg`

### Documentation
- [ ] Update `docs/api.md` - Add PortOne endpoints and examples
- [ ] Update `docs/payment-customization.md` - Add PortOne setup guide
- [ ] Update `README.md` - Add PortOne to providers list

### Testing
- [ ] Unit tests pass
- [ ] Integration test with PortOne sandbox
- [ ] Manual testing checklist complete

## Dependencies

**NPM Packages:**
- `@portone/browser-sdk` (installed in `web/site/`)

**Go Packages:**
- No new Go dependencies - uses standard library `net/http`, `encoding/json`, `crypto/hmac`

**External Services:**
- PortOne API: `https://api.portone.io`
- PortOne Developer Dashboard for credentials and webhook configuration

## Security Considerations

1. **API Secret Protection:**
   - Never expose `api_secret` to frontend
   - Only `store_id` and `channel_key` are public
   - Store secrets encrypted in database if possible

2. **Payment Verification:**
   - Always verify payment server-side before cart completion
   - Check amount, currency, cart_id match
   - Log all verification attempts for audit

3. **Webhook Security:**
   - Always verify webhook signature
   - Reject webhooks with invalid signatures
   - Log rejected webhooks as potential attacks

4. **SQL Injection:**
   - Use parameterized queries for all database operations
   - Never concatenate user input into SQL

5. **XSS Protection:**
   - Sanitize all user input in frontend
   - PortOne SDK handles payment UI (no custom HTML)

## Future Enhancements

1. **Payment Methods:**
   - Allow admin to select specific payment methods (card only, virtual account only, etc.)
   - Frontend passes `payMethod` parameter to SDK

2. **Installment Payments:**
   - Support Korean installment payments
   - Configure max installment months in settings

3. **Recurring Payments:**
   - Implement billing key issuance for subscriptions
   - Store billing keys securely

4. **Multi-Currency:**
   - Test with multiple currencies beyond KRW
   - Handle currency conversion if needed

5. **Payment Analytics:**
   - Track payment success/failure rates by provider
   - Dashboard showing PortOne vs other providers performance

6. **Refunds:**
   - Implement refund endpoint using PortOne API
   - Admin UI for refund management

## References

- [PortOne Documentation](https://developers.portone.io/)
- [PortOne Browser SDK](https://www.npmjs.com/package/@portone/browser-sdk)
- [PortOne REST API](https://developers.portone.io/api/rest-v2)
- [Flask Sample Project](/srv/portone-sample/flask-react)
