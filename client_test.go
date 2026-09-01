package scambus

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRequiresCredentials(t *testing.T) {
	t.Setenv("SCAMBUS_API_KEY_ID", "")
	t.Setenv("SCAMBUS_API_KEY_SECRET", "")
	t.Setenv("SCAMBUS_API_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	if _, err := New(WithAPIURL("https://example.test/api")); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("want ErrNoCredentials, got %v", err)
	}
}

func TestNewAppendsAPISuffix(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	c, err := New(WithAPIURL("https://example.test"), WithToken("t"))
	if err != nil {
		t.Fatal(err)
	}
	if c.APIURL() != "https://example.test/api" {
		t.Fatalf("got %q", c.APIURL())
	}
}

func TestNewKeepsExistingAPISuffix(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	c, err := New(WithAPIURL("https://example.test/api/"), WithToken("t"))
	if err != nil {
		t.Fatal(err)
	}
	if c.APIURL() != "https://example.test/api" {
		t.Fatalf("got %q", c.APIURL())
	}
}

func TestAPIKeyAuthWinsOverToken(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{})
	})
	t.Setenv("HOME", t.TempDir())
	c, err := New(WithAPIURL(srv.URL), WithAPIKey("kid", "secret"), WithToken("jwt"), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Media.Get(context.Background(), "m1"); err != nil {
		t.Fatal(err)
	}
	got := srv.last().Header.Get("X-API-Key")
	if got != "kid:secret" {
		t.Fatalf("X-API-Key = %q", got)
	}
	if srv.last().Header.Get("Authorization") != "" {
		t.Fatal("Authorization must not be sent alongside an API key")
	}
}

func TestBearerTokenAndUserAgent(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{})
	})
	c := srv.client(t)
	if _, err := c.Media.Get(context.Background(), "m1"); err != nil {
		t.Fatal(err)
	}
	if got := srv.last().Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := srv.last().Header.Get("User-Agent"); !strings.HasPrefix(got, "scambus-go-client/") {
		t.Fatalf("User-Agent = %q", got)
	}
}

func TestErrorMappingByStatus(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{401, ErrAuthentication},
		{403, ErrAuthentication},
		{400, ErrValidation},
		{404, ErrNotFound},
		{410, ErrCursorExpired},
		{416, ErrCursorExpired},
		{503, ErrServer},
	}
	for _, tc := range cases {
		srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, tc.status, map[string]string{"error": "boom"})
		})
		c := srv.client(t)
		_, err := c.Cases.Get(context.Background(), "c1")
		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: want %v, got %v", tc.status, tc.want, err)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("status %d: not an *APIError: %v", tc.status, err)
		}
		if apiErr.StatusCode != tc.status || apiErr.Message != "boom" {
			t.Fatalf("status %d: got %+v", tc.status, apiErr)
		}
	}
}

func TestRetriesTransientStatusThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.Header().Set("Retry-After", "0")
			writeJSON(t, w, 503, map[string]string{"error": "unavailable"})
			return
		}
		writeJSON(t, w, 200, Case{ID: "c1", Title: "ok"})
	})
	c := srv.client(t, WithMaxRetries(5), WithRetryMaxTime(10*time.Second))

	got, err := c.Cases.Get(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "ok" {
		t.Fatalf("got %+v", got)
	}
	if calls.Load() != 3 {
		t.Fatalf("want 3 attempts, got %d", calls.Load())
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeJSON(t, w, 400, map[string]string{"error": "bad"})
	})
	c := srv.client(t, WithMaxRetries(5))

	if _, err := c.Cases.Get(context.Background(), "c1"); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("want 1 attempt, got %d", calls.Load())
	}
}

func TestRetriesExhaustedReturnsLastError(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		writeJSON(t, w, 500, map[string]string{"error": "nope"})
	})
	c := srv.client(t, WithMaxRetries(2), WithRetryMaxTime(5*time.Second))

	_, err := c.Cases.Get(context.Background(), "c1")
	if !errors.Is(err, ErrServer) {
		t.Fatalf("got %v", err)
	}
	if len(srv.requests) != 3 {
		t.Fatalf("want 3 attempts, got %d", len(srv.requests))
	}
}

func TestContextCancelStopsRetries(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 503, map[string]string{"error": "unavailable"})
	})
	c := srv.client(t, WithMaxRetries(50), WithRetryMaxTime(time.Minute))

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	if _, err := c.Cases.Get(ctx, "c1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestNoContentIsNotAnError(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	c := srv.client(t)
	if err := c.Views.Delete(context.Background(), "v1"); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidJSONIsReported(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("<html>gateway</html>"))
	})
	c := srv.client(t)
	_, err := c.Cases.Get(context.Background(), "c1")
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("got %v", err)
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", time.Now().Add(3*time.Second).UTC().Format(http.TimeFormat))
	if d := retryAfter(resp); d < time.Second || d > 4*time.Second {
		t.Fatalf("got %v", d)
	}
}

func TestRetryAfterAbsent(t *testing.T) {
	if d := retryAfter(&http.Response{Header: http.Header{}}); d != -1 {
		t.Fatalf("got %v", d)
	}
}

func TestBackoffStaysWithinCeiling(t *testing.T) {
	for attempt := 1; attempt < 12; attempt++ {
		if d := backoff(attempt, retryBaseDelay); d < 0 || d > retryMaxBackoff {
			t.Fatalf("attempt %d gave %v", attempt, d)
		}
	}
}
