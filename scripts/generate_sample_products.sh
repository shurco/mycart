#!/bin/bash

# Script to generate 200 sample products via API
# Usage: ./scripts/generate_sample_products.sh

API_URL="${API_URL:-http://localhost:8080}"
TOKEN="${ADMIN_TOKEN}"

if [ -z "$TOKEN" ]; then
  echo "Error: ADMIN_TOKEN environment variable not set"
  echo "Usage: ADMIN_TOKEN=your_token ./scripts/generate_sample_products.sh"
  exit 1
fi

echo "Creating 200 sample products..."

categories=("Electronics" "Clothing" "Books" "Home & Garden" "Sports" "Toys" "Food" "Beauty" "Automotive" "Music")
adjectives=("Premium" "Deluxe" "Classic" "Modern" "Vintage" "Professional" "Eco-Friendly" "Smart" "Portable" "Ultimate")
products=("Widget" "Gadget" "Device" "Tool" "Kit" "Set" "Bundle" "Package" "Collection" "System")

for i in {1..200}; do
  category=${categories[$((RANDOM % ${#categories[@]}))]}
  adjective=${adjectives[$((RANDOM % ${#adjectives[@]}))]}
  product=${products[$((RANDOM % ${#products[@]}))]}

  name="$adjective $category $product $i"
  price=$((RANDOM % 50000 + 1000))  # Random price between $10 and $500
  quantity=$((RANDOM % 100 + 1))

  description="High-quality $adjective ${product,,} perfect for your $category needs. Features advanced technology and superior craftsmanship."
  brief="Premium $category item - Item #$i"

  curl -s -X POST "$API_URL/api/_/products" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{
      \"name\": \"$name\",
      \"brief\": \"$brief\",
      \"description\": \"$description\",
      \"amount\": $price,
      \"quantity\": $quantity,
      \"active\": true,
      \"metadata\": [],
      \"attributes\": [\"$category\", \"Sample\"],
      \"digital\": {\"type\": \"\"}
    }" > /dev/null

  if [ $((i % 20)) -eq 0 ]; then
    echo "Created $i products..."
  fi
done

echo "✓ Successfully created 200 sample products!"
