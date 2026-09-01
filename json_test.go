package scambus

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeUnmarshalAcceptedLayouts(t *testing.T) {
	cases := map[string]time.Time{
		`"2025-01-15T10:30:00Z"`:        time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		`"2025-01-15T10:30:00.123456Z"`: time.Date(2025, 1, 15, 10, 30, 0, 123456000, time.UTC),
		`"2025-01-15T10:30:00+02:00"`:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.FixedZone("", 7200)),
		`"2025-01-15T10:30:00"`:         time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		`"2025-01-15"`:                  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
	}
	for input, want := range cases {
		var got Time
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if !got.Equal(want) {
			t.Fatalf("%s: got %v, want %v", input, got.Time, want)
		}
	}
}

func TestTimeUnmarshalEmptyValues(t *testing.T) {
	for _, input := range []string{`null`, `""`} {
		var got Time
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if got.IsSet() {
			t.Fatalf("%s should be unset", input)
		}
	}
}

func TestTimeUnmarshalRejectsGarbage(t *testing.T) {
	var got Time
	if err := json.Unmarshal([]byte(`"not a date"`), &got); err == nil {
		t.Fatal("want an error")
	}
}

func TestTimeMarshalRoundTrip(t *testing.T) {
	original := NewTime(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var back Time
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Equal(original.Time) {
		t.Fatalf("got %v", back.Time)
	}
}

func TestTimeMarshalZeroIsNull(t *testing.T) {
	raw, err := json.Marshal(Time{})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "null" {
		t.Fatalf("got %s", raw)
	}
}

func TestConfidenceAcceptsObjectAndNumber(t *testing.T) {
	cases := map[string]float64{
		`{"score":0.95}`: 0.95,
		`0.42`:           0.42,
		`1`:              1,
	}
	for input, want := range cases {
		var got Confidence
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if !got.Set || got.Score != want {
			t.Fatalf("%s: got %+v", input, got)
		}
	}
}

func TestConfidenceUnsetForNullAndEmptyObject(t *testing.T) {
	for _, input := range []string{`null`, `{}`} {
		var got Confidence
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if got.Set {
			t.Fatalf("%s should leave Confidence unset", input)
		}
	}
}

func TestIdentifierDecodesWrappedConfidence(t *testing.T) {
	raw := `{"id":"i1","type":"phone","display_value":"+12125551234","confidence":{"score":0.9},"created_at":"2025-01-15T10:30:00Z"}`
	var got Identifier
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.Confidence.Score != 0.9 {
		t.Fatalf("got %+v", got.Confidence)
	}
	if !got.CreatedAt.IsSet() {
		t.Fatal("created_at should be set")
	}
}

func TestStreamIdentifierDecodesBareConfidence(t *testing.T) {
	raw := `{"id":"i1","type":"email","display_value":"a@b.test","confidence":0.75}`
	var got StreamIdentifierInfo
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.Confidence.Score != 0.75 {
		t.Fatalf("got %+v", got.Confidence)
	}
}

func TestReportDecodesHandlerShape(t *testing.T) {
	var report Report
	raw := `{"report_id":"r1","status":"completed","download_url":"https://x/y.pdf","status_url":"https://x/s","generated_at":"2025-01-15T10:30:00Z"}`
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatal(err)
	}
	if report.ID != "r1" || report.StatusURL == "" || !report.GeneratedAt.IsSet() {
		t.Fatalf("got %+v", report)
	}
	if !report.IsCompleted() || report.IsProcessing() {
		t.Fatalf("status helpers wrong for %+v", report)
	}
}

func TestDecodeDetails(t *testing.T) {
	details, err := DecodeDetails[PhoneCallDetails](map[string]any{
		"direction":     "inbound",
		"recording_url": "https://example.test/r.mp3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if details.Direction != "inbound" || details.RecordingURL != "https://example.test/r.mp3" {
		t.Fatalf("got %+v", details)
	}
}

func TestParseIdentifierDetailsByType(t *testing.T) {
	got, err := ParseIdentifierDetails("phone", map[string]any{"country_code": "+1", "number": "2125551234", "is_toll_free": false})
	if err != nil {
		t.Fatal(err)
	}
	phone, ok := got.(PhoneDetails)
	if !ok {
		t.Fatalf("got %T", got)
	}
	if phone.CountryCode != "+1" || phone.Number != "2125551234" {
		t.Fatalf("got %+v", phone)
	}

	unknown, err := ParseIdentifierDetails("mystery", map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := unknown.(map[string]any); !ok {
		t.Fatalf("unknown type should pass the map through, got %T", unknown)
	}
}

// Field names here were read from internal/models/journal_entry_details.go,
// not from the client, so a drift in either direction fails.
func TestDetailStructsMatchBackendFieldNames(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  []string
	}{
		{"phone_call", PhoneCallDetails{Direction: "inbound", StartTime: NewTime(time.Now()), Platform: "pstn"},
			[]string{"direction", "start_time", "platform"}},
		{"email", EmailDetails{Direction: "inbound", Subject: "s", MessageID: "m", SourceIP: "1.2.3.4", SPFResult: "pass"},
			[]string{"direction", "subject", "message_id", "source_ip", "spf_result"}},
		{"text_conversation", TextConversationDetails{Platform: "whatsapp", ParticipantCount: 2},
			[]string{"platform", "participant_count"}},
		{"activity_complete", ActivityCompleteDetails{CompletionReason: "manual", DurationSeconds: 5},
			[]string{"completion_reason", "start_time", "end_time", "duration_seconds"}},
		{"case_handoff", CaseHandoffDetails{FromUserID: "a", ToUserID: "b", Reason: "r"},
			[]string{"from_user_id", "to_user_id", "reason", "reassigned_tasks"}},
		{"scam_classification", ScamClassificationDetails{Summary: "s", ModelUsed: "m"},
			[]string{"composite_score", "prior_composite_score", "is_first_analysis", "model_used"}},
		{"funnel_action", FunnelActionDetails{FunnelID: "f", ActionType: "a", ClusterID: "c"},
			[]string{"funnel_id", "action_type", "cluster_id"}},
		{"task_update", TaskUpdateDetails{TaskID: "t", NewStatus: "done"},
			[]string{"task_id", "task_title", "old_status", "new_status"}},
	}

	for _, tc := range cases {
		raw, err := json.Marshal(tc.value)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		for _, field := range tc.want {
			if _, present := got[field]; !present {
				t.Errorf("%s is missing %q; got %s", tc.name, field, raw)
			}
		}
	}
}
