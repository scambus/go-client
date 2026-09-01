// Reads an export stream over SSE, reconnecting from the last cursor.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	scambus "github.com/scambus/go-client"
)

func main() {
	consumerKey := os.Getenv("SCAMBUS_CONSUMER_KEY")
	if consumerKey == "" {
		log.Fatal("set SCAMBUS_CONSUMER_KEY")
	}

	client, err := scambus.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = client.Consume.Subscribe(ctx, consumerKey, &scambus.SubscribeOptions{
		Cursor:    scambus.CursorEnd,
		Reconnect: true,
	}, func(m scambus.StreamMessage) error {
		msg, err := m.JournalEntry()
		if err != nil {
			return err
		}
		fmt.Printf("%s %s at %s\n", msg.Type, msg.Description, msg.PerformedAt.Format("15:04:05"))
		for _, identifier := range msg.Identifiers {
			fmt.Printf("  %s %s\n", identifier.Type, identifier.DisplayValue)
		}
		return nil
	})
	if err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
