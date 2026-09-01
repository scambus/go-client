package scambus

import "encoding/json"

type Identifier struct {
	ID             string           `json:"id"`
	Type           string           `json:"type"`
	DisplayValue   string           `json:"display_value"`
	Confidence     Confidence       `json:"confidence"`
	Data           map[string]any   `json:"data,omitempty"`
	CreatedAt      Time             `json:"created_at"`
	UpdatedAt      Time             `json:"updated_at"`
	IsTest         bool             `json:"is_test"`
	TagDisplay     []map[string]any `json:"tag_display,omitempty"`
	Label          string           `json:"label,omitempty"`
	Classification string           `json:"classification,omitempty"`
}

type IdentifierLookup struct {
	Type        string                    `json:"type"`
	Value       string                    `json:"value"`
	Confidence  *float64                  `json:"confidence,omitempty"`
	Label       string                    `json:"label,omitempty"`
	Ref         string                    `json:"ref,omitempty"`
	Enrichments map[string]EnrichedDetail `json:"enrichments,omitempty"`
}

type EnrichedDetail struct {
	Value  any    `json:"value"`
	Source string `json:"source"`
}

type Evidence struct {
	ID          string   `json:"id,omitempty"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Source      string   `json:"source,omitempty"`
	CollectedAt Time     `json:"collected_at,omitzero"`
	MediaIDs    []string `json:"media_ids,omitempty"`
	CreatedAt   Time     `json:"created_at,omitzero"`
	UpdatedAt   Time     `json:"updated_at,omitzero"`
	IsTest      bool     `json:"is_test,omitempty"`
}

type Media struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	FileName       string `json:"file_name"`
	MimeType       string `json:"mime_type"`
	FileSize       int64  `json:"file_size"`
	Notes          string `json:"notes,omitempty"`
	UploadedAt     Time   `json:"uploaded_at"`
	JournalEntryID string `json:"journal_entry_id,omitempty"`
}

type JournalEntry struct {
	ID                   string                     `json:"id"`
	Type                 string                     `json:"type"`
	Description          string                     `json:"description"`
	Details              map[string]any             `json:"details,omitempty"`
	PerformedAt          Time                       `json:"performed_at"`
	CreatedAt            Time                       `json:"created_at"`
	UpdatedAt            Time                       `json:"updated_at"`
	Identifiers          []Identifier               `json:"identifiers,omitempty"`
	OurIdentifiers       []Identifier               `json:"our_identifiers,omitempty"`
	Evidence             []Evidence                 `json:"evidence,omitempty"`
	CaseID               string                     `json:"case_id,omitempty"`
	StartTime            Time                       `json:"start_time"`
	EndTime              Time                       `json:"end_time"`
	ParentJournalEntryID string                     `json:"parent_journal_entry_id,omitempty"`
	BatchID              string                     `json:"batch_id,omitempty"`
	Tags                 []map[string]any           `json:"tags,omitempty"`
	EffectiveTags        []map[string]any           `json:"effective_tags,omitempty"`
	TotalKarma           *int                       `json:"total_karma,omitempty"`
	KarmaBreakdown       *KarmaBreakdown            `json:"karma_breakdown,omitempty"`
	IsDraft              bool                       `json:"is_draft"`
	IsTest               bool                       `json:"is_test"`
	DraftMetadata        map[string]any             `json:"draft_metadata,omitempty"`
	Signature            string                     `json:"signature,omitempty"`
	SignedBy             string                     `json:"signed_by,omitempty"`
	SignatureAlgorithm   string                     `json:"signature_algorithm,omitempty"`
	SignedAt             Time                       `json:"signed_at"`
	ChildEntries         []JournalEntry             `json:"child_entries,omitempty"`
	FailedIdentifiers    []FailedIdentifier         `json:"failed_identifiers,omitempty"`
	ExtractedIdentifiers []ExtractedIdentifier      `json:"extracted_identifiers,omitempty"`
	ExternalIdentifiers  []ExternalIdentifierRecord `json:"external_identifiers,omitempty"`
}

type JournalEntryChildSummary struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Platform             string `json:"platform,omitempty"`
	Direction            string `json:"direction,omitempty"`
	ParentJournalEntryID string `json:"parent_journal_entry_id,omitempty"`
	PerformedAt          Time   `json:"performed_at"`
}

type FailedIdentifier struct {
	Type   string `json:"type"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

type ExtractedIdentifier struct {
	Ref          string                          `json:"ref"`
	IdentifierID string                          `json:"identifier_id"`
	Type         string                          `json:"type"`
	Value        string                          `json:"value"`
	Label        string                          `json:"label,omitempty"`
	Confidence   *float64                        `json:"confidence,omitempty"`
	Occurrences  []ExtractedIdentifierOccurrence `json:"occurrences,omitempty"`
}

type ExtractedIdentifierOccurrence struct {
	MessageIndex int    `json:"message_index"`
	Field        string `json:"field"`
	Position     int    `json:"position"`
	Length       int    `json:"length"`
}

type IdentifierSummary struct {
	JournalEntryID string                `json:"journal_entry_id"`
	Total          int                   `json:"total"`
	ByType         []IdentifierTypeCount `json:"by_type"`
}

type IdentifierTypeCount struct {
	Type      string                   `json:"type"`
	Count     int                      `json:"count"`
	BySubtype []IdentifierSubtypeCount `json:"by_subtype,omitempty"`
}

type IdentifierSubtypeCount struct {
	Subtype string `json:"subtype"`
	Count   int    `json:"count"`
}

type ExternalIdentifierRecord struct {
	ID             string `json:"id"`
	JournalEntryID string `json:"journal_entry_id"`
	ExternalSystem string `json:"external_system"`
	ExternalID     string `json:"external_id"`
	Source         string `json:"source"`
	RawMatch       string `json:"raw_match,omitempty"`
	Link           string `json:"link,omitempty"`
	CreatedAt      Time   `json:"created_at"`
}

type ExternalSystem struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Pattern     string `json:"pattern,omitempty"`
	LinkFormat  string `json:"link_format,omitempty"`
}

type BatchCreateResult struct {
	Results   []BatchEntryResult `json:"results"`
	Total     int                `json:"total"`
	Succeeded int                `json:"succeeded"`
	Failed    int                `json:"failed"`
}

type BatchEntryResult struct {
	Index                int                   `json:"index"`
	Status               string                `json:"status"`
	ID                   string                `json:"id,omitempty"`
	Error                string                `json:"error,omitempty"`
	FailedIdentifiers    []FailedIdentifier    `json:"failed_identifiers,omitempty"`
	ExtractedIdentifiers []ExtractedIdentifier `json:"extracted_identifiers,omitempty"`
}

type KarmaComponent struct {
	Type        string `json:"type"`
	Amount      int    `json:"amount"`
	Description string `json:"description"`
	ConfigID    string `json:"config_id,omitempty"`
}

type KarmaBreakdown struct {
	Components []KarmaComponent `json:"components"`
	Total      int              `json:"total"`
}

type Case struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Notes     string `json:"notes,omitempty"`
	Status    string `json:"status,omitempty"`
	Priority  string `json:"priority,omitempty"`
	CreatedAt Time   `json:"created_at"`
	UpdatedAt Time   `json:"updated_at"`
	CreatedBy string `json:"created_by,omitempty"`
	IsTest    bool   `json:"is_test"`
}

type CaseComment struct {
	ID              string `json:"id"`
	CaseID          string `json:"case_id"`
	AuthorID        string `json:"author_id"`
	Content         string `json:"content"`
	IsReaction      bool   `json:"is_reaction"`
	Reaction        string `json:"reaction,omitempty"`
	Edited          bool   `json:"edited"`
	Deleted         bool   `json:"deleted"`
	ParentCommentID string `json:"parent_comment_id,omitempty"`
	CreatedAt       Time   `json:"created_at"`
	UpdatedAt       Time   `json:"updated_at"`
	DeletedAt       Time   `json:"deleted_at"`
}

type Tag struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	TagType            string     `json:"tag_type"`
	Description        string     `json:"description,omitempty"`
	Aliases            []string   `json:"aliases,omitempty"`
	ApplicableModels   []string   `json:"applicable_models,omitempty"`
	Color              string     `json:"color,omitempty"`
	Icon               string     `json:"icon,omitempty"`
	Active             bool       `json:"active"`
	IsSystem           bool       `json:"is_system"`
	IsGlobal           bool       `json:"is_global"`
	FlowUp             bool       `json:"flow_up"`
	FlowDown           bool       `json:"flow_down"`
	AllowDynamicValues bool       `json:"allow_dynamic_values"`
	AllocatesKarma     *int       `json:"allocates_karma,omitempty"`
	OwnerOrgID         string     `json:"owner_org_id,omitempty"`
	TagValues          []TagValue `json:"tag_values,omitempty"`
	CreatedAt          Time       `json:"created_at"`
	UpdatedAt          Time       `json:"updated_at"`
}

type TagValue struct {
	ID          string `json:"id"`
	TagID       string `json:"tag_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order"`
	Active      bool   `json:"active"`
	CreatedAt   Time   `json:"created_at"`
	UpdatedAt   Time   `json:"updated_at"`
}

type Notification struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	Timestamp        Time   `json:"timestamp"`
	NotificationText string `json:"notification_text"`
	Service          string `json:"service"`
	Read             bool   `json:"read"`
	Dismissed        bool   `json:"dismissed"`
	Link             string `json:"link,omitempty"`
	Icon             string `json:"icon,omitempty"`
	Severity         string `json:"severity"`
	EntityType       string `json:"entity_type,omitempty"`
	EntityID         string `json:"entity_id,omitempty"`
	CreatedAt        Time   `json:"created_at"`
	UpdatedAt        Time   `json:"updated_at"`
}

type Session struct {
	ID           string `json:"id"`
	JTI          string `json:"jti"`
	UserID       string `json:"user_id"`
	UserType     string `json:"user_type"`
	ClerkUserID  string `json:"clerk_user_id"`
	ExpiresAt    Time   `json:"expires_at"`
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	RevokedAt    Time   `json:"revoked_at"`
	RevokedBy    string `json:"revoked_by,omitempty"`
	RevokeReason string `json:"revoke_reason,omitempty"`
	CreatedAt    Time   `json:"created_at"`
}

type Passkey struct {
	ID             string   `json:"id"`
	UserID         string   `json:"user_id"`
	Name           string   `json:"name"`
	SignCount      int      `json:"sign_count"`
	Transports     []string `json:"transports,omitempty"`
	BackupEligible bool     `json:"backup_eligible"`
	BackupState    bool     `json:"backup_state"`
	CreatedAt      Time     `json:"created_at"`
	UpdatedAt      Time     `json:"updated_at"`
	LastUsedAt     Time     `json:"last_used_at"`
}

type TwoFactorStatus struct {
	Enabled      bool `json:"enabled"`
	PasskeyCount int  `json:"passkey_count,omitempty"`
}

type View struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	EntityType     string         `json:"entity_type"`
	Visibility     string         `json:"visibility"`
	ViewType       string         `json:"view_type"`
	Description    string         `json:"description,omitempty"`
	Alias          string         `json:"alias,omitempty"`
	FilterCriteria map[string]any `json:"filter_criteria,omitempty"`
	SortOrder      *SortOrder     `json:"sort_order,omitempty"`
	IsSystem       bool           `json:"is_system"`
	CreatedAt      Time           `json:"created_at"`
	UpdatedAt      Time           `json:"updated_at"`
	CreatedBy      string         `json:"created_by,omitempty"`
	OrganizationID string         `json:"organization_id,omitempty"`
}

type Persona struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	Personality       string                  `json:"personality"`
	Background        string                  `json:"background"`
	AddressLine1      string                  `json:"address_line1,omitempty"`
	AddressLine2      string                  `json:"address_line2,omitempty"`
	AddressCity       string                  `json:"address_city"`
	AddressState      string                  `json:"address_state"`
	AddressPostalCode string                  `json:"address_postal_code"`
	AddressCountry    string                  `json:"address_country"`
	OwnerOrgID        string                  `json:"owner_org_id,omitempty"`
	CreatedBy         string                  `json:"created_by,omitempty"`
	IsActive          bool                    `json:"is_active"`
	IsTest            bool                    `json:"is_test"`
	CreatedAt         Time                    `json:"created_at"`
	UpdatedAt         Time                    `json:"updated_at"`
	Identifiers       []PersonaIdentifierLink `json:"identifiers,omitempty"`
	Media             []PersonaMediaLink      `json:"media,omitempty"`
}

type PersonaIdentifierLink struct {
	PersonaID       string `json:"persona_id"`
	IdentifierID    string `json:"identifier_id"`
	Annotation      string `json:"annotation"`
	IdentifierValue string `json:"identifier_value,omitempty"`
	IdentifierType  string `json:"identifier_type,omitempty"`
	CreatedAt       Time   `json:"created_at"`
}

type PersonaMediaLink struct {
	PersonaID string `json:"persona_id"`
	MediaID   string `json:"media_id"`
	Category  string `json:"category"`
	Notes     string `json:"notes"`
	FileName  string `json:"file_name,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	FileSize  int64  `json:"file_size,omitempty"`
	CreatedAt Time   `json:"created_at"`
}

type Report struct {
	ID                string `json:"report_id"`
	ReportType        string `json:"report_type"`
	Status            string `json:"status"`
	IdentifierCount   int    `json:"identifier_count"`
	JournalEntryCount int    `json:"journal_entry_count"`
	EvidenceCount     int    `json:"evidence_count"`
	DownloadURL       string `json:"download_url,omitempty"`
	GeneratedAt       Time   `json:"generated_at"`
	ExpiresAt         Time   `json:"expires_at"`
	CreatedAt         Time   `json:"created_at"`
	ErrorMessage      string `json:"error_message,omitempty"`
}

func (r Report) IsCompleted() bool { return r.Status == "completed" }

func (r Report) IsFailed() bool { return r.Status == "failed" }

func (r Report) IsProcessing() bool { return r.Status == "pending" || r.Status == "processing" }

// UnmarshalJSON fills ID and ReportType from whichever key the endpoint used.
func (r *Report) UnmarshalJSON(data []byte) error {
	type alias Report
	var raw struct {
		alias
		AltID   string `json:"id"`
		AltType string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = Report(raw.alias)
	if r.ID == "" {
		r.ID = raw.AltID
	}
	if r.ReportType == "" {
		r.ReportType = raw.AltType
	}
	return nil
}

type Automation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
	OwnerOrgID  string `json:"owner_org_id,omitempty"`
	CreatedBy   string `json:"created_by,omitempty"`
	CreatedAt   Time   `json:"created_at"`
	UpdatedAt   Time   `json:"updated_at"`
}

type AutomationAPIKey struct {
	ID           string `json:"id"`
	AutomationID string `json:"automation_id"`
	Name         string `json:"name"`
	KeyID        string `json:"key_id,omitempty"`
	Secret       string `json:"secret,omitempty"`
	Prefix       string `json:"prefix,omitempty"`
	ExpiresAt    Time   `json:"expires_at"`
	RevokedAt    Time   `json:"revoked_at"`
	LastUsedAt   Time   `json:"last_used_at"`
	CreatedAt    Time   `json:"created_at"`
}

type IdentifierExclusion struct {
	ID             string `json:"id"`
	OwnerOrgID     string `json:"owner_org_id"`
	IdentifierType string `json:"identifier_type"`
	UniquenessKey  string `json:"uniqueness_key"`
	DisplayValue   string `json:"display_value"`
	Reason         string `json:"reason,omitempty"`
	CreatedByID    string `json:"created_by_id,omitempty"`
	CreatedAt      Time   `json:"created_at"`
}

type IdentifierURLReference struct {
	ID           string `json:"id"`
	IdentifierID string `json:"identifier_id"`
	URL          string `json:"url"`
	Hostname     string `json:"hostname"`
	Path         string `json:"path"`
	FirstSeenAt  Time   `json:"first_seen_at"`
	LastSeenAt   Time   `json:"last_seen_at"`
	SeenCount    int    `json:"seen_count"`
	OwnerOrgID   string `json:"owner_org_id,omitempty"`
	IsTest       bool   `json:"is_test"`
	CreatedAt    Time   `json:"created_at"`
}

type SpecialDomainRule struct {
	ID            string `json:"id"`
	Domain        string `json:"domain"`
	Category      string `json:"category"`
	PathDepth     int    `json:"path_depth"`
	StripQuery    bool   `json:"strip_query"`
	StripFragment bool   `json:"strip_fragment"`
	IsActive      bool   `json:"is_active"`
	IsDefault     bool   `json:"is_default"`
	CreatedAt     Time   `json:"created_at"`
	UpdatedAt     Time   `json:"updated_at"`
}

type URLConsolidationStatus struct {
	Status          string `json:"status"`
	StartedAt       Time   `json:"started_at"`
	CompletedAt     Time   `json:"completed_at"`
	TotalGroups     int    `json:"total_groups"`
	ProcessedGroups int    `json:"processed_groups"`
	Merged          int    `json:"merged"`
	Skipped         int    `json:"skipped"`
	Errors          int    `json:"errors"`
	LastError       string `json:"last_error,omitempty"`
}

type FileExport struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	SourceType   string         `json:"source_type"`
	SourceID     string         `json:"source_id,omitempty"`
	EntityType   string         `json:"entity_type"`
	Format       string         `json:"format"`
	Status       string         `json:"status"`
	RowCount     int            `json:"row_count"`
	FileSize     int64          `json:"file_size"`
	DownloadURL  string         `json:"download_url,omitempty"`
	Options      map[string]any `json:"format_options,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	ExpiresAt    Time           `json:"expires_at"`
	CreatedAt    Time           `json:"created_at"`
	UpdatedAt    Time           `json:"updated_at"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type Queue struct {
	ID                    string         `json:"id"`
	Name                  string         `json:"name"`
	Description           string         `json:"description"`
	FilterCriteria        map[string]any `json:"filter_criteria,omitempty"`
	CadenceDays           int            `json:"cadence_days"`
	CooldownHours         int            `json:"cooldown_hours"`
	MaxContactsPerCluster *int           `json:"max_contacts_per_cluster,omitempty"`
	RotationEnabled       bool           `json:"rotation_enabled"`
	PriorityMode          string         `json:"priority_mode"`
	AutoPopulate          bool           `json:"auto_populate"`
	ActorClusterID        string         `json:"actor_cluster_id,omitempty"`
	ActorClusterName      string         `json:"actor_cluster_name,omitempty"`
	RedisStreamKey        string         `json:"redis_stream_key,omitempty"`
	StreamVersion         int            `json:"stream_version"`
	OwnerOrgID            string         `json:"owner_org_id,omitempty"`
	CreatedBy             string         `json:"created_by,omitempty"`
	IsActive              bool           `json:"is_active"`
	IsTest                bool           `json:"is_test"`
	CreatedAt             Time           `json:"created_at"`
	UpdatedAt             Time           `json:"updated_at"`
}

type QueueItem struct {
	ID                  string         `json:"id"`
	QueueID             string         `json:"queue_id"`
	ClusterID           string         `json:"cluster_id"`
	RepresentativeID    string         `json:"representative_id"`
	State               string         `json:"state"`
	FunnelID            string         `json:"funnel_id,omitempty"`
	FunnelEntryID       string         `json:"funnel_entry_id,omitempty"`
	FunnelStageID       string         `json:"funnel_stage_id,omitempty"`
	PersonaID           string         `json:"persona_id,omitempty"`
	ActorClusterID      string         `json:"actor_cluster_id,omitempty"`
	SelectedChannel     string         `json:"selected_channel,omitempty"`
	JourneyState        map[string]any `json:"journey_state,omitempty"`
	Provenance          string         `json:"provenance"`
	ClaimedBy           string         `json:"claimed_by,omitempty"`
	ClaimedAt           Time           `json:"claimed_at"`
	LastContactedAt     Time           `json:"last_contacted_at"`
	LastContactedBy     string         `json:"last_contacted_by,omitempty"`
	ContactCount        int            `json:"contact_count"`
	NextContactAfter    Time           `json:"next_contact_after"`
	Priority            int            `json:"priority"`
	OwnerOrgID          string         `json:"owner_org_id,omitempty"`
	IsTest              bool           `json:"is_test"`
	CreatedAt           Time           `json:"created_at"`
	UpdatedAt           Time           `json:"updated_at"`
	RepresentativeValue string         `json:"representative_value,omitempty"`
	RepresentativeType  string         `json:"representative_type,omitempty"`
	ClusterSize         *int           `json:"cluster_size,omitempty"`
}

type QueueStats struct {
	TotalItems int `json:"total_items"`
	Pending    int `json:"pending"`
	Claimed    int `json:"claimed"`
	InProgress int `json:"in_progress"`
	Contacted  int `json:"contacted"`
	Cooldown   int `json:"cooldown"`
	Ready      int `json:"ready"`
	Completed  int `json:"completed"`
	Dropped    int `json:"dropped"`
}

type QueueStreamResponse struct {
	StreamKey     string               `json:"stream_key"`
	Cursor        string               `json:"cursor"`
	Messages      []QueueStreamMessage `json:"messages"`
	ClaimEndpoint string               `json:"claim_endpoint,omitempty"`
	SourceOfTruth string               `json:"source_of_truth"`
}

type QueueStreamMessage struct {
	Event               string         `json:"event"`
	QueueID             string         `json:"queue_id"`
	QueueItemID         string         `json:"queue_item_id"`
	ClusterID           string         `json:"cluster_id"`
	RepresentativeID    string         `json:"representative_id"`
	State               string         `json:"state"`
	ContactCount        int            `json:"contact_count"`
	Priority            int            `json:"priority"`
	StreamVersion       int            `json:"stream_version"`
	OccurredAt          Time           `json:"occurred_at"`
	IsTest              bool           `json:"is_test"`
	Cursor              string         `json:"cursor,omitempty"`
	PreviousQueueID     string         `json:"previous_queue_id,omitempty"`
	TargetQueueID       string         `json:"target_queue_id,omitempty"`
	RepresentativeType  string         `json:"representative_type,omitempty"`
	RepresentativeValue string         `json:"representative_value,omitempty"`
	FunnelID            string         `json:"funnel_id,omitempty"`
	FunnelEntryID       string         `json:"funnel_entry_id,omitempty"`
	FunnelStageID       string         `json:"funnel_stage_id,omitempty"`
	PersonaID           string         `json:"persona_id,omitempty"`
	ActorClusterID      string         `json:"actor_cluster_id,omitempty"`
	SelectedChannel     string         `json:"selected_channel,omitempty"`
	JourneyState        map[string]any `json:"journey_state,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type QueueContactLog struct {
	ID                  string `json:"id"`
	QueueItemID         string `json:"queue_item_id"`
	QueueID             string `json:"queue_id"`
	ClusterID           string `json:"cluster_id"`
	ContactedBy         string `json:"contacted_by"`
	Outcome             string `json:"outcome"`
	ContactedAt         Time   `json:"contacted_at"`
	Notes               string `json:"notes"`
	FunnelID            string `json:"funnel_id,omitempty"`
	FunnelEntryID       string `json:"funnel_entry_id,omitempty"`
	FunnelStageID       string `json:"funnel_stage_id,omitempty"`
	PersonaID           string `json:"persona_id,omitempty"`
	ActorClusterID      string `json:"actor_cluster_id,omitempty"`
	ContactIdentifierID string `json:"contact_identifier_id,omitempty"`
	JournalEntryID      string `json:"journal_entry_id,omitempty"`
	Channel             string `json:"channel,omitempty"`
}

type QueueItemEvent struct {
	ID                string         `json:"id"`
	QueueItemID       string         `json:"queue_item_id"`
	QueueID           string         `json:"queue_id"`
	ClusterID         string         `json:"cluster_id"`
	RepresentativeID  string         `json:"representative_id"`
	Event             string         `json:"event"`
	State             string         `json:"state"`
	ContactCount      int            `json:"contact_count"`
	Priority          int            `json:"priority"`
	StreamVersion     int            `json:"stream_version"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	IsTest            bool           `json:"is_test"`
	OccurredAt        Time           `json:"occurred_at"`
	PreviousQueueID   string         `json:"previous_queue_id,omitempty"`
	TargetQueueID     string         `json:"target_queue_id,omitempty"`
	FunnelID          string         `json:"funnel_id,omitempty"`
	FunnelEntryID     string         `json:"funnel_entry_id,omitempty"`
	FunnelStageID     string         `json:"funnel_stage_id,omitempty"`
	PersonaID         string         `json:"persona_id,omitempty"`
	ActorClusterID    string         `json:"actor_cluster_id,omitempty"`
	SelectedChannel   string         `json:"selected_channel,omitempty"`
	OwnerOrgID        string         `json:"owner_org_id,omitempty"`
	QueueName         string         `json:"queue_name,omitempty"`
	PreviousQueueName string         `json:"previous_queue_name,omitempty"`
	TargetQueueName   string         `json:"target_queue_name,omitempty"`
}

type QueueClusterIdentifier struct {
	ID         string   `json:"id"`
	Value      string   `json:"value"`
	Type       string   `json:"type"`
	IsOurs     bool     `json:"is_ours"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type ExportStream struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	DataType         string         `json:"data_type"`
	FilterCriteria   map[string]any `json:"filter_criteria,omitempty"`
	IdentifierTypes  []string       `json:"identifier_types,omitempty"`
	MinConfidence    float64        `json:"min_confidence"`
	MaxConfidence    float64        `json:"max_confidence"`
	IsActive         bool           `json:"is_active"`
	ConsumerKey      string         `json:"consumer_key,omitempty"`
	RetentionDays    int            `json:"retention_days"`
	FilterExpression string         `json:"filter_expression,omitempty"`
	CreatedAt        Time           `json:"created_at"`
	UpdatedAt        Time           `json:"updated_at"`
}

type StreamTagInfo struct {
	TagID    string `json:"tag_id"`
	TagTitle string `json:"tag_title"`
	Value    any    `json:"value,omitempty"`
	ValueID  string `json:"value_id,omitempty"`
}

type StreamEvidenceInfo struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Description    string   `json:"description,omitempty"`
	Source         string   `json:"source,omitempty"`
	CollectedAt    Time     `json:"collected_at"`
	MediaIDs       []string `json:"media_ids,omitempty"`
	OriginatorID   string   `json:"originator_id,omitempty"`
	OriginatorType string   `json:"originator_type,omitempty"`
}

type StreamJournalEntryInfo struct {
	ID             string               `json:"id"`
	Type           string               `json:"type"`
	Description    string               `json:"description"`
	PerformedAt    Time                 `json:"performed_at"`
	OriginatorID   string               `json:"originator_id,omitempty"`
	OriginatorType string               `json:"originator_type,omitempty"`
	Evidence       []StreamEvidenceInfo `json:"evidence,omitempty"`
}

type StreamOriginatorInfo struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type StreamIdentifierInfo struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	DisplayValue string         `json:"display_value"`
	Confidence   Confidence     `json:"confidence"`
	IsOurs       bool           `json:"is_ours"`
	Label        string         `json:"label,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
	CreatedAt    Time           `json:"created_at"`
	UpdatedAt    Time           `json:"updated_at"`
}

type IdentifierStreamMessage struct {
	MessageID              string                   `json:"message_id,omitempty"`
	Cursor                 string                   `json:"cursor,omitempty"`
	IdentifierID           string                   `json:"identifier_id"`
	Type                   string                   `json:"type"`
	DisplayValue           string                   `json:"display_value"`
	Details                map[string]any           `json:"details,omitempty"`
	Confidence             Confidence               `json:"confidence"`
	ModifiedAt             Time                     `json:"modified_at"`
	OriginatorID           string                   `json:"originator_id,omitempty"`
	Tags                   []StreamTagInfo          `json:"tags,omitempty"`
	TriggeringJournalEntry *StreamJournalEntryInfo  `json:"triggering_journal_entry,omitempty"`
	JournalEntries         []StreamJournalEntryInfo `json:"journal_entries,omitempty"`
	IsTest                 bool                     `json:"is_test"`
}

type JournalEntryStreamMessage struct {
	MessageID            string                 `json:"message_id,omitempty"`
	Cursor               string                 `json:"cursor,omitempty"`
	ID                   string                 `json:"id"`
	Type                 string                 `json:"type"`
	Description          string                 `json:"description"`
	Details              map[string]any         `json:"details,omitempty"`
	PerformedAt          Time                   `json:"performed_at"`
	Confidence           Confidence             `json:"confidence"`
	StartTime            Time                   `json:"start_time"`
	EndTime              Time                   `json:"end_time"`
	ParentJournalEntryID string                 `json:"parent_journal_entry_id,omitempty"`
	Originator           *StreamOriginatorInfo  `json:"originator,omitempty"`
	Identifiers          []StreamIdentifierInfo `json:"identifiers,omitempty"`
	Evidence             []StreamEvidenceInfo   `json:"evidence,omitempty"`
	IsTest               bool                   `json:"is_test"`
	LockedBy             string                 `json:"locked_by,omitempty"`
	LockedByName         string                 `json:"locked_by_name,omitempty"`
	LockedAt             Time                   `json:"locked_at"`
}
