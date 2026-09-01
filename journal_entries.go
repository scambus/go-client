package scambus

import (
	"context"
	"strings"
	"time"
)

// attachMedia appends one Evidence record per media item, because the API
// accepts at most one media id per record. It never writes through to the
// caller's slice.
func attachMedia(in *CreateEntryInput, media []Media, template Evidence) {
	if len(media) == 0 {
		return
	}
	evidence := make([]Evidence, 0, len(in.Evidence)+len(media))
	evidence = append(evidence, in.Evidence...)
	for _, m := range media {
		record := template
		record.MediaID = m.ID
		evidence = append(evidence, record)
	}
	in.Evidence = evidence
}

func anyMimePrefix(media []Media, prefix string) bool {
	for _, m := range media {
		if strings.HasPrefix(m.MimeType, prefix) {
			return true
		}
	}
	return false
}

func nowOr(t Time) Time {
	if t.IsSet() {
		return t
	}
	return NewTime(time.Now().UTC())
}

type DetectionInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Details                    *DetectionDetails
	PerformedAt                Time
	StartTime                  Time
	EndTime                    Time
	InProgress                 bool
}

func (s *JournalService) CreateDetection(ctx context.Context, in DetectionInput) (*JournalEntry, error) {
	entry := CreateEntryInput{
		Type:        EntryTypeDetection,
		PerformedAt: nowOr(in.PerformedAt),
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		InProgress:  in.InProgress,
	}
	in.applyCommon(&entry)
	if in.Details != nil {
		entry.Details = in.Details
	}

	evidenceType := "file"
	if anyMimePrefix(in.Media, "image/") {
		evidenceType = "screenshot"
	}
	attachMedia(&entry, in.Media, Evidence{
		Type:        evidenceType,
		Description: "Evidence for detection: " + in.Description,
		Source:      "Automated Detection",
		CollectedAt: nowOr(in.PerformedAt),
	})
	return s.Create(ctx, entry)
}

type PhoneCallInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Direction                  string
	StartTime                  Time
	EndTime                    Time
	RecordingURL               string
	TranscriptURL              string
	Transcript                 []ConversationMessage
	InProgress                 bool
	AIExtract                  bool
}

func (s *JournalService) CreatePhoneCall(ctx context.Context, in PhoneCallInput) (*JournalEntry, error) {
	entry := CreateEntryInput{
		Type:        EntryTypePhoneCall,
		PerformedAt: in.StartTime,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		InProgress:  in.InProgress,
		AIExtract:   in.AIExtract || len(in.Transcript) > 0,
		Details: PhoneCallDetails{
			Direction:     in.Direction,
			RecordingURL:  in.RecordingURL,
			TranscriptURL: in.TranscriptURL,
			Transcript:    in.Transcript,
		},
	}
	in.applyCommon(&entry)

	evidenceType := "file"
	if anyMimePrefix(in.Media, "audio/") {
		evidenceType = "recording"
	}
	attachMedia(&entry, in.Media, Evidence{
		Type:        evidenceType,
		Description: "Evidence for phone call: " + in.Description,
		Source:      "Phone Call Recording",
		CollectedAt: in.StartTime,
	})
	return s.Create(ctx, entry)
}

type EmailInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Direction                  string
	Subject                    string
	SentAt                     Time
	Body                       string
	HTMLBody                   string
	MessageID                  string
	Headers                    map[string]string
	Attachments                []string
	StartTime                  Time
	EndTime                    Time
	InProgress                 bool
}

func (s *JournalService) CreateEmail(ctx context.Context, in EmailInput) (*JournalEntry, error) {
	entry := CreateEntryInput{
		Type:        EntryTypeEmail,
		PerformedAt: in.SentAt,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		InProgress:  in.InProgress,
		Details: EmailDetails{
			Direction:   in.Direction,
			Subject:     in.Subject,
			SentAt:      in.SentAt,
			Body:        in.Body,
			HTMLBody:    in.HTMLBody,
			MessageID:   in.MessageID,
			Headers:     in.Headers,
			Attachments: in.Attachments,
		},
	}
	in.applyCommon(&entry)
	attachMedia(&entry, in.Media, Evidence{
		Type:        "screenshot",
		Description: "Evidence for email: " + in.Subject,
		Source:      "Email Communication",
		CollectedAt: in.SentAt,
	})
	return s.Create(ctx, entry)
}

type TextConversationInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Platform                   string
	StartTime                  Time
	EndTime                    Time
	InProgress                 bool
	AIExtract                  bool
	Details                    *TextConversationDetails
}

func (s *JournalService) CreateTextConversation(ctx context.Context, in TextConversationInput) (*JournalEntry, error) {
	details := TextConversationDetails{Platform: in.Platform}
	if in.Details != nil {
		details = *in.Details
		if details.Platform == "" {
			details.Platform = in.Platform
		}
	}

	entry := CreateEntryInput{
		Type:        EntryTypeTextConversation,
		PerformedAt: in.StartTime,
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		InProgress:  in.InProgress,
		AIExtract:   in.AIExtract,
		Details:     details,
	}
	in.applyCommon(&entry)
	attachMedia(&entry, in.Media, Evidence{
		Type:        "screenshot",
		Description: "Evidence for " + in.Platform + " conversation: " + in.Description,
		Source:      in.Platform + " Communication",
		CollectedAt: in.StartTime,
	})
	return s.Create(ctx, entry)
}

type ConversationContinuationInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	ParentEntryID              string
	Messages                   []ConversationMessage
	Reason                     string
	NonContiguous              bool
	AIExtract                  bool
}

// CreateConversationContinuation derives start and end times from the messages.
func (s *JournalService) CreateConversationContinuation(ctx context.Context, in ConversationContinuationInput) (*JournalEntry, error) {
	if len(in.Messages) == 0 {
		return nil, errNoMessages
	}

	start, end := in.Messages[0].Timestamp, in.Messages[0].Timestamp
	for _, m := range in.Messages[1:] {
		if m.Timestamp.Before(start.Time) {
			start = m.Timestamp
		}
		if m.Timestamp.After(end.Time) {
			end = m.Timestamp
		}
	}

	description := in.Description
	if description == "" {
		description = "Conversation continuation"
	}

	entry := CreateEntryInput{
		Type:        EntryTypeConversationContinuation,
		PerformedAt: start,
		StartTime:   start,
		EndTime:     end,
		AIExtract:   in.AIExtract,
		Details: ConversationContinuationDetails{
			Messages:      in.Messages,
			Reason:        in.Reason,
			NonContiguous: in.NonContiguous,
		},
	}
	in.applyCommon(&entry)
	entry.Description = description
	entry.ParentJournalEntryID = in.ParentEntryID

	attachMedia(&entry, in.Media, Evidence{
		Type:        "screenshot",
		Description: "Evidence for continuation: " + description,
		Source:      "Conversation Messages",
		CollectedAt: start,
	})
	return s.Create(ctx, entry)
}

type NoteInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Details                    *NoteDetails
	PerformedAt                Time
	StartTime                  Time
	EndTime                    Time
	InProgress                 bool
}

func (s *JournalService) CreateNote(ctx context.Context, in NoteInput) (*JournalEntry, error) {
	entry := CreateEntryInput{
		Type:        EntryTypeNote,
		PerformedAt: nowOr(in.PerformedAt),
		StartTime:   in.StartTime,
		EndTime:     in.EndTime,
		InProgress:  in.InProgress,
	}
	in.applyCommon(&entry)
	if in.Details != nil {
		entry.Details = in.Details
	}
	attachMedia(&entry, in.Media, Evidence{
		Type:        "document",
		Description: "Evidence for note: " + in.Description,
		Source:      "Note Attachment",
		CollectedAt: nowOr(in.PerformedAt),
	})
	return s.Create(ctx, entry)
}

type ImportInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Details                    *ImportDetails
	PerformedAt                Time
}

func (s *JournalService) CreateImport(ctx context.Context, in ImportInput) (*JournalEntry, error) {
	entry := CreateEntryInput{Type: EntryTypeImport, PerformedAt: nowOr(in.PerformedAt)}
	in.applyCommon(&entry)
	if in.Details != nil {
		entry.Details = in.Details
	}
	return s.Create(ctx, entry)
}

type ExportInput struct {
	Description                string
	Identifiers                []IdentifierLookup
	OurIdentifiers             []IdentifierLookup
	Evidence                   []Evidence
	Media                      []Media
	CaseID                     string
	Tags                       []TagLookup
	Metadata                   map[string]any
	ParentJournalEntryID       string
	Originator                 *OriginatorLookup
	IsTest                     bool
	ExternalIdentifiers        []ExternalIdentifierInput
	ExtractExternalIdentifiers bool
	Details                    *ExportDetails
	PerformedAt                Time
}

func (s *JournalService) CreateExport(ctx context.Context, in ExportInput) (*JournalEntry, error) {
	entry := CreateEntryInput{Type: EntryTypeExport, PerformedAt: nowOr(in.PerformedAt)}
	in.applyCommon(&entry)
	if in.Details != nil {
		entry.Details = in.Details
	}
	return s.Create(ctx, entry)
}

func (in DetectionInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in PhoneCallInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in EmailInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in TextConversationInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in ConversationContinuationInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in NoteInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in ImportInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}

func (in ExportInput) applyCommon(entry *CreateEntryInput) {
	entry.Description = in.Description
	entry.IdentifierLookups = in.Identifiers
	entry.OurIdentifierLookups = in.OurIdentifiers
	entry.Evidence = in.Evidence
	entry.CaseID = in.CaseID
	entry.TagLookups = in.Tags
	entry.Metadata = in.Metadata
	entry.ParentJournalEntryID = in.ParentJournalEntryID
	entry.Originator = in.Originator
	entry.IsTest = in.IsTest
	entry.ExternalIdentifiers = in.ExternalIdentifiers
	entry.ExtractExternalIdentifiers = in.ExtractExternalIdentifiers
}
