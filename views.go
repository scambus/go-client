package scambus

import (
	"context"
	"encoding/json"
)

type ViewService struct{ client *Client }

type ListViewsResult struct {
	Data       []View     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

func (s *ViewService) List(ctx context.Context, page *PageRequest) (*ListViewsResult, error) {
	var out ListViewsResult
	if err := s.client.get(ctx, "/views", page.values(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ViewService) Get(ctx context.Context, viewID string) (*View, error) {
	var out View
	if err := s.client.get(ctx, "/views/"+viewID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateViewInput struct {
	Name            string          `json:"name"`
	EntityType      string          `json:"entity_type"`
	Visibility      string          `json:"visibility,omitempty"`
	ViewType        string          `json:"view_type,omitempty"`
	Description     string          `json:"description,omitempty"`
	Alias           string          `json:"alias,omitempty"`
	FilterCriteria  *FilterCriteria `json:"filter_criteria,omitempty"`
	SortOrder       *SortOrder      `json:"sort_order,omitempty"`
	QueryString     string          `json:"query_string,omitempty"`
	DisplaySettings map[string]any  `json:"display_settings,omitempty"`
}

func (s *ViewService) Create(ctx context.Context, in CreateViewInput) (*View, error) {
	if in.Visibility == "" {
		in.Visibility = "organization"
	}
	if in.ViewType == "" {
		in.ViewType = "standard"
	}
	var out View
	if err := s.client.post(ctx, "/views", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateViewInput struct {
	Name            *string         `json:"name,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Visibility      *string         `json:"visibility,omitempty"`
	FilterCriteria  *FilterCriteria `json:"filter_criteria,omitempty"`
	SortOrder       *SortOrder      `json:"sort_order,omitempty"`
	QueryString     *string         `json:"query_string,omitempty"`
	DisplaySettings map[string]any  `json:"display_settings,omitempty"`
}

func (s *ViewService) Update(ctx context.Context, viewID string, in UpdateViewInput) (*View, error) {
	var out View
	if err := s.client.put(ctx, "/views/"+viewID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ViewService) Delete(ctx context.Context, viewID string) error {
	return s.client.delete(ctx, "/views/"+viewID)
}

type ExecuteViewResult struct {
	Data       []json.RawMessage `json:"data"`
	NextCursor string            `json:"nextCursor"`
	HasMore    bool              `json:"hasMore"`
	TotalCount int               `json:"totalCount"`
	View       *View             `json:"view,omitempty"`
}

func (r ExecuteViewResult) JournalEntries() ([]JournalEntry, error) {
	return decodeMessages[JournalEntry](r.Data)
}

func (r ExecuteViewResult) Identifiers() ([]Identifier, error) {
	return decodeMessages[Identifier](r.Data)
}

type ExecuteViewOptions struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (s *ViewService) Execute(ctx context.Context, viewID string, opts *ExecuteViewOptions) (*ExecuteViewResult, error) {
	body := ExecuteViewOptions{}
	if opts != nil {
		body = *opts
	}
	var out ExecuteViewResult
	if err := s.client.post(ctx, "/views/"+viewID+"/execute", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ViewService) MyJournalEntries(ctx context.Context) (*View, error) {
	var out View
	if err := s.client.get(ctx, "/views/my-journal-entries", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ViewService) MyPinboard(ctx context.Context) (*View, error) {
	var out View
	if err := s.client.get(ctx, "/views/my-pinboard", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *ViewService) ExecuteMyJournalEntries(ctx context.Context, opts *ExecuteViewOptions) (*ExecuteViewResult, error) {
	view, err := s.MyJournalEntries(ctx)
	if err != nil {
		return nil, err
	}
	return s.Execute(ctx, view.ID, opts)
}

func (s *ViewService) ExecuteMyPinboard(ctx context.Context, opts *ExecuteViewOptions) (*ExecuteViewResult, error) {
	view, err := s.MyPinboard(ctx)
	if err != nil {
		return nil, err
	}
	return s.Execute(ctx, view.ID, opts)
}
