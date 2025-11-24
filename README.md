# Pizza Order DAG Demo

A simple project demonstrating Temporal workflows with a DAG (Directed Acyclic Graph) for managing component dependencies.

## Concept

This project simulates a pizza ordering system where each step depends on previous steps:

```
Payment → Make Dough → Add Toppings → Bake Pizza → Deliver
```

Each step is a "component" in the DAG, and the workflow waits for user actions to complete each step.

## Architecture

```
Mobile/Web App  →  HTTP Server  →  Temporal Client  →  Temporal Server
                                                              ↓
                                                         Worker Process
                                                         (Runs Workflows)
                                                              ↓
                                                          Database
                                                    (Stores Workflow State)
```

## Prerequisites

- Docker and Docker Compose
- Go 1.21+

## Quick Start

### 1. Start Temporal Server

```bash
docker-compose up -d
```

This starts:
- Temporal Server (localhost:7233)
- Temporal UI (http://localhost:8233)

### 2. Start the Worker

```bash
go run worker/main.go
```

The worker listens for workflow tasks from Temporal.

### 3. Start the API Server

```bash
go run main.go
```

API server runs on http://localhost:8080

## API Endpoints

### Create a New Pizza Order

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_name": "John Doe"}'
```

Response:
```json
{
  "order_id": "pizza-orders/abc-123",
  "state": "IN_PROGRESS",
  "components": [
    {"type": "PAYMENT", "state": "INCOMPLETE", "dependsOn": []},
    {"type": "MAKE_DOUGH", "state": "NEEDS_INIT", "dependsOn": ["PAYMENT"]},
    {"type": "ADD_TOPPINGS", "state": "NEEDS_INIT", "dependsOn": ["MAKE_DOUGH"]},
    {"type": "BAKE_PIZZA", "state": "NEEDS_INIT", "dependsOn": ["ADD_TOPPINGS"]},
    {"type": "DELIVER", "state": "NEEDS_INIT", "dependsOn": ["BAKE_PIZZA"]}
  ]
}
```

### Get Order Status

```bash
curl http://localhost:8080/orders/pizza-orders/abc-123
```

### Complete Payment

```bash
curl -X POST http://localhost:8080/orders/pizza-orders/abc-123/payment
```

### Make Dough

```bash
curl -X POST http://localhost:8080/orders/pizza-orders/abc-123/make-dough
```

### Add Toppings

```bash
curl -X POST http://localhost:8080/orders/pizza-orders/abc-123/add-toppings
```

### Bake Pizza

```bash
curl -X POST http://localhost:8080/orders/pizza-orders/abc-123/bake
```

### Deliver Pizza

```bash
curl -X POST http://localhost:8080/orders/pizza-orders/abc-123/deliver
```

## Example Flow

```bash
# 1. Create order
ORDER_ID=$(curl -s -X POST http://localhost:8080/orders -H "Content-Type: application/json" -d '{"customer_name": "Alice"}' | jq -r '.order_id')

echo "Order created: $ORDER_ID"

# 2. Check status
curl http://localhost:8080/orders/$ORDER_ID | jq

# 3. Complete payment
curl -X POST http://localhost:8080/orders/$ORDER_ID/payment

# 4. Close the terminal and reopen it tomorrow...
# The workflow is still running in Temporal!

# 5. Check status again (DAG state is preserved!)
curl http://localhost:8080/orders/$ORDER_ID | jq

# 6. Continue with next steps
curl -X POST http://localhost:8080/orders/$ORDER_ID/make-dough
curl -X POST http://localhost:8080/orders/$ORDER_ID/add-toppings
curl -X POST http://localhost:8080/orders/$ORDER_ID/bake
curl -X POST http://localhost:8080/orders/$ORDER_ID/deliver

# 7. Final status - order complete!
curl http://localhost:8080/orders/$ORDER_ID | jq
```

## Key Concepts Demonstrated

### 1. DAG (Directed Acyclic Graph)
- Components with dependencies (`types/dag.go`)
- Automatic dependency checking
- State transitions (NEEDS_INIT → INCOMPLETE → COMPLETED)

### 2. Temporal Workflow
- Long-running workflow that waits for user actions
- State persists across server restarts
- Workflow can run for hours, days, or weeks

### 3. Temporal Primitives
- **Updates**: Modify workflow state (complete steps)
- **Queries**: Read current state without modifications
- **Await**: Block until all components complete

### 4. State Persistence
- All state stored in Temporal's database
- Workflow can be queried anytime to get current status
- Resume from any point even after app restarts

## Temporal UI

View your workflows at: http://localhost:8233

You can see:
- Running workflows
- Workflow history (all events)
- Current state
- Pending tasks

## Project Structure

```
├── main.go              # HTTP API server
├── worker/main.go       # Temporal worker
├── types/
│   ├── dag.go          # DAG implementation
│   └── models.go       # Data structures
├── workflow/
│   ├── pizza_workflow.go  # Temporal workflow definition
│   └── activities.go      # Temporal activities (none in this demo)
└── docker-compose.yml     # Temporal server setup
```

## How It Works

1. **Create Order**: Starts a Temporal workflow with initial DAG
2. **Workflow Waits**: Uses `workflow.Await()` to block until all components complete
3. **User Actions**: Send HTTP requests to complete each step
4. **Updates**: Each request triggers a Temporal Update that modifies the DAG
5. **State Persists**: Temporal stores the updated DAG in its database
6. **Query State**: GET requests use Temporal Query to read current state
7. **Completion**: When all components are done, workflow completes

## Troubleshooting

**Worker not connecting to Temporal:**
```bash
# Check if Temporal is running
docker-compose ps

# View Temporal logs
docker-compose logs temporal
```

**Port already in use:**
```bash
# Change ports in docker-compose.yml and main.go
```

**Workflow not found:**
```bash
# Make sure worker is running
# Check Temporal UI for workflow status
```
