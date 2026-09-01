package scambus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Time accepts RFC3339 with or without a zone; a naive value is read as UTC.
type Time struct{ time.Time }

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func NewTime(t time.Time) Time { return Time{t} }

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(t.UTC().Format(time.RFC3339Nano))), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("scambus: timestamp is not a string: %w", err)
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	for _, layout := range timeLayouts {
		parsed, err := time.Parse(layout, s)
		if err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("scambus: unrecognised timestamp %q", s)
}

func (t Time) IsSet() bool { return !t.IsZero() }

// Confidence decodes both the object form the entity endpoints return
// ({"score": 0.95}) and the bare number the stream endpoints return.
type Confidence struct {
	Score float64
	Set   bool
}

func NewConfidence(score float64) Confidence { return Confidence{Score: score, Set: true} }

func (c Confidence) MarshalJSON() ([]byte, error) {
	if !c.Set {
		return []byte("null"), nil
	}
	return json.Marshal(c.Score)
}

func (c *Confidence) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*c = Confidence{}
		return nil
	}
	if data[0] == '{' {
		var wrapper struct {
			Score *float64 `json:"score"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return fmt.Errorf("scambus: decode confidence object: %w", err)
		}
		if wrapper.Score == nil {
			*c = Confidence{}
			return nil
		}
		*c = Confidence{Score: *wrapper.Score, Set: true}
		return nil
	}
	var score float64
	if err := json.Unmarshal(data, &score); err != nil {
		return fmt.Errorf("scambus: decode confidence: %w", err)
	}
	*c = Confidence{Score: score, Set: true}
	return nil
}
