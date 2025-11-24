package workflow

import (
	"fmt"
	"time"

	"pizza-order-dag-demo/activities"
	"pizza-order-dag-demo/types"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// Workflow and task queue name
	PizzaOrderWorkflowName = "PizzaOrderWorkflow"
	PizzaOrderTaskQueue    = "pizza-order-queue"

	// Query name
	QueryOrderState = "QueryOrderState"

	// Update names
	UpdateCompletePayment     = "CompletePayment"
	UpdateMakeDough           = "MakeDough"
	UpdateAddToppings         = "AddToppings"
	UpdateBakePizza           = "BakePizza"
	UpdateDeliver             = "Deliver"
)

// PizzaOrderInput is the input to start a new pizza order workflow
type PizzaOrderInput struct {
	OrderID         string
	CustomerName    string
	CustomerEmail   string
	CustomerPhone   string
	DeliveryAddress string
	Amount          float64 // Pizza price
}

// PizzaOrderWorkflow is the main Temporal workflow
// This is the KEY function - it runs in the Temporal worker
func PizzaOrderWorkflow(ctx workflow.Context, input *PizzaOrderInput) (*types.PizzaOrder, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting pizza order workflow", "orderID", input.OrderID, "customer", input.CustomerName)

	// 1. Initialize the workflow state (THIS IS JUST A REGULAR GO VARIABLE!)
	state := &types.PizzaOrder{
		OrderID:         input.OrderID,
		CustomerName:    input.CustomerName,
		CustomerEmail:   input.CustomerEmail,
		CustomerPhone:   input.CustomerPhone,
		DeliveryAddress: input.DeliveryAddress,
		State:           types.OrderStateInProgress,
		DAG:             types.NewPizzaOrderDAG(), // Create the component graph
		CreateTime:      workflow.Now(ctx),
		UpdateTime:      workflow.Now(ctx),
	}

	logger.Info("Initial DAG state", "components", state.DAG.GetComponents())

	// 2. Setup Query Handler - allows external systems to READ current state
	err := workflow.SetQueryHandler(ctx, QueryOrderState, func() (*types.PizzaOrder, error) {
		logger.Info("Query received - returning current state")
		return state, nil // Just return the current state variable!
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set query handler: %w", err)
	}

	// 3. Setup Update Handlers - allows external systems to MODIFY state
	// Each update handler modifies the state variable and returns it
	// Temporal automatically stores the returned state!

	err = workflow.SetUpdateHandler(ctx, UpdateCompletePayment, func() (*types.PizzaOrder, error) {
		logger.Info("Processing payment - calling payment gateway activity")

		// Configure activity options (timeout, retry policy, etc.)
		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 3,
			},
		}
		activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

		// Call payment activity (non-deterministic operation!)
		paymentInput := activities.PaymentInput{
			OrderID:      state.OrderID,
			CustomerName: state.CustomerName,
			Amount:       input.Amount,
		}

		var paymentResult activities.PaymentResult
		err := workflow.ExecuteActivity(activityCtx, "ProcessPayment", paymentInput).Get(activityCtx, &paymentResult)
		if err != nil {
			logger.Error("Payment failed", "error", err)
			return nil, fmt.Errorf("payment processing failed: %w", err)
		}

		// Store payment result
		state.PaymentTxnID = paymentResult.TransactionID
		state.PaymentAmount = paymentResult.Amount

		// Send confirmation notification
		var notifErr error
		workflow.ExecuteActivity(activityCtx, "SendOrderConfirmation",
			state.OrderID, state.CustomerName, state.CustomerEmail).Get(activityCtx, &notifErr)
		// Ignore notification errors - not critical

		if err := state.DAG.CompleteComponent(types.ComponentPayment); err != nil {
			return nil, err
		}
		state.UpdateTime = workflow.Now(ctx)
		logger.Info("Payment completed", "txnID", paymentResult.TransactionID, "nextComponent", state.DAG.GetNextComponent())
		return state, nil
	})
	if err != nil {
		return nil, err
	}

	err = workflow.SetUpdateHandler(ctx, UpdateMakeDough, func() (*types.PizzaOrder, error) {
		logger.Info("Processing make dough")
		if err := state.DAG.CompleteComponent(types.ComponentMakeDough); err != nil {
			return nil, err
		}
		state.UpdateTime = workflow.Now(ctx)
		logger.Info("Dough made", "nextComponent", state.DAG.GetNextComponent())
		return state, nil
	})
	if err != nil {
		return nil, err
	}

	err = workflow.SetUpdateHandler(ctx, UpdateAddToppings, func() (*types.PizzaOrder, error) {
		logger.Info("Processing add toppings")
		if err := state.DAG.CompleteComponent(types.ComponentAddToppings); err != nil {
			return nil, err
		}
		state.UpdateTime = workflow.Now(ctx)
		logger.Info("Toppings added", "nextComponent", state.DAG.GetNextComponent())
		return state, nil
	})
	if err != nil {
		return nil, err
	}

	err = workflow.SetUpdateHandler(ctx, UpdateBakePizza, func() (*types.PizzaOrder, error) {
		logger.Info("Processing bake pizza")
		if err := state.DAG.CompleteComponent(types.ComponentBakePizza); err != nil {
			return nil, err
		}
		state.UpdateTime = workflow.Now(ctx)
		logger.Info("Pizza baked", "nextComponent", state.DAG.GetNextComponent())
		return state, nil
	})
	if err != nil {
		return nil, err
	}

	err = workflow.SetUpdateHandler(ctx, UpdateDeliver, func() (*types.PizzaOrder, error) {
		logger.Info("Processing delivery - calling delivery service activity")

		activityOptions := workflow.ActivityOptions{
			StartToCloseTimeout: 30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 3,
			},
		}
		activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

		// Call delivery activity (non-deterministic operation!)
		deliveryInput := activities.DeliveryInput{
			OrderID:         state.OrderID,
			CustomerName:    state.CustomerName,
			DeliveryAddress: state.DeliveryAddress,
			EstimatedTime:   30, // 30 minutes
		}

		var deliveryResult activities.DeliveryResult
		err := workflow.ExecuteActivity(activityCtx, "ScheduleDelivery", deliveryInput).Get(activityCtx, &deliveryResult)
		if err != nil {
			logger.Error("Delivery scheduling failed", "error", err)
			return nil, fmt.Errorf("delivery scheduling failed: %w", err)
		}

		// Store delivery result
		state.DeliveryID = deliveryResult.DeliveryID
		state.DriverName = deliveryResult.DriverName
		state.TrackingURL = deliveryResult.TrackingURL
		state.EstimatedArrival = &deliveryResult.EstimatedArrival

		// Send delivery notification
		var notifErr error
		workflow.ExecuteActivity(activityCtx, "SendDeliveryNotification",
			state.CustomerName, deliveryResult.DriverName, deliveryResult.EstimatedArrival).Get(activityCtx, &notifErr)
		// Ignore notification errors - not critical

		if err := state.DAG.CompleteComponent(types.ComponentDeliver); err != nil {
			return nil, err
		}
		state.UpdateTime = workflow.Now(ctx)
		logger.Info("Delivery scheduled", "deliveryID", deliveryResult.DeliveryID, "driver", deliveryResult.DriverName)
		return state, nil
	})
	if err != nil {
		return nil, err
	}

	// 4. Wait for all components to complete
	// This is where the workflow "blocks" waiting for user actions
	logger.Info("Waiting for all components to complete...")

	err = workflow.Await(ctx, func() bool {
		// This function is called after every update
		// It checks if we should continue waiting or not
		completed := state.IsDone()
		if completed {
			logger.Info("All components completed!")
		}
		return completed
	})
	if err != nil {
		return nil, err
	}

	// 5. All done! Mark order as completed
	state.State = types.OrderStateCompleted
	state.UpdateTime = workflow.Now(ctx)

	logger.Info("Pizza order workflow completed successfully!")

	// 6. Return final state - this becomes the workflow result
	return state, nil
}

// Helper function to create workflow ID
func CreateWorkflowID(customerName string) string {
	return fmt.Sprintf("pizza-orders/%s-%d", customerName, time.Now().Unix())
}
