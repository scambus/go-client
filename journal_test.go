package scambus

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func journalEntryResponse(entry map[string]any) map[string]any {
	return map[string]any{
		"journal_entry": map[string]any{"journal_entry": entry, "can_edit": true},
		"cases":         []any{},
	}
}

func createEntryServer(t *testing.T, entry map[string]any, created map[string]any) *recordingServer {
	t.Helper()
	return newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(t, w, 201, created)
			return
		}
		writeJSON(t, w, 200, journalEntryResponse(entry))
	})
}

func TestCreateEntryUnwrapsNestedGetResponse(t *testing.T) {
	srv := createEntryServer(t,
		map[string]any{"id": "e1", "type": "note", "description": "hello"},
		map[string]any{"id": "e1"},
	)
	c := srv.client(t)

	entry, err := c.Journal.Create(context.Background(), CreateEntryInput{
		Type:        EntryTypeNote,
		Description: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID != "e1" || entry.Description != "hello" {
		t.Fatalf("got %+v", entry)
	}
	if srv.requests[0].Path != "/api/journal-entries" {
		t.Fatalf("post path %q", srv.requests[0].Path)
	}
	if srv.requests[1].Path != "/api/journal-entries/e1" {
		t.Fatalf("get path %q", srv.requests[1].Path)
	}
}

func TestCreateEntryCarriesFailedAndExtractedIdentifiers(t *testing.T) {
	srv := createEntryServer(t,
		map[string]any{"id": "e1", "type": "detection", "description": "d"},
		map[string]any{
			"id":                 "e1",
			"failed_identifiers": []any{map[string]any{"type": "phone", "value": "bad", "reason": "invalid"}},
			"extracted_identifiers": []any{
				map[string]any{"ref": "r1", "identifier_id": "i1", "type": "email", "value": "a@b.test"},
			},
		},
	)
	c := srv.client(t)

	entry, err := c.Journal.Create(context.Background(), CreateEntryInput{Type: EntryTypeDetection, Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entry.FailedIdentifiers) != 1 || entry.FailedIdentifiers[0].Reason != "invalid" {
		t.Fatalf("failed identifiers: %+v", entry.FailedIdentifiers)
	}
	if len(entry.ExtractedIdentifiers) != 1 || entry.ExtractedIdentifiers[0].IdentifierID != "i1" {
		t.Fatalf("extracted identifiers: %+v", entry.ExtractedIdentifiers)
	}
}

func TestCreateEntryRejectsMissingType(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1"}, map[string]any{"id": "e1"})
	c := srv.client(t)
	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{Description: "x"}); err == nil {
		t.Fatal("want a validation error")
	}
	if len(srv.requests) != 0 {
		t.Fatal("must not hit the API")
	}
}

func TestCreateEntryDefaultsEndTimeToStartTime(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "note"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC))
	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{
		Type:        EntryTypeNote,
		Description: "x",
		StartTime:   start,
	}); err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	srv.requests[0].decode(t, &body)
	if body["end_time"] != body["start_time"] {
		t.Fatalf("end_time %v should equal start_time %v", body["end_time"], body["start_time"])
	}
}

func TestCreateEntryOmitsEndTimeWhenInProgress(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "phone_call"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC))
	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{
		Type:        EntryTypePhoneCall,
		Description: "x",
		StartTime:   start,
		InProgress:  true,
	}); err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	srv.requests[0].decode(t, &body)
	if _, present := body["end_time"]; present {
		t.Fatalf("end_time must be absent, got %v", body["end_time"])
	}
	if body["start_time"] == nil {
		t.Fatal("start_time must be present")
	}
}

func TestCreateEntryKeepsExplicitEndTime(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "note"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC))
	end := NewTime(time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC))
	if _, err := c.Journal.Create(context.Background(), CreateEntryInput{
		Type: EntryTypeNote, Description: "x", StartTime: start, EndTime: end,
	}); err != nil {
		t.Fatal(err)
	}

	var body struct {
		StartTime Time `json:"start_time"`
		EndTime   Time `json:"end_time"`
	}
	srv.requests[0].decode(t, &body)
	if !body.EndTime.Equal(end.Time) {
		t.Fatalf("got %v", body.EndTime.Time)
	}
}

func TestCreatePhoneCallBuildsDetailsAndEnablesExtraction(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "phone_call"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC))
	_, err := c.Journal.CreatePhoneCall(context.Background(), PhoneCallInput{
		Description: "inbound scam call",
		Direction:   "inbound",
		StartTime:   start,
		EndTime:     NewTime(start.Add(5 * time.Minute)),
		Transcript: []ConversationMessage{
			{Index: 0, MessageID: "m0", Timestamp: start, Body: "hello"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Type      string           `json:"type"`
		AIExtract bool             `json:"ai_extract"`
		Details   PhoneCallDetails `json:"details"`
	}
	srv.requests[0].decode(t, &body)
	if body.Type != "phone_call" {
		t.Fatalf("type %q", body.Type)
	}
	if !body.AIExtract {
		t.Fatal("a transcript should enable ai_extract")
	}
	if body.Details.Direction != "inbound" || len(body.Details.Transcript) != 1 {
		t.Fatalf("details %+v", body.Details)
	}
}

func TestCreateDetectionBuildsEvidenceFromMedia(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "detection"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	_, err := c.Journal.CreateDetection(context.Background(), DetectionInput{
		Description: "phishing site",
		Media:       []Media{{ID: "m1", MimeType: "image/png"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Evidence []Evidence `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if len(body.Evidence) != 1 {
		t.Fatalf("want one evidence record, got %d", len(body.Evidence))
	}
	if body.Evidence[0].Type != "screenshot" {
		t.Fatalf("evidence type %q", body.Evidence[0].Type)
	}
	if body.Evidence[0].MediaID != "m1" {
		t.Fatalf("media id %q", body.Evidence[0].MediaID)
	}
}

func TestEachMediaItemGetsItsOwnEvidenceRecord(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "detection"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	_, err := c.Journal.CreateDetection(context.Background(), DetectionInput{
		Description: "three shots",
		Media: []Media{
			{ID: "m1", MimeType: "image/png"},
			{ID: "m2", MimeType: "image/png"},
			{ID: "m3", MimeType: "image/png"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Evidence []Evidence `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if len(body.Evidence) != 3 {
		t.Fatalf("the API accepts one media id per record: got %d records", len(body.Evidence))
	}
	for i, want := range []string{"m1", "m2", "m3"} {
		if body.Evidence[i].MediaID != want {
			t.Fatalf("record %d has media id %q, want %q", i, body.Evidence[i].MediaID, want)
		}
	}
}

func TestAttachMediaDoesNotMutateCallerEvidence(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "detection"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	shared := []Evidence{{Type: "document", Description: "mine"}}
	in := DetectionInput{Description: "first", Evidence: shared, Media: []Media{{ID: "m1"}}}

	if _, err := c.Journal.CreateDetection(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Journal.CreateDetection(context.Background(), in); err != nil {
		t.Fatal(err)
	}

	if len(shared) != 1 {
		t.Fatalf("caller slice was written through: len %d", len(shared))
	}
	var second struct {
		Evidence []Evidence `json:"evidence"`
	}
	srv.requests[2].decode(t, &second)
	if len(second.Evidence) != 2 {
		t.Fatalf("reuse leaked media between entries: %d records on the second call", len(second.Evidence))
	}
}

func TestCreateDetectionAppendsMediaToSuppliedEvidence(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "detection"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	_, err := c.Journal.CreateDetection(context.Background(), DetectionInput{
		Description: "phishing site",
		Evidence:    []Evidence{{Type: "document", Description: "mine", MediaID: "m0"}},
		Media:       []Media{{ID: "m1", MimeType: "image/png"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Evidence []Evidence `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if len(body.Evidence) != 2 {
		t.Fatalf("want the supplied record plus one per media, got %d", len(body.Evidence))
	}
	if body.Evidence[0].Description != "mine" || body.Evidence[0].MediaID != "m0" {
		t.Fatalf("supplied evidence must survive, got %+v", body.Evidence[0])
	}
	if body.Evidence[1].MediaID != "m1" {
		t.Fatalf("media record %+v", body.Evidence[1])
	}
}

func TestCreatePhoneCallPicksRecordingEvidenceForAudio(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1", "type": "phone_call"}, map[string]any{"id": "e1"})
	c := srv.client(t)

	start := NewTime(time.Now().UTC())
	_, err := c.Journal.CreatePhoneCall(context.Background(), PhoneCallInput{
		Description: "call", Direction: "inbound", StartTime: start, EndTime: start,
		Media: []Media{{ID: "m1", MimeType: "audio/flac"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		Evidence []Evidence `json:"evidence"`
	}
	srv.requests[0].decode(t, &body)
	if len(body.Evidence) != 1 || body.Evidence[0].Type != "recording" {
		t.Fatalf("got %+v", body.Evidence)
	}
}

func TestCreateConversationContinuationDerivesTimeSpan(t *testing.T) {
	srv := createEntryServer(t,
		map[string]any{"id": "e2", "type": "conversation_continuation"},
		map[string]any{"id": "e2"},
	)
	c := srv.client(t)

	base := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	_, err := c.Journal.CreateConversationContinuation(context.Background(), ConversationContinuationInput{
		ParentJournalEntryID: "e1",
		Messages: []ConversationMessage{
			{Index: 1, MessageID: "b", Timestamp: NewTime(base.Add(time.Hour)), Body: "later"},
			{Index: 0, MessageID: "a", Timestamp: NewTime(base), Body: "earlier"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		StartTime            Time   `json:"start_time"`
		EndTime              Time   `json:"end_time"`
		ParentJournalEntryID string `json:"parent_journal_entry_id"`
		Description          string `json:"description"`
	}
	srv.requests[0].decode(t, &body)
	if !body.StartTime.Equal(base) {
		t.Fatalf("start %v", body.StartTime.Time)
	}
	if !body.EndTime.Equal(base.Add(time.Hour)) {
		t.Fatalf("end %v", body.EndTime.Time)
	}
	if body.ParentJournalEntryID != "e1" {
		t.Fatalf("parent %q", body.ParentJournalEntryID)
	}
	if body.Description != "Conversation continuation" {
		t.Fatalf("description %q", body.Description)
	}
}

func TestCreateConversationContinuationRejectsEmptyMessages(t *testing.T) {
	srv := createEntryServer(t, map[string]any{"id": "e1"}, map[string]any{"id": "e1"})
	c := srv.client(t)
	_, err := c.Journal.CreateConversationContinuation(context.Background(), ConversationContinuationInput{ParentJournalEntryID: "e1"})
	if err == nil {
		t.Fatal("want an error")
	}
	if len(srv.requests) != 0 {
		t.Fatal("must not hit the API")
	}
}

func TestListEntriesUnwrapsDataItems(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"data": []any{
				map[string]any{"journal_entry": map[string]any{"id": "e1", "type": "note"}, "can_edit": true},
				map[string]any{"journal_entry": map[string]any{"id": "e2", "type": "detection"}, "can_edit": false},
			},
		})
	})
	c := srv.client(t)

	entries, err := c.Journal.List(context.Background(), &ListEntriesOptions{Type: EntryTypeNote, Page: 2, PageSize: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].ID != "e2" {
		t.Fatalf("got %+v", entries)
	}
	if q := srv.last().Query; q != "page=2&pageSize=5&type=note" {
		t.Fatalf("query %q", q)
	}
}

func TestQueryMergesFilterAndStructuralFields(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"data":    []any{map[string]any{"id": "e1", "type": "note", "description": "x"}},
			"hasMore": false, "count": 1,
		})
	})
	c := srv.client(t)

	result, err := c.Journal.Query(context.Background(), QueryEntriesInput{
		Filter:             &FilterCriteria{SearchQuery: "phishing", Types: []string{"detection"}, MinConfidence: Ptr(0.8)},
		IncludeIdentifiers: true,
		OrderDesc:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 1 || len(result.Data) != 1 {
		t.Fatalf("got %+v", result)
	}

	var body map[string]any
	srv.last().decode(t, &body)
	if body["search_query"] != "phishing" {
		t.Fatalf("search_query %v", body["search_query"])
	}
	if body["order_by"] != "performed_at" {
		t.Fatalf("order_by default is wrong: %v", body["order_by"])
	}
	if body["include_identifiers"] != true {
		t.Fatalf("include_identifiers %v", body["include_identifiers"])
	}
	if body["min_confidence"] != 0.8 {
		t.Fatalf("min_confidence %v", body["min_confidence"])
	}
}

func TestQueryAllFollowsCursors(t *testing.T) {
	page := 0
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			writeJSON(t, w, 200, map[string]any{
				"data":       []any{map[string]any{"id": "e1", "type": "note"}},
				"nextCursor": "c2",
				"hasMore":    true,
			})
			return
		}
		writeJSON(t, w, 200, map[string]any{
			"data":    []any{map[string]any{"id": "e2", "type": "note"}},
			"hasMore": false,
		})
	})
	c := srv.client(t)

	var ids []string
	err := c.Journal.QueryAll(context.Background(), QueryEntriesInput{}, func(e JournalEntry) error {
		ids = append(ids, e.ID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "e1" || ids[1] != "e2" {
		t.Fatalf("got %v", ids)
	}

	var second map[string]any
	srv.requests[1].decode(t, &second)
	if second["cursor"] != "c2" {
		t.Fatalf("cursor %v", second["cursor"])
	}
}

func TestCompleteActivityComputesDuration(t *testing.T) {
	start := time.Date(2025, 5, 1, 8, 0, 0, 0, time.UTC)
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writeJSON(t, w, 201, map[string]any{"id": "e2"})
			return
		}
		writeJSON(t, w, 200, journalEntryResponse(map[string]any{
			"id": "e1", "type": "phone_call", "start_time": start.Format(time.RFC3339),
		}))
	})
	c := srv.client(t)

	_, err := c.Journal.CompleteActivity(context.Background(), "e1", NewTime(start.Add(90*time.Second)), "", "")
	if err != nil {
		t.Fatal(err)
	}

	var post capturedRequest
	for _, req := range srv.requests {
		if req.Method == http.MethodPost {
			post = req
			break
		}
	}
	var body struct {
		Type    string                  `json:"type"`
		Details ActivityCompleteDetails `json:"details"`
	}
	if err := json.Unmarshal(post.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "activity_complete" {
		t.Fatalf("type %q", body.Type)
	}
	if body.Details.DurationSeconds != 90 {
		t.Fatalf("duration %d", body.Details.DurationSeconds)
	}
	if body.Details.CompletionReason != "manual" {
		t.Fatalf("reason %q", body.Details.CompletionReason)
	}
}

func TestCompleteActivityRefusesEntryWithoutStartTime(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, journalEntryResponse(map[string]any{"id": "e1", "type": "note"}))
	})
	c := srv.client(t)
	if _, err := c.Journal.CompleteActivity(context.Background(), "e1", Time{}, "", ""); err == nil {
		t.Fatal("want an error")
	}
}

func TestBatchCreateWrapsEntries(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, map[string]any{
			"results": []any{map[string]any{"index": 0, "status": "created", "id": "e1"}},
			"summary": map[string]any{"total": 1, "succeeded": 1, "failed": 0},
		})
	})
	c := srv.client(t)

	result, err := c.Journal.CreateBatch(context.Background(), []CreateEntryInput{
		{Type: EntryTypeNote, Description: "one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.Succeeded != 1 || result.Results[0].ID != "e1" {
		t.Fatalf("got %+v", result)
	}

	var body struct {
		Entries []map[string]any `json:"entries"`
	}
	srv.last().decode(t, &body)
	if len(body.Entries) != 1 || body.Entries[0]["type"] != "note" {
		t.Fatalf("got %+v", body.Entries)
	}
}

func TestIdentifierSummaryPassesTypeFilter(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, IdentifierSummary{JournalEntryID: "e1", Total: 3})
	})
	c := srv.client(t)

	summary, err := c.Journal.IdentifierSummary(context.Background(), "e1", IdentifierTypePhone)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 3 {
		t.Fatalf("got %+v", summary)
	}
	if srv.last().Query != "type=phone" {
		t.Fatalf("query %q", srv.last().Query)
	}
}

// The eight typed creators each copy the same common fields into
// CreateEntryInput. A field added to one and forgotten in another stops
// being sent for exactly that entry type, which is close to invisible.
func TestEveryCreatorCarriesTheSameCommonFields(t *testing.T) {
	common := []string{
		"Description", "Identifiers", "OurIdentifiers", "Evidence", "Media",
		"CaseID", "Tags", "Metadata", "ParentJournalEntryID", "Originator",
		"IsTest", "ExternalIdentifiers", "ExtractExternalIdentifiers",
	}
	inputs := []any{
		DetectionInput{}, PhoneCallInput{}, EmailInput{}, TextConversationInput{},
		ConversationContinuationInput{}, NoteInput{}, ImportInput{}, ExportInput{},
	}

	for _, in := range inputs {
		typ := reflect.TypeOf(in)
		for _, field := range common {
			if _, ok := typ.FieldByName(field); !ok {
				t.Errorf("%s is missing the common field %s", typ.Name(), field)
			}
		}
	}
}

// applyCommon is copied per input type; this catches a copy that forgets to
// forward one of the fields.
func TestApplyCommonForwardsEveryField(t *testing.T) {
	populated := DetectionInput{
		Description:                "d",
		Identifiers:                []IdentifierLookup{{Type: "phone", Value: "+1"}},
		OurIdentifiers:             []IdentifierLookup{{Type: "phone", Value: "+2"}},
		Evidence:                   []Evidence{{Type: "document"}},
		CaseID:                     "c1",
		Tags:                       []TagLookup{{TagName: "t"}},
		Metadata:                   map[string]any{"k": "v"},
		ParentJournalEntryID:       "e0",
		Originator:                 &OriginatorLookup{Type: "user", Identifier: "u1"},
		IsTest:                     true,
		ExternalIdentifiers:        []ExternalIdentifierInput{{ExternalSystem: "s", ExternalID: "x"}},
		ExtractExternalIdentifiers: true,
	}

	var entry CreateEntryInput
	populated.applyCommon(&entry)

	checks := map[string]bool{
		"Description":                entry.Description == "d",
		"Identifiers":                len(entry.IdentifierLookups) == 1,
		"OurIdentifiers":             len(entry.OurIdentifierLookups) == 1,
		"Evidence":                   len(entry.Evidence) == 1,
		"CaseID":                     entry.CaseID == "c1",
		"Tags":                       len(entry.TagLookups) == 1,
		"Metadata":                   entry.Metadata["k"] == "v",
		"ParentJournalEntryID":       entry.ParentJournalEntryID == "e0",
		"Originator":                 entry.Originator != nil,
		"IsTest":                     entry.IsTest,
		"ExternalIdentifiers":        len(entry.ExternalIdentifiers) == 1,
		"ExtractExternalIdentifiers": entry.ExtractExternalIdentifiers,
	}
	for field, ok := range checks {
		if !ok {
			t.Errorf("applyCommon did not forward %s", field)
		}
	}
}
