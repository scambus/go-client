package scambus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryCarriesFilterIncludeFlags(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"data": []any{}, "hasMore": false})
	})
	c := srv.client(t)

	_, err := c.Journal.Query(context.Background(), QueryEntriesInput{
		Filter: &FilterCriteria{IncludeIdentifiers: Ptr(true), IncludeEvidence: Ptr(true)},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	srv.last().decode(t, &body)
	if body["include_identifiers"] != true || body["include_evidence"] != true {
		t.Fatalf("filter flags were dropped: %+v", body)
	}
}

func TestQueryOptionsAlsoSetIncludeFlags(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"data": []any{}, "hasMore": false})
	})
	c := srv.client(t)

	if _, err := c.Journal.Query(context.Background(), QueryEntriesInput{IncludeIdentifiers: true}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["include_identifiers"] != true {
		t.Fatalf("got %+v", body)
	}
}

func TestQueryAllStopsOnAStuckCursor(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeJSON(t, w, 200, map[string]any{"data": []any{}, "nextCursor": "stuck", "hasMore": true})
	})
	c := srv.client(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.Journal.QueryAll(ctx, QueryEntriesInput{Cursor: "stuck"}, func(JournalEntry) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("a cursor that does not advance must stop the walk: %d requests", calls.Load())
	}
}

func TestCreateReturnsIDWhenReloadFails(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(t, w, 201, map[string]any{"id": "e1"})
			return
		}
		writeJSON(t, w, 500, map[string]string{"error": "boom"})
	})
	c := srv.client(t, WithMaxRetries(0))

	entry, err := c.Journal.Create(context.Background(), CreateEntryInput{Type: EntryTypeNote, Description: "x"})
	if err == nil {
		t.Fatal("want an error describing the failed reload")
	}
	if entry == nil || entry.ID != "e1" {
		t.Fatalf("the id of the committed entry must survive: %+v", entry)
	}
}

func TestCompleteActivityRefusesNegativeDuration(t *testing.T) {
	start := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, journalEntryResponse(map[string]any{
			"id": "e1", "type": "phone_call", "start_time": start.Format(time.RFC3339),
		}))
	})
	c := srv.client(t)

	_, err := c.Journal.CompleteActivity(context.Background(), "e1", NewTime(start.Add(-time.Hour)), "", "")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestExplicitEndTimeSurvivesInProgress(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "note"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC))
	end := NewTime(start.Add(time.Hour))
	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{
		Type: EntryTypeNote, Description: "x", StartTime: start, EndTime: end, InProgress: true,
	}); err != nil {
		t.Fatal(err)
	}

	var body struct {
		EndTime Time `json:"end_time"`
	}
	srv.requests[0].decode(t, &body)
	if !body.EndTime.Equal(end.Time) {
		t.Fatalf("an explicit end time must not be dropped, got %v", body.EndTime.Time)
	}
}

func TestDisableAIExtractBeatsATranscript(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "phone_call"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Now().UTC())
	_, err := c.Journal.CreatePhoneCall(context.Background(), PhoneCallInput{
		Description: "call", Direction: "inbound", StartTime: start, EndTime: start,
		Transcript:       []ConversationMessage{{Index: 0, MessageID: "m", Timestamp: start, Body: "hi"}},
		DisableAIExtract: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.requests[0].decode(t, &body)
	if _, present := body["ai_extract"]; present {
		t.Fatalf("ai_extract should be absent, got %v", body["ai_extract"])
	}
}

func TestImportAndExportAttachMedia(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "import"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	if _, err := c.Journal.CreateImport(context.Background(), ImportInput{
		Description: "csv", Media: []Media{{ID: "m1"}},
	}); err != nil {
		t.Fatal(err)
	}
	var body struct {
		Evidence []Evidence `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if len(body.Evidence) != 1 || body.Evidence[0].MediaID != "m1" {
		t.Fatalf("import media was dropped: %+v", body.Evidence)
	}
}

func TestTagValueOrderZeroIsSent(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, TagValue{ID: "v1"})
	})
	c := srv.client(t)

	if _, err := c.Tags.CreateValue(context.Background(), "t1", CreateTagValue{Title: "First"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if _, present := body["order"]; !present {
		t.Fatalf("order 0 is a real position and must be sent: %+v", body)
	}
}

func TestZeroDetailTimestampsAreOmitted(t *testing.T) {
	raw, err := json.Marshal(NoteDetails{Content: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "null") {
		t.Fatalf("an unset timestamp must be omitted, not sent as null: %s", raw)
	}
}

func TestQueueUpdateResendsEveryFieldFromCurrentState(t *testing.T) {
	var put map[string]any
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(t, w, 200, Queue{
				ID: "q1", Name: "live", Description: "keep me",
				CadenceDays: 7, CooldownHours: 24, PriorityMode: "fifo",
				RotationEnabled: true, AutoPopulate: true, IsActive: true,
			})
			return
		}
		body, _ := json.Marshal(map[string]any{})
		_ = body
		writeJSON(t, w, 200, Queue{ID: "q1"})
	})
	c := srv.client(t)

	if _, err := c.Queues.Update(context.Background(), "q1", QueuePatch{Name: Ptr("renamed")}); err != nil {
		t.Fatal(err)
	}
	srv.last().decode(t, &put)

	if put["name"] != "renamed" {
		t.Fatalf("patch not applied: %+v", put)
	}
	for field, want := range map[string]any{
		"description":      "keep me",
		"is_active":        true,
		"auto_populate":    true,
		"rotation_enabled": true,
		"priority_mode":    "fifo",
	} {
		if put[field] != want {
			t.Fatalf("%s was reset to %v; the server writes omitted fields as zero", field, put[field])
		}
	}
}

func TestTwoFactorUsesTheEnableKey(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"success": true, "enabled": true})
	})
	c := srv.client(t)

	result, err := c.Sessions.SetTwoFactor(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Enabled {
		t.Fatalf("got %+v", result)
	}

	var body map[string]any
	srv.last().decode(t, &body)
	if body["enable"] != true {
		t.Fatalf("the server reads \"enable\"; sending %q turns 2FA off: %+v", "enabled", body)
	}
}

func TestAPIKeySecretIsReadFromTheKeyField(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"id": "k1", "name": "primary", "key": "sk_live_secret", "secret_id": "k1", "is_active": true,
		})
	})
	c := srv.client(t)

	key, err := c.Automations.CreateAPIKey(context.Background(), "a1", "primary", "")
	if err != nil {
		t.Fatal(err)
	}
	if key.Key != "sk_live_secret" {
		t.Fatalf("the secret is returned once and must not be dropped: %+v", key)
	}
}

func TestDownloadFailureLeavesTheExistingFileIntact(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 404, map[string]string{"error": "gone"})
	})
	c := srv.client(t, WithMaxRetries(0))

	path := filepath.Join(t.TempDir(), "report.pdf")
	if err := os.WriteFile(path, []byte("PREVIOUS GOOD REPORT"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := c.Reports.DownloadToFile(context.Background(), "r1", path); err == nil {
		t.Fatal("want an error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "PREVIOUS GOOD REPORT" {
		t.Fatalf("a failed download truncated the existing file: %q", got)
	}
}

func TestDownloadedReportIsNotWorldReadable(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("%PDF-1.4"))
	})
	c := srv.client(t)

	path := filepath.Join(t.TempDir(), "evidence.pdf")
	if err := c.Reports.DownloadToFile(context.Background(), "r1", path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("evidence must not be group- or world-readable: %v", info.Mode().Perm())
	}
}
