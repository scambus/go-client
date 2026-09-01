# SCAMBUS Go Client

Official Go client for SCAMBUS. Submit scam reports, search identifiers and
cases, manage export streams, and receive live data over SSE or WebSocket.

```bash
go get github.com/scambus/go-client
```

## Features

- **Submit scam reports** — phone calls, emails, text conversations, continuations, detections, notes, imports and exports
- **Tagging** — apply boolean and valued tags when you create an entry
- **Search** — identifiers, cases and journal entries, with cursor pagination helpers
- **Views** — create, execute and manage saved queries
- **In-progress activities** — start an open activity and close it later
- **Work queues** — claim, contact, complete, drop and move items
- **Data streams** — create export streams for journal entries or identifier state changes
- **Stream consumption** — poll for batches, or subscribe over SSE with automatic resume
- **Real-time updates** — WebSocket client for notifications and live streams
- **Cases and evidence** — manage investigations and attach media
- **Reports** — generate and download signed PDF reports
- **Automations** — create automation identities and rotate API keys
- **Automatic authentication** — shares cached credentials with the SCAMBUS CLI

## Quick start

```go
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

	entry, err := client.Journal.CreateDetection(context.Background(), scambus.DetectionInput{
		Description: "Automated phishing detection",
		Identifiers: []scambus.IdentifierLookup{
			{Type: "phone", Value: "+12125551234", Confidence: scambus.Ptr(0.9)},
			{Type: "email", Value: "scammer@example.com", Confidence: scambus.Ptr(0.95)},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(entry.ID)
}
```

## Authentication

`New` resolves credentials in this order: explicit options, environment
variables, then `~/.scambus/config.json` written by the CLI. An API key wins
over a bearer token when both are present.

```go
client, err := scambus.New(
	scambus.WithAPIURL("https://scambus.net/api"),
	scambus.WithAPIKey(keyID, keySecret),
)
```

| Variable | Purpose |
|----------|---------|
| `SCAMBUS_API_KEY_ID` | API key id |
| `SCAMBUS_API_KEY_SECRET` | API key secret |
| `SCAMBUS_API_TOKEN` | Bearer token |
| `SCAMBUS_API_URL` | Base URL (`SCAMBUS_URL` also works) |

Run `scambus auth login` with the CLI and the Go client picks the token up with
no configuration at all.

## Options

```go
client, err := scambus.New(
	scambus.WithTimeout(30*time.Second),
	scambus.WithMaxRetries(10),
	scambus.WithRetryMaxTime(5*time.Minute),
	scambus.WithHTTPClient(myClient),
	scambus.WithLogger(slog.Default()),
)
```

Transient failures (408, 429, 5xx and connection errors) retry with full-jitter
exponential backoff and honour `Retry-After`. Cancelling the context stops the
retry loop.

## Services

Calls are grouped by resource. Every method takes a `context.Context` first.

| Service | Covers |
|---------|--------|
| `client.Journal` | Journal entries, batch create, query, activities |
| `client.Identifiers` | Identifiers, exclusions, URL references, lookup builders |
| `client.Cases` | Cases and case comments |
| `client.Comments` | Comment edit and delete |
| `client.Queues` | Work queues and queue items |
| `client.Streams` | Export streams, recovery, backfill |
| `client.Consume` | Stream polling, info and SSE subscription |
| `client.Views` | Saved views and execution |
| `client.Tags` | Tags, tag values, effective tags, tag history |
| `client.Search` | Identifier and case search |
| `client.Notifications` | Notifications |
| `client.Sessions` | Sessions, passkeys, two-factor |
| `client.Personas` | Personas and persona media |
| `client.Reports` | PDF report generation and download |
| `client.Automations` | Automation identities and API keys |
| `client.FileExports` | CSV and file exports |
| `client.Media` | Media upload and lookup |
| `client.Admin` | Special domain rules, URL consolidation |

## Errors

Every failure carries an `*APIError` and matches a sentinel through `errors.Is`.

```go
_, err := client.Cases.Get(ctx, caseID)
switch {
case errors.Is(err, scambus.ErrNotFound):
	// no such case
case errors.Is(err, scambus.ErrAuthentication):
	// bad or expired credentials
case errors.Is(err, scambus.ErrValidation):
	// the request was rejected
}

var apiErr *scambus.APIError
if errors.As(err, &apiErr) {
	log.Printf("%d %s: %s", apiErr.StatusCode, apiErr.Endpoint, apiErr.Message)
}
```

Sentinels: `ErrAuthentication`, `ErrValidation`, `ErrNotFound`,
`ErrRateLimited`, `ErrCursorExpired`, `ErrServer`, `ErrNoCredentials`.

## Journal entries

Typed constructors fill the details payload for you:

```go
start := scambus.NewTime(time.Now().UTC())

entry, err := client.Journal.CreatePhoneCall(ctx, scambus.PhoneCallInput{
	Description: "Inbound tech support scam",
	Direction:   "inbound",
	StartTime:   start,
	InProgress:  true,
	Transcript: []scambus.ConversationMessage{
		{Index: 0, MessageID: "m0", Timestamp: start, Body: "This is Microsoft support."},
	},
})
```

A transcript turns on AI identifier extraction. Close an in-progress activity
later:

```go
done, err := client.Journal.CompleteActivity(ctx, entry.ID, scambus.NewTime(time.Now().UTC()), "manual", "")
```

Attach media and the client builds the evidence record:

```go
media, _ := client.Media.UploadFile(ctx, "screenshot.png", &scambus.MediaUpload{Notes: "Phishing page"})

entry, err := client.Journal.CreateDetection(ctx, scambus.DetectionInput{
	Description: "Phishing site",
	Media:       []scambus.Media{*media},
})
```

For an entry type without a constructor, use `Journal.Create` with
`CreateEntryInput`.

## Filtering

`FilterCriteria` is the one filter shape used by search, query, views, export
streams and file exports. Optional scalars are pointers; use `scambus.Ptr`.

```go
filter := &scambus.FilterCriteria{
	Types:          []string{"phone", "email"},
	MinConfidence:  scambus.Ptr(0.8),
	CreatedAfter:   "2025-01-01T00:00:00Z",
	Country:        "US",
	ExcludedTypes:  []string{"note"},
}

result, err := client.Journal.Query(ctx, scambus.QueryEntriesInput{
	Filter:             filter,
	IncludeIdentifiers: true,
	OrderDesc:          true,
})
```

`Journal.QueryAll` and `Search.IdentifiersAll` walk every page for you.

## Consuming a stream

Consume methods take the stream's **consumer key**, not its id.

### Polling

```go
cursor := scambus.CursorStart
for {
	result, err := client.Consume.Poll(ctx, consumerKey, &scambus.PollOptions{
		Cursor: cursor,
		Order:  scambus.SortAsc,
		Limit:  100,
	})
	if err != nil {
		return err
	}

	messages, err := result.IdentifierMessages()
	if err != nil {
		return err
	}
	for _, msg := range messages {
		fmt.Println(msg.Type, msg.DisplayValue, msg.Confidence.Score)
	}

	if result.NextCursor != "" {
		cursor = result.NextCursor
	}
	if !result.HasMore {
		time.Sleep(5 * time.Second)
	}
}
```

| Cursor | Meaning |
|--------|---------|
| `scambus.CursorStart` (`"0"`) | From the beginning of the stream |
| `scambus.CursorEnd` (`"$"`) | Only messages arriving from now on |
| `"1735689600000-0"` | Resume from a specific message |

### SSE

`Subscribe` handles the historical `batch` replay and live `message` events,
tracks the cursor, and resumes from it on reconnect.

```go
err := client.Consume.Subscribe(ctx, consumerKey, &scambus.SubscribeOptions{
	Cursor:    scambus.CursorEnd,
	Reconnect: true,
}, func(m scambus.StreamMessage) error {
	msg, err := m.JournalEntry()
	if err != nil {
		return err
	}
	fmt.Println(msg.Type, msg.Description)
	return nil
})
```

Return `scambus.ErrStopSubscription` from the callback to end the loop cleanly.
Any other error stops the loop and is returned.

### WebSocket

```go
ws, err := client.NewWebSocket()
if err != nil {
	return err
}

err = ws.ListenStream(ctx, streamID, scambus.CursorEnd, false, func(m scambus.StreamMessage) {
	identifier, err := m.Identifier()
	if err != nil {
		return
	}
	fmt.Println(identifier.DisplayValue)
})
```

`ws.On(channel, event, handler)` registers extra handlers and returns an
unregister function. Use `"*"` as the event to receive every message on a
channel.

## Identifier details

`Identifier.Data` and stream message details are loosely typed. Decode them
into the struct for the identifier type:

```go
details, err := scambus.ParseIdentifierDetails(identifier.Type, identifier.Data)
if phone, ok := details.(scambus.PhoneDetails); ok {
	fmt.Println(phone.CountryCode, phone.Number, phone.IsTollFree)
}
```

Journal entry details work the same way:

```go
call, err := scambus.DecodeDetails[scambus.PhoneCallDetails](entry.Details)
```

| Type | Struct |
|------|--------|
| `phone` | `PhoneDetails` |
| `email` | `IdentifierEmailDetails` |
| `url` | `URLDetails` |
| `bank_account` | `BankAccountDetails` |
| `crypto_wallet` | `CryptoWalletDetails` |
| `social_media` | `SocialMediaDetails` |
| `zelle` | `ZelleDetails` |
| `payment_token` | `PaymentTokenDetails` |

## Composite identifiers

Bank accounts and payment tokens carry a JSON-encoded value. Builders validate
the input and encode it:

```go
bank, err := scambus.BankAccountLookup(scambus.BankAccountInput{
	Account:     "123456789",
	Routing:     "021000021",
	Institution: "Chase",
	Confidence:  scambus.Ptr(0.9),
})

venmo, err := scambus.VenmoLookup("@scammer_handle", "Some Name", nil)
chime, err := scambus.ChimeLookup("$JohnDoe", "John", nil)
```

## Timestamps and confidence

`scambus.Time` accepts RFC3339 with or without a zone and reads a naive value
as UTC. It embeds `time.Time`, so all the usual methods work.

`scambus.Confidence` decodes both shapes the API returns: `{"score": 0.95}` on
entity endpoints and a bare `0.95` on stream endpoints. Read `Confidence.Score`,
and check `Confidence.Set` to tell an absent value from zero.

## Examples

Runnable programs live in [`examples/`](examples):

| Example | Shows |
|---------|-------|
| `simple-detection` | Submit a detection with identifiers and tags |
| `phone-call` | Upload media, file a call, complete the activity |
| `consume-poll` | Poll a stream and resume from a cursor |
| `consume-sse` | Subscribe over SSE with reconnect |
| `websocket-stream` | Live stream and notifications over WebSocket |
| `search-and-views` | Paginated search, saved as a view |
| `queue-worker` | Claim, contact and complete queue items |
| `reports` | Generate and download a signed PDF report |

## Development

```bash
go test ./...
go test -race ./...
go test -cover ./...
go vet ./...
```

## License

MIT. See [LICENSE](LICENSE).
