package scambus

import (
	"context"
	"net/url"
	"strconv"
)

type StreamService struct{ client *Client }

type ListStreamsOptions struct {
	ActiveOnly bool
	Page       int
	PageSize   int
}

type ListStreamsResult struct {
	Data       []ExportStream `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

func (s *StreamService) List(ctx context.Context, opts *ListStreamsOptions) (*ListStreamsResult, error) {
	q := url.Values{}
	if opts != nil {
		// The handler filters only on active=true; there is no way to ask
		// for inactive streams, so a false value is left off.
		if opts.ActiveOnly {
			q.Set("active", "true")
		}
		if opts.Page > 0 {
			q.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PageSize > 0 {
			q.Set("pageSize", strconv.Itoa(opts.PageSize))
		}
	}
	var out ListStreamsResult
	if err := s.client.get(ctx, "/export-streams", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *StreamService) Get(ctx context.Context, streamID string) (*ExportStream, error) {
	var out ExportStream
	if err := s.client.get(ctx, "/export-streams/"+streamID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateStreamInput struct {
	Name                  string          `json:"name"`
	Description           string          `json:"description,omitempty"`
	DataType              StreamDataType  `json:"data_type"`
	FilterCriteria        *FilterCriteria `json:"filter_criteria,omitempty"`
	FilterExpression      string          `json:"filter_expression,omitempty"`
	IsActive              *bool           `json:"is_active,omitempty"`
	RetentionDays         *int            `json:"retention_days,omitempty"`
	BackfillHistorical    bool            `json:"backfill_historical"`
	BackfillFromDate      string          `json:"backfill_from_date,omitempty"`
	IncludeOriginator     bool            `json:"include_originator,omitempty"`
	IncludeJournalEntries bool            `json:"include_journal_entries,omitempty"`
	BatchSize             *int            `json:"batch_size,omitempty"`
	RateLimitPerMinute    *int            `json:"rate_limit_per_minute,omitempty"`
	SharedOrgIDs          []string        `json:"shared_org_ids,omitempty"`
}

func (s *StreamService) Create(ctx context.Context, in CreateStreamInput) (*ExportStream, error) {
	if in.DataType == "" {
		in.DataType = StreamDataJournalEntry
	}
	var out ExportStream
	if err := s.client.post(ctx, "/export-streams", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateTemporaryStreamInput struct {
	Name                  string          `json:"name,omitempty"`
	DataType              StreamDataType  `json:"data_type"`
	ViewID                string          `json:"viewId,omitempty"`
	FilterCriteria        *FilterCriteria `json:"filter_criteria,omitempty"`
	FilterExpression      string          `json:"filter_expression,omitempty"`
	IncludeOriginator     bool            `json:"include_originator,omitempty"`
	IncludeJournalEntries bool            `json:"include_journal_entries,omitempty"`
	BatchSize             *int            `json:"batch_size,omitempty"`
}

func (s *StreamService) CreateTemporary(ctx context.Context, in CreateTemporaryStreamInput) (*ExportStream, error) {
	if in.DataType == "" {
		in.DataType = StreamDataIdentifier
	}
	var out ExportStream
	if err := s.client.post(ctx, "/export-streams/temporary", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *StreamService) Delete(ctx context.Context, streamID string) error {
	return s.client.delete(ctx, "/export-streams/"+streamID)
}

// RecoverResult reports a 202; recovery then runs asynchronously.
type RecoverResult struct {
	Status           string `json:"status"`
	Message          string `json:"message,omitempty"`
	StreamID         string `json:"stream_id"`
	StreamName       string `json:"stream_name,omitempty"`
	IgnoreCheckpoint bool   `json:"ignore_checkpoint"`
	ClearStream      bool   `json:"clear_stream"`
	TriggeredBy      string `json:"triggered_by,omitempty"`
}

func (s *StreamService) Recover(ctx context.Context, streamID string, ignoreCheckpoint, clearStream bool) (*RecoverResult, error) {
	q := url.Values{}
	if ignoreCheckpoint {
		q.Set("ignore_checkpoint", "true")
	}
	if !clearStream {
		q.Set("clear_stream", "false")
	}
	var out RecoverResult
	if err := s.client.call(ctx, request{method: "POST", endpoint: "/export-streams/" + streamID + "/recover", query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *StreamService) RecoveryInfo(ctx context.Context, streamID string) (map[string]any, error) {
	var out map[string]any
	if err := s.client.get(ctx, "/export-streams/"+streamID+"/recovery-info", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type RecoveryHistoryOptions struct {
	Limit    int
	Offset   int
	StreamID string
}

func (s *StreamService) RecoveryHistory(ctx context.Context, opts *RecoveryHistoryOptions) (map[string]any, error) {
	q := url.Values{}
	if opts != nil {
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		if opts.StreamID != "" {
			q.Set("stream_id", opts.StreamID)
		}
	}
	var out map[string]any
	if err := s.client.get(ctx, "/redis/recovery/history", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *StreamService) BackfillIdentifiers(ctx context.Context, streamID, fromDate string) (map[string]any, error) {
	q := url.Values{}
	if fromDate != "" {
		q.Set("fromDate", fromDate)
	}
	var out map[string]any
	if err := s.client.call(ctx, request{method: "POST", endpoint: "/export-streams/" + streamID + "/backfill-identifiers", query: q}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
