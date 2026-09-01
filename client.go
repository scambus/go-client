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

// 408 and 429 mean the server did not process the request, so any method may
// be replayed. A 5xx may follow a committed write, so only idempotent methods
// are retried on those.
var (
	notProcessedStatus = map[int]bool{408: true, 429: true}
	serverFaultStatus  = map[int]bool{500: true, 502: true, 503: true, 504: true}
)

func retryableFor(method string, status int) bool {
	if notProcessedStatus[status] {
		return true
	}
	return serverFaultStatus[status] && idempotent(method)
}

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
	timeout      time.Duration
	maxResponse  int64

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

// WithTimeout bounds a single non-streaming request. It never mutates a
// client passed to WithHTTPClient.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithMaxResponseBytes caps how much of a JSON response body is read.
func WithMaxResponseBytes(n int64) Option { return func(c *Client) { c.maxResponse = n } }

func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

func WithRetryMaxTime(d time.Duration) Option { return func(c *Client) { c.retryMaxTime = d } }

func WithLogger(l *slog.Logger) Option { return func(c *Client) { c.logger = l } }

// New resolves credentials from options, then the SCAMBUS_* environment
// variables, then ~/.scambus/config.json.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		logger:       slog.New(slog.DiscardHandler),
		maxRetries:   10,
		retryMaxTime: 5 * time.Minute,
		timeout:      30 * time.Second,
		maxResponse:  64 << 20,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}
	// Credentials ride in headers; Go strips Authorization across hosts but
	// forwards X-API-Key, so a redirect would hand the key to the new host.
	if c.httpClient.CheckRedirect == nil {
		c.httpClient.CheckRedirect = refuseRedirect
	}

	cfg := loadCLIConfig()
	c.apiURL = resolveAPIURL(c.apiURL, cfg)
	c.apiURL = ensureAPIPath(c.apiURL)

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

func refuseRedirect(req *http.Request, via []*http.Request) error {
	return fmt.Errorf("scambus: refusing redirect to %s: the API does not redirect, and following one would disclose credentials", req.URL.Redacted())
}

// ensureAPIPath appends /api only when the base URL carries no path of its
// own, so a versioned or proxied base is left alone.
func ensureAPIPath(base string) string {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Path == "" || parsed.Path == "/" {
		return strings.TrimRight(base, "/") + "/api"
	}
	return strings.TrimRight(base, "/")
}

// idempotent reports whether replaying the method is safe. A POST or PATCH
// may already have been committed when the response was lost.
func idempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

// safeEndpoint escapes each path segment and refuses anything that would
// leave the intended path or start a query.
func safeEndpoint(endpoint string) (string, error) {
	if strings.ContainsAny(endpoint, "?#") {
		return "", fmt.Errorf("%w: path %q must not contain %q or %q", ErrValidation, endpoint, "?", "#")
	}
	segments := strings.Split(strings.TrimLeft(endpoint, "/"), "/")
	for i, seg := range segments {
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("%w: path %q must not contain a %q segment", ErrValidation, endpoint, seg)
		}
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/"), nil
}

type request struct {
	method      string
	endpoint    string
	body        any
	rawBody     []byte
	contentType string
	query       url.Values
	accept      string
	stream      bool
}

func (c *Client) newHTTPRequest(ctx context.Context, r request) (*http.Request, error) {
	endpoint, err := safeEndpoint(r.endpoint)
	if err != nil {
		return nil, err
	}
	u := c.apiURL + "/" + endpoint
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
// maxRetries and retryMaxTime. A non-idempotent request is replayed only when
// the server states it did not process it, so a lost response never turns one
// write into several. The caller owns closing the response body.
func (c *Client) do(ctx context.Context, r request) (*http.Response, error) {
	start := time.Now()
	attempt := 0

	if c.timeout > 0 && !r.stream {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

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
			// A dropped connection may still have delivered the request.
			if !idempotent(r.method) || attempt >= c.maxRetries || elapsed >= c.retryMaxTime {
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

		if retryableFor(r.method, resp.StatusCode) {
			elapsed := time.Since(start)
			if attempt < c.maxRetries && elapsed < c.retryMaxTime {
				attempt++
				delay := retryAfter(resp)
				if delay == 0 {
					delay = retryBaseDelay
				}
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

	limit := c.maxResponse
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return fmt.Errorf("scambus: read %s %s response: %w", r.method, r.endpoint, err)
	}
	if int64(len(body)) > limit {
		return fmt.Errorf("scambus: %s %s response exceeds %d bytes", r.method, r.endpoint, limit)
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
