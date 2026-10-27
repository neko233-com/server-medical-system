// Package chat implements the in-hospital instant messaging used by the
// floating chat widget: direct (private) messages between online colleagues and
// a single hospital-wide group, delivered in real time over WebSocket.
package chat

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/neko233-com/server-medical-system/internal/models"
)

// ---------------------------------------------------------------------------
//  Wire protocol
// ---------------------------------------------------------------------------

// GroupKeyHospital is the well-known identifier of the single hospital-wide
// group every member can post to.
const GroupKeyHospital = "hospital"

// InFrame is a message received from a browser client.
type InFrame struct {
	Type string `json:"type"` // message | ping | read
	// message frames:
	Scope       string `json:"scope"`        // "direct" | "group"
	ReceiverID  uint   `json:"receiver_id"`  // for direct messages
	ContentType string `json:"content_type"` // text | image
	Content     string `json:"content"`
	FileName    string `json:"file_name"`
	OrigName    string `json:"orig_name"`
	FileSize    int64  `json:"file_size"`
}

// OutFrame is a message pushed to browser clients. Field names intentionally
// mirror models.ChatMessage so the frontend can render either shape directly.
type OutFrame struct {
	Type string `json:"type"` // message | presence | online | error
	// message payload (flattened from the persisted record)
	models.ChatMessage
	// presence payload
	Online []PresenceItem `json:"online,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// PresenceItem describes one connected user for the online roster.
type PresenceItem struct {
	UserID   uint   `json:"user_id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`
	LastSeen string `json:"last_seen,omitempty"`
}

// ---------------------------------------------------------------------------
//  Client
// ---------------------------------------------------------------------------

// Client is one open WebSocket connection belonging to a user. A user may have
// several (multiple tabs / devices); every client receives its own copy.
type Client struct {
	UserID uint
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
}

// Hub tracks every connected client and routes messages between them.
type Hub struct {
	mu     sync.RWMutex
	conns  map[uint]map[*Client]struct{}
	roster map[uint]PresenceItem
}

// NewHub constructs an empty hub.
func NewHub() *Hub {
	return &Hub{
		conns:  map[uint]map[*Client]struct{}{},
		roster: map[uint]PresenceItem{},
	}
}

// Register adds a client and announces the new roster to everyone.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	if h.conns[c.UserID] == nil {
		h.conns[c.UserID] = map[*Client]struct{}{}
	}
	h.conns[c.UserID][c] = struct{}{}
	h.mu.Unlock()
	h.broadcastPresence()
	log.Printf("chat: user %d connected (%d connections)", c.UserID, h.ConnectionCount())
}

// Unregister removes a client and announces the updated roster.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if set, ok := h.conns[c.UserID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.conns, c.UserID)
		}
	}
	h.mu.Unlock()
	h.broadcastPresence()
	log.Printf("chat: user %d disconnected", c.UserID)
}

// ConnectionCount returns the number of live sockets.
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for _, set := range h.conns {
		n += len(set)
	}
	return n
}

// PutRoster stores the display info of a user so presence frames can be built
// without touching the database on every connection change.
func (h *Hub) PutRoster(item PresenceItem) {
	h.mu.Lock()
	h.roster[item.UserID] = item
	h.mu.Unlock()
}

// IsOnline reports whether the user currently holds at least one connection.
func (h *Hub) IsOnline(uid uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	set, ok := h.conns[uid]
	return ok && len(set) > 0
}

// OnlineSnapshot returns the roster entries of all currently connected users.
func (h *Hub) OnlineSnapshot() []PresenceItem {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]PresenceItem, 0, len(h.conns))
	for uid := range h.conns {
		item, ok := h.roster[uid]
		if !ok {
			item = PresenceItem{UserID: uid, Name: "User " + strconv.Itoa(int(uid))}
		}
		out = append(out, item)
	}
	return out
}

// SendUser delivers payload to every connection owned by uid. Returns false when
// the user has no live connection (message was persisted but not pushed).
func (h *Hub) SendUser(uid uint, payload []byte) bool {
	h.mu.RLock()
	set := h.conns[uid]
	clients := make([]*Client, 0, len(set))
	for c := range set {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	if len(clients) == 0 {
		return false
	}
	for _, c := range clients {
		// Non-blocking: a slow client must never stall the sender.
		select {
		case c.send <- payload:
		default:
			go func(cl *Client) { cl.conn.Close() }(c)
		}
	}
	return true
}

// Broadcast delivers payload to every connected client.
func (h *Hub) Broadcast(payload []byte) {
	h.mu.RLock()
	var all []*Client
	for _, set := range h.conns {
		for c := range set {
			all = append(all, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range all {
		select {
		case c.send <- payload:
		default:
			go func(cl *Client) { cl.conn.Close() }(c)
		}
	}
}

// broadcastPresence pushes the current roster to every client.
func (h *Hub) broadcastPresence() {
	frame := OutFrame{Type: "presence", Online: h.OnlineSnapshot()}
	buf, err := json.Marshal(frame)
	if err != nil {
		return
	}
	h.Broadcast(buf)
}

// PresenceFrame builds an "online" frame for a single recipient (used right
// after a successful handshake).
func PresenceFrame(items []PresenceItem) []byte {
	frame := OutFrame{Type: "online", Online: items}
	buf, err := json.Marshal(frame)
	if err != nil {
		return []byte(`{"type":"online","online":[]}`)
	}
	return buf
}

// ---------------------------------------------------------------------------
//  Read / write pumps
// ---------------------------------------------------------------------------

// ReadPump reads frames until the socket closes, handing each one to handler.
func (c *Client) ReadPump(handler func(*Client, InFrame)) {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512 << 10) // 512KB max frame
	_ = c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var f InFrame
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}
		handler(c, f)
	}
}

// WritePump flushes queued frames and emits heartbeats.
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// NewClient wraps a websocket connection into a hub client.
func NewClient(hub *Hub, uid uint, conn *websocket.Conn) *Client {
	return &Client{UserID: uid, hub: hub, conn: conn, send: make(chan []byte, 128)}
}

// Queue places a payload into the client's outbound queue.
func (c *Client) Queue(payload []byte) { c.send <- payload }

// SendChan exposes the outbound queue for non-blocking sends by the hub owner.
func (c *Client) SendChan() chan []byte { return c.send }

// Close terminates the underlying socket.
func (c *Client) Close() { c.conn.Close() }

// PeerKeyFor returns the canonical, order-independent conversation key for a
// pair of user ids, so A→B and B→A resolve to the same private thread.
func PeerKeyFor(a, b uint) string {
	return peerKey(a, b)
}

// peerKey builds the key; see PeerKeyFor.
func peerKey(a, b uint) string {
	if a > b {
		a, b = b, a
	}
	return strconv.Itoa(int(a)) + "-" + strconv.Itoa(int(b))
}
