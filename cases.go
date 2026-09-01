package scambus

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type CaseService struct{ client *Client }

type ListCasesOptions struct {
	Page     int
	PageSize int
	Status   string
	Priority string
	Category string
}

type listCasesResponse struct {
	Data       []Case     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

func (s *CaseService) List(ctx context.Context, opts *ListCasesOptions) ([]Case, error) {
	q := url.Values{}
	page, pageSize := 1, 25
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PageSize > 0 {
			pageSize = opts.PageSize
		}
		for key, value := range map[string]string{
			"status": opts.Status, "priority": opts.Priority, "category": opts.Category,
		} {
			if value != "" {
				q.Set(key, value)
			}
		}
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))

	var out listCasesResponse
	if err := s.client.get(ctx, "/cases", q, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (s *CaseService) Get(ctx context.Context, caseID string) (*Case, error) {
	var out Case
	if err := s.client.get(ctx, "/cases/"+caseID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type CreateCaseInput struct {
	Title    string         `json:"title"`
	Notes    string         `json:"notes,omitempty"`
	Status   string         `json:"status,omitempty"`
	Priority string         `json:"priority,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	IsTest   bool           `json:"is_test,omitempty"`
}

func (s *CaseService) Create(ctx context.Context, in CreateCaseInput) (*Case, error) {
	if in.Status == "" {
		in.Status = "open"
	}
	if in.Priority == "" {
		in.Priority = "medium"
	}
	var out Case
	if err := s.client.post(ctx, "/cases", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Resolution values accepted when closing a case.
const (
	ResolutionResolved    = "resolved"
	ResolutionUnresolved  = "unresolved"
	ResolutionTransferred = "transferred"
	ResolutionDuplicate   = "duplicate"
)

var validResolutions = map[string]bool{
	ResolutionResolved: true, ResolutionUnresolved: true,
	ResolutionTransferred: true, ResolutionDuplicate: true,
}

type UpdateCaseInput struct {
	Title    *string `json:"title,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	Status   *string `json:"status,omitempty"`
	Priority *string `json:"priority,omitempty"`
	IsTest   *bool   `json:"is_test,omitempty"`

	// Both are required, and must be non-empty, when Status is "closed".
	Resolution   *string `json:"resolution,omitempty"`
	ClosureNotes *string `json:"closure_notes,omitempty"`
}

func (in UpdateCaseInput) empty() bool {
	return in.Title == nil && in.Notes == nil && in.Status == nil && in.Priority == nil &&
		in.IsTest == nil && in.Resolution == nil && in.ClosureNotes == nil
}

func (in UpdateCaseInput) validate() error {
	if in.Status == nil || *in.Status != "closed" {
		return nil
	}
	if in.Resolution == nil || *in.Resolution == "" {
		return fmt.Errorf("%w: closing a case requires a resolution", ErrValidation)
	}
	if !validResolutions[*in.Resolution] {
		return fmt.Errorf("%w: resolution %q must be one of resolved, unresolved, transferred, duplicate",
			ErrValidation, *in.Resolution)
	}
	if in.ClosureNotes == nil || *in.ClosureNotes == "" {
		return fmt.Errorf("%w: closing a case requires closure notes", ErrValidation)
	}
	return nil
}

// Update reloads the case afterwards because the API answers 204.
func (s *CaseService) Update(ctx context.Context, caseID string, in UpdateCaseInput) (*Case, error) {
	if in.empty() {
		return nil, fmt.Errorf("%w: at least one field must be set", ErrValidation)
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	if err := s.client.put(ctx, "/cases/"+caseID, in, nil); err != nil {
		return nil, err
	}
	return s.Get(ctx, caseID)
}

func (s *CaseService) Delete(ctx context.Context, caseID string) error {
	return s.client.delete(ctx, "/cases/"+caseID)
}

func (s *CaseService) ListComments(ctx context.Context, caseID string) ([]CaseComment, error) {
	var out struct {
		Comments []CaseComment `json:"comments"`
		Total    int           `json:"total"`
	}
	if err := s.client.get(ctx, "/cases/"+caseID+"/comments", nil, &out); err != nil {
		return nil, err
	}
	return out.Comments, nil
}

func (s *CaseService) CreateComment(ctx context.Context, caseID, content, parentCommentID string) (*CaseComment, error) {
	body := map[string]string{"content": content}
	if parentCommentID != "" {
		body["parent_comment_id"] = parentCommentID
	}
	var out CaseComment
	if err := s.client.post(ctx, "/cases/"+caseID+"/comments", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CaseService) CommentCount(ctx context.Context, caseID string) (int, error) {
	var out struct {
		Count int `json:"count"`
	}
	if err := s.client.get(ctx, "/cases/"+caseID+"/comments/count", nil, &out); err != nil {
		return 0, err
	}
	return out.Count, nil
}

type CommentService struct{ client *Client }

func (s *CommentService) Update(ctx context.Context, commentID, content string) (*CaseComment, error) {
	var out CaseComment
	if err := s.client.put(ctx, "/comments/"+commentID, map[string]string{"content": content}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CommentService) Delete(ctx context.Context, commentID string) error {
	return s.client.delete(ctx, "/comments/"+commentID)
}
