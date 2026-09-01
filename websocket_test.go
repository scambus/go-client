package scambus

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestWebSocketURLRewritesProductionHost(t *testing.T) {
	cases := map[string]string{
		"https://scambus.net/api":      "wss://live.scambus.net/api/ws",
		"https://live.scambus.net/api": "wss://live.scambus.net/api/ws",
		"http://localhost:8080/api":    "ws://localhost:8080/api/ws",
		"https://staging.example/api":  "wss://staging.example/api/ws",
	}
	for input, want := range cases {
		got, err := websocketURL(input)
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		if got != want {
			t.Fatalf("%s: got %q, want %q", input, got, want)
		}
	}
}

type wsTestServer struct {
	*httptest.Server
	received chan []byte
	authOK   chan string
}

func newWSServer(t *testing.T, onConn func(ctx context.Context, c *websocket.Conn)) *wsTestServer {
	t.Helper()
	srv := &wsTestServer{received: make(chan []byte, 16), authOK: make(chan string, 4)}
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		srv.authOK <- r.Header.Get("Authorization")

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		onConn(r.Context(), conn)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// drain keeps reading so the connection answers the close handshake.
func drain(ctx context.Context, conn *websocket.Conn) {
	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}

func wsClient(t *testing.T, serverURL string, opts ...WSOption) *WSClient {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	c, err := New(WithAPIURL(serverURL), WithToken("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	ws, err := c.NewWebSocket(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestWebSocketDispatchesToChannelHandlers(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		messages := []string{
			`{"type":"connected","data":{"connectionId":"c1"}}`,
			`{"type":"heartbeat"}`,
			`{"type":"event","channel":"notifications","event":"notification","data":{"id":"n1","notification_text":"hello"}}`,
			`{"type":"event","channel":"notifications","event":"other","data":{"id":"n2"}}`,
		}
		for _, m := range messages {
			if err := conn.Write(ctx, websocket.MessageText, []byte(m)); err != nil {
				return
			}
		}
		drain(ctx, conn)
	})

	ws := wsClient(t, srv.URL)

	var mu sync.Mutex
	var matched, wildcard int
	done := make(chan struct{})

	ws.On("notifications", "notification", func(msg WSMessage) {
		var n Notification
		if err := json.Unmarshal(msg.Data, &n); err != nil {
			t.Error(err)
		}
		if n.NotificationText != "hello" {
			t.Errorf("got %+v", n)
		}
		mu.Lock()
		matched++
		mu.Unlock()
	})
	ws.On("notifications", "*", func(WSMessage) {
		mu.Lock()
		wildcard++
		if wildcard == 2 {
			close(done)
		}
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go ws.Run(ctx)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for messages")
	}

	mu.Lock()
	defer mu.Unlock()
	if matched != 1 {
		t.Fatalf("event handler ran %d times", matched)
	}
	if wildcard != 2 {
		t.Fatalf("wildcard handler ran %d times", wildcard)
	}
}

func TestWebSocketSendsAuthHeader(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) { drain(ctx, conn) })
	ws := wsClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ws.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	select {
	case got := <-srv.authOK:
		if got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
	case <-ctx.Done():
		t.Fatal("no connection observed")
	}
}

func TestWebSocketSubscribeSendsAction(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		conn.Write(ctx, websocket.MessageText, data)
		drain(ctx, conn)
	})
	ws := wsClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ws.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.SubscribeStream(ctx, "s1", "", true); err != nil {
		t.Fatal(err)
	}

	ws.mu.Lock()
	conn := ws.conn
	ws.mu.Unlock()
	_, echoed, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var sent map[string]any
	if err := json.Unmarshal(echoed, &sent); err != nil {
		t.Fatal(err)
	}
	if sent["action"] != "subscribe" || sent["channel"] != "stream:s1" {
		t.Fatalf("got %+v", sent)
	}
	if sent["cursor"] != CursorEnd {
		t.Fatalf("cursor should default to %q, got %v", CursorEnd, sent["cursor"])
	}
	if sent["include_test"] != true {
		t.Fatalf("include_test %v", sent["include_test"])
	}
}

func TestWebSocketSendWithoutConnectionFails(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) { drain(ctx, conn) })
	ws := wsClient(t, srv.URL)

	if err := ws.SubscribeStream(context.Background(), "s1", "", false); err == nil {
		t.Fatal("want an error")
	}
}

func TestWebSocketUnsubscribeHandlerStopsDelivery(t *testing.T) {
	ws := &WSClient{handlers: map[string]map[string][]registeredHandler{}}
	calls := 0
	off := ws.On("chan", "event", func(WSMessage) { calls++ })

	ws.dispatch(WSMessage{Channel: "chan", Event: "event"})
	off()
	ws.dispatch(WSMessage{Channel: "chan", Event: "event"})

	if calls != 1 {
		t.Fatalf("got %d calls", calls)
	}
}

// Unregistering out of order must remove the handler that was registered,
// not whichever one now sits at the same index.
func TestWebSocketUnregisterIsPositionIndependent(t *testing.T) {
	ws := &WSClient{handlers: map[string]map[string][]registeredHandler{}}

	var fired []string
	offA := ws.On("c", "e", func(WSMessage) { fired = append(fired, "A") })
	ws.On("c", "e", func(WSMessage) { fired = append(fired, "B") })
	offC := ws.On("c", "e", func(WSMessage) { fired = append(fired, "C") })

	offA()
	offC()
	ws.dispatch(WSMessage{Channel: "c", Event: "e"})

	if len(fired) != 1 || fired[0] != "B" {
		t.Fatalf("wrong handler removed: got %v, want [B]", fired)
	}
}

func TestWebSocketRunStopsOnContextCancel(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) { drain(ctx, conn) })
	ws := wsClient(t, srv.URL, WithWSReconnectDelay(time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := ws.Run(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestWebSocketGivesUpAfterMaxReconnects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	ws := wsClient(t, srv.URL, WithWSMaxReconnects(2), WithWSReconnectDelay(time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := ws.Run(ctx); err == nil {
		t.Fatal("want an error once reconnects run out")
	}
}

func TestWebSocketListenNotifications(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		conn.Write(ctx, websocket.MessageText,
			[]byte(`{"type":"event","channel":"notifications","event":"notification","data":{"id":"n1","notification_text":"alert","severity":"warning"}}`))
		drain(ctx, conn)
	})
	ws := wsClient(t, srv.URL, WithWSLogger(slog.New(slog.DiscardHandler)))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got := make(chan Notification, 1)
	go ws.ListenNotifications(ctx, func(n Notification) { got <- n })

	select {
	case n := <-got:
		if n.ID != "n1" || n.Severity != "warning" {
			t.Fatalf("got %+v", n)
		}
	case <-ctx.Done():
		t.Fatal("no notification received")
	}
}

func TestWebSocketListenStreamSubscribesAndDelivers(t *testing.T) {
	subscribed := make(chan map[string]any, 1)
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var action map[string]any
		json.Unmarshal(data, &action)
		subscribed <- action

		conn.Write(ctx, websocket.MessageText,
			[]byte(`{"type":"event","channel":"stream:s1","event":"message","data":{"identifier_id":"i1","cursor":"1-0"}}`))
		drain(ctx, conn)
	})
	ws := wsClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got := make(chan StreamMessage, 1)
	go ws.ListenStream(ctx, "s1", "0-0", false, func(m StreamMessage) { got <- m })

	select {
	case action := <-subscribed:
		if action["channel"] != "stream:s1" || action["cursor"] != "0-0" {
			t.Fatalf("subscribe payload %+v", action)
		}
	case <-ctx.Done():
		t.Fatal("no subscribe message received")
	}

	select {
	case m := <-got:
		msg, err := m.Identifier()
		if err != nil {
			t.Fatal(err)
		}
		if msg.IdentifierID != "i1" || m.Cursor != "1-0" {
			t.Fatalf("got %+v", msg)
		}
	case <-ctx.Done():
		t.Fatal("no stream message received")
	}
}

func TestWebSocketUnsubscribeStream(t *testing.T) {
	received := make(chan map[string]any, 2)
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var action map[string]any
			json.Unmarshal(data, &action)
			received <- action
		}
	})
	ws := wsClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ws.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	if err := ws.UnsubscribeStream(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-received:
		if action["action"] != "unsubscribe" || action["channel"] != "stream:s1" {
			t.Fatalf("got %+v", action)
		}
	case <-ctx.Done():
		t.Fatal("no unsubscribe message received")
	}
}

func TestWebSocketResubscribesAfterReconnect(t *testing.T) {
	var subscribes atomic.Int32
	var cursors []string
	var mu sync.Mutex
	connections := 0

	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		mu.Lock()
		connections++
		first := connections == 1
		mu.Unlock()

		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var action map[string]any
		json.Unmarshal(data, &action)
		if action["action"] == "subscribe" {
			subscribes.Add(1)
			mu.Lock()
			cursors = append(cursors, action["cursor"].(string))
			mu.Unlock()
		}

		conn.Write(ctx, websocket.MessageText,
			[]byte(`{"type":"event","channel":"stream:s1","event":"message","data":{"identifier_id":"i1","cursor":"1-7"}}`))

		if first {
			conn.Close(websocket.StatusInternalError, "drop")
			return
		}
		drain(ctx, conn)
	})

	ws := wsClient(t, srv.URL, WithWSReconnectDelay(time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	received := make(chan struct{}, 8)
	go ws.ListenStream(ctx, "s1", CursorEnd, false, func(StreamMessage) { received <- struct{}{} })

	for range 2 {
		select {
		case <-received:
		case <-ctx.Done():
			t.Fatalf("only %d subscribe frames sent; the stream went deaf after reconnect", subscribes.Load())
		}
	}

	if subscribes.Load() < 2 {
		t.Fatalf("reconnect must replay the subscription: %d frames", subscribes.Load())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(cursors) < 2 || cursors[1] != "1-7" {
		t.Fatalf("reconnect must resume from the last cursor, got %v", cursors)
	}
}

func TestWebSocketCloseStopsRun(t *testing.T) {
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) { drain(ctx, conn) })
	ws := wsClient(t, srv.URL, WithWSReconnectDelay(time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := ws.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- ws.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	ws.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close should end Run cleanly, got %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Close did not stop Run; it reconnected instead")
	}
}

func TestListenNotificationsSubscribes(t *testing.T) {
	subscribed := make(chan string, 1)
	srv := newWSServer(t, func(ctx context.Context, conn *websocket.Conn) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var action map[string]any
		json.Unmarshal(data, &action)
		subscribed <- action["channel"].(string)
		drain(ctx, conn)
	})
	ws := wsClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go ws.ListenNotifications(ctx, func(Notification) {})

	select {
	case channel := <-subscribed:
		if channel != "notifications" {
			t.Fatalf("channel %q", channel)
		}
	case <-ctx.Done():
		t.Fatal("no subscribe frame sent; notifications would never arrive")
	}
}
