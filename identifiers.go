package scambus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type IdentifierService struct{ client *Client }

type ListIdentifiersOptions struct {
	Type     IdentifierType
	Page     int
	PageSize int
}

type listIdentifiersResponse struct {
	Data       []Identifier `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

func (s *IdentifierService) List(ctx context.Context, opts *ListIdentifiersOptions) ([]Identifier, error) {
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

	var out listIdentifiersResponse
	if err := s.client.get(ctx, "/identifiers", q, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (s *IdentifierService) Get(ctx context.Context, identifierID string) (*Identifier, error) {
	var out Identifier
	if err := s.client.get(ctx, "/identifiers/"+identifierID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type URLReferencesOptions struct {
	Page     int
	PageSize int
	Sort     string
	Order    string
}

type URLReferencesResult struct {
	URLReferences []IdentifierURLReference `json:"url_references"`
	Total         int                      `json:"total"`
	Page          int                      `json:"page"`
	PageSize      int                      `json:"page_size"`
}

func (s *IdentifierService) URLReferences(ctx context.Context, identifierID string, opts *URLReferencesOptions) (*URLReferencesResult, error) {
	page, pageSize, sort, order := 1, 25, "last_seen_at", SortDesc
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PageSize > 0 {
			pageSize = opts.PageSize
		}
		if opts.Sort != "" {
			sort = opts.Sort
		}
		if opts.Order != "" {
			order = opts.Order
		}
	}
	q := url.Values{
		"page":      {strconv.Itoa(page)},
		"page_size": {strconv.Itoa(pageSize)},
		"sort":      {sort},
		"order":     {order},
	}

	var out URLReferencesResult
	if err := s.client.get(ctx, "/identifiers/"+identifierID+"/url-references", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type listExclusionsResponse struct {
	Data       []IdentifierExclusion `json:"data"`
	Pagination Pagination            `json:"pagination"`
}

func (s *IdentifierService) ListExclusions(ctx context.Context, page, pageSize int) ([]IdentifierExclusion, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	q := url.Values{"page": {strconv.Itoa(page)}, "pageSize": {strconv.Itoa(pageSize)}}

	var out listExclusionsResponse
	if err := s.client.get(ctx, "/identifier-exclusions", q, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

type CreateExclusionInput struct {
	IdentifierID   string `json:"identifier_id,omitempty"`
	IdentifierType string `json:"identifier_type,omitempty"`
	Value          string `json:"value,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

func (s *IdentifierService) CreateExclusion(ctx context.Context, in CreateExclusionInput) (*IdentifierExclusion, error) {
	if in.IdentifierID == "" && (in.IdentifierType == "" || in.Value == "") {
		return nil, fmt.Errorf("%w: provide IdentifierID, or both IdentifierType and Value", ErrValidation)
	}
	var out IdentifierExclusion
	if err := s.client.post(ctx, "/identifier-exclusions", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *IdentifierService) DeleteExclusion(ctx context.Context, exclusionID string) error {
	return s.client.delete(ctx, "/identifier-exclusions/"+exclusionID)
}

type BankAccountInput struct {
	Account      string
	Routing      string
	Institution  string
	Owner        string
	OwnerAddress string
	Country      string
	Confidence   *float64
}

// BankAccountLookup packs bank fields into the JSON-encoded lookup value the
// API expects.
func BankAccountLookup(in BankAccountInput) (IdentifierLookup, error) {
	if in.Account == "" || in.Routing == "" || in.Institution == "" {
		return IdentifierLookup{}, fmt.Errorf("%w: account, routing and institution are required", ErrValidation)
	}
	value, err := json.Marshal(BankAccountDetails{
		AccountNumber: in.Account,
		Routing:       in.Routing,
		Institution:   in.Institution,
		Owner:         in.Owner,
		OwnerAddress:  in.OwnerAddress,
		Country:       in.Country,
	})
	if err != nil {
		return IdentifierLookup{}, err
	}
	return IdentifierLookup{
		Type:       IdentifierTypeBankAccount,
		Value:      string(value),
		Confidence: in.Confidence,
	}, nil
}

var (
	venmoUserID   = regexp.MustCompile(`^\d{16,19}$`)
	venmoUsername = regexp.MustCompile(`^@[a-zA-Z0-9_-]{5,30}$`)
	chimeSign     = regexp.MustCompile(`^\$[a-zA-Z0-9_]{1,20}$`)
)

// VenmoLookup accepts an @username, a 16-19 digit user id, or a
// venmo.com/code QR URL.
func VenmoLookup(identifier, name string, confidence *float64) (IdentifierLookup, error) {
	identifier = strings.TrimSpace(identifier)
	details := PaymentTokenDetails{Service: string(PaymentServiceVenmo), Name: name}

	switch {
	case strings.HasPrefix(identifier, "http://"), strings.HasPrefix(identifier, "https://"):
		parsed, err := url.Parse(identifier)
		if err != nil {
			return IdentifierLookup{}, fmt.Errorf("%w: invalid Venmo URL: %w", ErrValidation, err)
		}
		if !strings.EqualFold(parsed.Hostname(), "venmo.com") {
			return IdentifierLookup{}, fmt.Errorf("%w: Venmo URL must be from venmo.com, got %q", ErrValidation, parsed.Hostname())
		}
		if strings.TrimRight(parsed.Path, "/") != "/code" {
			return IdentifierLookup{}, fmt.Errorf("%w: Venmo URL must use the /code path, got %q", ErrValidation, parsed.Path)
		}
		userID := parsed.Query().Get("user_id")
		if !venmoUserID.MatchString(userID) {
			return IdentifierLookup{}, fmt.Errorf("%w: Venmo user_id must be 16-19 digits, got %q", ErrValidation, userID)
		}
		details.Identifier = identifier
	case strings.HasPrefix(identifier, "@"):
		if !venmoUsername.MatchString(identifier) {
			return IdentifierLookup{}, fmt.Errorf("%w: Venmo @username must be 5-30 alphanumeric, hyphen or underscore characters", ErrValidation)
		}
		details.Identifier = identifier
	case venmoUserID.MatchString(identifier):
		details.Identifier = identifier
	default:
		return IdentifierLookup{}, fmt.Errorf("%w: Venmo identifier must be an @username, a 16-19 digit user id, or a venmo.com/code URL", ErrValidation)
	}

	return paymentTokenLookup(details, confidence)
}

// ChimeLookup accepts a $ChimeSign.
func ChimeLookup(chimesign, name string, confidence *float64) (IdentifierLookup, error) {
	chimesign = strings.TrimSpace(chimesign)
	if !chimeSign.MatchString(chimesign) {
		return IdentifierLookup{}, fmt.Errorf("%w: Chime identifier must be a $ChimeSign of 1-20 alphanumeric or underscore characters", ErrValidation)
	}
	return paymentTokenLookup(PaymentTokenDetails{
		Service:    string(PaymentServiceChime),
		Identifier: chimesign,
		Name:       name,
	}, confidence)
}

func paymentTokenLookup(details PaymentTokenDetails, confidence *float64) (IdentifierLookup, error) {
	value, err := json.Marshal(details)
	if err != nil {
		return IdentifierLookup{}, err
	}
	return IdentifierLookup{
		Type:       IdentifierTypePaymentToken,
		Value:      string(value),
		Confidence: confidence,
	}, nil
}
