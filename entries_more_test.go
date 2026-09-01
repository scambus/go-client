package scambus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateEmailSendsDetails(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "email"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	sentAt := NewTime(time.Date(2025, 2, 2, 8, 0, 0, 0, time.UTC))
	_, err := c.Journal.CreateEmail(context.Background(), EmailInput{
		Description: "phishing email",
		Direction:   "inbound",
		Subject:     "Your account is locked",
		SentAt:      sentAt,
		Body:        "click here",
		Headers:     map[string]string{"Return-Path": "a@b.test"},
		Media:       []Media{{ID: "m1", MimeType: "image/png"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Type        string       `json:"type"`
		PerformedAt Time         `json:"performed_at"`
		Details     EmailDetails `json:"details"`
		Evidence    Evidence     `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if body.Type != "email" {
		t.Fatalf("type %q", body.Type)
	}
	if body.Details.Subject != "Your account is locked" || body.Details.Direction != "inbound" {
		t.Fatalf("details %+v", body.Details)
	}
	if !body.PerformedAt.Equal(sentAt.Time) {
		t.Fatalf("performed_at %v", body.PerformedAt.Time)
	}
	if body.Evidence.Type != "screenshot" || len(body.Evidence.MediaIDs) != 1 {
		t.Fatalf("evidence %+v", body.Evidence)
	}
}

func TestCreateTextConversationKeepsSuppliedDetails(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "text_conversation"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Now().UTC())
	_, err := c.Journal.CreateTextConversation(context.Background(), TextConversationInput{
		Description: "whatsapp thread",
		Platform:    "whatsapp",
		StartTime:   start,
		EndTime:     start,
		Details:     &TextConversationDetails{ConversationID: "abc", ParticipantCount: Ptr(3)},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Details TextConversationDetails `json:"details"`
	}
	srv.requests[0].decode(t, &body)
	if body.Details.Platform != "whatsapp" {
		t.Fatalf("platform must be filled from the input, got %q", body.Details.Platform)
	}
	if body.Details.ConversationID != "abc" || *body.Details.ParticipantCount != 3 {
		t.Fatalf("details %+v", body.Details)
	}
}

func TestCreateNoteDefaultsPerformedAt(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "note"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	before := time.Now().UTC().Add(-time.Second)
	_, err := c.Journal.CreateNote(context.Background(), NoteInput{
		Description: "observed pattern",
		Details:     &NoteDetails{Content: "text", NotedAt: NewTime(time.Now().UTC())},
		Media:       []Media{{ID: "m1", MimeType: "application/pdf"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		PerformedAt Time     `json:"performed_at"`
		Evidence    Evidence `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if body.PerformedAt.Before(before) {
		t.Fatalf("performed_at %v should default to now", body.PerformedAt.Time)
	}
	if body.Evidence.Type != "document" {
		t.Fatalf("evidence type %q", body.Evidence.Type)
	}
}

func TestCreateImportAndExport(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "import"}, map[string]any{"id": "e1"})
	c := srv.client(t)
	ctx := context.Background()

	if _, err := c.Journal.CreateImport(ctx, ImportInput{
		Description: "csv load",
		Details:     &ImportDetails{Source: "partner", RecordCount: 42, ImportedAt: NewTime(time.Now().UTC())},
	}); err != nil {
		t.Fatal(err)
	}
	var imported struct {
		Type    string        `json:"type"`
		Details ImportDetails `json:"details"`
	}
	srv.requests[0].decode(t, &imported)
	if imported.Type != "import" || imported.Details.RecordCount != 42 {
		t.Fatalf("got %+v", imported)
	}

	if _, err := c.Journal.CreateExport(ctx, ExportInput{
		Description: "csv dump",
		Details:     &ExportDetails{Destination: "partner", RecordCount: 7, ExportedAt: NewTime(time.Now().UTC())},
	}); err != nil {
		t.Fatal(err)
	}
	var exported struct {
		Type    string        `json:"type"`
		Details ExportDetails `json:"details"`
	}
	srv.requests[2].decode(t, &exported)
	if exported.Type != "export" || exported.Details.RecordCount != 7 {
		t.Fatalf("got %+v", exported)
	}
}

func TestCommonFieldsReachTheWire(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "note"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	_, err := c.Journal.CreateNote(context.Background(), NoteInput{
		Description:                "everything set",
		Identifiers:                []IdentifierLookup{{Type: "phone", Value: "+12125551234"}},
		OurIdentifiers:             []IdentifierLookup{{Type: "phone", Value: "+13125550000"}},
		CaseID:                     "c1",
		Tags:                       []TagLookup{{TagName: "ScamType", TagValue: "Phishing"}},
		Metadata:                   map[string]any{"batch": "b1"},
		ParentJournalEntryID:       "e0",
		Originator:                 &OriginatorLookup{Type: "user", Identifier: "u1", CreateIfNotExist: true},
		IsTest:                     true,
		ExternalIdentifiers:        []ExternalIdentifierInput{{ExternalSystem: "sys", ExternalID: "x1"}},
		ExtractExternalIdentifiers: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var body createEntryBody
	srv.requests[0].decode(t, &body)
	if len(body.IdentifierLookups) != 1 || len(body.OurIdentifierLookups) != 1 {
		t.Fatalf("identifier lookups %+v", body)
	}
	if body.CaseID != "c1" || body.ParentJournalEntryID != "e0" {
		t.Fatalf("ids %+v", body)
	}
	if len(body.TagLookups) != 1 || body.TagLookups[0].TagValue != "Phishing" {
		t.Fatalf("tags %+v", body.TagLookups)
	}
	if body.OriginatorLookup == nil || !body.OriginatorLookup.CreateIfNotExist {
		t.Fatalf("originator %+v", body.OriginatorLookup)
	}
	if !body.IsTest || !body.ExtractExternalIdentifiers {
		t.Fatalf("flags %+v", body)
	}
	if body.Metadata["batch"] != "b1" {
		t.Fatalf("metadata %+v", body.Metadata)
	}
}

func TestPollResultJournalEntryMessages(t *testing.T) {
	result := PollResult{Messages: []json.RawMessage{
		json.RawMessage(`{"id":"e1","type":"detection","confidence":0.5}`),
	}}
	messages, err := result.JournalEntryMessages()
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].ID != "e1" {
		t.Fatalf("got %+v", messages)
	}
}

func TestAPIErrorMessageIncludesRoute(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "missing", Method: "GET", Endpoint: "/cases/c1"}
	if got := err.Error(); got != "scambus: GET /cases/c1: 404 missing" {
		t.Fatalf("got %q", got)
	}
	bare := &APIError{StatusCode: 500, Message: "oops"}
	if got := bare.Error(); got != "scambus: 500 oops" {
		t.Fatalf("got %q", got)
	}
}

func TestAPIErrorFallsBackToStatusText(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	c := srv.client(t, WithMaxRetries(0), WithLogger(slog.New(slog.NewTextHandler(os.Stderr, nil))))

	_, err := c.Cases.Get(context.Background(), "c1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %v", err)
	}
	if apiErr.Message != "Bad Gateway" {
		t.Fatalf("message %q", apiErr.Message)
	}
}

func TestDownloadFileExport(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("id,value\n1,x\n"))
	})
	c := srv.client(t)

	var buf bytes.Buffer
	n, err := c.FileExports.Download(context.Background(), "f1", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(buf.Len()) || buf.String() != "id,value\n1,x\n" {
		t.Fatalf("got %q (%d bytes)", buf.String(), n)
	}
	if srv.last().Path != "/api/file-exports/f1/download" {
		t.Fatalf("path %q", srv.last().Path)
	}

	path := filepath.Join(t.TempDir(), "out", "export.csv")
	if err := c.FileExports.DownloadToFile(context.Background(), "f1", path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "id,value\n1,x\n" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchIdentifiersAllPaginates(t *testing.T) {
	page := 0
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			writeJSON(t, w, 200, map[string]any{
				"data":       []any{map[string]any{"id": "i1", "type": "phone", "display_value": "+1"}},
				"nextCursor": "c2",
				"hasMore":    true,
			})
			return
		}
		writeJSON(t, w, 200, map[string]any{
			"data":    []any{map[string]any{"id": "i2", "type": "email", "display_value": "a@b.test"}},
			"hasMore": false,
		})
	})
	c := srv.client(t)

	var ids []string
	err := c.Search.IdentifiersAll(context.Background(), SearchIdentifiersInput{}, func(i Identifier) error {
		ids = append(ids, i.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %v", ids)
	}
}

func TestExecuteMyViewsResolveThenExecute(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(t, w, 200, View{ID: "v1", EntityType: "identifier"})
			return
		}
		writeJSON(t, w, 200, map[string]any{
			"data":        []any{map[string]any{"id": "i1", "type": "phone", "display_value": "+1"}},
			"entity_type": "identifier",
		})
	})
	c := srv.client(t)
	ctx := context.Background()

	result, err := c.Views.ExecuteMyPinboard(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	identifiers, err := result.Identifiers()
	if err != nil {
		t.Fatal(err)
	}
	if len(identifiers) != 1 || identifiers[0].ID != "i1" {
		t.Fatalf("got %+v", identifiers)
	}
	if srv.requests[0].Path != "/api/views/my-pinboard" {
		t.Fatalf("path %q", srv.requests[0].Path)
	}

	if _, err := c.Views.ExecuteMyJournalEntries(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if srv.requests[2].Path != "/api/views/my-journal-entries" {
		t.Fatalf("path %q", srv.requests[2].Path)
	}
}

func TestConfidenceHelpers(t *testing.T) {
	raw, err := json.Marshal(NewConfidence(0.25))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "0.25" {
		t.Fatalf("got %s", raw)
	}
	unset, err := json.Marshal(Confidence{})
	if err != nil {
		t.Fatal(err)
	}
	if string(unset) != "null" {
		t.Fatalf("got %s", unset)
	}
}

func TestReportIsFailed(t *testing.T) {
	if !(Report{Status: "failed"}).IsFailed() {
		t.Fatal("failed report not detected")
	}
	if (Report{Status: "completed"}).IsFailed() {
		t.Fatal("completed report reported as failed")
	}
}
