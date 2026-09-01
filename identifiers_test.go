package scambus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestBankAccountLookupEncodesValue(t *testing.T) {
	lookup, err := BankAccountLookup(BankAccountInput{
		Account: "123456789", Routing: "021000021", Institution: "Chase",
		Owner: "J Doe", Confidence: Ptr(0.9),
	})
	if err != nil {
		t.Fatal(err)
	}
	if lookup.Type != "bank_account" {
		t.Fatalf("type %q", lookup.Type)
	}
	var details BankAccountDetails
	if err := json.Unmarshal([]byte(lookup.Value), &details); err != nil {
		t.Fatal(err)
	}
	if details.AccountNumber != "123456789" || details.Routing != "021000021" || details.Institution != "Chase" {
		t.Fatalf("got %+v", details)
	}
	if lookup.Confidence == nil || *lookup.Confidence != 0.9 {
		t.Fatalf("confidence %v", lookup.Confidence)
	}
}

func TestBankAccountLookupRequiresCoreFields(t *testing.T) {
	if _, err := BankAccountLookup(BankAccountInput{Account: "1"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestVenmoLookupAcceptedForms(t *testing.T) {
	cases := []string{
		"@scammer_one",
		"1234567890123456",
		"https://venmo.com/code?user_id=1234567890123456",
	}
	for _, input := range cases {
		lookup, err := VenmoLookup(input, "Some Name", nil)
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		var details PaymentTokenDetails
		if err := json.Unmarshal([]byte(lookup.Value), &details); err != nil {
			t.Fatal(err)
		}
		if details.Service != "venmo" || details.Identifier != input {
			t.Fatalf("%s: got %+v", input, details)
		}
	}
}

func TestVenmoLookupRejectedForms(t *testing.T) {
	cases := map[string]string{
		"wrong host":     "https://paypal.com/code?user_id=1234567890123456",
		"wrong path":     "https://venmo.com/u?user_id=1234567890123456",
		"missing userid": "https://venmo.com/code",
		"short userid":   "https://venmo.com/code?user_id=123",
		"short handle":   "@abc",
		"bare word":      "scammer",
	}
	for name, input := range cases {
		if _, err := VenmoLookup(input, "", nil); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s (%s): want a validation error, got %v", name, input, err)
		}
	}
}

func TestChimeLookup(t *testing.T) {
	lookup, err := ChimeLookup("$JohnDoe", "John", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lookup.Value, `"identifier":"$JohnDoe"`) {
		t.Fatalf("got %s", lookup.Value)
	}
	for _, bad := range []string{"JohnDoe", "$", "$" + strings.Repeat("a", 21), "$john-doe"} {
		if _, err := ChimeLookup(bad, "", nil); !errors.Is(err, ErrValidation) {
			t.Fatalf("%q: want a validation error, got %v", bad, err)
		}
	}
}

func TestListIdentifiersDefaultsPaging(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"data": []any{map[string]any{"id": "i1", "type": "phone", "display_value": "+1", "confidence": map[string]any{"score": 0.5}}},
		})
	})
	c := srv.client(t)

	got, err := c.Identifiers.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Confidence.Score != 0.5 {
		t.Fatalf("got %+v", got)
	}
	if srv.last().Query != "page=1&pageSize=25" {
		t.Fatalf("query %q", srv.last().Query)
	}
}

func TestCreateExclusionRequiresIdentity(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, IdentifierExclusion{ID: "x1"})
	})
	c := srv.client(t)

	if _, err := c.Identifiers.CreateExclusion(context.Background(), CreateExclusionInput{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
	if len(srv.requests) != 0 {
		t.Fatal("must not hit the API")
	}

	if _, err := c.Identifiers.CreateExclusion(context.Background(), CreateExclusionInput{IdentifierID: "i1"}); err != nil {
		t.Fatal(err)
	}
}

func TestURLReferencesDefaults(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"url_references": []any{}, "total": 0})
	})
	c := srv.client(t)

	if _, err := c.Identifiers.URLReferences(context.Background(), "i1", nil); err != nil {
		t.Fatal(err)
	}
	want := "order=desc&page=1&page_size=25&sort=last_seen_at"
	if srv.last().Query != want {
		t.Fatalf("query %q, want %q", srv.last().Query, want)
	}
}
