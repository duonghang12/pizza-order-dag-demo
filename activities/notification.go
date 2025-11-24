package activities

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// NotificationInput represents notification data
type NotificationInput struct {
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	Message       string
	Type          string // "SMS", "EMAIL", "PUSH"
}

// NotificationActivities holds notification-related activities
type NotificationActivities struct{}

// SendNotification simulates calling a notification service (Twilio, SendGrid, etc.)
func (a *NotificationActivities) SendNotification(ctx context.Context, input NotificationInput) error {
	// Simulate API call latency
	time.Sleep(time.Duration(200+rand.Intn(500)) * time.Millisecond)

	// Simulate random failures (2% chance)
	if rand.Float64() < 0.02 {
		return fmt.Errorf("notification service temporarily unavailable")
	}

	var destination string
	switch input.Type {
	case "SMS":
		destination = input.CustomerPhone
	case "EMAIL":
		destination = input.CustomerEmail
	default:
		destination = input.CustomerName
	}

	fmt.Printf("✓ %s sent to %s: %s\n", input.Type, destination, input.Message)
	return nil
}

// SendOrderConfirmation sends order confirmation notification
func (a *NotificationActivities) SendOrderConfirmation(ctx context.Context, orderID, customerName, customerEmail string) error {
	return a.SendNotification(ctx, NotificationInput{
		CustomerName:  customerName,
		CustomerEmail: customerEmail,
		Message:       fmt.Sprintf("Order %s confirmed! Your pizza is being prepared.", orderID),
		Type:          "EMAIL",
	})
}

// SendDeliveryNotification sends delivery status notification
func (a *NotificationActivities) SendDeliveryNotification(ctx context.Context, customerName, driverName string, eta time.Time) error {
	return a.SendNotification(ctx, NotificationInput{
		CustomerName: customerName,
		Message:      fmt.Sprintf("Your pizza is on the way! Driver: %s, ETA: %s", driverName, eta.Format("3:04 PM")),
		Type:         "SMS",
	})
}
