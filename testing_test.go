package scambus

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type capturedRequest struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   []byte
}

type recordingServer struct {
	*httptest.Server
	t        *testing.T
	requests []capturedRequest
}

func newServer(t *testing.T, handler http.HandlerFunc) *recordingServer {
	t.Helper()
	rec := &recordingServer{t: t}
	rec.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.requests = append(rec.requests, capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
			Header: r.Header.Clone(),
			Body:   body,
		})
		handler(w, r)
	}))
	t.Cleanup(rec.Close)
	return rec
}

func (s *recordingServer) client(t *testing.T, opts ...Option) *Client {
	t.Helper()
	t.Setenv("SCAMBUS_API_KEY_ID", "")
	t.Setenv("SCAMBUS_API_KEY_SECRET", "")
	t.Setenv("SCAMBUS_API_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	all := append([]Option{WithAPIURL(s.URL), WithToken("test-token"), WithMaxRetries(0)}, opts...)
	c, err := New(all...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func (s *recordingServer) last() capturedRequest {
	s.t.Helper()
	if len(s.requests) == 0 {
		s.t.Fatal("no requests recorded")
	}
	return s.requests[len(s.requests)-1]
}

func (r capturedRequest) decode(t *testing.T, out any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, out); err != nil {
		t.Fatalf("decode request body %q: %v", r.Body, err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
