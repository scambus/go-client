// Submits a detection with two identifiers and a tag.
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	entry, err := client.Journal.CreateDetection(ctx, scambus.DetectionInput{
		Description: "Automated phishing detection",
		Identifiers: []scambus.IdentifierLookup{
			{Type: scambus.IdentifierTypePhone, Value: "+12125551234", Confidence: scambus.Ptr(0.9)},
			{Type: scambus.IdentifierTypeEmail, Value: "scammer@example.com", Confidence: scambus.Ptr(0.95)},
		},
		Tags: []scambus.TagLookup{
			{TagName: "ScamType", TagValue: "Phishing"},
		},
		Details: &scambus.DetectionDetails{
			Data: map[string]any{"category": "phishing", "score": 0.92},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("created %s with %d identifiers\n", entry.ID, len(entry.Identifiers))
	for _, failed := range entry.FailedIdentifiers {
		fmt.Printf("  rejected %s %s: %s\n", failed.Type, failed.Value, failed.Reason)
	}
}
