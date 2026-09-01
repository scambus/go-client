package scambus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type WSMessage struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel"`
	Event   string          `json:"event"`
	Data    json.RawMessage `json:"data"`
}

// WSHandler receives the `data` payload of a matching message. A handler
// registered for the "*" event receives the whole message instead.
type WSHandler func(WSMessage)

type WSClient struct {
	wsURL      string
	authHeader [2]string
	logger     *slog.Logger

	maxReconnects  int
	reconnectDelay time.Duration

	mu        sync.Mutex
	conn      *websocket.Conn
	handlers  map[string]map[string][]WSHandler
	connected chan struct{}
}

type WSOption func(*WSClient)

func WithWSMaxReconnects(n int) WSOption { return func(c *WSClient) { c.maxReconnects = n } }

func WithWSReconnectDelay(d time.Duration) WSOption {
	return func(c *WSClient) { c.reconnectDelay = d }
}

func WithWSLogger(l *slog.Logger) WSOption { return func(c *WSClient) { c.logger = l } }

// NewWebSocket builds a WebSocket client reusing this client's credentials.
func (c *Client) NewWebSocket(opts ...WSOption) (*WSClient, error) {
	wsURL, err := websocketURL(c.apiURL)
	if err != nil {
		return nil, err
	}
	ws := &WSClient{
		wsURL:          wsURL,
		authHeader:     c.authHeader,
		logger:         c.logger,
		maxReconnects:  10,
		reconnectDelay: time.Second,
		handlers:       map[string]map[string][]WSHandler{},
		connected:      make(chan struct{}),
	}
	for _, opt := range opts {
		opt(ws)
	}
	return ws, nil
}

// websocketURL sends production traffic to live.scambus.net, which reaches the
// ALB directly; CloudFront does not carry WebSocket over VPC origins.
func websocketURL(apiURL string) (string, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("scambus: parse api url: %w", err)
	}
	scheme := "ws"
	if parsed.Scheme == "https" {
		scheme = "wss"
	}
	host := parsed.Host
	if strings.Contains(host, "scambus.net") && !strings.HasPrefix(host, "live.") {
		host = "live.scambus.net"
	}
	return scheme + "://" + host + parsed.Path + "/ws", nil
}

// On registers a handler for a channel and event. Pass "*" as the event to
// receive every message on the channel. The returned function unregisters it.
func (w *WSClient) On(channel, event string, handler WSHandler) func() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.handlers[channel] == nil {
		w.handlers[channel] = map[string][]WSHandler{}
	}
	index := len(w.handlers[channel][event])
	w.handlers[channel][event] = append(w.handlers[channel][event], handler)

	return func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		list := w.handlers[channel][event]
		if index < len(list) {
			w.handlers[channel][event] = append(list[:index], list[index+1:]...)
		}
	}
}

func (w *WSClient) Connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, w.wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			w.authHeader[0]: {w.authHeader[1]},
			"User-Agent":    {userAgent},
		},
	})
	if err != nil {
		return fmt.Errorf("scambus: websocket dial %s: %w", w.wsURL, err)
	}
	conn.SetReadLimit(32 * 1024 * 1024)

	w.mu.Lock()
	w.conn = conn
	select {
	case <-w.connected:
	default:
		close(w.connected)
	}
	w.mu.Unlock()
	return nil
}

func (w *WSClient) Close() error {
	w.mu.Lock()
	conn := w.conn
	w.conn = nil
	w.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close(websocket.StatusNormalClosure, "")
}

// Run reads messages until ctx is cancelled, reconnecting with backoff.
func (w *WSClient) Run(ctx context.Context) error {
	attempts := 0
	delay := w.reconnectDelay

	for {
		w.mu.Lock()
		conn := w.conn
		w.mu.Unlock()

		if conn == nil {
			if err := w.Connect(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if attempts >= w.maxReconnects {
					return err
				}
				attempts++
				if err := sleepCtx(ctx, jitter(delay)); err != nil {
					return err
				}
				delay = min(delay*2, time.Minute)
				continue
			}
			attempts, delay = 0, w.reconnectDelay
			continue
		}

		err := w.readLoop(ctx, conn)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		status := websocket.CloseStatus(err)
		if status == websocket.StatusNormalClosure {
			return nil
		}

		w.mu.Lock()
		w.conn = nil
		w.mu.Unlock()

		// 1012 is a planned server restart; come straight back.
		if status == 1012 {
			w.logger.Info("scambus websocket restarting with server")
			attempts, delay = 0, w.reconnectDelay
			if err := sleepCtx(ctx, 100*time.Millisecond); err != nil {
				return err
			}
			continue
		}

		if attempts >= w.maxReconnects {
			return err
		}
		attempts++
		w.logger.Warn("scambus websocket reconnecting", "attempt", attempts, "delay", delay, "error", err)
		if err := sleepCtx(ctx, jitter(delay)); err != nil {
			return err
		}
		delay = min(delay*2, time.Minute)
	}
}

func (w *WSClient) readLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			w.logger.Warn("scambus websocket message is not valid JSON", "error", err)
			continue
		}
		if msg.Type == "heartbeat" || msg.Type == "connected" {
			continue
		}
		w.dispatch(msg)
	}
}

func (w *WSClient) dispatch(msg WSMessage) {
	w.mu.Lock()
	byEvent := w.handlers[msg.Channel]
	matched := append([]WSHandler(nil), byEvent[msg.Event]...)
	wildcard := append([]WSHandler(nil), byEvent["*"]...)
	w.mu.Unlock()

	for _, handler := range matched {
		handler(msg)
	}
	for _, handler := range wildcard {
		handler(msg)
	}
}

func (w *WSClient) send(ctx context.Context, payload any) error {
	w.mu.Lock()
	conn := w.conn
	w.mu.Unlock()
	if conn == nil {
		return errors.New("scambus: websocket is not connected")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func (w *WSClient) SubscribeStream(ctx context.Context, streamID, cursor string, includeTest bool) error {
	if cursor == "" {
		cursor = CursorEnd
	}
	return w.send(ctx, map[string]any{
		"action":       "subscribe",
		"channel":      "stream:" + streamID,
		"cursor":       cursor,
		"include_test": includeTest,
	})
}

func (w *WSClient) UnsubscribeStream(ctx context.Context, streamID string) error {
	return w.send(ctx, map[string]any{
		"action":  "unsubscribe",
		"channel": "stream:" + streamID,
	})
}

// ListenNotifications connects and runs until ctx is cancelled.
func (w *WSClient) ListenNotifications(ctx context.Context, fn func(Notification)) error {
	w.On("notifications", "notification", func(msg WSMessage) {
		var n Notification
		if err := json.Unmarshal(msg.Data, &n); err != nil {
			w.logger.Warn("scambus notification decode failed", "error", err)
			return
		}
		fn(n)
	})
	return w.Run(ctx)
}

// ListenStream connects, subscribes, and runs until ctx is cancelled.
func (w *WSClient) ListenStream(ctx context.Context, streamID, cursor string, includeTest bool, fn func(StreamMessage)) error {
	w.On("stream:"+streamID, "message", func(msg WSMessage) {
		var envelope struct {
			Cursor string `json:"cursor"`
		}
		_ = json.Unmarshal(msg.Data, &envelope)
		fn(StreamMessage{Cursor: envelope.Cursor, Raw: msg.Data})
	})

	if err := w.Connect(ctx); err != nil {
		return err
	}
	if err := w.SubscribeStream(ctx, streamID, cursor, includeTest); err != nil {
		return err
	}
	return w.Run(ctx)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func jitter(d time.Duration) time.Duration {
	return d + time.Duration(rand.Float64()*0.25*float64(d))
}
