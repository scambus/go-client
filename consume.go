package scambus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// ConsumeService reads export streams. Every method takes the stream's
// consumer key, not its stream id.
type ConsumeService struct{ client *Client }

type PollOptions struct {
	Cursor      string
	Order       string
	Limit       int
	IncludeTest *bool
}

type PollResult struct {
	Messages   []json.RawMessage `json:"messages"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

func (r PollResult) IdentifierMessages() ([]IdentifierStreamMessage, error) {
	return decodeMessages[IdentifierStreamMessage](r.Messages)
}

func (r PollResult) JournalEntryMessages() ([]JournalEntryStreamMessage, error) {
	return decodeMessages[JournalEntryStreamMessage](r.Messages)
}

func decodeMessages[T any](raw []json.RawMessage) ([]T, error) {
	out := make([]T, 0, len(raw))
	for _, item := range raw {
		var msg T
		if err := json.Unmarshal(item, &msg); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, nil
}

func (s *ConsumeService) Poll(ctx context.Context, consumerKey string, opts *PollOptions) (*PollResult, error) {
	q := url.Values{"order": {SortAsc}}
	if opts != nil {
		if opts.Cursor != "" {
			q.Set("cursor", opts.Cursor)
		}
		if opts.Order != "" {
			q.Set("order", opts.Order)
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.IncludeTest != nil {
			q.Set("include_test", strconv.FormatBool(*opts.IncludeTest))
		}
	}

	resp, err := s.client.do(ctx, request{
		method:   http.MethodGet,
		endpoint: "/consume/" + consumerKey + "/poll",
		query:    q,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return &PollResult{}, nil
	}

	var out PollResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

type StreamInfo struct {
	Stream        string `json:"stream"`
	StreamID      string `json:"stream_id,omitempty"`
	Name          string `json:"name,omitempty"`
	DataType      string `json:"data_type,omitempty"`
	MessageCount  int64  `json:"message_count"`
	FirstID       string `json:"first_id,omitempty"`
	LastID        string `json:"last_id,omitempty"`
	RetentionDays int    `json:"retention_days,omitempty"`
	IsActive      bool   `json:"is_active"`
}

func (s *ConsumeService) Info(ctx context.Context, consumerKey string) (*StreamInfo, error) {
	var out StreamInfo
	if err := s.client.get(ctx, "/consume/"+consumerKey+"/info", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
