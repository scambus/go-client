package scambus

import (
	"context"
	"net/url"
	"strconv"
)

type AdminService struct{ client *Client }

type ListDomainRulesOptions struct {
	Category string
	Active   *bool
}

func (s *AdminService) ListSpecialDomainRules(ctx context.Context, opts *ListDomainRulesOptions) ([]SpecialDomainRule, error) {
	q := url.Values{}
	if opts != nil {
		if opts.Category != "" {
			q.Set("category", opts.Category)
		}
		if opts.Active != nil {
			q.Set("active", strconv.FormatBool(*opts.Active))
		}
	}
	var out []SpecialDomainRule
	if err := s.client.get(ctx, "/admin/special-domain-rules", q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type CreateDomainRuleInput struct {
	Domain        string `json:"domain"`
	Category      string `json:"category"`
	PathDepth     int    `json:"path_depth"`
	StripQuery    bool   `json:"strip_query"`
	StripFragment bool   `json:"strip_fragment"`
}

func (s *AdminService) CreateSpecialDomainRule(ctx context.Context, in CreateDomainRuleInput) (*SpecialDomainRule, error) {
	var out SpecialDomainRule
	if err := s.client.post(ctx, "/admin/special-domain-rules", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateDomainRuleInput struct {
	Domain        *string `json:"domain,omitempty"`
	Category      *string `json:"category,omitempty"`
	PathDepth     *int    `json:"path_depth,omitempty"`
	StripQuery    *bool   `json:"strip_query,omitempty"`
	StripFragment *bool   `json:"strip_fragment,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

func (s *AdminService) UpdateSpecialDomainRule(ctx context.Context, ruleID string, in UpdateDomainRuleInput) (*SpecialDomainRule, error) {
	var out SpecialDomainRule
	if err := s.client.put(ctx, "/admin/special-domain-rules/"+ruleID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AdminService) DeleteSpecialDomainRule(ctx context.Context, ruleID string) error {
	return s.client.delete(ctx, "/admin/special-domain-rules/"+ruleID)
}

func (s *AdminService) StartURLConsolidation(ctx context.Context) (*URLConsolidationStatus, error) {
	var out URLConsolidationStatus
	if err := s.client.post(ctx, "/admin/url-consolidation/start", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AdminService) URLConsolidationStatus(ctx context.Context) (*URLConsolidationStatus, error) {
	var out URLConsolidationStatus
	if err := s.client.get(ctx, "/admin/url-consolidation/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *AdminService) CancelURLConsolidation(ctx context.Context) error {
	return s.client.post(ctx, "/admin/url-consolidation/cancel", nil, nil)
}
