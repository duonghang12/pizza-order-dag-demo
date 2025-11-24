#!/bin/bash

# Demo script showing complete pizza order flow

set -e

echo "===================================="
echo "Pizza Order DAG Demo"
echo "===================================="
echo ""

# 1. Create a new order
echo "1. Creating new pizza order..."
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_name": "Alice"}')

ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.order_id')
echo "   Order created: $ORDER_ID"
echo "   Initial state:"
echo "$ORDER_RESPONSE" | jq '.components[] | {type, state, dependsOn}'
echo ""

sleep 2

# 2. Check status
echo "2. Checking order status..."
curl -s http://localhost:8080/orders/${ORDER_ID#pizza-orders/} | jq '.components[] | {type, state}'
echo ""

sleep 2

# 3. Complete payment
echo "3. Completing payment..."
curl -s -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/payment | jq '{
  step: "payment",
  next_step: (.components[] | select(.state == "INCOMPLETE") | .type)
}'
echo ""

sleep 2

# 4. Make dough
echo "4. Making dough..."
curl -s -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/make-dough | jq '{
  step: "make-dough",
  next_step: (.components[] | select(.state == "INCOMPLETE") | .type)
}'
echo ""

sleep 2

# 5. Add toppings
echo "5. Adding toppings..."
curl -s -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/add-toppings | jq '{
  step: "add-toppings",
  next_step: (.components[] | select(.state == "INCOMPLETE") | .type)
}'
echo ""

sleep 2

# 6. Bake pizza
echo "6. Baking pizza..."
curl -s -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/bake | jq '{
  step: "bake",
  next_step: (.components[] | select(.state == "INCOMPLETE") | .type)
}'
echo ""

sleep 2

# 7. Deliver
echo "7. Delivering pizza..."
curl -s -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/deliver | jq '{
  step: "deliver",
  state: .state
}'
echo ""

sleep 2

# 8. Final status
echo "8. Final order status:"
curl -s http://localhost:8080/orders/${ORDER_ID#pizza-orders/} | jq '{
  order_id,
  customer_name,
  state,
  components: [.components[] | {type, state, completeTime}]
}'
echo ""

echo "===================================="
echo "Pizza order complete! 🍕"
echo "===================================="
echo ""
echo "Check Temporal UI: http://localhost:8233"
echo "Workflow ID: $ORDER_ID"
