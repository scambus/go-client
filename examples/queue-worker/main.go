// Claims queue items in a loop, records contact, and completes them.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	scambus "github.com/scambus/go-client"
)

func main() {
	queueID := os.Getenv("SCAMBUS_QUEUE_ID")
	if queueID == "" {
		log.Fatal("set SCAMBUS_QUEUE_ID")
	}

	client, err := scambus.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for ctx.Err() == nil {
		item, err := client.Queues.Claim(ctx, queueID)
		if err != nil {
			// A missing or inaccessible queue is an error; only an empty
			// queue yields a nil item.
			log.Fatal(err)
		}
		if item == nil {
			time.Sleep(10 * time.Second)
			continue
		}

		fmt.Printf("claimed %s (%s %s)\n", item.ID, item.RepresentativeType, item.RepresentativeValue)

		identifiers, err := client.Queues.ItemClusterIdentifiers(ctx, queueID, item.ID, "target")
		if err != nil {
			log.Fatal(err)
		}
		for _, identifier := range identifiers {
			fmt.Printf("  %s %s\n", identifier.Type, identifier.Value)
		}

		if err := client.Queues.RecordContact(ctx, queueID, item.ID, scambus.ContactInput{
			Notes: "Outbound attempt from the Go worker",
		}); err != nil {
			log.Fatal(err)
		}
		if err := client.Queues.Complete(ctx, queueID, item.ID, scambus.ItemActionInput{
			Outcome: "engaged",
		}); err != nil {
			log.Fatal(err)
		}
	}
}
