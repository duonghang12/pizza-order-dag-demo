package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"pizza-order-dag-demo/types"
	"pizza-order-dag-demo/workflow"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

var temporalClient client.Client

func main() {
	// 1. Connect to Temporal
	var err error
	temporalClient, err = client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer temporalClient.Close()

	// 2. Setup HTTP routes
	http.HandleFunc("/orders", handleOrders)
	http.HandleFunc("/orders/", handleOrderActions)

	// 3. Start server
	log.Println("API Server starting on :8080")
	log.Println("\nEndpoints:")
	log.Println("  POST   /orders                         - Create new pizza order")
	log.Println("  GET    /orders/{orderID}               - Get order status")
	log.Println("  POST   /orders/{orderID}/payment       - Complete payment")
	log.Println("  POST   /orders/{orderID}/make-dough    - Make dough")
	log.Println("  POST   /orders/{orderID}/add-toppings  - Add toppings")
	log.Println("  POST   /orders/{orderID}/bake          - Bake pizza")
	log.Println("  POST   /orders/{orderID}/deliver       - Deliver pizza")
	log.Println("\nReady to accept requests...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleOrders handles POST /orders (create new order)
func handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		createOrder(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOrderActions handles GET and POST for specific orders
func handleOrderActions(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /orders/{orderID}/{action}
	path := strings.TrimPrefix(r.URL.Path, "/orders/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	orderID := "pizza-orders/" + parts[0]

	// GET /orders/{orderID} - get status
	if r.Method == http.MethodGet && len(parts) == 1 {
		getOrderStatus(w, r, orderID)
		return
	}

	// POST /orders/{orderID}/{action} - complete a step
	if r.Method == http.MethodPost && len(parts) == 2 {
		action := parts[1]
		completeStep(w, r, orderID, action)
		return
	}

	http.Error(w, "Invalid request", http.StatusBadRequest)
}

// createOrder creates a new pizza order workflow
func createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerName    string  `json:"customer_name"`
		CustomerEmail   string  `json:"customer_email"`
		CustomerPhone   string  `json:"customer_phone"`
		DeliveryAddress string  `json:"delivery_address"`
		Amount          float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.CustomerName == "" {
		http.Error(w, "customer_name is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.CustomerEmail == "" {
		req.CustomerEmail = fmt.Sprintf("%s@example.com", req.CustomerName)
	}
	if req.CustomerPhone == "" {
		req.CustomerPhone = "+1-555-0100"
	}
	if req.DeliveryAddress == "" {
		req.DeliveryAddress = "123 Main St, San Francisco, CA"
	}
	if req.Amount == 0 {
		req.Amount = 19.99 // Default pizza price
	}

	// Generate workflow ID
	orderID := fmt.Sprintf("pizza-orders/%s", uuid.New().String())

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        orderID,
		TaskQueue: workflow.PizzaOrderTaskQueue,
	}

	input := &workflow.PizzaOrderInput{
		OrderID:         orderID,
		CustomerName:    req.CustomerName,
		CustomerEmail:   req.CustomerEmail,
		CustomerPhone:   req.CustomerPhone,
		DeliveryAddress: req.DeliveryAddress,
		Amount:          req.Amount,
	}

	we, err := temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, workflow.PizzaOrderWorkflow, input)
	if err != nil {
		log.Printf("Failed to start workflow: %v", err)
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	log.Printf("Started workflow - OrderID: %s, WorkflowID: %s, RunID: %s",
		orderID, we.GetID(), we.GetRunID())

	// Query the workflow to get initial state
	var state types.PizzaOrder
	value, err := temporalClient.QueryWorkflow(r.Context(), orderID, "", workflow.QueryOrderState)
	if err != nil {
		log.Printf("Failed to query workflow: %v", err)
		// Return basic response even if query fails
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"order_id":      orderID,
			"customer_name": req.CustomerName,
			"state":         "IN_PROGRESS",
		})
		return
	}

	if err := value.Get(&state); err != nil {
		log.Printf("Failed to decode state: %v", err)
		http.Error(w, "Failed to get order state", http.StatusInternalServerError)
		return
	}

	// Return the full state including DAG
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":      state.OrderID,
		"customer_name": state.CustomerName,
		"state":         state.State,
		"components":    state.DAG.GetComponents(),
		"create_time":   state.CreateTime,
	})
}

// getOrderStatus queries the workflow for current state
func getOrderStatus(w http.ResponseWriter, r *http.Request, orderID string) {
	// Query workflow (read-only, doesn't modify state)
	value, err := temporalClient.QueryWorkflow(r.Context(), orderID, "", workflow.QueryOrderState)
	if err != nil {
		log.Printf("Failed to query workflow %s: %v", orderID, err)
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	var state types.PizzaOrder
	if err := value.Get(&state); err != nil {
		log.Printf("Failed to decode state: %v", err)
		http.Error(w, "Failed to get order state", http.StatusInternalServerError)
		return
	}

	// Return state including DAG
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":      state.OrderID,
		"customer_name": state.CustomerName,
		"state":         state.State,
		"components":    state.DAG.GetComponents(),
		"create_time":   state.CreateTime,
		"update_time":   state.UpdateTime,
	})
}

// completeStep sends an update to complete a component
func completeStep(w http.ResponseWriter, r *http.Request, orderID, action string) {
	// Map action to update name
	var updateName string
	switch action {
	case "payment":
		updateName = workflow.UpdateCompletePayment
	case "make-dough":
		updateName = workflow.UpdateMakeDough
	case "add-toppings":
		updateName = workflow.UpdateAddToppings
	case "bake":
		updateName = workflow.UpdateBakePizza
	case "deliver":
		updateName = workflow.UpdateDeliver
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	// Send update to workflow (this modifies state!)
	updateHandle, err := temporalClient.UpdateWorkflow(r.Context(), client.UpdateWorkflowOptions{
		WorkflowID:   orderID,
		UpdateName:   updateName,
		WaitForStage: client.WorkflowUpdateStageCompleted, // Wait for result
	})
	if err != nil {
		log.Printf("Failed to update workflow %s: %v", orderID, err)
		http.Error(w, fmt.Sprintf("Failed to complete step: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the updated state
	var state types.PizzaOrder
	err = updateHandle.Get(r.Context(), &state)
	if err != nil {
		log.Printf("Failed to get update result: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get result: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Completed step %s for order %s", action, orderID)

	// Return updated state
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":      state.OrderID,
		"customer_name": state.CustomerName,
		"state":         state.State,
		"components":    state.DAG.GetComponents(),
		"update_time":   state.UpdateTime,
	})
}
