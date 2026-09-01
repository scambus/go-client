package scambus

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SSEEvent struct {
	Event string
	Data  []byte
	ID    string
}

type SubscribeOptions struct {
	Cursor      string
	IncludeTest bool

	// Reconnect resumes from the last seen cursor after a dropped connection.
	Reconnect       bool
	MaxReconnects   int
	ReconnectDelay  time.Duration
	MaxReconnectGap time.Duration
}

// StreamMessage is one decoded payload from an SSE `batch` or `message` event.
type StreamMessage struct {
	Cursor string
	Raw    json.RawMessage
}

func (m StreamMessage) Identifier() (IdentifierStreamMessage, error) {
	var out IdentifierStreamMessage
	err := json.Unmarshal(m.Raw, &out)
	return out, err
}

func (m StreamMessage) JournalEntry() (JournalEntryStreamMessage, error) {
	var out JournalEntryStreamMessage
	err := json.Unmarshal(m.Raw, &out)
	return out, err
}

// Subscribe opens the SSE endpoint for a consumer key and calls fn for every
// message, until ctx is cancelled or fn returns an error.
func (s *ConsumeService) Subscribe(ctx context.Context, consumerKey string, opts *SubscribeOptions, fn func(StreamMessage) error) error {
	settings := SubscribeOptions{
		Cursor:          CursorEnd,
		ReconnectDelay:  time.Second,
		MaxReconnectGap: time.Minute,
		MaxReconnects:   -1,
	}
	if opts != nil {
		settings = *opts
		if settings.Cursor == "" {
			settings.Cursor = CursorEnd
		}
		if settings.ReconnectDelay <= 0 {
			settings.ReconnectDelay = time.Second
		}
		if settings.MaxReconnectGap <= 0 {
			settings.MaxReconnectGap = time.Minute
		}
		if !settings.Reconnect {
			settings.MaxReconnects = 0
		} else if settings.MaxReconnects == 0 {
			settings.MaxReconnects = -1
		}
	}

	cursor := settings.Cursor
	attempts := 0
	delay := settings.ReconnectDelay

	for {
		err := s.subscribeOnce(ctx, consumerKey, &cursor, settings.IncludeTest, fn)
		switch {
		case errors.Is(err, errStopSubscription):
			return nil
		case ctx.Err() != nil:
			return ctx.Err()
		case err != nil:
			if !retryableSSEError(err) || !settings.Reconnect {
				return err
			}
			s.client.logger.Warn("scambus SSE connection lost", "consumer_key", consumerKey, "error", err)
		}

		if !settings.Reconnect {
			return err
		}
		if settings.MaxReconnects >= 0 && attempts >= settings.MaxReconnects {
			return err
		}
		attempts++

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < settings.MaxReconnectGap {
			delay = min(delay*2, settings.MaxReconnectGap)
		}
	}
}

var errStopSubscription = errors.New("scambus: subscription stopped by caller")

// retryableSSEError is true for transport faults and transient server states;
// a rejected key or a bad consumer key will not fix itself.
func retryableSSEError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return true
	}
	return errors.Is(err, ErrServer) || errors.Is(err, ErrRateLimited)
}

func (s *ConsumeService) subscribeOnce(ctx context.Context, consumerKey string, cursor *string, includeTest bool, fn func(StreamMessage) error) error {
	q := url.Values{
		"cursor":       {*cursor},
		"include_test": {strconv.FormatBool(includeTest)},
	}
	req, err := s.client.newHTTPRequest(ctx, request{
		method:   http.MethodGet,
		endpoint: "/consume/" + consumerKey + "/stream",
		query:    q,
		accept:   "text/event-stream",
	})
	if err != nil {
		return err
	}
	req.Header.Set("Cache-Control", "no-cache")

	// The stream is long-lived; the shared client timeout would cut it off.
	httpClient := *s.client.httpClient
	httpClient.Timeout = 0

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return s.client.errorFromResponse(request{method: req.Method, endpoint: "/consume/" + consumerKey + "/stream"}, resp)
	}

	for event, err := range parseSSE(resp.Body) {
		if err != nil {
			return err
		}
		switch event.Event {
		case "batch":
			var batch []json.RawMessage
			if err := json.Unmarshal(event.Data, &batch); err != nil {
				return fmt.Errorf("scambus: decode SSE batch: %w", err)
			}
			for _, raw := range batch {
				if err := deliver(raw, cursor, fn); err != nil {
					return err
				}
			}
		case "message":
			if err := deliver(event.Data, cursor, fn); err != nil {
				return err
			}
		case "error":
			return fmt.Errorf("scambus: SSE stream error: %s", strings.TrimSpace(string(event.Data)))
		}
	}
	return nil
}

func deliver(raw json.RawMessage, cursor *string, fn func(StreamMessage) error) error {
	var envelope struct {
		Cursor string `json:"cursor"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if envelope.Cursor != "" {
		*cursor = envelope.Cursor
	}
	if err := fn(StreamMessage{Cursor: envelope.Cursor, Raw: raw}); err != nil {
		if errors.Is(err, ErrStopSubscription) {
			return errStopSubscription
		}
		return err
	}
	return nil
}

// ErrStopSubscription ends a Subscribe loop cleanly when returned by its callback.
var ErrStopSubscription = errors.New("scambus: stop subscription")

func parseSSE(r io.Reader) func(func(SSEEvent, error) bool) {
	return func(yield func(SSEEvent, error) bool) {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

		var event SSEEvent
		var data bytes.Buffer

		flush := func() bool {
			if data.Len() == 0 && event.Event == "" && event.ID == "" {
				return true
			}
			event.Data = bytes.Clone(bytes.TrimSuffix(data.Bytes(), []byte("\n")))
			ok := yield(event, nil)
			event = SSEEvent{}
			data.Reset()
			return ok
		}

		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "":
				if !flush() {
					return
				}
			case strings.HasPrefix(line, ":"):
			default:
				field, value, found := strings.Cut(line, ":")
				if found {
					value = strings.TrimPrefix(value, " ")
				}
				switch field {
				case "event":
					event.Event = value
				case "id":
					event.ID = value
				case "data":
					data.WriteString(value)
					data.WriteByte('\n')
				}
			}
		}
		if err := scanner.Err(); err != nil {
			yield(SSEEvent{}, err)
			return
		}
		flush()
	}
}
