package scambus

import (
	"net/url"
	"strconv"
)

type IdentifierType string

const (
	IdentifierTypePhone        IdentifierType = "phone"
	IdentifierTypeEmail        IdentifierType = "email"
	IdentifierTypeURL          IdentifierType = "url"
	IdentifierTypeBankAccount  IdentifierType = "bank_account"
	IdentifierTypeCryptoWallet IdentifierType = "crypto_wallet"
	IdentifierTypeSocialMedia  IdentifierType = "social_media"
	IdentifierTypeZelle        IdentifierType = "zelle"
	IdentifierTypePaymentToken IdentifierType = "payment_token"
)

func (t IdentifierType) String() string { return string(t) }

var validIdentifierTypes = map[IdentifierType]bool{
	IdentifierTypePhone: true, IdentifierTypeEmail: true, IdentifierTypeURL: true,
	IdentifierTypeBankAccount: true, IdentifierTypeCryptoWallet: true,
	IdentifierTypeSocialMedia: true, IdentifierTypeZelle: true, IdentifierTypePaymentToken: true,
}

func (t IdentifierType) Valid() bool { return validIdentifierTypes[t] }

type PaymentService string

const (
	PaymentServiceZelle   PaymentService = "zelle"
	PaymentServiceCashApp PaymentService = "cashapp"
	PaymentServicePayPal  PaymentService = "paypal"
	PaymentServiceWise    PaymentService = "wise"
	PaymentServiceKrak    PaymentService = "krak"
	PaymentServiceVenmo   PaymentService = "venmo"
	PaymentServiceChime   PaymentService = "chime"
)

type EntryType string

const (
	EntryTypePhoneCall                EntryType = "phone_call"
	EntryTypeEmail                    EntryType = "email"
	EntryTypeTextConversation         EntryType = "text_conversation"
	EntryTypeConversationContinuation EntryType = "conversation_continuation"
	EntryTypeWebInteraction           EntryType = "web_interaction"
	EntryTypeDetection                EntryType = "detection"
	EntryTypeImport                   EntryType = "import"
	EntryTypeExport                   EntryType = "export"
	EntryTypeNote                     EntryType = "note"
	EntryTypeTagOperation             EntryType = "tag_operation"
	EntryTypeConfidenceOperation      EntryType = "confidence_operation"
	EntryTypeRedaction                EntryType = "redaction"
	EntryTypeCaseUpdate               EntryType = "case_update"
	EntryTypeCaseIdentifierLink       EntryType = "case_identifier_link"
	EntryTypeCaseIdentifierUnlink     EntryType = "case_identifier_unlink"
	EntryTypeKarmaAdjustment          EntryType = "karma_adjustment"
	EntryTypeActivityComplete         EntryType = "activity_complete"
	EntryTypeData                     EntryType = "data"
	EntryTypeTaskUpdate               EntryType = "task_update"
	EntryTypeTaskAssignment           EntryType = "task_assignment"
	EntryTypeCaseHandoff              EntryType = "case_handoff"
	EntryTypeEvidenceReview           EntryType = "evidence_review"
	EntryTypeFinancialTransaction     EntryType = "financial_transaction"
	EntryTypeFalsePositive            EntryType = "false_positive"
	EntryTypeEvidenceOperation        EntryType = "evidence_operation"
)

func (t EntryType) String() string { return string(t) }

type StreamDataType string

const (
	StreamDataJournalEntry StreamDataType = "journal_entry"
	StreamDataIdentifier   StreamDataType = "identifier"
)

const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

// Cursor values accepted by the stream consumer endpoints.
const (
	CursorStart = "0"
	CursorEnd   = "$"
)

type TagLookup struct {
	TagName       string `json:"tag_name"`
	TagValue      string `json:"tag_value,omitempty"`
	TagValueTitle string `json:"tag_value_title,omitempty"`
}

type SortOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type OriginatorLookup struct {
	Type             string `json:"type"`
	Identifier       string `json:"identifier"`
	CreateIfNotExist bool   `json:"create_if_not_exists"`
}

type ExternalIdentifierInput struct {
	ExternalSystem string `json:"external_system"`
	ExternalID     string `json:"external_id"`
	Source         string `json:"source,omitempty"`
	RawMatch       string `json:"raw_match,omitempty"`
	Link           string `json:"link,omitempty"`
}

// FilterCriteria is the shared filter shape used by search, query, views,
// export streams and file exports. Zero fields are omitted from the request.
type FilterCriteria struct {
	SearchQuery       string `json:"search_query,omitempty"`
	NegateSearchQuery *bool  `json:"negate_search_query,omitempty"`

	Status             []string `json:"status,omitempty"`
	Priority           []string `json:"priority,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	TagsAny            []string `json:"tags_any,omitempty"`
	Types              []string `json:"types,omitempty"`
	OriginatorIDs      []string `json:"originator_ids,omitempty"`
	ProxyOriginatorIDs []string `json:"proxy_originator_ids,omitempty"`
	OriginatorTypes    []string `json:"originator_types,omitempty"`
	OrgIDs             []string `json:"org_ids,omitempty"`

	Type           string `json:"type,omitempty"`
	IdentifierType string `json:"identifier_type,omitempty"`

	ConfidenceRange string   `json:"confidence_range,omitempty"`
	MinConfidence   *float64 `json:"min_confidence,omitempty"`
	MaxConfidence   *float64 `json:"max_confidence,omitempty"`

	CreatedAfter            string `json:"created_after,omitempty"`
	CreatedBefore           string `json:"created_before,omitempty"`
	PerformedAfter          string `json:"performed_after,omitempty"`
	PerformedBefore         string `json:"performed_before,omitempty"`
	DiscoveredAfter         string `json:"discovered_after,omitempty"`
	DiscoveredBefore        string `json:"discovered_before,omitempty"`
	ConfidenceChangedAfter  string `json:"confidence_changed_after,omitempty"`
	ConfidenceChangedBefore string `json:"confidence_changed_before,omitempty"`

	OriginatorFilterType string `json:"originator_filter_type,omitempty"`

	Details         map[string]any `json:"details,omitempty"`
	ExcludedDetails map[string]any `json:"excluded_details,omitempty"`

	HasMedia             *bool `json:"has_media,omitempty"`
	HasNotes             *bool `json:"has_notes,omitempty"`
	IsOurs               *bool `json:"is_ours,omitempty"`
	HumanReviewed        *bool `json:"human_reviewed,omitempty"`
	ExcludeHumanReviewed *bool `json:"exclude_human_reviewed,omitempty"`
	UserPinned           *bool `json:"user_pinned,omitempty"`
	IsTest               *bool `json:"is_test,omitempty"`
	IncludeIdentifiers   *bool `json:"include_identifiers,omitempty"`
	IncludeEvidence      *bool `json:"include_evidence,omitempty"`

	TollFree           *bool  `json:"toll_free,omitempty"`
	Institution        string `json:"institution,omitempty"`
	Platform           string `json:"platform,omitempty"`
	Service            string `json:"service,omitempty"`
	AreaCode           string `json:"area_code,omitempty"`
	Region             string `json:"region,omitempty"`
	Country            string `json:"country,omitempty"`
	State              string `json:"state,omitempty"`
	DomainCategory     string `json:"domain_category,omitempty"`
	IsPrivateSuffix    *bool  `json:"is_private_suffix,omitempty"`
	RoutingNumberOwner string `json:"routing_number_owner,omitempty"`

	Networks       []string `json:"networks,omitempty"`
	FormatClass    string   `json:"format_class,omitempty"`
	IsContract     *bool    `json:"is_contract,omitempty"`
	Sanctioned     *bool    `json:"sanctioned,omitempty"`
	SanctionsList  []string `json:"sanctions_list,omitempty"`
	KnownLabels    []string `json:"known_labels,omitempty"`
	EnrichedAfter  string   `json:"enriched_after,omitempty"`
	EnrichedBefore string   `json:"enriched_before,omitempty"`

	AssignedTo string `json:"assigned_to,omitempty"`

	TagNames         []string `json:"tag_names,omitempty"`
	ExcludedTagNames []string `json:"excluded_tag_names,omitempty"`

	ExcludedStatus             []string `json:"excluded_status,omitempty"`
	ExcludedPriority           []string `json:"excluded_priority,omitempty"`
	ExcludedTags               []string `json:"excluded_tags,omitempty"`
	ExcludedTypes              []string `json:"excluded_types,omitempty"`
	ExcludedOriginatorTypes    []string `json:"excluded_originator_types,omitempty"`
	ExcludedOriginatorIDs      []string `json:"excluded_originator_ids,omitempty"`
	ExcludedProxyOriginatorIDs []string `json:"excluded_proxy_originator_ids,omitempty"`
	ExcludedOrgIDs             []string `json:"excluded_org_ids,omitempty"`

	ExcludedOriginatorFilterType string `json:"excluded_originator_filter_type,omitempty"`

	NegateHasMedia *bool `json:"negate_has_media,omitempty"`
	NegateIsOurs   *bool `json:"negate_is_ours,omitempty"`
}

func Ptr[T any](v T) *T { return &v }

// PageRequest carries the paging parameters the API actually reads. The
// server ignores "limit"; page size is "pageSize", default 25, maximum 100.
type PageRequest struct {
	Page     int
	PageSize int
}

func (p *PageRequest) values() url.Values {
	if p == nil {
		return nil
	}
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(p.PageSize))
	}
	if len(q) == 0 {
		return nil
	}
	return q
}
