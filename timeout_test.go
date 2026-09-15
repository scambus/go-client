package scambus

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

// slowBody sends the headers and the first half of a JSON body, then pauses before the rest,
// so the body is still arriving after the client has the response headers.
func slowBody(first, rest string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(first))
		w.(http.Flusher).Flush()
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(rest))
	}
}

func TestATimedRequestReadsABodyThatArrivesAfterTheHeaders(t *testing.T) {
	srv := newServer(t, slowBody(`{"data":[{"id":"t1",`, `"title":"Scam Type"}],"pagination":{"total_pages":1}}`))
	c := srv.client(t, WithTimeout(5*time.Second))

	result, err := c.Tags.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].Title != "Scam Type" {
		t.Fatalf("got %+v", result.Data)
	}
}

func TestAPollReadsABodyThatArrivesAfterTheHeaders(t *testing.T) {
	srv := newServer(t, slowBody(`{"messages":[],`, `"next_cursor":"1-2","has_more":false}`))
	c := srv.client(t, WithTimeout(5*time.Second))

	result, err := c.Consume.Poll(context.Background(), "key", nil)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if result.NextCursor != "1-2" {
		t.Fatalf("got %+v", result)
	}
}

func TestTheTimeoutStillBoundsTheBodyRead(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[`))
		w.(http.Flusher).Flush()
		time.Sleep(time.Second)
	})
	c := srv.client(t, WithTimeout(200*time.Millisecond))

	_, err := c.Tags.List(context.Background(), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want a deadline error, got %v", err)
	}
}
