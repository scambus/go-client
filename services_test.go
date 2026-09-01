package scambus

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUploadMediaSendsMultipart(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, Media{ID: "m1", FileName: "shot.png", MimeType: "image/png"})
	})
	c := srv.client(t)

	media, err := c.Media.Upload(context.Background(), "shot.png", bytes.NewReader([]byte("pixels")), &MediaUpload{
		Notes:          "phishing page",
		JournalEntryID: "e1",
		Metadata:       map[string]any{"width": 800},
	})
	if err != nil {
		t.Fatal(err)
	}
	if media.ID != "m1" {
		t.Fatalf("got %+v", media)
	}

	req := srv.last()
	mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type %q (%v)", req.Header.Get("Content-Type"), err)
	}

	reader := multipart.NewReader(bytes.NewReader(req.Body), params["boundary"])
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(part)
		fields[part.FormName()] = string(body)
	}

	if fields["file"] != "pixels" {
		t.Fatalf("file part %q", fields["file"])
	}
	if fields["notes"] != "phishing page" || fields["journalEntryId"] != "e1" {
		t.Fatalf("fields %+v", fields)
	}
	if fields["metadata"] != `{"width":800}` {
		t.Fatalf("metadata %q", fields["metadata"])
	}
}

func TestUploadFileReadsFromDisk(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, Media{ID: "m2"})
	})
	c := srv.client(t)

	path := filepath.Join(t.TempDir(), "evidence.txt")
	if err := os.WriteFile(path, []byte("contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	media, err := c.Media.UploadFile(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if media.ID != "m2" {
		t.Fatalf("got %+v", media)
	}
}

func TestUploadFileMissingPath(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {})
	c := srv.client(t)
	if _, err := c.Media.UploadFile(context.Background(), filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Fatal("want an error")
	}
}

func TestUpdateCaseRefetchesAfter204(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(t, w, 200, Case{ID: "c1", Title: "renamed", Status: "closed"})
	})
	c := srv.client(t)

	updated, err := c.Cases.Update(context.Background(), "c1", UpdateCaseInput{Status: Ptr("closed")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "closed" || updated.Title != "renamed" {
		t.Fatalf("got %+v", updated)
	}
	if len(srv.requests) != 2 {
		t.Fatalf("want PUT then GET, got %d requests", len(srv.requests))
	}
}

func TestUpdateCaseRejectsEmptyPatch(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {})
	c := srv.client(t)
	if _, err := c.Cases.Update(context.Background(), "c1", UpdateCaseInput{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
	if len(srv.requests) != 0 {
		t.Fatal("must not hit the API")
	}
}

func TestCreateCaseAppliesDefaults(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, Case{ID: "c1"})
	})
	c := srv.client(t)

	if _, err := c.Cases.Create(context.Background(), CreateCaseInput{Title: "New case"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["status"] != "open" || body["priority"] != "medium" {
		t.Fatalf("got %+v", body)
	}
}

func TestClaimQueueItemReturnsNilWhenEmpty(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 404, map[string]string{"error": "no claimable items"})
	})
	c := srv.client(t)

	item, err := c.Queues.Claim(context.Background(), "q1")
	if err != nil {
		t.Fatalf("an empty queue is not an error: %v", err)
	}
	if item != nil {
		t.Fatalf("got %+v", item)
	}
}

func TestClaimQueueItemPropagatesOtherErrors(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 401, map[string]string{"error": "nope"})
	})
	c := srv.client(t)

	if _, err := c.Queues.Claim(context.Background(), "q1"); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("got %v", err)
	}
}

func TestQueueItemActionsHitTheRightPaths(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	c := srv.client(t)
	ctx := context.Background()

	if err := c.Queues.Complete(ctx, "q1", "i1", ItemActionInput{Outcome: "engaged"}); err != nil {
		t.Fatal(err)
	}
	if got := srv.last().Path; got != "/api/queues/q1/items/i1/complete" {
		t.Fatalf("path %q", got)
	}

	if err := c.Queues.Drop(ctx, "q1", "i1", ItemActionInput{Reason: "duplicate"}); err != nil {
		t.Fatal(err)
	}
	if got := srv.last().Path; got != "/api/queues/q1/items/i1/drop" {
		t.Fatalf("path %q", got)
	}

	if err := c.Queues.Move(ctx, "q1", "i1", MoveItemInput{TargetQueueID: "q2"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["target_queue_id"] != "q2" {
		t.Fatalf("body %+v", body)
	}
}

func TestMoveQueueItemRequiresTarget(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {})
	c := srv.client(t)
	if err := c.Queues.Move(context.Background(), "q1", "i1", MoveItemInput{}); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestReadQueueStreamDefaultsToStartCursor(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, QueueStreamResponse{StreamKey: "k", Cursor: "1-0"})
	})
	c := srv.client(t)

	if _, err := c.Queues.ReadStream(context.Background(), "q1", nil); err != nil {
		t.Fatal(err)
	}
	if srv.last().Query != "cursor=0" {
		t.Fatalf("query %q", srv.last().Query)
	}
}

func TestCreateStreamDefaultsDataType(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, ExportStream{ID: "s1", DataType: "journal_entry"})
	})
	c := srv.client(t)

	if _, err := c.Streams.Create(context.Background(), CreateStreamInput{Name: "feed"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["data_type"] != "journal_entry" {
		t.Fatalf("got %v", body["data_type"])
	}
}

func TestCreateTemporaryStreamDefaultsToIdentifiers(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, ExportStream{ID: "s2", DataType: "identifier"})
	})
	c := srv.client(t)

	if _, err := c.Streams.CreateTemporary(context.Background(), CreateTemporaryStreamInput{}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["data_type"] != "identifier" {
		t.Fatalf("got %v", body["data_type"])
	}
}

func TestPollDecodesTypedMessages(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"messages": []any{
				map[string]any{"identifier_id": "i1", "type": "phone", "display_value": "+1", "confidence": 0.9, "cursor": "1-0"},
			},
			"next_cursor": "1-1",
			"has_more":    true,
		})
	})
	c := srv.client(t)

	result, err := c.Consume.Poll(context.Background(), "ck", &PollOptions{Cursor: CursorStart, Limit: 100, IncludeTest: Ptr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if result.NextCursor != "1-1" || !result.HasMore {
		t.Fatalf("got %+v", result)
	}

	messages, err := result.IdentifierMessages()
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Confidence.Score != 0.9 || messages[0].Cursor != "1-0" {
		t.Fatalf("got %+v", messages)
	}
	if srv.last().Path != "/api/consume/ck/poll" {
		t.Fatalf("path %q", srv.last().Path)
	}
	if srv.last().Query != "cursor=0&include_test=false&limit=100&order=asc" {
		t.Fatalf("query %q", srv.last().Query)
	}
}

func TestPollTreatsNoContentAsEmpty(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	c := srv.client(t)

	result, err := c.Consume.Poll(context.Background(), "ck", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 0 || result.HasMore {
		t.Fatalf("got %+v", result)
	}
}

func TestPollSurfacesCursorExpiry(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 410, map[string]string{"error": "cursor outside retention window"})
	})
	c := srv.client(t)

	if _, err := c.Consume.Poll(context.Background(), "ck", nil); !errors.Is(err, ErrCursorExpired) {
		t.Fatalf("got %v", err)
	}
}

func TestExecuteViewDecodesByEntityType(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"data":        []any{map[string]any{"id": "e1", "type": "note", "description": "x"}},
			"entity_type": "journal",
			"count":       1,
		})
	})
	c := srv.client(t)

	result, err := c.Views.Execute(context.Background(), "v1", &ExecuteViewOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := result.JournalEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].ID != "e1" {
		t.Fatalf("got %+v", entries)
	}
}

func TestGenerateViewReportPicksReportKind(t *testing.T) {
	var reportPath string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(t, w, 200, View{ID: "v1", EntityType: "identifier"})
			return
		}
		reportPath = r.URL.Path
		writeJSON(t, w, 202, map[string]any{"report_id": "r1", "status": "pending"})
	})
	c := srv.client(t)

	report, err := c.Reports.GenerateViewReport(context.Background(), "v1", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.ID != "r1" {
		t.Fatalf("got %+v", report)
	}
	if reportPath != "/api/reports/identifiers" {
		t.Fatalf("path %q", reportPath)
	}
}

func TestGenerateViewReportRejectsUnsupportedEntity(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, View{ID: "v1", EntityType: "case"})
	})
	c := srv.client(t)

	if _, err := c.Reports.GenerateViewReport(context.Background(), "v1", false, false); !errors.Is(err, ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestWaitForReportPollsUntilDone(t *testing.T) {
	calls := 0
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := "processing"
		if calls >= 3 {
			status = "completed"
		}
		writeJSON(t, w, 200, map[string]any{"report_id": "r1", "status": status})
	})
	c := srv.client(t)

	report, err := c.Reports.Wait(context.Background(), "r1", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !report.IsCompleted() {
		t.Fatalf("got %+v", report)
	}
	if calls != 3 {
		t.Fatalf("want 3 polls, got %d", calls)
	}
}

func TestDownloadReportToFile(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("%PDF-1.4"))
	})
	c := srv.client(t)

	path := filepath.Join(t.TempDir(), "nested", "report.pdf")
	if err := c.Reports.DownloadToFile(context.Background(), "r1", path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "%PDF-1.4" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchIdentifiersDefaultsLimit(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{"data": []any{}, "hasMore": false})
	})
	c := srv.client(t)

	if _, err := c.Search.Identifiers(context.Background(), SearchIdentifiersInput{
		Filter: &FilterCriteria{SearchQuery: "scam"},
	}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["limit"] != float64(100) {
		t.Fatalf("limit %v", body["limit"])
	}
	if body["search_query"] != "scam" {
		t.Fatalf("search_query %v", body["search_query"])
	}
}

func TestNotificationUnreadCount(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]int{"count": 7})
	})
	c := srv.client(t)

	count, err := c.Notifications.UnreadCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 7 {
		t.Fatalf("got %d", count)
	}
}

func TestTagCreateDefaultsType(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, Tag{ID: "t1", Title: "Threat"})
	})
	c := srv.client(t)

	if _, err := c.Tags.Create(context.Background(), CreateTagInput{Title: "Threat", FlowUp: true}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["tag_type"] != "valued" {
		t.Fatalf("got %v", body["tag_type"])
	}
}

func TestAdminDomainRuleQuery(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, []SpecialDomainRule{{ID: "r1", Domain: "example.test"}})
	})
	c := srv.client(t)

	if _, err := c.Admin.ListSpecialDomainRules(context.Background(), &ListDomainRulesOptions{
		Category: "social", Active: Ptr(true),
	}); err != nil {
		t.Fatal(err)
	}
	if srv.last().Query != "active=true&category=social" {
		t.Fatalf("query %q", srv.last().Query)
	}
}

func TestPersonaAddMediaDefaultsCategory(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 201, PersonaMediaLink{PersonaID: "p1", MediaID: "m1", Category: "other"})
	})
	c := srv.client(t)

	if _, err := c.Personas.AddMedia(context.Background(), "p1", PersonaMediaInput{MediaID: "m1"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	srv.last().decode(t, &body)
	if body["category"] != "other" {
		t.Fatalf("got %v", body["category"])
	}
}
