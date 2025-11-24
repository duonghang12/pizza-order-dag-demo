package activities

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// DeliveryInput represents delivery request data
type DeliveryInput struct {
	OrderID         string
	CustomerName    string
	DeliveryAddress string
	EstimatedTime   int // minutes
}

// DeliveryResult represents delivery service response
type DeliveryResult struct {
	DeliveryID       string
	DriverName       string
	EstimatedArrival time.Time
	TrackingURL      string
	Status           string
}

// DeliveryActivities holds delivery-related activities
type DeliveryActivities struct{}

// ScheduleDelivery simulates calling a delivery service API (Uber, DoorDash, etc.)
func (a *DeliveryActivities) ScheduleDelivery(ctx context.Context, input DeliveryInput) (*DeliveryResult, error) {
	// Simulate API call latency
	time.Sleep(time.Duration(300+rand.Intn(700)) * time.Millisecond)

	// Simulate random failures (5% chance - no drivers available)
	if rand.Float64() < 0.05 {
		return nil, fmt.Errorf("no delivery drivers available in your area")
	}

	// Random driver names for simulation
	drivers := []string{"John Smith", "Maria Garcia", "James Wilson", "Emma Johnson", "Ali Hassan"}

	result := &DeliveryResult{
		DeliveryID:       fmt.Sprintf("DEL-%s", generateRandomID(10)),
		DriverName:       drivers[rand.Intn(len(drivers))],
		EstimatedArrival: time.Now().Add(time.Duration(input.EstimatedTime) * time.Minute),
		TrackingURL:      fmt.Sprintf("https://tracking.example.com/%s", generateRandomID(12)),
		Status:           "DRIVER_ASSIGNED",
	}

	fmt.Printf("✓ Delivery scheduled: Driver %s will arrive in ~%d minutes (ID: %s)\n",
		result.DriverName, input.EstimatedTime, result.DeliveryID)

	return result, nil
}

// UpdateDeliveryStatus simulates checking delivery status
func (a *DeliveryActivities) UpdateDeliveryStatus(ctx context.Context, deliveryID string) (string, error) {
	time.Sleep(time.Duration(200+rand.Intn(300)) * time.Millisecond)

	statuses := []string{"DRIVER_ASSIGNED", "PICKED_UP", "IN_TRANSIT", "DELIVERED"}
	status := statuses[rand.Intn(len(statuses))]

	fmt.Printf("✓ Delivery status updated: %s -> %s\n", deliveryID, status)
	return status, nil
}
