# Testing PortOne Integration - Quick Guide

## 🚀 Quick Start (3 Steps)

### 1. Build and Start

```bash
# Make script executable (first time only)
chmod +x dev-start.sh

# Start development environment
./dev-start.sh -d
```

**Expected output:**
```
✅ myCart is running in the background!

Access points:
  • Storefront:    http://localhost:8080
  • Admin Panel:   http://localhost:8080/admin
```

### 2. Initial Setup

Open http://localhost:8080 in your browser and complete the installation:

- **Email**: admin@example.com
- **Password**: admin123 (or your choice)
- **Domain**: localhost:8080

Click **Install** and wait a few seconds.

### 3. Configure PortOne

1. Login at http://localhost:8080/admin
2. Go to **Settings → Payment**
3. Click on **PortOne** card
4. Enter your credentials:
   ```
   Store ID:     store-xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
   Channel Key:  channel-xxxxx
   API Secret:   secret-xxxxx
   ```
5. Toggle **Active** ON
6. Click **Save**

## ✅ Verify Installation

### Check Backend

```bash
# Test PortOne config endpoint
curl http://localhost:8080/api/cart/portone-config

# Expected response:
# {
#   "success": true,
#   "message": "PortOne config",
#   "result": {
#     "store_id": "your-store-id",
#     "channel_key": "your-channel-key"
#   }
# }
```

### Check Frontend

1. Go to http://localhost:8080/admin
2. Create a test product:
   - Name: "Test Product"
   - Price: 10.00
   - Active: ON
3. Visit storefront: http://localhost:8080
4. Add product to cart
5. Go to checkout
6. **Verify PortOne appears** as payment option

## 🧪 Test Payment Flow

### Prerequisites

1. **PortOne Test Account**: Sign up at https://portone.io
2. **Create Test Channel**: In PortOne console, create a test channel
3. **Get Test Credentials**: Copy Store ID, Channel Key, API Secret
4. **Configure in myCart**: Use the credentials in admin panel

### Test Transaction

1. **Add Product to Cart**
   - Visit http://localhost:8080
   - Click product
   - Click "ADD TO CART"

2. **Checkout**
   - Go to cart
   - Enter email: test@example.com
   - Select **PortOne** payment method
   - Click **CHECKOUT**

3. **Complete Payment**
   - PortOne payment modal should appear
   - Use test card:
     - Card: 4000-0000-0000-0002
     - Expiry: 12/25
     - CVC: 123
   - Submit payment

4. **Verify Success**
   - Should redirect to success page
   - Check admin panel for order

## 📊 View Logs

```bash
# Follow logs in real-time
./dev-start.sh logs

# Look for these log entries:
# - "PortOne config loaded"
# - "Payment verification successful"
# - "Webhook received"
```

## 🔧 Troubleshooting

### PortOne Not Showing

```bash
# Check if settings are saved
docker exec mycart-dev sqlite3 /lc_base/data.db \
  "SELECT key, value FROM setting WHERE key LIKE 'portone_%';"

# Should show:
# portone_active|true
# portone_store_id|your-store-id
# portone_channel_key|your-channel-key
# portone_api_secret|your-secret
```

### Payment Fails

1. **Check Browser Console** (F12) for JavaScript errors
2. **Check Backend Logs**: `./dev-start.sh logs`
3. **Verify Credentials**: Settings → Payment → PortOne
4. **Check PortOne Console**: View transaction status

### Container Issues

```bash
# Restart container
./dev-start.sh restart

# Rebuild from scratch
./dev-start.sh down
./dev-start.sh

# Reset database (WARNING: deletes all data)
rm -rf docker/lc_base/data.db*
./dev-start.sh restart
```

## 🌐 Test Webhook Locally

### Using ngrok

```bash
# Install ngrok from https://ngrok.com
# Run ngrok
ngrok http 8080

# Copy HTTPS URL (e.g., https://abc123.ngrok.io)
# Configure in PortOne Console:
# Webhook URL: https://abc123.ngrok.io/api/payment/portone/webhook
# Events: Transaction.Paid, Transaction.VirtualAccountIssued
```

### Verify Webhook

```bash
# Watch for webhook events in logs
./dev-start.sh logs | grep "webhook\|PortOne"

# Test webhook manually
curl -X POST http://localhost:8080/api/payment/portone/webhook \
  -H "Content-Type: application/json" \
  -H "PortOne-Signature: test-signature" \
  -d '{"type":"Transaction.Paid","data":{"paymentId":"test-123"}}'
```

## 📱 Test Different Payment Methods

PortOne supports multiple payment methods:

1. **Credit/Debit Cards**
   - Visa: 4000-0000-0000-0002
   - Mastercard: 5200-0000-0000-0007

2. **Virtual Account**
   - Select virtual account in PortOne UI
   - Note the account number
   - Simulate deposit in PortOne console

3. **Mobile Payments**
   - Samsung Pay (test mode)
   - Kakao Pay (test mode)

## 🎯 Production Deployment

Once testing is complete:

1. **Use Production Credentials**
   - Real Store ID
   - Real Channel Key
   - Real API Secret

2. **Configure Webhook**
   - Set webhook URL: `https://yourdomain.com/api/payment/portone/webhook`
   - In PortOne console

3. **Deploy**
   - Use production docker-compose.yml
   - Set up SSL/TLS
   - Configure proper domain

## 🛑 Stopping Development Environment

```bash
# Stop (keeps data)
./dev-start.sh stop

# Shutdown (removes containers)
./dev-start.sh down

# Clean everything (removes volumes - DELETES DATA)
docker compose -f docker-compose.dev.yml down -v
```

## 📚 Additional Resources

- **Full Setup Guide**: `DEV_SETUP.md`
- **Implementation Summary**: `PORTONE_IMPLEMENTATION_SUMMARY.md`
- **API Documentation**: `docs/api.md`
- **Payment Customization**: `docs/payment-customization.md`
- **PortOne Docs**: https://developers.portone.io/

## ✨ Features to Test

- [ ] Admin panel PortOne settings
- [ ] PortOne appears in payment methods
- [ ] Payment modal opens correctly
- [ ] Payment completes successfully
- [ ] Webhook receives notifications
- [ ] Order appears in admin panel
- [ ] Email notifications sent
- [ ] Multi-language support (EN/ZH)
- [ ] Payment verification with backend
- [ ] Error handling (invalid card, etc.)

## 🎉 Success Criteria

✅ PortOne configuration saves in admin panel  
✅ PortOne appears on checkout page  
✅ Payment modal opens when selected  
✅ Test payment completes successfully  
✅ Backend verifies payment correctly  
✅ Order is created in database  
✅ Customer receives confirmation email  

---

**Need Help?** Check `DEV_SETUP.md` for detailed troubleshooting guide.
