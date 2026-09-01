package scambus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func collectSSE(t *testing.T, body string) []SSEEvent {
	t.Helper()
	var events []SSEEvent
	for event, err := range parseSSE(strings.NewReader(body)) {
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func TestParseSSENamedEvents(t *testing.T) {
	events := collectSSE(t, "event: connected\ndata: {\"stream\":\"s1\"}\n\nevent: message\ndata: {\"id\":\"e1\"}\n\n")
	if len(events) != 2 {
		t.Fatalf("got %d events: %+v", len(events), events)
	}
	if events[0].Event != "connected" || string(events[0].Data) != `{"stream":"s1"}` {
		t.Fatalf("got %+v", events[0])
	}
	if events[1].Event != "message" {
		t.Fatalf("got %+v", events[1])
	}
}

func TestParseSSEIgnoresHeartbeatComments(t *testing.T) {
	events := collectSSE(t, ": heartbeat\n\nevent: message\ndata: {}\n\n: heartbeat\n\n")
	if len(events) != 1 || events[0].Event != "message" {
		t.Fatalf("got %+v", events)
	}
}

func TestParseSSEJoinsMultilineData(t *testing.T) {
	events := collectSSE(t, "event: message\ndata: {\"a\":1,\ndata: \"b\":2}\n\n")
	if len(events) != 1 {
		t.Fatalf("got %+v", events)
	}
	if string(events[0].Data) != "{\"a\":1,\n\"b\":2}" {
		t.Fatalf("got %q", events[0].Data)
	}
}

func TestParseSSEHandlesMissingTrailingBlankLine(t *testing.T) {
	events := collectSSE(t, "event: message\ndata: {\"id\":\"e1\"}\n")
	if len(events) != 1 || string(events[0].Data) != `{"id":"e1"}` {
		t.Fatalf("got %+v", events)
	}
}

func sseServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *recordingServer {
	t.Helper()
	return newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		handler(w, r)
	})
}

func TestSubscribeDeliversBatchThenMessages(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "event: connected\ndata: {\"stream\":\"s1\"}\n\n")
		fmt.Fprint(w, "event: batch\ndata: [{\"identifier_id\":\"i1\",\"cursor\":\"1-0\"},{\"identifier_id\":\"i2\",\"cursor\":\"1-1\"}]\n\n")
		fmt.Fprint(w, "event: message\ndata: {\"identifier_id\":\"i3\",\"cursor\":\"1-2\"}\n\n")
	})
	c := srv.client(t)

	var ids []string
	err := c.Consume.Subscribe(context.Background(), "ck", nil, func(m StreamMessage) error {
		msg, err := m.Identifier()
		if err != nil {
			return err
		}
		ids = append(ids, msg.IdentifierID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != "i1" || ids[2] != "i3" {
		t.Fatalf("got %v", ids)
	}
	if srv.last().Header.Get("Accept") != "text/event-stream" {
		t.Fatalf("Accept %q", srv.last().Header.Get("Accept"))
	}
	if srv.last().Query != "cursor=%24&include_test=false" {
		t.Fatalf("query %q", srv.last().Query)
	}
}

func TestSubscribeStopsWhenCallbackSignals(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		for i := range 5 {
			fmt.Fprintf(w, "event: message\ndata: {\"identifier_id\":\"i%d\",\"cursor\":\"1-%d\"}\n\n", i, i)
		}
	})
	c := srv.client(t)

	seen := 0
	err := c.Consume.Subscribe(context.Background(), "ck", nil, func(m StreamMessage) error {
		seen++
		if seen == 2 {
			return ErrStopSubscription
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen != 2 {
		t.Fatalf("got %d messages", seen)
	}
}

func TestSubscribePropagatesCallbackError(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "event: message\ndata: {\"identifier_id\":\"i1\"}\n\n")
	})
	c := srv.client(t)

	sentinel := errors.New("downstream write failed")
	err := c.Consume.Subscribe(context.Background(), "ck", nil, func(StreamMessage) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v", err)
	}
}

func TestSubscribeReportsServerErrorEvent(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "event: error\ndata: {\"error\":\"stream rebuilding\"}\n\n")
	})
	c := srv.client(t)

	err := c.Consume.Subscribe(context.Background(), "ck", nil, func(StreamMessage) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "stream rebuilding") {
		t.Fatalf("got %v", err)
	}
}

func TestSubscribeDoesNotRetryAuthFailure(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeJSON(t, w, 401, map[string]string{"error": "bad key"})
	})
	c := srv.client(t)

	err := c.Consume.Subscribe(context.Background(), "ck", &SubscribeOptions{
		Reconnect:      true,
		MaxReconnects:  3,
		ReconnectDelay: time.Millisecond,
	}, func(StreamMessage) error { return nil })

	if !errors.Is(err, ErrAuthentication) {
		t.Fatalf("got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("want 1 attempt, got %d", calls.Load())
	}
}

func TestSubscribeReconnectsResumingFromLastCursor(t *testing.T) {
	var connections atomic.Int32
	var cursors []string
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		cursors = append(cursors, r.URL.Query().Get("cursor"))
		if connections.Add(1) == 1 {
			fmt.Fprint(w, "event: message\ndata: {\"identifier_id\":\"i1\",\"cursor\":\"1-0\"}\n\n")
			return
		}
		fmt.Fprint(w, "event: message\ndata: {\"identifier_id\":\"i2\",\"cursor\":\"1-1\"}\n\n")
	})
	c := srv.client(t)

	var ids []string
	err := c.Consume.Subscribe(context.Background(), "ck", &SubscribeOptions{
		Reconnect:      true,
		ReconnectDelay: time.Millisecond,
	}, func(m StreamMessage) error {
		msg, _ := m.Identifier()
		ids = append(ids, msg.IdentifierID)
		if len(ids) == 2 {
			return ErrStopSubscription
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cursors) != 2 {
		t.Fatalf("want 2 connections, got %v", cursors)
	}
	if cursors[0] != CursorEnd {
		t.Fatalf("first cursor %q", cursors[0])
	}
	if cursors[1] != "1-0" {
		t.Fatalf("reconnect must resume from the last cursor, got %q", cursors[1])
	}
	if len(ids) != 2 {
		t.Fatalf("got %v", ids)
	}
}

func TestSubscribeHonoursContextCancel(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		for {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(5 * time.Millisecond):
				fmt.Fprint(w, ": heartbeat\n\n")
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	})
	c := srv.client(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := c.Consume.Subscribe(ctx, "ck", &SubscribeOptions{Reconnect: true, ReconnectDelay: time.Millisecond},
		func(StreamMessage) error { return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestSubscribeDecodesJournalEntryMessages(t *testing.T) {
	srv := sseServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "event: message\ndata: {\"id\":\"e1\",\"type\":\"phone_call\",\"confidence\":1,\"identifiers\":[{\"id\":\"i1\",\"type\":\"phone\",\"display_value\":\"+12125551234\",\"confidence\":1}]}\n\n")
	})
	c := srv.client(t)

	var got JournalEntryStreamMessage
	err := c.Consume.Subscribe(context.Background(), "ck", nil, func(m StreamMessage) error {
		msg, err := m.JournalEntry()
		got = msg
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "e1" || len(got.Identifiers) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got.Identifiers[0].DisplayValue != "+12125551234" {
		t.Fatalf("got %+v", got.Identifiers[0])
	}
}
