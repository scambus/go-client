package scambus

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type JournalService struct{ client *Client }

type CreateEntryInput struct {
	Type                       EntryType
	Description                string
	Details                    any
	PerformedAt                Time
	CaseID                     string
	IdentifierLookups          []IdentifierLookup
	OurIdentifierLookups       []IdentifierLookup
	Evidence                   []Evidence
	Originator                 *OriginatorLookup
	ParentJournalEntryID       string
	TagLookups                 []TagLookup
	StartTime                  Time
	EndTime                    Time
	InProgress                 bool
	Metadata                   map[string]any
	IsTest                     bool
	IsNSFW                     bool
	AIExtract                  bool
	RetractedIdentifierIDs     []string
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
}

type createEntryBody struct {
	Type                       string                    `json:"type"`
	Description                string                    `json:"description"`
	Details                    any                       `json:"details,omitempty"`
	PerformedAt                Time                      `json:"performed_at,omitzero"`
	CaseID                     string                    `json:"case_id,omitempty"`
	IdentifierLookups          []IdentifierLookup        `json:"identifier_lookups,omitempty"`
	OurIdentifierLookups       []IdentifierLookup        `json:"our_identifier_lookups,omitempty"`
	Evidence                   []Evidence                `json:"evidence,omitempty"`
	OriginatorLookup           *OriginatorLookup         `json:"originator_lookup,omitempty"`
	ParentJournalEntryID       string                    `json:"parent_journal_entry_id,omitempty"`
	TagLookups                 []TagLookup               `json:"tag_lookups,omitempty"`
	StartTime                  Time                      `json:"start_time,omitzero"`
	EndTime                    Time                      `json:"end_time,omitzero"`
	Metadata                   map[string]any            `json:"metadata,omitempty"`
	IsTest                     bool                      `json:"is_test,omitempty"`
	IsNSFW                     bool                      `json:"is_nsfw,omitempty"`
	AIExtract                  bool                      `json:"ai_extract,omitempty"`
	RetractedIdentifierIDs     []string                  `json:"retracted_identifier_ids,omitempty"`
	ExternalIdentifiers        []ExternalIdentifierInput `json:"external_identifiers,omitempty"`
	ExtractExternalIdentifiers bool                      `json:"extract_external_identifiers,omitempty"`
}

type createEntryResponse struct {
	ID                   string                `json:"id"`
	FailedIdentifiers    []FailedIdentifier    `json:"failed_identifiers"`
	ExtractedIdentifiers []ExtractedIdentifier `json:"extracted_identifiers"`
}

func (in CreateEntryInput) body() createEntryBody {
	b := createEntryBody{
		Type:                       string(in.Type),
		Description:                in.Description,
		Details:                    in.Details,
		PerformedAt:                in.PerformedAt,
		CaseID:                     in.CaseID,
		IdentifierLookups:          in.IdentifierLookups,
		OurIdentifierLookups:       in.OurIdentifierLookups,
		Evidence:                   in.Evidence,
		OriginatorLookup:           in.Originator,
		ParentJournalEntryID:       in.ParentJournalEntryID,
		TagLookups:                 in.TagLookups,
		StartTime:                  in.StartTime,
		Metadata:                   in.Metadata,
		IsTest:                     in.IsTest,
		IsNSFW:                     in.IsNSFW,
		AIExtract:                  in.AIExtract,
		RetractedIdentifierIDs:     in.RetractedIdentifierIDs,
		ExternalIdentifiers:        in.ExternalIdentifiers,
		ExtractExternalIdentifiers: in.ExtractExternalIdentifiers,
	}

	switch {
	case in.EndTime.IsSet():
		b.EndTime = in.EndTime
	case in.InProgress:
		// An open activity carries no end time.
	case in.StartTime.IsSet():
		b.EndTime = in.StartTime
	}
	return b
}

// Create posts the entry, then reloads it because the API returns only an id.
func (s *JournalService) Create(ctx context.Context, in CreateEntryInput) (*JournalEntry, error) {
	if in.Type == "" {
		return nil, fmt.Errorf("%w: entry type is required", ErrValidation)
	}

	var created createEntryResponse
	if err := s.client.post(ctx, "/journal-entries", in.body(), &created); err != nil {
		return nil, err
	}

	entry, err := s.Get(ctx, created.ID)
	if err != nil {
		// The entry exists; only the reload failed. Hand back the id so the
		// caller is not left without a handle on what it just wrote.
		return &JournalEntry{
			ID:                   created.ID,
			Type:                 string(in.Type),
			Description:          in.Description,
			FailedIdentifiers:    created.FailedIdentifiers,
			ExtractedIdentifiers: created.ExtractedIdentifiers,
		}, fmt.Errorf("scambus: created journal entry %s but could not load it: %w", created.ID, err)
	}
	entry.FailedIdentifiers = created.FailedIdentifiers
	entry.ExtractedIdentifiers = created.ExtractedIdentifiers
	return entry, nil
}

func (s *JournalService) CreateBatch(ctx context.Context, entries []CreateEntryInput) (*BatchCreateResult, error) {
	bodies := make([]createEntryBody, len(entries))
	for i, e := range entries {
		bodies[i] = e.body()
	}
	var out BatchCreateResult
	if err := s.client.post(ctx, "/journal-entries/batch", map[string]any{"entries": bodies}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type getEntryResponse struct {
	JournalEntry struct {
		JournalEntry JournalEntry `json:"journal_entry"`
		CanEdit      bool         `json:"can_edit"`
	} `json:"journal_entry"`
	Cases []Case `json:"cases"`
}

func (s *JournalService) Get(ctx context.Context, entryID string) (*JournalEntry, error) {
	var out getEntryResponse
	if err := s.client.get(ctx, "/journal-entries/"+entryID, nil, &out); err != nil {
		return nil, err
	}
	entry := out.JournalEntry.JournalEntry
	return &entry, nil
}

func (s *JournalService) Delete(ctx context.Context, entryID string) error {
	return s.client.delete(ctx, "/journal-entries/"+entryID)
}

type ListEntriesOptions struct {
	Type     EntryType
	Page     int
	PageSize int
}

type listEntriesResponse struct {
	Data []struct {
		JournalEntry JournalEntry `json:"journal_entry"`
		CanEdit      bool         `json:"can_edit"`
	} `json:"data"`
	Pagination Pagination `json:"pagination"`
}

func (s *JournalService) List(ctx context.Context, opts *ListEntriesOptions) ([]JournalEntry, error) {
	q := url.Values{}
	page, pageSize := 1, 25
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PageSize > 0 {
			pageSize = opts.PageSize
		}
		if opts.Type != "" {
			q.Set("type", string(opts.Type))
		}
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))

	var out listEntriesResponse
	if err := s.client.get(ctx, "/journal-entries", q, &out); err != nil {
		return nil, err
	}
	entries := make([]JournalEntry, 0, len(out.Data))
	for _, item := range out.Data {
		entries = append(entries, item.JournalEntry)
	}
	return entries, nil
}

type QueryEntriesInput struct {
	Filter               *FilterCriteria
	OrderBy              string
	OrderDesc            bool
	Cursor               string
	Limit                int
	IncludeIdentifiers   bool
	IncludeEvidence      bool
	IncludeOriginator    bool
	IncludeChildren      bool
	ParentJournalEntryID string
}

type QueryEntriesResult struct {
	Data           []JournalEntry `json:"data"`
	NextCursor     string         `json:"nextCursor"`
	HasMore        bool           `json:"hasMore"`
	Count          int            `json:"count"`
	EstimatedTotal *int           `json:"estimatedTotal"`
}

// queryEntriesBody must not redeclare any JSON name that FilterCriteria
// already carries: Go resolves the collision by depth, the outer field wins,
// and the caller's filter value is dropped without a word.
type queryEntriesBody struct {
	*FilterCriteria
	OrderBy              string `json:"order_by,omitempty"`
	OrderDesc            bool   `json:"order_desc"`
	IncludeOriginator    bool   `json:"include_originator,omitempty"`
	IncludeChildren      bool   `json:"include_children,omitempty"`
	Cursor               string `json:"cursor,omitempty"`
	Limit                int    `json:"limit,omitempty"`
	ParentJournalEntryID string `json:"parent_journal_entry_id,omitempty"`
}

func (s *JournalService) Query(ctx context.Context, in QueryEntriesInput) (*QueryEntriesResult, error) {
	filter := FilterCriteria{}
	if in.Filter != nil {
		filter = *in.Filter
	}
	if in.IncludeIdentifiers {
		filter.IncludeIdentifiers = Ptr(true)
	}
	if in.IncludeEvidence {
		filter.IncludeEvidence = Ptr(true)
	}
	orderBy := in.OrderBy
	if orderBy == "" {
		orderBy = "performed_at"
	}

	body := queryEntriesBody{
		FilterCriteria:       &filter,
		OrderBy:              orderBy,
		OrderDesc:            in.OrderDesc,
		IncludeOriginator:    in.IncludeOriginator,
		IncludeChildren:      in.IncludeChildren,
		Cursor:               in.Cursor,
		Limit:                in.Limit,
		ParentJournalEntryID: in.ParentJournalEntryID,
	}

	var out QueryEntriesResult
	if err := s.client.post(ctx, "/journal/query", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QueryAll walks every page of a query, calling fn for each entry.
func (s *JournalService) QueryAll(ctx context.Context, in QueryEntriesInput, fn func(JournalEntry) error) error {
	for {
		page, err := s.Query(ctx, in)
		if err != nil {
			return err
		}
		for _, entry := range page.Data {
			if err := fn(entry); err != nil {
				return err
			}
		}
		if !page.HasMore || page.NextCursor == "" || page.NextCursor == in.Cursor {
			return nil
		}
		in.Cursor = page.NextCursor
	}
}

func (s *JournalService) InProgress(ctx context.Context) ([]JournalEntry, error) {
	var out []JournalEntry
	if err := s.client.get(ctx, "/journal-entries/in-progress", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompleteActivity closes an in-progress entry with an activity_complete child.
func (s *JournalService) CompleteActivity(ctx context.Context, parentID string, endTime Time, reason, description string) (*JournalEntry, error) {
	parent, err := s.Get(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if !parent.StartTime.IsSet() {
		return nil, fmt.Errorf("%w: entry %s has no start_time", ErrValidation, parentID)
	}
	if !endTime.IsSet() {
		endTime = NewTime(time.Now().UTC())
	}
	if reason == "" {
		reason = "manual"
	}
	if description == "" {
		description = "Activity completed (" + reason + ")"
	}

	duration := endTime.Sub(parent.StartTime.Time)
	if duration < 0 {
		return nil, fmt.Errorf("%w: end time %s precedes the entry's start time %s",
			ErrValidation, endTime.Format(time.RFC3339), parent.StartTime.Format(time.RFC3339))
	}

	return s.Create(ctx, CreateEntryInput{
		Type:                 EntryTypeActivityComplete,
		Description:          description,
		ParentJournalEntryID: parentID,
		Details: ActivityCompleteDetails{
			CompletionReason: reason,
			StartTime:        parent.StartTime,
			EndTime:          endTime,
			DurationSeconds:  int(duration.Seconds()),
		},
	})
}

func (s *JournalService) ExtractedIdentifiers(ctx context.Context, entryID string) ([]ExtractedIdentifier, error) {
	var out []ExtractedIdentifier
	if err := s.client.get(ctx, "/journal-entries/"+entryID+"/extracted-identifiers", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *JournalService) IdentifierSummary(ctx context.Context, entryID string, identifierType IdentifierType) (*IdentifierSummary, error) {
	q := url.Values{}
	if identifierType != "" {
		q.Set("type", string(identifierType))
	}
	var out IdentifierSummary
	if err := s.client.get(ctx, "/journal-entries/"+entryID+"/identifier-summary", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *JournalService) ExternalSystems(ctx context.Context) ([]ExternalSystem, error) {
	var out []ExternalSystem
	if err := s.client.get(ctx, "/external-systems", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

var errNoMessages = errors.New("scambus: messages must not be empty")
