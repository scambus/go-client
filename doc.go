/*
Package scambus is the official Go client for the SCAMBUS API.

It submits scam reports as journal entries, searches identifiers, cases and
entries, manages export streams, and receives live data over SSE or WebSocket.

# Authentication

New resolves credentials in this order: explicit options, the SCAMBUS_*
environment variables, then ~/.scambus/config.json written by the CLI.

	client, err := scambus.New()

An API key takes precedence over a bearer token when both are available.

	client, err := scambus.New(
		scambus.WithAPIURL("https://scambus.net/api"),
		scambus.WithAPIKey(keyID, keySecret),
	)

# Services

Calls are grouped by resource on the client: Journal, Identifiers, Cases,
Queues, Streams, Consume, Views, Tags, Search, Notifications, Sessions,
Personas, Reports, Automations, FileExports, Media, Comments and Admin.

	entry, err := client.Journal.CreateDetection(ctx, scambus.DetectionInput{
		Description: "Automated phishing detection",
		Identifiers: []scambus.IdentifierLookup{
			{Type: string(scambus.IdentifierTypePhone), Value: "+12125551234", Confidence: scambus.Ptr(0.9)},
		},
	})

# Errors

Every failure carries an *APIError and matches a sentinel through errors.Is:
ErrAuthentication, ErrValidation, ErrNotFound, ErrRateLimited,
ErrCursorExpired and ErrServer.

	if errors.Is(err, scambus.ErrNotFound) {
		// ...
	}

# Retries

Transient failures retry with full-jitter exponential backoff, honouring
Retry-After and bounded by WithMaxRetries and WithRetryMaxTime. A 408 or 429
means the server did not process the request, so any method is replayed. A 5xx
or a dropped connection may follow a committed write, so only idempotent
methods are replayed there; a failed POST is reported, never repeated.
Cancelling the context stops the retry loop.

The client refuses HTTP redirects. Credentials travel in headers, and Go
forwards a custom header such as X-API-Key across a host boundary.

A *Client is safe for concurrent use by any number of goroutines. A *WSClient
is not.

# Consuming streams

Consume methods take the stream's consumer key, not its id.

	err := client.Consume.Subscribe(ctx, consumerKey,
		&scambus.SubscribeOptions{Cursor: scambus.CursorEnd, Reconnect: true},
		func(m scambus.StreamMessage) error {
			msg, err := m.Identifier()
			if err != nil {
				return err
			}
			fmt.Println(msg.DisplayValue, msg.Confidence.Score)
			return nil
		})
*/
package scambus
