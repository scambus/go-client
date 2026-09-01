package scambus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ReportService struct{ client *Client }

type DateRange struct {
	Start Time `json:"start,omitzero"`
	End   Time `json:"end,omitzero"`
}

type IdentifierReportInput struct {
	IdentifierIDs         []string   `json:"identifier_ids,omitempty"`
	ViewID                string     `json:"view_id,omitempty"`
	IncludeJournalEntries bool       `json:"include_journal_entries"`
	IncludeEvidence       bool       `json:"include_evidence"`
	SignReport            bool       `json:"sign_report"`
	DateRange             *DateRange `json:"date_range,omitempty"`
}

func (s *ReportService) GenerateIdentifierReport(ctx context.Context, in IdentifierReportInput) (*Report, error) {
	var out Report
	if err := s.client.post(ctx, "/reports/identifiers", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type JournalEntryReportInput struct {
	JournalEntryIDs    []string   `json:"journal_entry_ids,omitempty"`
	ViewID             string     `json:"view_id,omitempty"`
	IncludeIdentifiers bool       `json:"include_identifiers"`
	IncludeEvidence    bool       `json:"include_evidence"`
	IncludeParentChain bool       `json:"include_parent_chain"`
	SignReport         bool       `json:"sign_report"`
	DateRange          *DateRange `json:"date_range,omitempty"`
}

func (s *ReportService) GenerateJournalEntryReport(ctx context.Context, in JournalEntryReportInput) (*Report, error) {
	var out Report
	if err := s.client.post(ctx, "/reports/journal-entries", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GenerateViewReport picks the report kind from the view's entity type.
func (s *ReportService) GenerateViewReport(ctx context.Context, viewID string, includeEvidence, signReport bool) (*Report, error) {
	view, err := s.client.Views.Get(ctx, viewID)
	if err != nil {
		return nil, err
	}
	switch view.EntityType {
	case "identifier", "identifiers":
		return s.GenerateIdentifierReport(ctx, IdentifierReportInput{
			ViewID:                viewID,
			IncludeJournalEntries: true,
			IncludeEvidence:       includeEvidence,
			SignReport:            signReport,
		})
	case "journal", "journal_entry", "journal_entries":
		return s.GenerateJournalEntryReport(ctx, JournalEntryReportInput{
			ViewID:             viewID,
			IncludeIdentifiers: true,
			IncludeEvidence:    includeEvidence,
			SignReport:         signReport,
		})
	default:
		return nil, fmt.Errorf("%w: reports support identifier and journal views, not %q", ErrValidation, view.EntityType)
	}
}

func (s *ReportService) Status(ctx context.Context, reportID string) (*Report, error) {
	var out Report
	if err := s.client.get(ctx, "/reports/"+reportID+"/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Wait polls until the report leaves the processing state or ctx expires.
func (s *ReportService) Wait(ctx context.Context, reportID string, pollInterval time.Duration) (*Report, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	for {
		report, err := s.Status(ctx, reportID)
		if err != nil {
			return nil, err
		}
		if !report.IsProcessing() {
			return report, nil
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *ReportService) Download(ctx context.Context, reportID string, w io.Writer) (int64, error) {
	resp, err := s.client.do(ctx, request{method: http.MethodGet, endpoint: "/reports/" + reportID + "/download"})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return io.Copy(w, resp.Body)
}

func (s *ReportService) DownloadToFile(ctx context.Context, reportID, path string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := s.Download(ctx, reportID, f); err != nil {
		return err
	}
	return f.Close()
}
