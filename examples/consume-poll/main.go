// Polls an export stream in a loop, resuming from the last cursor.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

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

	ctx := context.Background()
	cursor := scambus.CursorStart

	for {
		result, err := client.Consume.Poll(ctx, consumerKey, &scambus.PollOptions{
			Cursor: cursor,
			Order:  scambus.SortAsc,
			Limit:  100,
		})
		if errors.Is(err, scambus.ErrCursorExpired) {
			log.Print("cursor fell outside retention, restarting from the beginning")
			cursor = scambus.CursorStart
			continue
		}
		if err != nil {
			log.Fatal(err)
		}

		messages, err := result.IdentifierMessages()
		if err != nil {
			log.Fatal(err)
		}
		for _, msg := range messages {
			fmt.Printf("%s %s confidence=%.2f\n", msg.Type, msg.DisplayValue, msg.Confidence.Score)
		}

		if result.NextCursor != "" {
			cursor = result.NextCursor
		}
		if !result.HasMore {
			time.Sleep(5 * time.Second)
		}
	}
}
