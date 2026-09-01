package scambus

import (
	"encoding/json"
	"fmt"
)

type PhoneCallDetails struct {
	Direction     string                `json:"direction"`
	RecordingURL  string                `json:"recording_url,omitempty"`
	TranscriptURL string                `json:"transcript_url,omitempty"`
	Transcript    []ConversationMessage `json:"transcript,omitempty"`
}

type EmailDetails struct {
	Direction   string            `json:"direction,omitempty"`
	Subject     string            `json:"subject,omitempty"`
	SentAt      Time              `json:"sent_at,omitzero"`
	Body        string            `json:"body,omitempty"`
	HTMLBody    string            `json:"html_body,omitempty"`
	MessageID   string            `json:"message_id,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []string          `json:"attachments,omitempty"`
}

type TextConversationDetails struct {
	Platform         string         `json:"platform"`
	ConversationType string         `json:"conversation_type,omitempty"`
	ConversationID   string         `json:"conversation_id,omitempty"`
	FirstMessageAt   Time           `json:"first_message_at,omitzero"`
	LastMessageAt    Time           `json:"last_message_at,omitzero"`
	SourceType       string         `json:"source_type,omitempty"`
	Subject          string         `json:"subject,omitempty"`
	ParticipantCount *int           `json:"participant_count,omitempty"`
	ExportFormat     string         `json:"export_format,omitempty"`
	CollectionMethod string         `json:"collection_method,omitempty"`
	ChainOfCustody   []CustodyEvent `json:"chain_of_custody,omitempty"`
	PlatformMetadata map[string]any `json:"platform_metadata,omitempty"`
}

type ConversationContinuationDetails struct {
	Messages      []ConversationMessage `json:"messages"`
	Reason        string                `json:"reason,omitempty"`
	NonContiguous bool                  `json:"non_contiguous"`
}

type ConversationMessage struct {
	Index                   int                    `json:"index"`
	MessageID               string                 `json:"message_id"`
	Timestamp               Time                   `json:"timestamp,omitzero"`
	Body                    string                 `json:"body"`
	IsOutgoing              bool                   `json:"is_outgoing"`
	MessageType             string                 `json:"message_type,omitempty"`
	PhoneCallJournalEntryID string                 `json:"phone_call_journal_entry_id,omitempty"`
	SenderRef               string                 `json:"sender_ref,omitempty"`
	SenderDisplayName       string                 `json:"sender_display_name,omitempty"`
	IdentifierRefs          []MessageIdentifierRef `json:"identifier_refs,omitempty"`
	Attachments             []MessageAttachment    `json:"attachments,omitempty"`
	Timezone                string                 `json:"timezone,omitempty"`
	BodyHTML                string                 `json:"body_html,omitempty"`
	Subject                 string                 `json:"subject,omitempty"`
	IsDeleted               bool                   `json:"is_deleted"`
	IsEdited                bool                   `json:"is_edited"`
	DeliveryStatus          string                 `json:"delivery_status,omitempty"`
	DeliveryTimestamp       Time                   `json:"delivery_timestamp,omitzero"`
	ReadTimestamp           Time                   `json:"read_timestamp,omitzero"`
	InReplyTo               string                 `json:"in_reply_to,omitempty"`
	Headers                 map[string]string      `json:"headers,omitempty"`
	PlatformMetadata        map[string]any         `json:"platform_metadata,omitempty"`
}

type MessageIdentifierRef struct {
	Ref      string `json:"ref"`
	Field    string `json:"field"`
	Position *int   `json:"position,omitempty"`
	Length   *int   `json:"length,omitempty"`
}

type MessageAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   *int64 `json:"size_bytes,omitempty"`
	URL         string `json:"url,omitempty"`
	MediaID     string `json:"media_id,omitempty"`
}

type CustodyEvent struct {
	Timestamp Time   `json:"timestamp,omitzero"`
	Event     string `json:"event"`
	Method    string `json:"method"`
	Actor     string `json:"actor,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

type DetectionDetails struct {
	Data       map[string]any `json:"data,omitempty"`
	Category   string         `json:"category,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	Confidence *float64       `json:"confidence,omitempty"`
}

type ImportDetails struct {
	Source      string `json:"source"`
	RecordCount int    `json:"record_count"`
	ImportedAt  Time   `json:"imported_at,omitzero"`
	FileName    string `json:"file_name,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type ExportDetails struct {
	Destination string `json:"destination"`
	RecordCount int    `json:"record_count"`
	ExportedAt  Time   `json:"exported_at,omitzero"`
	FileName    string `json:"file_name,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type ContactDetails struct {
	Method      string `json:"method"`
	Direction   string `json:"direction"`
	ContactedAt Time   `json:"contacted_at,omitzero"`
	Duration    *int   `json:"duration,omitempty"`
	Outcome     string `json:"outcome,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type ResearchDetails struct {
	Topic        string   `json:"topic"`
	ResearchedAt Time     `json:"researched_at,omitzero"`
	Sources      []string `json:"sources,omitempty"`
	Findings     string   `json:"findings,omitempty"`
	Confidence   *float64 `json:"confidence,omitempty"`
}

type AnalysisDetails struct {
	AnalysisType string         `json:"analysis_type"`
	Findings     string         `json:"findings"`
	AnalyzedAt   Time           `json:"analyzed_at,omitzero"`
	Confidence   *float64       `json:"confidence,omitempty"`
	Metrics      map[string]any `json:"metrics,omitempty"`
}

type ActionDetails struct {
	ActionType string `json:"action_type"`
	TakenAt    Time   `json:"taken_at,omitzero"`
	Outcome    string `json:"outcome,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type ObservationDetails struct {
	ObservationType string `json:"observation_type"`
	ObservedAt      Time   `json:"observed_at,omitzero"`
	Data            string `json:"data"`
	Significance    string `json:"significance,omitempty"`
}

type NoteDetails struct {
	Content  string `json:"content"`
	NotedAt  Time   `json:"noted_at,omitzero"`
	Category string `json:"category,omitempty"`
}

type UpdateDetails struct {
	UpdateType    string `json:"update_type"`
	UpdatedAt     Time   `json:"updated_at,omitzero"`
	Changes       string `json:"changes"`
	PreviousValue string `json:"previous_value,omitempty"`
	NewValue      string `json:"new_value,omitempty"`
}

type ActivityCompleteDetails struct {
	CompletionReason string `json:"completion_reason"`
	StartTime        Time   `json:"start_time,omitzero"`
	EndTime          Time   `json:"end_time,omitzero"`
	DurationSeconds  int    `json:"duration_seconds"`
}

type TagOperationDetails struct {
	Operation  string `json:"operation"`
	TagID      string `json:"tag_id"`
	TagValueID string `json:"tag_value_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type IdentifierConfidenceUpdate struct {
	IdentifierID  string  `json:"identifier_id"`
	PreviousScore float64 `json:"previous_score"`
	NewScore      float64 `json:"new_score"`
	Reason        string  `json:"reason,omitempty"`
}

type ConfidenceOperationDetails struct {
	Identifiers []IdentifierConfidenceUpdate `json:"identifiers"`
	Reason      string                       `json:"reason,omitempty"`
	Metadata    map[string]any               `json:"metadata,omitempty"`
}

type RedactionDetails struct {
	IdentifierID   string   `json:"identifier_id"`
	IdentifierType string   `json:"identifier_type"`
	OriginalHash   string   `json:"original_hash"`
	RedactedFields []string `json:"redacted_fields"`
	RedactedAt     Time     `json:"redacted_at,omitzero"`
	Reason         string   `json:"reason,omitempty"`
}

type CaseUpdateDetails struct {
	CaseID     string         `json:"case_id"`
	UpdateType string         `json:"update_type"`
	NewValue   string         `json:"new_value"`
	OldValue   string         `json:"old_value,omitempty"`
	Notes      string         `json:"notes,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type CaseIdentifierLinkDetails struct {
	Operation       string         `json:"operation"`
	CaseID          string         `json:"case_id"`
	IdentifierID    string         `json:"identifier_id"`
	Reason          string         `json:"reason,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CaseRef         string         `json:"case_ref,omitempty"`
	CaseTitle       string         `json:"case_title,omitempty"`
	IdentifierType  string         `json:"identifier_type,omitempty"`
	IdentifierValue string         `json:"identifier_value,omitempty"`
}

type KarmaAdjustmentDetails struct {
	Amount      int             `json:"amount"`
	Reason      string          `json:"reason"`
	TriggerType string          `json:"trigger_type"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
	Breakdown   *KarmaBreakdown `json:"breakdown,omitempty"`
}

type PhoneDetails struct {
	CountryCode string `json:"country_code,omitempty"`
	Number      string `json:"number,omitempty"`
	AreaCode    string `json:"area_code,omitempty"`
	IsTollFree  *bool  `json:"is_toll_free,omitempty"`
	Region      string `json:"region,omitempty"`
}

type IdentifierEmailDetails struct {
	Email string `json:"email,omitempty"`
}

type URLDetails struct {
	URL string `json:"url,omitempty"`
}

type BankAccountDetails struct {
	AccountNumber string `json:"account_number,omitempty"`
	Routing       string `json:"routing,omitempty"`
	Institution   string `json:"institution,omitempty"`
	Owner         string `json:"owner,omitempty"`
	OwnerAddress  string `json:"owner_address,omitempty"`
	Country       string `json:"country,omitempty"`
	Address       string `json:"address,omitempty"`
	Swift         string `json:"swift,omitempty"`
	IBAN          string `json:"iban,omitempty"`
	AccountType   string `json:"account_type,omitempty"`
}

type CryptoWalletDetails struct {
	Address  string `json:"address,omitempty"`
	Currency string `json:"currency,omitempty"`
	Network  string `json:"network,omitempty"`
}

type SocialMediaDetails struct {
	Platform string `json:"platform,omitempty"`
	Handle   string `json:"handle,omitempty"`
}

type ZelleDetails struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type PaymentTokenDetails struct {
	Service          string `json:"service,omitempty"`
	Identifier       string `json:"identifier,omitempty"`
	Type             string `json:"type,omitempty"`
	Name             string `json:"name,omitempty"`
	SourceURLCreated string `json:"source_url_created,omitempty"`
	SourceURLPrinted string `json:"source_url_printed,omitempty"`
}

// DecodeDetails re-decodes a loosely typed details map into a concrete
// details struct, e.g. DecodeDetails[PhoneCallDetails](entry.Details).
func DecodeDetails[T any](data map[string]any) (T, error) {
	var out T
	if len(data) == 0 {
		return out, nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return out, fmt.Errorf("scambus: encode details: %w", err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("scambus: decode details: %w", err)
	}
	return out, nil
}

// ParseIdentifierDetails decodes Identifier.Data into the struct matching the
// identifier type. Unknown types return the map unchanged.
func ParseIdentifierDetails(identifierType string, data map[string]any) (any, error) {
	switch IdentifierType(identifierType) {
	case IdentifierTypePhone:
		return DecodeDetails[PhoneDetails](data)
	case IdentifierTypeEmail:
		return DecodeDetails[IdentifierEmailDetails](data)
	case IdentifierTypeURL:
		return DecodeDetails[URLDetails](data)
	case IdentifierTypeBankAccount:
		return DecodeDetails[BankAccountDetails](data)
	case IdentifierTypeCryptoWallet:
		return DecodeDetails[CryptoWalletDetails](data)
	case IdentifierTypeSocialMedia:
		return DecodeDetails[SocialMediaDetails](data)
	case IdentifierTypeZelle:
		return DecodeDetails[ZelleDetails](data)
	case IdentifierTypePaymentToken:
		return DecodeDetails[PaymentTokenDetails](data)
	default:
		return data, nil
	}
}
