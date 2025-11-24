package types

import "time"

// ComponentType represents different steps in pizza order
type ComponentType string

const (
	ComponentPayment     ComponentType = "PAYMENT"
	ComponentMakeDough   ComponentType = "MAKE_DOUGH"
	ComponentAddToppings ComponentType = "ADD_TOPPINGS"
	ComponentBakePizza   ComponentType = "BAKE_PIZZA"
	ComponentDeliver     ComponentType = "DELIVER"
)

// ComponentState tracks progress of each component
type ComponentState string

const (
	StateNeedsInit  ComponentState = "NEEDS_INIT"  // Not ready to start yet (dependencies not met)
	StateIncomplete ComponentState = "INCOMPLETE"  // Ready to work on, but not done
	StateCompleted  ComponentState = "COMPLETED"   // Done!
)

// Component represents a single step in the pizza order
type Component struct {
	Type         ComponentType   `json:"type"`
	State        ComponentState  `json:"state"`
	DependsOn    []ComponentType `json:"dependsOn"`    // Which steps must complete first
	UpdateTime   time.Time       `json:"updateTime"`
	CompleteTime *time.Time      `json:"completeTime"` // nil if not completed
}

// OrderState represents the overall state of a pizza order
type OrderState string

const (
	OrderStateInProgress OrderState = "IN_PROGRESS"
	OrderStateCompleted  OrderState = "COMPLETED"
)

// PizzaOrder is the complete workflow state
type PizzaOrder struct {
	OrderID         string       `json:"order_id"`
	CustomerName    string       `json:"customer_name"`
	CustomerEmail   string       `json:"customer_email,omitempty"`
	CustomerPhone   string       `json:"customer_phone,omitempty"`
	DeliveryAddress string       `json:"delivery_address,omitempty"`
	State           OrderState   `json:"state"`
	DAG             *DAG         `json:"components"` // The component graph
	CreateTime      time.Time    `json:"create_time"`
	UpdateTime      time.Time    `json:"update_time"`

	// Activity results
	PaymentTxnID    string     `json:"payment_txn_id,omitempty"`
	PaymentAmount   float64    `json:"payment_amount,omitempty"`
	DeliveryID      string     `json:"delivery_id,omitempty"`
	DriverName      string     `json:"driver_name,omitempty"`
	TrackingURL     string     `json:"tracking_url,omitempty"`
	EstimatedArrival *time.Time `json:"estimated_arrival,omitempty"`
}

// Clone creates a deep copy of the order
func (po *PizzaOrder) Clone() *PizzaOrder {
	clone := &PizzaOrder{
		OrderID:         po.OrderID,
		CustomerName:    po.CustomerName,
		CustomerEmail:   po.CustomerEmail,
		CustomerPhone:   po.CustomerPhone,
		DeliveryAddress: po.DeliveryAddress,
		State:           po.State,
		CreateTime:      po.CreateTime,
		UpdateTime:      po.UpdateTime,
		PaymentTxnID:    po.PaymentTxnID,
		PaymentAmount:   po.PaymentAmount,
		DeliveryID:      po.DeliveryID,
		DriverName:      po.DriverName,
		TrackingURL:     po.TrackingURL,
	}

	if po.EstimatedArrival != nil {
		t := *po.EstimatedArrival
		clone.EstimatedArrival = &t
	}

	if po.DAG != nil {
		clone.DAG = po.DAG.Clone()
	}

	return clone
}

// IsDone checks if all components are completed
func (po *PizzaOrder) IsDone() bool {
	if po.DAG == nil {
		return false
	}
	return po.DAG.AllComponentsCompleted()
}
