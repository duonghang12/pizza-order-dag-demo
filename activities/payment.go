package activities

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// PaymentInput represents payment request data
type PaymentInput struct {
	OrderID      string
	CustomerName string
	Amount       float64
}

// PaymentResult represents payment response
type PaymentResult struct {
	TransactionID string
	Status        string
	Amount        float64
	Timestamp     time.Time
}

// PaymentActivities holds payment-related activities
type PaymentActivities struct{}

// ProcessPayment simulates calling a payment gateway API (Stripe, PayPal, etc.)
// This is a non-deterministic activity that should NEVER be in workflow code!
func (a *PaymentActivities) ProcessPayment(ctx context.Context, input PaymentInput) (*PaymentResult, error) {
	// Simulate API call latency
	time.Sleep(time.Duration(500+rand.Intn(1000)) * time.Millisecond)

	// Simulate random payment failures (10% chance)
	if rand.Float64() < 0.1 {
		return nil, fmt.Errorf("payment gateway error: insufficient funds or card declined")
	}

	// Simulate successful payment
	result := &PaymentResult{
		TransactionID: fmt.Sprintf("TXN-%d-%s", time.Now().Unix(), generateRandomID(8)),
		Status:        "SUCCESS",
		Amount:        input.Amount,
		Timestamp:     time.Now(),
	}

	fmt.Printf("✓ Payment processed: %s for $%.2f (TxnID: %s)\n",
		input.CustomerName, result.Amount, result.TransactionID)

	return result, nil
}

// RefundPayment simulates refunding a payment
func (a *PaymentActivities) RefundPayment(ctx context.Context, transactionID string) error {
	time.Sleep(time.Duration(300+rand.Intn(700)) * time.Millisecond)

	fmt.Printf("✓ Payment refunded: %s\n", transactionID)
	return nil
}
