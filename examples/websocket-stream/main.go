// Receives export stream messages and notifications over WebSocket.
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
	streamID := os.Getenv("SCAMBUS_STREAM_ID")
	if streamID == "" {
		log.Fatal("set SCAMBUS_STREAM_ID")
	}

	client, err := scambus.New()
	if err != nil {
		log.Fatal(err)
	}
	ws, err := client.NewWebSocket()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ws.On("notifications", "notification", func(msg scambus.WSMessage) {
		fmt.Printf("notification: %s\n", msg.Data)
	})

	err = ws.ListenStream(ctx, streamID, scambus.CursorEnd, false, func(m scambus.StreamMessage) {
		identifier, err := m.Identifier()
		if err != nil {
			log.Print(err)
			return
		}
		fmt.Printf("%s %s confidence=%.2f\n", identifier.Type, identifier.DisplayValue, identifier.Confidence.Score)
	})
	if err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
