package scambus

import "context"

type SearchService struct{ client *Client }

type SearchIdentifiersInput struct {
	Filter                *FilterCriteria
	Limit                 int
	Cursor                string
	IncludeJournalEntries bool
}

type SearchIdentifiersResult struct {
	Data           []Identifier `json:"data"`
	NextCursor     string       `json:"nextCursor"`
	HasMore        bool         `json:"hasMore"`
	EstimatedTotal *int         `json:"estimatedTotal"`
}

type searchIdentifiersBody struct {
	*FilterCriteria
	Limit                 int    `json:"limit,omitempty"`
	Cursor                string `json:"cursor,omitempty"`
	IncludeJournalEntries bool   `json:"include_journal_entries,omitempty"`
}

func (s *SearchService) Identifiers(ctx context.Context, in SearchIdentifiersInput) (*SearchIdentifiersResult, error) {
	filter := in.Filter
	if filter == nil {
		filter = &FilterCriteria{}
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 100
	}

	var out SearchIdentifiersResult
	body := searchIdentifiersBody{
		FilterCriteria:        filter,
		Limit:                 limit,
		Cursor:                in.Cursor,
		IncludeJournalEntries: in.IncludeJournalEntries,
	}
	if err := s.client.post(ctx, "/search/identifiers", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// IdentifiersAll walks every page of an identifier search.
func (s *SearchService) IdentifiersAll(ctx context.Context, in SearchIdentifiersInput, fn func(Identifier) error) error {
	for {
		page, err := s.Identifiers(ctx, in)
		if err != nil {
			return err
		}
		for _, identifier := range page.Data {
			if err := fn(identifier); err != nil {
				return err
			}
		}
		if !page.HasMore || page.NextCursor == "" {
			return nil
		}
		in.Cursor = page.NextCursor
	}
}

// SearchCasesInput is camelCase on the wire, unlike the identifier search.
type SearchCasesInput struct {
	SearchQuery        string   `json:"searchQuery,omitempty"`
	Status             []string `json:"status,omitempty"`
	Priority           []string `json:"priority,omitempty"`
	Category           string   `json:"category,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	AssignedTo         string   `json:"assignedTo,omitempty"`
	CreatedBy          string   `json:"createdBy,omitempty"`
	Resolution         string   `json:"resolution,omitempty"`
	CreatedAfter       string   `json:"createdAfter,omitempty"`
	CreatedBefore      string   `json:"createdBefore,omitempty"`
	UpdatedAfter       string   `json:"updatedAfter,omitempty"`
	UpdatedBefore      string   `json:"updatedBefore,omitempty"`
	ClosedAfter        string   `json:"closedAfter,omitempty"`
	ClosedBefore       string   `json:"closedBefore,omitempty"`
	IncludesIdentifier string   `json:"includesIdentifier,omitempty"`
	TeamMemberInvolved string   `json:"teamMemberInvolved,omitempty"`
	Cursor             string   `json:"cursor,omitempty"`
	Limit              int      `json:"limit,omitempty"`
	IsTest             *bool    `json:"isTest,omitempty"`
}

type SearchCasesResult struct {
	Data       []Case `json:"data"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
}

func (s *SearchService) Cases(ctx context.Context, in SearchCasesInput) (*SearchCasesResult, error) {
	if in.Limit <= 0 {
		in.Limit = 50
	}
	var out SearchCasesResult
	if err := s.client.post(ctx, "/search/cases", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
