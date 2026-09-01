package scambus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Version   = "0.1.0"
	userAgent = "scambus-go-client/" + Version

	retryBaseDelay    = 1 * time.Second
	retryMaxBackoff   = 20 * time.Second
	retryThrottleBase = 2 * time.Second
)

var retryableStatus = map[int]bool{408: true, 429: true, 500: true, 502: true, 503: true, 504: true}

type Client struct {
	apiURL       string
	httpClient   *http.Client
	logger       *slog.Logger
	authHeader   [2]string
	apiKeyID     string
	apiKeySecret string
	apiToken     string
	maxRetries   int
	retryMaxTime time.Duration

	Media         *MediaService
	Journal       *JournalService
	Identifiers   *IdentifierService
	Cases         *CaseService
	Comments      *CommentService
	Queues        *QueueService
	Streams       *StreamService
	Consume       *ConsumeService
	FileExports   *FileExportService
	Views         *ViewService
	Tags          *TagService
	Search        *SearchService
	Notifications *NotificationService
	Sessions      *SessionService
	Personas      *PersonaService
	Reports       *ReportService
	Automations   *AutomationService
	Admin         *AdminService
}

type Option func(*Client)

func WithAPIURL(u string) Option { return func(c *Client) { c.apiURL = u } }

func WithAPIKey(id, secret string) Option {
	return func(c *Client) { c.apiKeyID, c.apiKeySecret = id, secret }
}

func WithToken(token string) Option { return func(c *Client) { c.apiToken = token } }

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

func WithRetryMaxTime(d time.Duration) Option { return func(c *Client) { c.retryMaxTime = d } }

func WithLogger(l *slog.Logger) Option { return func(c *Client) { c.logger = l } }

// New resolves credentials from options, then the SCAMBUS_* environment
// variables, then ~/.scambus/config.json.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       slog.New(slog.DiscardHandler),
		maxRetries:   10,
		retryMaxTime: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}

	cfg := loadCLIConfig()
	c.apiURL = resolveAPIURL(c.apiURL, cfg)
	if !strings.HasSuffix(c.apiURL, "/api") {
		c.apiURL += "/api"
	}

	if c.apiKeyID == "" {
		c.apiKeyID = os.Getenv("SCAMBUS_API_KEY_ID")
	}
	if c.apiKeySecret == "" {
		c.apiKeySecret = os.Getenv("SCAMBUS_API_KEY_SECRET")
	}

	switch {
	case c.apiKeyID != "" && c.apiKeySecret != "":
		c.authHeader = [2]string{"X-API-Key", c.apiKeyID + ":" + c.apiKeySecret}
	default:
		c.apiToken = resolveToken(c.apiToken, cfg)
		if c.apiToken == "" {
			return nil, ErrNoCredentials
		}
		c.authHeader = [2]string{"Authorization", "Bearer " + c.apiToken}
	}

	c.registerServices()
	return c, nil
}

func (c *Client) registerServices() {
	c.Media = &MediaService{c}
	c.Journal = &JournalService{c}
	c.Identifiers = &IdentifierService{c}
	c.Cases = &CaseService{c}
	c.Comments = &CommentService{c}
	c.Queues = &QueueService{c}
	c.Streams = &StreamService{c}
	c.Consume = &ConsumeService{c}
	c.FileExports = &FileExportService{c}
	c.Views = &ViewService{c}
	c.Tags = &TagService{c}
	c.Search = &SearchService{c}
	c.Notifications = &NotificationService{c}
	c.Sessions = &SessionService{c}
	c.Personas = &PersonaService{c}
	c.Reports = &ReportService{c}
	c.Automations = &AutomationService{c}
	c.Admin = &AdminService{c}
}

func (c *Client) APIURL() string { return c.apiURL }

type request struct {
	method      string
	endpoint    string
	body        any
	rawBody     []byte
	contentType string
	query       url.Values
	accept      string
}

func (c *Client) newHTTPRequest(ctx context.Context, r request) (*http.Request, error) {
	u := c.apiURL + "/" + strings.TrimLeft(r.endpoint, "/")
	if len(r.query) > 0 {
		u += "?" + r.query.Encode()
	}

	var body io.Reader
	payload := r.rawBody
	if r.body != nil {
		encoded, err := json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("scambus: encode request body: %w", err)
		}
		payload = encoded
	}
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, r.method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set(c.authHeader[0], c.authHeader[1])
	req.Header.Set("User-Agent", userAgent)
	switch {
	case r.contentType != "":
		req.Header.Set("Content-Type", r.contentType)
	case r.body != nil:
		req.Header.Set("Content-Type", "application/json")
	}
	if r.accept != "" {
		req.Header.Set("Accept", r.accept)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		if payload == nil {
			return nil, nil
		}
		return io.NopCloser(bytes.NewReader(payload)), nil
	}
	return req, nil
}

// do retries transient failures with full-jitter backoff, bounded by both
// maxRetries and retryMaxTime. The caller owns closing the response body.
func (c *Client) do(ctx context.Context, r request) (*http.Response, error) {
	start := time.Now()
	attempt := 0

	for {
		req, err := c.newHTTPRequest(ctx, r)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			elapsed := time.Since(start)
			if attempt >= c.maxRetries || elapsed >= c.retryMaxTime {
				return nil, fmt.Errorf("scambus: %s %s failed after %d retries over %s: %w",
					r.method, r.endpoint, attempt, elapsed.Round(time.Second), err)
			}
			attempt++
			if sleepErr := c.wait(ctx, backoff(attempt, retryBaseDelay), start); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if resp.StatusCode < 400 {
			return resp, nil
		}

		if retryableStatus[resp.StatusCode] {
			elapsed := time.Since(start)
			if attempt < c.maxRetries && elapsed < c.retryMaxTime {
				attempt++
				delay := retryAfter(resp)
				if delay < 0 {
					base := retryBaseDelay
					if resp.StatusCode == http.StatusTooManyRequests {
						base = retryThrottleBase
					}
					delay = backoff(attempt, base)
				}
				c.logger.Warn("retrying scambus request",
					"status", resp.StatusCode, "method", r.method, "endpoint", r.endpoint,
					"attempt", attempt, "delay", delay)
				_ = resp.Body.Close()
				if sleepErr := c.wait(ctx, delay, start); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
		}

		return nil, c.errorFromResponse(r, resp)
	}
}

func (c *Client) wait(ctx context.Context, d time.Duration, start time.Time) error {
	if remaining := c.retryMaxTime - time.Since(start); d > remaining {
		d = remaining
	}
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func backoff(attempt int, base time.Duration) time.Duration {
	ceiling := float64(base) * math.Pow(2, float64(attempt))
	if ceiling > float64(retryMaxBackoff) {
		ceiling = float64(retryMaxBackoff)
	}
	return time.Duration(rand.Float64() * ceiling)
}

func retryAfter(resp *http.Response) time.Duration {
	header := resp.Header.Get("Retry-After")
	if header == "" {
		return -1
	}
	if secs, err := strconv.ParseFloat(header, 64); err == nil {
		return min(time.Duration(secs*float64(time.Second)), retryMaxBackoff)
	}
	if at, err := http.ParseTime(header); err == nil {
		return min(max(time.Until(at), 0), retryMaxBackoff)
	}
	return -1
}

func (c *Client) errorFromResponse(r request, resp *http.Response) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Method:     r.method,
		Endpoint:   r.endpoint,
		Body:       body,
		kind:       kindForStatus(resp.StatusCode),
	}

	var payload map[string]any
	if json.Unmarshal(body, &payload) == nil {
		apiErr.Data = payload
		if msg, ok := payload["error"].(string); ok {
			apiErr.Message = msg
		} else if msg, ok := payload["message"].(string); ok {
			apiErr.Message = msg
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(body))
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}

func (c *Client) call(ctx context.Context, r request, out any) error {
	resp, err := c.do(ctx, r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("scambus: read %s %s response: %w", r.method, r.endpoint, err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return fmt.Errorf("scambus: decode %s %s response (status %d): %w: %s",
			r.method, r.endpoint, resp.StatusCode, err, preview)
	}
	return nil
}

func (c *Client) get(ctx context.Context, endpoint string, q url.Values, out any) error {
	return c.call(ctx, request{method: http.MethodGet, endpoint: endpoint, query: q}, out)
}

func (c *Client) post(ctx context.Context, endpoint string, body, out any) error {
	return c.call(ctx, request{method: http.MethodPost, endpoint: endpoint, body: body}, out)
}

func (c *Client) put(ctx context.Context, endpoint string, body, out any) error {
	return c.call(ctx, request{method: http.MethodPut, endpoint: endpoint, body: body}, out)
}

func (c *Client) patch(ctx context.Context, endpoint string, body, out any) error {
	return c.call(ctx, request{method: http.MethodPatch, endpoint: endpoint, body: body}, out)
}

func (c *Client) delete(ctx context.Context, endpoint string) error {
	return c.call(ctx, request{method: http.MethodDelete, endpoint: endpoint}, nil)
}
