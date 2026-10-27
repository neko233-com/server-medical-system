package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/chat"
	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
	"github.com/neko233-com/server-medical-system/internal/models"
)

// =============================================================================
//  ChatHandler — 院内即时通讯（浮窗聊天）
// =============================================================================
//  能力：
//   • 在线同事互相**私聊**（direct）
//   • 全院**大群**（group，key = "hospital"）
//   • 支持**文字 + 图片**消息
//   • WebSocket 实时投递 + 心跳保活 + 重连友好
//   • 消息自动持久化，按 31 天窗口 + LRU 容量回收（见 internal/chat/retention.go）
//
//  鉴权说明：浏览器 WebSocket 握手无法携带自定义 Header，因此 WS 端点通过
//  ?token= 传递 JWT，并在处理器内部用 middleware.TokenUserID 校验（与 HTTP
//  请求完全相同的校验逻辑）。REST 端点仍走常规 Bearer 中间件。
// =============================================================================

// ChatHandler wires chat REST endpoints and the WebSocket upgrade.
type ChatHandler struct {
	db        *gorm.DB
	cfg       *config.Config
	hub       *chat.Hub
	upgrader  websocket.Upgrader
	uploadDir string // <cfg.UploadDir>/chat
}

// RegisterChat mounts every chat route. The WS endpoint authenticates itself via
// a query token, while REST endpoints use the standard bearer middleware.
func RegisterChat(r gin.IRouter, db *gorm.DB, cfg *config.Config, hub *chat.Hub) {
	h := &ChatHandler{
		db:  db,
		cfg: cfg,
		hub: hub,
		upgrader: websocket.Upgrader{
			// The SPA may be served from another origin in development
			// (Vite dev server on :5173), so cross-origin upgrades are allowed.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		uploadDir: filepath.Join(cfg.UploadDir, "chat"),
	}
	_ = os.MkdirAll(h.uploadDir, 0o755)

	// REST API — standard bearer-token protected.
	authed := r.Group("")
	authed.Use(middleware.AuthRequired(cfg))
	{
		authed.GET("/chat/online", h.Online)
		authed.GET("/chat/history", h.History)
		authed.POST("/chat/images", h.UploadImage)
		authed.GET("/chat/files/:file", h.ServeFile)
	}

	// WebSocket — self-authenticating via ?token= (headers are unavailable here).
	r.GET("/chat/ws", h.ServeWS)
}

// ---------------------------------------------------------------------------
//  REST
// ---------------------------------------------------------------------------

// Online returns currently connected users (roster with tints for avatars).
func (h *ChatHandler) Online(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.hub.OnlineSnapshot()})
}

// History returns the most recent messages for a conversation.
//
//	?peer=<userID>  → private thread with that user
//	?group=1        → the hospital-wide group
//	&limit=<n>      → how many messages to return (default 50, max 200)
func (h *ChatHandler) History(c *gin.Context) {
	uid, _ := c.Get("uid")
	me := uid.(uint)
	limit := atoui(c.Query("limit"))
	n := int(limit)
	if n <= 0 || n > 200 {
		n = 50
	}

	query := h.db.Model(&models.ChatMessage{})
	if c.Query("group") == "1" || c.Query("group") == "true" {
		query = query.Where("scope = ? AND group_key = ?", "group", chat.GroupKeyHospital)
	} else {
		peer := atoui(c.Query("peer"))
		if peer == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "peer_required"})
			return
		}
		query = query.Where("scope = ? AND peer_key = ?", "direct", chat.PeerKeyFor(me, peer))
	}

	var items []models.ChatMessage
	// Fetch newest-first then reverse in memory: this avoids dialect-specific
	// subquery syntax and keeps SQLite / Postgres / MySQL behaviour identical.
	query.Order("id desc").Limit(n).Find(&items)
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// UploadImage stores a chat image and returns its descriptor, which the client
// then references from an image message frame.
func (h *ChatHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "invalid_request")})
		return
	}
	if file.Size > 10<<20 { // 10MB chat images
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "file_too_large")})
		return
	}
	ct := file.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(c, "unsupported_file_type")})
		return
	}
	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	stored := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
	stored = filepath.Base(stored)
	dst := filepath.Join(h.uploadDir, stored)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(c, "server_error")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"file_name": stored,
		"orig_name": file.Filename,
		"file_size": file.Size,
	})
}

// ServeFile streams a stored chat image. Images are private to authenticated
// users, so they are never served as static assets.
func (h *ChatHandler) ServeFile(c *gin.Context) {
	name := filepath.Base(c.Param("file")) // prevent path traversal
	path := filepath.Join(h.uploadDir, name)
	f, err := os.Open(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	defer f.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Cache-Control", "private, max-age=86400")
	_, _ = io.Copy(c.Writer, f)
}

// ---------------------------------------------------------------------------
//  WebSocket
// ---------------------------------------------------------------------------

// ServeWS upgrades the connection and pumps frames until the client leaves.
func (h *ChatHandler) ServeWS(c *gin.Context) {
	token := c.Query("token")
	uid, err := middleware.TokenUserID(h.cfg, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var user models.User
	if h.db.First(&user, uid).Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := chat.NewClient(h.hub, uid, conn)

	// Seed the roster so presence frames can be rendered without DB lookups.
	h.hub.PutRoster(chat.PresenceItem{
		UserID: user.ID,
		Name:   user.Name,
		Role:   user.Role,
		Avatar: initial(user.Name),
		Status: "online",
	})
	h.hub.Register(client)

	// Immediately hand the new client the current roster.
	client.Queue(chat.PresenceFrame(h.hub.OnlineSnapshot()))

	go client.WritePump()
	client.ReadPump(h.handleFrame)
}

// handleFrame processes one inbound frame from a client.
func (h *ChatHandler) handleFrame(client *chat.Client, f chat.InFrame) {
	switch f.Type {
	case "ping":
		// Nothing to do: the connection's read deadline was already refreshed.
		return
	case "message":
		h.handleMessage(client, f)
	}
}

// handleMessage persists an inbound message and fans it out to recipients.
func (h *ChatHandler) handleMessage(client *chat.Client, f chat.InFrame) {
	isGroup := f.Scope == "group"
	if f.ContentType == "" {
		f.ContentType = "text"
	}
	if f.ContentType != "text" && f.ContentType != "image" {
		h.sendError(client, "unsupported content type")
		return
	}
	if !isGroup && f.ReceiverID == 0 {
		h.sendError(client, "receiver required for direct message")
		return
	}
	if f.ContentType == "text" && strings.TrimSpace(f.Content) == "" {
		return // ignore empty compositions
	}

	var sender models.User
	if h.db.First(&sender, client.UserID).Error != nil {
		h.sendError(client, "sender missing")
		return
	}

	msg := models.ChatMessage{
		SenderID:    client.UserID,
		SenderName:  sender.Name,
		ContentType: f.ContentType,
		Content:     f.Content,
		FileName:    filepath.Base(f.FileName),
		OrigName:    f.OrigName,
		FileSize:    f.FileSize,
		CreatedAt:   time.Now(),
	}
	if isGroup {
		msg.Scope = "group"
		msg.GroupKey = chat.GroupKeyHospital
		msg.ReceiverID = 0
	} else {
		msg.Scope = "direct"
		msg.PeerKey = chat.PeerKeyFor(client.UserID, f.ReceiverID)
		msg.ReceiverID = f.ReceiverID
	}

	if err := h.db.Create(&msg).Error; err != nil {
		h.sendError(client, "persist failed")
		return
	}

	frame := chat.OutFrame{Type: "message", ChatMessage: msg}
	payload, err := json.Marshal(frame)
	if err != nil {
		return
	}

	if isGroup {
		// Everyone in the hospital sees group traffic.
		h.hub.Broadcast(payload)
		return
	}
	// Direct: deliver to the recipient and echo to the sender's other sessions
	// so multi-tab usage stays consistent.
	h.hub.SendUser(msg.ReceiverID, payload)
	h.hub.SendUser(client.UserID, payload)

	// Tell the sender whether the peer is currently reachable.
	if !h.hub.IsOnline(msg.ReceiverID) {
		h.sendError(client, "recipient offline; message stored")
	}
}

// sendError pushes a non-fatal error notice to a single client.
func (h *ChatHandler) sendError(client *chat.Client, msg string) {
	payload, err := json.Marshal(chat.OutFrame{Type: "error", Error: msg})
	if err != nil {
		return
	}
	select {
	case client.SendChan() <- payload:
	default:
	}
}

// initial returns the first character of a name, used for generated avatars.
func initial(name string) string {
	if name == "" {
		return "?"
	}
	runes := []rune(name)
	return strings.ToUpper(string(runes[0]))
}
