package sioyun

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ── WebSocket 客户端 ─────────────────────────────────────────────────────

// WSClient WebSocket 实时推送客户端。
type WSClient struct {
	cfg           Config
	conn          *websocket.Conn
	mu            sync.Mutex
	subscriptions map[string]bool
	handlers      map[string][]EventHandler
	done          chan struct{}
	reconnect     bool
}

// EventHandler 事件处理回调。
type EventHandler func(event GatewayEvent)

// GatewayEvent 网关推送的事件。
type GatewayEvent struct {
	Type    string      `json:"type"`
	Event   string      `json:"event,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Channel string      `json:"channel,omitempty"`
	Code    int         `json:"code,omitempty"`
	Msg     string      `json:"msg,omitempty"`
}

// NewWSClient 创建 WebSocket 客户端。
func NewWSClient(cfg Config) *WSClient {
	return &WSClient{
		cfg:           cfg,
		subscriptions: make(map[string]bool),
		handlers:      make(map[string][]EventHandler),
		done:          make(chan struct{}),
		reconnect:     true,
	}
}

// Connect 建立 WebSocket 连接并鉴权。
func (w *WSClient) Connect(ctx context.Context) error {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := randomString(16)

	u, err := url.Parse(w.cfg.BaseURL)
	if err != nil {
		return err
	}
	wsPath := u.Path + "/ws"
	// 签名使用完整路径，与后端 sign_auth.go ctx.Path() 对齐
	signature := sign(w.cfg.SecretKey, "GET", wsPath, timestamp, nonce, "")
	u.Path = u.Path + "/ws"

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s%s?access_key=%s&timestamp=%s&nonce=%s&signature=%s",
		scheme, u.Host, u.Path,
		w.cfg.AccessKey, timestamp, nonce, signature,
	)

	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: false},
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	header.Set("X-Request-Id", "ws-"+newID())

	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("sioyun ws: dial failed: %w", err)
	}

	w.mu.Lock()
	w.conn = conn
	w.mu.Unlock()

	// 恢复订阅
	for ch := range w.subscriptions {
		_ = w.subscribe(ch)
	}

	// 启动接收循环
	go w.readLoop()
	go w.heartbeat()

	return nil
}

// Subscribe 订阅事件频道，支持通配符。
// 频道示例："payment.*", "sms.delivered", "partner.*"
func (w *WSClient) Subscribe(channel string) error {
	w.subscriptions[channel] = true
	if w.conn == nil {
		return nil // 连接未建立，暂存订阅
	}
	return w.subscribe(channel)
}

// Unsubscribe 取消订阅。
func (w *WSClient) Unsubscribe(channel string) error {
	delete(w.subscriptions, channel)
	if w.conn == nil {
		return nil
	}
	msg := map[string]string{"type": "unsubscribe", "channel": channel}
	return w.writeJSON(msg)
}

// On 注册事件处理器。event 支持通配符（"payment.*" 匹配 "payment.success"）。
func (w *WSClient) On(event string, handler EventHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[event] = append(w.handlers[event], handler)
}

// Close 断开 WebSocket 连接。
func (w *WSClient) Close() error {
	w.reconnect = false
	close(w.done)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// ── 内部方法 ────────────────────────────────────────────────────────────

func (w *WSClient) subscribe(channel string) error {
	msg := map[string]string{"type": "subscribe", "channel": channel}
	return w.writeJSON(msg)
}

func (w *WSClient) writeJSON(v interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn == nil {
		return fmt.Errorf("sioyun ws: not connected")
	}
	return w.conn.WriteJSON(v)
}

func (w *WSClient) readLoop() {
	for {
		select {
		case <-w.done:
			return
		default:
		}

		var event GatewayEvent
		if err := w.conn.ReadJSON(&event); err != nil {
			if w.reconnect {
				w.reconnectWithBackoff()
			}
			return
		}

		switch event.Type {
		case "event":
			w.dispatch(event)
		case "subscribed":
			// 订阅确认，无需处理
		case "pong":
			// 心跳响应
		case "error":
			w.dispatch(event)
		}
	}
}

func (w *WSClient) dispatch(event GatewayEvent) {
	w.mu.Lock()
	handlers := make([]EventHandler, 0)
	if h, ok := w.handlers[event.Event]; ok {
		handlers = append(handlers, h...)
	}
	for pattern, h := range w.handlers {
		if matchPattern(pattern, event.Event) {
			handlers = append(handlers, h...)
		}
	}
	w.mu.Unlock()

	for _, handler := range handlers {
		handler(event)
	}
}

func (w *WSClient) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.mu.Lock()
			if w.conn != nil {
				_ = w.conn.WriteJSON(map[string]string{"type": "ping"})
			}
			w.mu.Unlock()
		}
	}
}

func (w *WSClient) reconnectWithBackoff() {
	delays := []time.Duration{1, 2, 4, 8, 16, 30, 60}
	for _, d := range delays {
		select {
		case <-w.done:
			return
		case <-time.After(d * time.Second):
		}
		if err := w.Connect(context.Background()); err == nil {
			return
		}
	}
}

func matchPattern(pattern, event string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == event {
		return true
	}
	// 简单通配符匹配（不依赖第三方库）
	if len(pattern) > 2 && pattern[len(pattern)-2:] == ".*" {
		prefix := pattern[:len(pattern)-2]
		return len(event) > len(prefix) && event[:len(prefix)] == prefix && event[len(prefix)] == '.'
	}
	return false
}
