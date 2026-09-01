// Generates a signed identifier report and downloads the PDF.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	scambus "github.com/scambus/go-client"
)

func main() {
	viewID := os.Getenv("SCAMBUS_VIEW_ID")
	if viewID == "" {
		log.Fatal("set SCAMBUS_VIEW_ID")
	}

	client, err := scambus.New()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report, err := client.Reports.GenerateViewReport(ctx, viewID, true, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("report %s is %s\n", report.ID, report.Status)

	report, err = client.Reports.Wait(ctx, report.ID, 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	if report.IsFailed() {
		log.Fatalf("report failed: %s", report.ErrorMessage)
	}

	if err := client.Reports.DownloadToFile(ctx, report.ID, "report.pdf"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("saved report.pdf (%d identifiers, %d entries)\n", report.IdentifierCount, report.JournalEntryCount)
}
