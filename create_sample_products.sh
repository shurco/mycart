#!/bin/bash

# Simple script to create 200 sample products via API
# First start the server: ./mycart serve
# Then get your auth token from admin login
# Run: ./create_sample_products.sh YOUR_TOKEN

TOKEN="$1"
API_URL="http://localhost:8080"

if [ -z "$TOKEN" ]; then
  echo "Usage: ./create_sample_products.sh YOUR_AUTH_TOKEN"
  exit 1
fi

for i in {1..200}; do
  name="Sample Product $i"
  price=$((1000 + i * 100))
  
  curl -s -X POST "$API_URL/api/_/products" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{
      \"name\": \"$name\",
      \"brief\": \"Sample product #$i\",
      \"description\": \"This is a sample product for testing.\",
      \"amount\": $price,
      \"quantity\": 100,
      \"active\": true
    }" > /dev/null
  
  [ $((i % 20)) -eq 0 ] && echo "Created $i products..."
done

echo "✓ Created 200 products!"
