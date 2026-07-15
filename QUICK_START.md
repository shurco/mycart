# 🚀 Quick Start - Testing PortOne Integration

## One-Line Start

```bash
./dev-start.sh -d && sleep 5 && ./test-portone.sh
```

This will:
1. Build and start myCart in Docker
2. Run integration tests
3. Show you what to do next

---

## Manual 3-Step Start

### Step 1: Start myCart

```bash
./dev-start.sh -d
```

**Wait 10-15 seconds** for the build to complete.

### Step 2: Open Browser

Go to: **http://localhost:8080**

Complete installation:
- Email: `admin@example.com`
- Password: `admin123`
- Domain: `localhost:8080`

### Step 3: Configure PortOne

1. Login at **http://localhost:8080/admin**
2. Go to **Settings → Payment**
3. Click **PortOne** card
4. Enter credentials and toggle Active ON
5. Click **Save**

---

## Test Payment

1. Visit **http://localhost:8080**
2. Add product to cart
3. Checkout → Select **PortOne**
4. Enter email and click **CHECKOUT**
5. Use test card: `4000-0000-0000-0002`

---

## Useful Commands

```bash
# View logs
./dev-start.sh logs

# Test endpoints
./test-portone.sh

# Restart
./dev-start.sh restart

# Stop
./dev-start.sh stop

# Remove everything
./dev-start.sh down
```

---

## PortOne Test Credentials

Get your test credentials from:
**https://portone.io** → Console → Stores → Your Store

You need:
- Store ID (format: `store-xxxxx-xxxx-...`)
- Channel Key
- API Secret (V2)

---

## Verify Integration

### Backend Check
```bash
curl http://localhost:8080/api/cart/portone-config
# Should return: {"success":true,"result":{"store_id":"...","channel_key":"..."}}
```

### Frontend Check
1. Go to cart page
2. Look for "PortOne" in payment methods
3. Should see description: "Credit card, virtual account, and mobile payment"

---

## Troubleshooting

### PortOne not showing?
```bash
# Check settings
docker exec mycart-dev sqlite3 /lc_base/data.db \
  "SELECT * FROM setting WHERE key LIKE 'portone_%';"

# Check if Active is 'true'
```

### Build failed?
```bash
# Clean rebuild
./dev-start.sh down
docker system prune -f
./dev-start.sh
```

### Need fresh database?
```bash
rm -rf docker/lc_base/data.db*
./dev-start.sh restart
```

---

## Access Points

- **Storefront**: http://localhost:8080
- **Admin**: http://localhost:8080/admin  
- **API**: http://localhost:8080/api
- **Health**: http://localhost:8080/ping

---

## More Documentation

- **Detailed Setup**: `DEV_SETUP.md`
- **Testing Guide**: `TESTING_PORTONE.md`
- **Implementation**: `PORTONE_IMPLEMENTATION_SUMMARY.md`
- **API Docs**: `docs/api.md`

---

**That's it! You're ready to test PortOne integration.** 🎉
