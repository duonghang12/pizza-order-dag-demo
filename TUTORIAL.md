# Pizza Order DAG Tutorial

This tutorial explains how the DAG and Temporal workflow work together, step by step.

## Understanding the Flow

### 1. The DAG Structure (types/dag.go)

The DAG is **just a regular Go struct** - nothing special about it!

```go
type DAG struct {
    components []*Component  // Just a slice of components!
}

type Component struct {
    Type         ComponentType   // PAYMENT, MAKE_DOUGH, etc.
    State        ComponentState  // NEEDS_INIT, INCOMPLETE, COMPLETED
    DependsOn    []ComponentType // Which steps must complete first
    UpdateTime   time.Time
    CompleteTime *time.Time      // nil until completed
}
```

**Key Point**: This is NOT a Temporal feature - it's your application data structure.

### 2. How State is Stored in Temporal

When you create an order, here's what happens:

```
1. HTTP POST /orders
   │
2. API calls: temporalClient.ExecuteWorkflow()
   │
3. Temporal Server creates workflow execution
   │
4. Worker picks up task and runs PizzaOrderWorkflow()
   │
5. Workflow creates state variable:
   state := &PizzaOrder{
       OrderID: "pizza-orders/abc-123",
       DAG: NewPizzaOrderDAG(),  ← Creates component graph
   }
   │
6. Workflow sets up query handler:
   workflow.SetQueryHandler("QueryOrderState", func() {
       return state  ← Returns the variable
   })
   │
7. Workflow waits:
   workflow.Await(func() bool {
       return state.DAG.AllComponentsCompleted()
   })
```

**At this point:**
- Workflow is BLOCKED waiting
- State variable exists in worker's memory
- Temporal stores initial events in database

### 3. User Completes Payment

```
1. HTTP POST /orders/abc-123/payment
   │
2. API calls: temporalClient.UpdateWorkflow("CompletePayment")
   │
3. Temporal Server finds running workflow
   │
4. Worker calls update handler:
   func() (*PizzaOrder, error) {
       state.DAG.CompleteComponent("PAYMENT")  ← Modifies variable
       return state  ← Returns modified state
   }
   │
5. Temporal serializes state to JSON:
   {
     "order_id": "pizza-orders/abc-123",
     "dag": {
       "components": [
         {"type": "PAYMENT", "state": "COMPLETED"},  ← Changed!
         {"type": "MAKE_DOUGH", "state": "INCOMPLETE"},  ← Auto-updated!
         ...
       ]
     }
   }
   │
6. Temporal STORES this JSON in database as event
   │
7. Response sent back to API
```

**Key Point**: The return value from the update handler is what gets stored!

### 4. User Closes App and Returns Later

```
1. HTTP GET /orders/abc-123
   │
2. API calls: temporalClient.QueryWorkflow("QueryOrderState")
   │
3. Temporal routes to Worker
   │
4. Worker has two options:

   Option A: Workflow still in memory
   - Just returns state variable directly

   Option B: Workflow was restarted
   - Replays events from database
   - Rebuilds state by:
     * Creating initial state
     * Applying each update event
     * Ends up with same state as before!
   │
5. Query handler returns state
   │
6. API returns to user - shows PAYMENT completed, MAKE_DOUGH ready
```

**Key Point**: State is preserved either in memory or reconstructed from events!

### 5. The DAG Updates Dependencies Automatically

When you complete PAYMENT, look what happens in the DAG:

```go
func (d *DAG) CompleteComponent(componentType ComponentType) error {
    // 1. Mark component as completed
    component.State = StateCompleted
    component.CompleteTime = &now

    // 2. Check all other components
    d.updateDependentComponents()  ← Magic happens here!
}

func (d *DAG) updateDependentComponents() {
    for _, component := range d.components {
        if component.State != StateNeedsInit {
            continue  // Skip already started components
        }

        // Check if all dependencies are complete
        allDepsComplete := true
        for _, depType := range component.DependsOn {
            dep := d.GetComponent(depType)
            if dep.State != StateCompleted {
                allDepsComplete = false
                break
            }
        }

        // Move to INCOMPLETE if ready!
        if allDepsComplete {
            component.State = StateIncomplete  ← Now user can work on it
        }
    }
}
```

**Example Timeline:**

```
Initial State:
  PAYMENT:      INCOMPLETE  (ready to start - no dependencies)
  MAKE_DOUGH:   NEEDS_INIT  (waiting for PAYMENT)
  ADD_TOPPINGS: NEEDS_INIT  (waiting for MAKE_DOUGH)

After completing PAYMENT:
  PAYMENT:      COMPLETED   ✓
  MAKE_DOUGH:   INCOMPLETE  ← Auto-changed! (dependency met)
  ADD_TOPPINGS: NEEDS_INIT  (still waiting)

After completing MAKE_DOUGH:
  PAYMENT:      COMPLETED   ✓
  MAKE_DOUGH:   COMPLETED   ✓
  ADD_TOPPINGS: INCOMPLETE  ← Auto-changed! (dependency met)
```

### 6. Where is Everything Stored?

```
┌─────────────────────────────────────────────────┐
│ YOUR APPLICATION (Go processes)                 │
│                                                 │
│  Worker Memory:                                 │
│    state = {                                    │
│      OrderID: "abc-123"                         │
│      DAG: { components: [...] }                 │
│    }                                            │
│                                                 │
└──────────────────┬──────────────────────────────┘
                   │ Updates sent via gRPC
                   ▼
┌─────────────────────────────────────────────────┐
│ TEMPORAL SERVER                                 │
│                                                 │
│  Stores events in database:                     │
│    Event 1: WorkflowStarted                     │
│    Event 2: UpdateCompleted {                   │
│               result: <state JSON>              │
│             }                                   │
│    Event 3: UpdateCompleted {                   │
│               result: <updated state JSON>      │
│             }                                   │
│                                                 │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│ DATABASE (SQLite in dev, PostgreSQL in prod)   │
│                                                 │
│  Table: workflow_executions                     │
│    workflow_id: pizza-orders/abc-123            │
│    state: RUNNING                               │
│    ...                                          │
│                                                 │
│  Table: history_events                          │
│    event_id: 1                                  │
│    event_type: WorkflowStarted                  │
│    event_id: 2                                  │
│    event_type: UpdateCompleted                  │
│    event_data: {"order_id": "abc-123", ...}  ←  │
│                                                 │
└─────────────────────────────────────────────────┘
```

### 7. The Key Temporal Methods

Here's what each Temporal method does:

```go
// START WORKFLOW
client.ExecuteWorkflow(ctx, options, PizzaOrderWorkflow, input)
// → Creates new workflow execution
// → Database: INSERT into workflow_executions
// → Worker starts running PizzaOrderWorkflow()

// QUERY (Read State)
client.QueryWorkflow(ctx, workflowID, "", "QueryOrderState")
// → Asks worker for current state
// → NO database write
// → Just returns state variable

// UPDATE (Modify State)
client.UpdateWorkflow(ctx, UpdateWorkflowOptions{
    WorkflowID: "abc-123",
    UpdateName: "CompletePayment",
})
// → Calls update handler in workflow
// → Handler modifies state and returns it
// → Temporal serializes return value to JSON
// → Database: INSERT new event with JSON
// → Returns updated state to caller

// AWAIT (Block Until Condition)
workflow.Await(ctx, func() bool {
    return state.DAG.AllComponentsCompleted()
})
// → Checks condition function
// → If false: creates timer, workflow yields (pauses)
// → No CPU used while waiting
// → When Update arrives: wakes up, checks condition again
// → If true: continues past Await
```

## Testing the Demo

### Test 1: Create and Complete Order

```bash
# Terminal 1: Start Temporal
make start-temporal

# Terminal 2: Start Worker
make start-worker

# Terminal 3: Start API Server
make start-server

# Terminal 4: Run test
make test
```

### Test 2: State Persistence

```bash
# 1. Create order and complete payment
ORDER_ID=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_name": "Bob"}' | jq -r '.order_id')

curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/payment

# 2. Kill the worker (Ctrl+C in Terminal 2)

# 3. Check status - STILL WORKS! (querying Temporal DB)
curl http://localhost:8080/orders/${ORDER_ID#pizza-orders/} | jq

# 4. Restart worker
make start-worker

# 5. Continue order - workflow resumes!
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/make-dough
```

### Test 3: Dependency Validation

```bash
# Try to skip steps - should fail!
ORDER_ID=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_name": "Charlie"}' | jq -r '.order_id')

# Try to bake without making dough - FAILS
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/bake

# Must follow order
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/payment
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/make-dough
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/add-toppings
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/bake
curl -X POST http://localhost:8080/orders/${ORDER_ID#pizza-orders/}/deliver
```

## Summary

1. **DAG is just data** - a Go struct you define
2. **Temporal stores the data** - in its database as JSON
3. **Updates modify state** - return value gets stored
4. **Queries read state** - from memory or replayed from events
5. **Await blocks efficiently** - no CPU used while waiting
6. **State persists forever** - until workflow completes

The "magic" is that Temporal:
- Serializes your state to JSON
- Stores it reliably in a database
- Reconstructs it when needed
- Guarantees exactly-once execution

There's no special DAG tracking - it's just event sourcing with a nice API!
