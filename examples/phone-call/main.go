// Uploads a recording, files a phone call with a transcript, and completes it.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	scambus "github.com/scambus/go-client"
)

func main() {
	client, err := scambus.New()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	media, err := client.Media.UploadFile(ctx, "recording.flac", &scambus.MediaUpload{
		Notes: "Inbound call recording",
	})
	if err != nil {
		log.Fatal(err)
	}

	start := scambus.NewTime(time.Now().UTC().Add(-5 * time.Minute))
	entry, err := client.Journal.CreatePhoneCall(ctx, scambus.PhoneCallInput{
		Description: "Inbound tech support scam",
		Direction:   "inbound",
		StartTime:   start,
		Media:       []scambus.Media{*media},
		InProgress:  true,
		Transcript: []scambus.ConversationMessage{
			{Index: 0, MessageID: "m0", Timestamp: start, Body: "This is Microsoft support."},
			{Index: 1, MessageID: "m1", Timestamp: scambus.NewTime(start.Add(time.Minute)),
				Body: "Wire the payment to 021000021 / 123456789.", IsOutgoing: false},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("call %s is in progress\n", entry.ID)

	done, err := client.Journal.CompleteActivity(ctx, entry.ID, scambus.NewTime(time.Now().UTC()), "manual", "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("closed with %s\n", done.ID)
}
