package scambus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type MediaService struct{ client *Client }

type MediaUpload struct {
	Notes          string
	JournalEntryID string
	Metadata       map[string]any
}

func (s *MediaService) UploadFile(ctx context.Context, path string, opts *MediaUpload) (*Media, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("scambus: open media file: %w", err)
	}
	defer f.Close()
	return s.Upload(ctx, filepath.Base(path), f, opts)
}

func (s *MediaService) Upload(ctx context.Context, filename string, r io.Reader, opts *MediaUpload) (*Media, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if opts != nil {
		fields := map[string]string{"notes": opts.Notes, "journalEntryId": opts.JournalEntryID}
		if len(opts.Metadata) > 0 {
			encoded, err := json.Marshal(opts.Metadata)
			if err != nil {
				return nil, fmt.Errorf("scambus: encode media metadata: %w", err)
			}
			fields["metadata"] = string(encoded)
		}
		for name, value := range fields {
			if value == "" {
				continue
			}
			if err := w.WriteField(name, value); err != nil {
				return nil, err
			}
		}
	}

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("scambus: read media file: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	var out Media
	err = s.client.call(ctx, request{
		method:      http.MethodPost,
		endpoint:    "/media/upload",
		rawBody:     buf.Bytes(),
		contentType: w.FormDataContentType(),
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *MediaService) Get(ctx context.Context, mediaID string) (*Media, error) {
	var out Media
	if err := s.client.get(ctx, "/media/"+mediaID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
