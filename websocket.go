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

	mu         sync.Mutex
	conn       *websocket.Conn
	closed     bool
	nextID     uint64
	handlers   map[string]map[string][]registeredHandler
	subscribed map[string]streamSubscription
	channels   []string
}

type registeredHandler struct {
	id uint64
	fn WSHandler
}

type streamSubscription struct {
	cursor      string
	includeTest bool
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
		handlers:       map[string]map[string][]registeredHandler{},
		subscribed:     map[string]streamSubscription{},
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
		w.handlers[channel] = map[string][]registeredHandler{}
	}
	w.nextID++
	id := w.nextID
	w.handlers[channel][event] = append(w.handlers[channel][event], registeredHandler{id: id, fn: handler})

	return func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		list := w.handlers[channel][event]
		for i, h := range list {
			if h.id == id {
				w.handlers[channel][event] = append(list[:i], list[i+1:]...)
				return
			}
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
	previous := w.conn
	w.conn = conn
	w.mu.Unlock()

	if previous != nil {
		_ = previous.Close(websocket.StatusNormalClosure, "replaced")
	}
	return w.resubscribe(ctx)
}

// resubscribe replays every stream subscription, because the server holds
// them per connection and a reconnect starts with none.
func (w *WSClient) resubscribe(ctx context.Context) error {
	w.mu.Lock()
	pending := make(map[string]streamSubscription, len(w.subscribed))
	for id, sub := range w.subscribed {
		pending[id] = sub
	}
	w.mu.Unlock()

	w.mu.Lock()
	channels := append([]string(nil), w.channels...)
	w.mu.Unlock()

	for streamID, sub := range pending {
		if err := w.sendSubscribe(ctx, streamID, sub); err != nil {
			return err
		}
	}
	for _, channel := range channels {
		if err := w.send(ctx, map[string]any{"action": "subscribe", "channel": channel}); err != nil {
			return err
		}
	}
	return nil
}

func (w *WSClient) Close() error {
	w.mu.Lock()
	conn := w.conn
	w.conn = nil
	w.closed = true
	w.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close(websocket.StatusNormalClosure, "")
}

func (w *WSClient) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// Run reads messages until ctx is cancelled, reconnecting with backoff.
func (w *WSClient) Run(ctx context.Context) error {
	attempts := 0
	delay := w.reconnectDelay

	for {
		if w.isClosed() {
			return nil
		}

		w.mu.Lock()
		conn := w.conn
		w.mu.Unlock()

		if conn == nil {
			if err := w.Connect(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if w.maxReconnects >= 0 && attempts >= w.maxReconnects {
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
		if status == websocket.StatusNormalClosure || w.isClosed() {
			_ = w.Close()
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

		if w.maxReconnects >= 0 && attempts >= w.maxReconnects {
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
	matched := append([]registeredHandler(nil), byEvent[msg.Event]...)
	wildcard := append([]registeredHandler(nil), byEvent["*"]...)
	w.mu.Unlock()

	for _, handler := range matched {
		handler.fn(msg)
	}
	for _, handler := range wildcard {
		handler.fn(msg)
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
	sub := streamSubscription{cursor: cursor, includeTest: includeTest}

	w.mu.Lock()
	w.subscribed[streamID] = sub
	w.mu.Unlock()

	return w.sendSubscribe(ctx, streamID, sub)
}

func (w *WSClient) sendSubscribe(ctx context.Context, streamID string, sub streamSubscription) error {
	return w.send(ctx, map[string]any{
		"action":       "subscribe",
		"channel":      "stream:" + streamID,
		"cursor":       sub.cursor,
		"include_test": sub.includeTest,
	})
}

func (w *WSClient) UnsubscribeStream(ctx context.Context, streamID string) error {
	w.mu.Lock()
	delete(w.subscribed, streamID)
	w.mu.Unlock()

	return w.send(ctx, map[string]any{
		"action":  "unsubscribe",
		"channel": "stream:" + streamID,
	})
}

// ListenNotifications subscribes to the notifications channel and runs until
// ctx is cancelled. The server subscribes nothing on connect, so the frame
// has to be sent explicitly.
func (w *WSClient) ListenNotifications(ctx context.Context, fn func(Notification)) error {
	w.On("notifications", "notification", func(msg WSMessage) {
		var n Notification
		if err := json.Unmarshal(msg.Data, &n); err != nil {
			w.logger.Warn("scambus notification decode failed", "error", err)
			return
		}
		fn(n)
	})

	if err := w.Connect(ctx); err != nil {
		return err
	}
	if err := w.SubscribeChannel(ctx, "notifications"); err != nil {
		return err
	}
	return w.Run(ctx)
}

// SubscribeChannel subscribes to a non-stream channel and replays the
// subscription after a reconnect.
func (w *WSClient) SubscribeChannel(ctx context.Context, channel string) error {
	w.mu.Lock()
	w.channels = append(w.channels, channel)
	w.mu.Unlock()
	return w.send(ctx, map[string]any{"action": "subscribe", "channel": channel})
}

// ListenStream connects, subscribes, and runs until ctx is cancelled.
func (w *WSClient) ListenStream(ctx context.Context, streamID, cursor string, includeTest bool, fn func(StreamMessage)) error {
	w.On("stream:"+streamID, "message", func(msg WSMessage) {
		var envelope struct {
			Cursor string `json:"cursor"`
		}
		_ = json.Unmarshal(msg.Data, &envelope)
		if envelope.Cursor != "" {
			w.mu.Lock()
			if sub, ok := w.subscribed[streamID]; ok {
				sub.cursor = envelope.Cursor
				w.subscribed[streamID] = sub
			}
			w.mu.Unlock()
		}
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
