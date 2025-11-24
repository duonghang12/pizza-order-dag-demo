# Quick Start Guide

Get the demo running in 3 minutes!

## Prerequisites

- Docker & Docker Compose installed
- Go 1.21+ installed
- `jq` installed (for test script) - optional

## Steps

### 1. Start Temporal Server

```bash
docker-compose up -d
```

Wait 10 seconds for Temporal to be ready.

### 2. Install Dependencies

```bash
go mod download
```

### 3. Start Worker (Terminal 1)

```bash
go run worker/main.go
```

You should see:
```
Worker starting...
Task Queue: pizza-order-queue
Registered Workflows: PizzaOrderWorkflow

Waiting for workflow tasks...
```

### 4. Start API Server (Terminal 2)

```bash
go run main.go
```

You should see:
```
API Server starting on :8080

Endpoints:
  POST   /orders                         - Create new pizza order
  GET    /orders/{orderID}               - Get order status
  ...

Ready to accept requests...
```

### 5. Create Your First Order (Terminal 3)

```bash
# Create order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_name": "Alice"}'
```

You'll get a response like:
```json
{
  "order_id": "pizza-orders/abc-123",
  "state": "IN_PROGRESS",
  "components": [
    {"type": "PAYMENT", "state": "INCOMPLETE", "dependsOn": []},
    {"type": "MAKE_DOUGH", "state": "NEEDS_INIT", "dependsOn": ["PAYMENT"]},
    ...
  ]
}
```

### 6. Complete Steps

```bash
# Save order ID from previous response
ORDER_ID="pizza-orders/abc-123"  # Replace with your actual ID

# Complete payment
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/payment

# Make dough
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/make-dough

# Add toppings
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/add-toppings

# Bake
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/bake

# Deliver
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/deliver
```

### 7. Check Status Anytime

```bash
curl http://localhost:8080/orders/${ORDER_ID#pizza-orders/}
```

### 8. View in Temporal UI

Open http://localhost:8233 in your browser

You can see:
- All running workflows
- Workflow history (every event)
- Input/output of each step

## Automated Test

Run the complete flow automatically:

```bash
./test-flow.sh
```

## What to Notice

1. **DAG State Changes**: After completing PAYMENT, notice MAKE_DOUGH automatically changes to INCOMPLETE
2. **Persistence**: Kill the worker, check status (still works!), restart worker
3. **Dependency Enforcement**: Try to skip steps - it will fail
4. **Long-Running**: Leave a workflow incomplete for hours/days - it will still be there!

## Next Steps

Read TUTORIAL.md to understand:
- How the DAG works
- How Temporal stores state
- How resumption works
- Where everything is stored

## Cleanup

```bash
docker-compose down -v
```
