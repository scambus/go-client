package scambus

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type CaseService struct{ client *Client }

type ListCasesOptions struct {
	Page        int
	Limit       int
	Status      string
	Priority    string
	Category    string
	IncludeTest bool
}

type listCasesResponse struct {
	Data       []Case     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

func (s *CaseService) List(ctx context.Context, opts *ListCasesOptions) ([]Case, error) {
	q := url.Values{}
	page, limit := 1, 20
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		for key, value := range map[string]string{
			"status": opts.Status, "priority": opts.Priority, "category": opts.Category,
		} {
			if value != "" {
				q.Set(key, value)
			}
		}
		if opts.IncludeTest {
			q.Set("includeTest", "true")
		}
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))

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

type UpdateCaseInput struct {
	Title    *string `json:"title,omitempty"`
	Notes    *string `json:"notes,omitempty"`
	Status   *string `json:"status,omitempty"`
	Priority *string `json:"priority,omitempty"`
	IsTest   *bool   `json:"is_test,omitempty"`
}

func (in UpdateCaseInput) empty() bool {
	return in.Title == nil && in.Notes == nil && in.Status == nil && in.Priority == nil && in.IsTest == nil
}

// Update reloads the case afterwards because the API answers 204.
func (s *CaseService) Update(ctx context.Context, caseID string, in UpdateCaseInput) (*Case, error) {
	if in.empty() {
		return nil, fmt.Errorf("%w: at least one field must be set", ErrValidation)
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
	var out []CaseComment
	if err := s.client.get(ctx, "/cases/"+caseID+"/comments", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *CaseService) CreateComment(ctx context.Context, caseID, content, parentCommentID string) (*CaseComment, error) {
	body := map[string]string{"content": content}
	if parentCommentID != "" {
		body["parentCommentId"] = parentCommentID
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
