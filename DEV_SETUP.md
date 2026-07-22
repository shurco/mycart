# Development Setup for PortOne Integration Testing

This guide will help you build and test the PortOne payment gateway integration in a Docker container.

## Quick Start

### Option 1: Using Docker Compose (Recommended)

```bash
# Build and start the development container
docker compose -f docker-compose.dev.yml up --build

# Or run in detached mode
docker compose -f docker-compose.dev.yml up --build -d

# View logs
docker compose -f docker-compose.dev.yml logs -f mycart-dev

# Stop the containers
docker compose -f docker-compose.dev.yml down
```

### Option 2: Using Docker directly

```bash
# Build the image
docker build -f Dockerfile.dev -t mycart:dev .

# Run the container
docker run -d \
  --name mycart-dev \
  -p 8080:8080 \
  -v $(pwd)/docker/lc_base:/lc_base \
  -v $(pwd)/docker/lc_digitals:/lc_digitals \
  -v $(pwd)/docker/lc_uploads:/lc_uploads \
  mycart:dev

# View logs
docker logs -f mycart-dev

# Stop and remove
docker stop mycart-dev && docker rm mycart-dev
```

## Access Points

After starting the container:

- **Admin Panel**: http://localhost:8080/admin
- **Storefront**: http://localhost:8080
- **API**: http://localhost:8080/api
- **Health Check**: http://localhost:8080/ping

With Nginx proxy (if using docker-compose):
- **Admin Panel**: http://localhost/admin
- **Storefront**: http://localhost

## Initial Setup

### 1. First Time Installation

When you first access the application, you'll be prompted to install:

1. Navigate to http://localhost:8080
2. Complete the installation wizard:
   - Set admin email
   - Set admin password
   - Set domain (use `localhost:8080` or your actual domain)
3. Click "Install"

### 2. Configure PortOne

After installation:

1. Login to admin panel: http://localhost:8080/admin
2. Navigate to **Settings → Payment → PortOne**
3. Enter your PortOne credentials:
   - **Store ID**: Your PortOne store ID (format: `store-xxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`)
   - **Channel Key**: Your PortOne channel key
   - **API Secret**: Your PortOne V2 API secret
4. Toggle **Active** to enable PortOne
5. Click **Save**

### 3. Test Payment Flow

#### Using PortOne Test Environment

1. Sign up at https://portone.io
2. Create a test store and channel
3. Use test credentials in myCart admin panel
4. Add a product in admin panel
5. Go to storefront and add product to cart
6. At checkout, select PortOne as payment method
7. Use PortOne test card numbers:
   - Card: 4000-0000-0000-0002 (Visa)
   - Expiry: Any future date
   - CVC: Any 3 digits

## Verifying PortOne Integration

### Check Database Migration

```bash
# Access the container
docker exec -it mycart-dev sh

# Check if PortOne settings exist in database
ls -la /lc_base/data.db

# Exit container
exit
```

### Check Backend Endpoints

```bash
# Check PortOne config endpoint
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

# Check payment methods
curl http://localhost:8080/api/cart/payment

# Expected response should include:
# {
#   "success": true,
#   "message": "Payment list",
#   "result": {
#     ...
#     "portone": true,
#     ...
#   }
# }
```

### Check Frontend Integration

1. **Admin Panel**:
   - Check Settings → Payment → PortOne is visible
   - Verify all form fields are present
   - Test save/load functionality

2. **Storefront**:
   - Add product to cart
   - Go to checkout
   - Verify PortOne appears in payment methods
   - Check PortOne description displays correctly

## Troubleshooting

### Container won't start

```bash
# Check logs
docker compose -f docker-compose.dev.yml logs mycart-dev

# Common issues:
# 1. Port 8080 already in use - change port in docker-compose.dev.yml
# 2. Build failed - check Go/Node versions
# 3. Permission issues - check volume permissions
```

### Database Issues

```bash
# Reset database (WARNING: deletes all data)
rm -rf docker/lc_base/data.db*

# Restart container to trigger fresh installation
docker compose -f docker-compose.dev.yml restart mycart-dev
```

### Frontend Build Issues

```bash
# Rebuild with fresh dependencies
docker compose -f docker-compose.dev.yml build --no-cache
```

### PortOne Not Appearing

1. Check admin settings were saved:
   ```bash
   docker exec mycart-dev cat /lc_base/data.db | strings | grep portone
   ```

2. Check browser console for errors
3. Verify API endpoints return correct data
4. Check that Active toggle is ON in admin panel

## Development Tips

### Hot Reload (Not Available in Current Setup)

For development with hot reload, you would need to:
1. Run the Go backend with `air` or similar tool
2. Run frontend dev servers separately
3. Configure CORS appropriately

### Viewing Logs

```bash
# Follow all logs
docker compose -f docker-compose.dev.yml logs -f

# Follow specific service
docker compose -f docker-compose.dev.yml logs -f mycart-dev

# Last 100 lines
docker compose -f docker-compose.dev.yml logs --tail=100 mycart-dev
```

### Accessing Database

```bash
# Install sqlite3 in container
docker exec -it mycart-dev apk add sqlite

# Query database
docker exec -it mycart-dev sqlite3 /lc_base/data.db "SELECT * FROM setting WHERE key LIKE 'portone_%';"
```

## Testing Webhook

To test PortOne webhook locally:

### Option 1: Using ngrok

```bash
# Install ngrok
# https://ngrok.com/download

# Expose local port
ngrok http 8080

# Copy the HTTPS URL (e.g., https://abc123.ngrok.io)
# Configure in PortOne console:
# Webhook URL: https://abc123.ngrok.io/api/payment/portone/webhook
```

### Option 2: Using localhost.run

```bash
# SSH tunnel
ssh -R 80:localhost:8080 localhost.run

# Use provided URL in PortOne webhook configuration
```

### Verify Webhook

```bash
# Check webhook logs
docker compose -f docker-compose.dev.yml logs -f mycart-dev | grep webhook
```

## Production Deployment

When ready for production:

1. Use `docker-compose.yml` instead of `docker-compose.dev.yml`
2. Set proper environment variables
3. Use production PortOne credentials
4. Configure proper domain and SSL
5. Set up proper webhook URL

## Clean Up

```bash
# Stop and remove containers
docker compose -f docker-compose.dev.yml down

# Remove volumes (deletes all data)
docker compose -f docker-compose.dev.yml down -v

# Remove images
docker rmi mycart:dev
```

## Support

- Implementation Summary: `PORTONE_IMPLEMENTATION_SUMMARY.md`
- API Documentation: `docs/api.md`
- Payment Customization: `docs/payment-customization.md`
- GitHub Issues: https://github.com/shurco/mycart/issues
