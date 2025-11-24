package main

import (
	"log"

	"pizza-order-dag-demo/activities"
	"pizza-order-dag-demo/workflow"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// 1. Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer c.Close()

	// 2. Create worker that listens on the task queue
	w := worker.New(c, workflow.PizzaOrderTaskQueue, worker.Options{})

	// 3. Register workflow
	w.RegisterWorkflow(workflow.PizzaOrderWorkflow)

	// 4. Register activities
	paymentActivities := &activities.PaymentActivities{}
	w.RegisterActivity(paymentActivities.ProcessPayment)
	w.RegisterActivity(paymentActivities.RefundPayment)

	deliveryActivities := &activities.DeliveryActivities{}
	w.RegisterActivity(deliveryActivities.ScheduleDelivery)
	w.RegisterActivity(deliveryActivities.UpdateDeliveryStatus)

	notificationActivities := &activities.NotificationActivities{}
	w.RegisterActivity(notificationActivities.SendNotification)
	w.RegisterActivity(notificationActivities.SendOrderConfirmation)
	w.RegisterActivity(notificationActivities.SendDeliveryNotification)

	// 5. Start worker
	log.Println("Worker starting...")
	log.Println("Task Queue:", workflow.PizzaOrderTaskQueue)
	log.Println("Registered Workflows:", workflow.PizzaOrderWorkflowName)
	log.Println("Registered Activities: Payment, Delivery, Notification")
	log.Println("\nWaiting for workflow tasks...")

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
