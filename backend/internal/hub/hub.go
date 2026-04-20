package hub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type InboundHandler func(ctx context.Context, roomID string, c *Client, payload ChatInbound)
type MemberHandler func(ctx context.Context, roomID string, c *Client)
type ChatInbound struct {
	Content           string `json:"content"`
	ClientMsgID       string `json:"clientMsgId,omitempty"`
	ReplyToMessageID  *int64 `json:"replyToMessageId,omitempty"`
	ReplyToSenderName string `json:"replyToSenderName,omitempty"`
	ReplyToPreview    string `json:"replyToPreview,omitempty"`
}

type Manager struct {
	logger        *slog.Logger
	hubs          map[string]*Hub
	mu            sync.Mutex
	onInboundChat InboundHandler
	onConnect     MemberHandler
	onDisconnect  MemberHandler
}

func NewManager(logger *slog.Logger, onInbound InboundHandler, onConnect, onDisconnect MemberHandler) *Manager {
	return &Manager{
		logger:        logger,
		hubs:          make(map[string]*Hub),
		onInboundChat: onInbound,
		onConnect:     onConnect,
		onDisconnect:  onDisconnect,
	}
}

func (m *Manager) GetOrCreate(roomID string) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.hubs[roomID]; ok {
		return h
	}
	h := &Hub{
		roomID:     roomID,
		manager:    m,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 128),
	}
	m.hubs[roomID] = h
	go h.run()
	return h
}

func (m *Manager) RemoveIfEmpty(roomID string, h *Hub) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(h.clients) == 0 {
		delete(m.hubs, roomID)
	}
}

func (m *Manager) Broadcast(roomID, typ string, payload any) {
	h := m.GetOrCreate(roomID)
	packet, _ := json.Marshal(map[string]any{
		"type":      typ,
		"payload":   payload,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	h.broadcast <- packet
}

type Hub struct {
	roomID     string
	manager    *Manager
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
			if h.manager.onConnect != nil {
				h.manager.onConnect(context.Background(), h.roomID, c)
			}
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				if h.manager.onDisconnect != nil {
					h.manager.onDisconnect(context.Background(), h.roomID, c)
				}
				if len(h.clients) == 0 {
					h.manager.RemoveIfEmpty(h.roomID, h)
				}
			}
		case packet := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- packet:
				default:
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

type Client struct {
	Conn     *websocket.Conn
	SendChan chan []byte
	UserID   string
	Nickname string
	RoomID   string
	Identity string
	send     chan []byte
}

func NewClient(conn *websocket.Conn, roomID, userID, nickname, identity string) *Client {
	ch := make(chan []byte, 128)
	return &Client{
		Conn:     conn,
		RoomID:   roomID,
		UserID:   userID,
		Nickname: nickname,
		Identity: identity,
		send:     ch,
		SendChan: ch,
	}
}

func (h *Hub) ServeClient(c *Client) {
	h.register <- c
	go c.writePump()
	go c.readPump(h)
}

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.unregister <- c
		_ = c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		var event struct {
			Type string `json:"type"`
			ChatInbound
		}
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}
		if event.Type == "chat" && h.manager.onInboundChat != nil {
			h.manager.onInboundChat(context.Background(), h.roomID, c, event.ChatInbound)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
