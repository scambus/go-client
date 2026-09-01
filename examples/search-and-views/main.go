// Searches identifiers, then saves the same filter as a view.
package main

import (
	"context"
	"fmt"
	"log"

	scambus "github.com/scambus/go-client"
)

func main() {
	client, err := scambus.New()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	filter := &scambus.FilterCriteria{
		IdentifierType: string(scambus.IdentifierTypePhone),
		MinConfidence:  scambus.Ptr(0.8),
		CreatedAfter:   "2025-01-01T00:00:00Z",
		Country:        "US",
	}

	count := 0
	err = client.Search.IdentifiersAll(ctx, scambus.SearchIdentifiersInput{
		Filter: filter,
		Limit:  200,
	}, func(identifier scambus.Identifier) error {
		count++
		fmt.Printf("%s %s %.2f\n", identifier.Type, identifier.DisplayValue, identifier.Confidence.Score)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d identifiers matched\n", count)

	view, err := client.Views.Create(ctx, scambus.CreateViewInput{
		Name:           "High confidence US contacts",
		EntityType:     "identifier",
		FilterCriteria: filter,
		SortOrder:      &scambus.SortOrder{Field: "created_at", Direction: scambus.SortDesc},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("saved view %s\n", view.ID)
}
