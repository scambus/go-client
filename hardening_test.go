package scambus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCredentialsNeverFollowARedirect(t *testing.T) {
	var harvested http.Header
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		harvested = r.Header.Clone()
		writeJSON(t, w, 200, map[string]any{})
	}))
	defer attacker.Close()

	victim := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL+"/harvest", http.StatusFound)
	})
	t.Setenv("HOME", t.TempDir())
	c, err := New(WithAPIURL(victim.URL), WithAPIKey("kid", "SECRET"), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.Cases.Get(context.Background(), "c1"); err == nil {
		t.Fatal("following the redirect must fail")
	}
	if harvested.Get("X-API-Key") != "" {
		t.Fatalf("API key leaked to the redirect target: %q", harvested.Get("X-API-Key"))
	}
}

func TestPathSegmentsCannotEscapeOrInjectQuery(t *testing.T) {
	cases := map[string]string{
		"query injection":  "c1?admin=true",
		"fragment":         "c1#frag",
		"parent traversal": "../admin/url-consolidation/start",
		"nested traversal": "c1/../../automations",
		"current dir":      "./c1",
	}
	for name, id := range cases {
		srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{})
		})
		c := srv.client(t)
		if _, err := c.Cases.Get(context.Background(), id); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s (%q): want a validation error, got %v", name, id, err)
		}
		if len(srv.requests) != 0 {
			t.Fatalf("%s: request must not be sent, got path %q", name, srv.requests[0].Path)
		}
	}
}

func TestPathSegmentsAreEscaped(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{})
	})
	c := srv.client(t)
	if _, err := c.Cases.Get(context.Background(), "a b%c"); err != nil {
		t.Fatal(err)
	}
	if got := srv.last().Path; got != "/api/cases/a b%c" {
		t.Fatalf("server decoded path %q", got)
	}
}

func TestPostIsNotRetriedOnServerFault(t *testing.T) {
	var posts atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		w.Header().Set("Retry-After", "0")
		writeJSON(t, w, 502, map[string]string{"error": "bad gateway"})
	})
	c := srv.client(t, WithMaxRetries(5), WithRetryMaxTime(10*time.Second))

	_, err := c.Journal.Create(context.Background(), CreateEntryInput{Type: EntryTypeNote, Description: "x"})
	if !errors.Is(err, ErrServer) {
		t.Fatalf("got %v", err)
	}
	if posts.Load() != 1 {
		t.Fatalf("a 5xx must not replay a POST: %d attempts", posts.Load())
	}
}

func TestPostIsRetriedWhenServerSaysItDidNotProcess(t *testing.T) {
	var posts atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if posts.Add(1) < 2 {
				w.Header().Set("Retry-After", "0")
				writeJSON(t, w, 429, map[string]string{"error": "slow down"})
				return
			}
			writeJSON(t, w, 201, map[string]any{"id": "e1"})
			return
		}
		writeJSON(t, w, 200, journalEntryResponse(map[string]any{"id": "e1", "type": "note"}))
	})
	c := srv.client(t, WithMaxRetries(5), WithRetryMaxTime(10*time.Second))

	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{Type: EntryTypeNote, Description: "x"}); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 {
		t.Fatalf("429 should replay a POST: %d attempts", posts.Load())
	}
}

func TestIdempotentMethodsStillRetryOnServerFault(t *testing.T) {
	var gets atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if gets.Add(1) < 3 {
			w.Header().Set("Retry-After", "0")
			writeJSON(t, w, 503, map[string]string{"error": "unavailable"})
			return
		}
		writeJSON(t, w, 200, Case{ID: "c1"})
	})
	c := srv.client(t, WithMaxRetries(5), WithRetryMaxTime(10*time.Second))

	if _, err := c.Cases.Get(context.Background(), "c1"); err != nil {
		t.Fatal(err)
	}
	if gets.Load() != 3 {
		t.Fatalf("GET should retry: %d attempts", gets.Load())
	}
}

func TestPostConnectionFailureIsNotReplayed(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	c, err := New(WithAPIURL(srv.URL), WithToken("t"), WithMaxRetries(5), WithRetryMaxTime(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{Type: EntryTypeNote, Description: "x"}); err == nil {
		t.Fatal("want an error")
	}
	if hits.Load() != 1 {
		t.Fatalf("a dropped connection must not replay a POST: %d attempts", hits.Load())
	}
}

func TestOversizedResponseIsRefused(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"` + strings.Repeat("A", 4096) + `"}`))
	})
	c := srv.client(t, WithMaxResponseBytes(512))

	if _, err := c.Cases.Get(context.Background(), "c1"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("got %v", err)
	}
}

func TestVersionedBaseURLIsLeftAlone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cases := map[string]string{
		"https://host":         "https://host/api",
		"https://host/":        "https://host/api",
		"https://host/api":     "https://host/api",
		"https://host/api/":    "https://host/api",
		"https://host/api/v2":  "https://host/api/v2",
		"https://host/gateway": "https://host/gateway",
	}
	for in, want := range cases {
		c, err := New(WithAPIURL(in), WithToken("t"))
		if err != nil {
			t.Fatal(err)
		}
		if c.APIURL() != want {
			t.Fatalf("%s -> %s, want %s", in, c.APIURL(), want)
		}
	}
}
