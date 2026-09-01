package scambus

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type FileExportService struct{ client *Client }

type CreateFileExportInput struct {
	SourceType     string          `json:"source_type"`
	SourceID       string          `json:"source_id,omitempty"`
	EntityType     string          `json:"entity_type"`
	Format         string          `json:"format"`
	Name           string          `json:"name,omitempty"`
	FilterCriteria *FilterCriteria `json:"filter_criteria,omitempty"`
	Columns        []string        `json:"columns,omitempty"`
	Limit          *int            `json:"limit,omitempty"`
	DateRangeStart string          `json:"date_range_start,omitempty"`
	DateRangeEnd   string          `json:"date_range_end,omitempty"`
	IncludeOurs    bool            `json:"include_ours,omitempty"`
	FormatOptions  map[string]any  `json:"format_options,omitempty"`
}

func (s *FileExportService) Create(ctx context.Context, in CreateFileExportInput) (*FileExport, error) {
	if in.Format == "" {
		in.Format = "csv"
	}
	var out FileExport
	if err := s.client.post(ctx, "/file-exports", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *FileExportService) List(ctx context.Context) ([]FileExport, error) {
	var out []FileExport
	if err := s.client.get(ctx, "/file-exports", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *FileExportService) Get(ctx context.Context, exportID string) (*FileExport, error) {
	var out FileExport
	if err := s.client.get(ctx, "/file-exports/"+exportID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *FileExportService) Rename(ctx context.Context, exportID, name string) (*FileExport, error) {
	var out FileExport
	if err := s.client.patch(ctx, "/file-exports/"+exportID+"/rename", map[string]string{"name": name}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *FileExportService) Delete(ctx context.Context, exportID string) error {
	return s.client.delete(ctx, "/file-exports/"+exportID)
}

func (s *FileExportService) Download(ctx context.Context, exportID string, w io.Writer) (int64, error) {
	resp, err := s.client.do(ctx, request{method: http.MethodGet, endpoint: "/file-exports/" + exportID + "/download"})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return io.Copy(w, resp.Body)
}

func (s *FileExportService) DownloadToFile(ctx context.Context, exportID, path string) error {
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
	if _, err := s.Download(ctx, exportID, f); err != nil {
		return err
	}
	return f.Close()
}
