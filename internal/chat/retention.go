package chat

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/models"
)

// =============================================================================
//  自动留存（31 天 LRU）
// =============================================================================
//  聊天数据增长很快且价值随时间衰减，因此后台常驻一个清理任务，按两条规则回收空间：
//
//   1) 时间窗口：删除早于 RetentionDays（默认 31 天）的消息；
//   2) LRU 容量：单个会话最多保留 MaxPerConversation 条最新消息，
//      超出部分从最旧的一条开始淘汰（典型的 LRU 语义）。
//
//  被删除的**图片附件会同步从磁盘移除**，避免 uploads/ 里堆积孤儿文件。
//  任务幂等且只删除数据，重复执行安全。
// =============================================================================

// DefaultRetentionDays is how long chat history is kept unless overridden.
const DefaultRetentionDays = 31

// DefaultMaxPerConversation caps how many rows each conversation keeps.
const DefaultMaxPerConversation = 2000

// StartRetention launches the background cleanup loop and returns immediately.
// interval controls how often cleanup runs (defaults to 1 hour when <= 0).
func StartRetention(db *gorm.DB, uploadDir string, days, max int, interval time.Duration) {
	if days <= 0 {
		days = DefaultRetentionDays
	}
	if max <= 0 {
		max = DefaultMaxPerConversation
	}
	if interval <= 0 {
		interval = time.Hour
	}
	chatDir := filepath.Join(uploadDir, "chat")

	go func() {
		// Run once at startup so long downtime does not delay space reclamation.
		purge(db, chatDir, days, max)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			purge(db, chatDir, days, max)
		}
	}()
	log.Printf("chat: retention started (keep %dd, max %d/conversation, every %s)", days, max, interval)
}

// purge performs one cleanup pass: time-window deletion, LRU trimming, then
// removal of any file no longer referenced by a message.
func purge(db *gorm.DB, chatDir string, days, max int) {
	total := int64(0)

	// --- 1) 时间窗口：删除 31 天前的历史 ---
	cutoff := time.Now().AddDate(0, 0, -days)
	total += deleteWithFiles(db, chatDir, "created_at < ?", cutoff)

	// --- 2) LRU 容量：每个会话仅保留最新 max 条 ---
	total += Trim(db, chatDir, max)

	// --- 3) 孤儿文件清理：磁盘上有但数据库中已无引用的图片 ---
	total += sweepOrphanFiles(db, chatDir)

	if total > 0 {
		log.Printf("chat: retention pass removed %d item(s)", total)
	}
}

// Trim applies LRU capacity limiting to **every** conversation (both the
// hospital group and all private threads), keeping only the newest `max` rows.
// It returns the number of rows deleted.
func Trim(db *gorm.DB, chatDir string, max int) int64 {
	if max <= 0 {
		max = DefaultMaxPerConversation
	}
	cut := time.Now().AddDate(0, 0, -DefaultRetentionDays)
	var removed int64
	for _, conv := range AllConversations(db) {
		var ids []uint
		// Skip rows already scheduled for time-window deletion.
		db.Model(&models.ChatMessage{}).
			Where(conv.Where, conv.Args...).
			Where("created_at >= ?", cut).
			Order("id desc").
			Offset(max).
			Pluck("id", &ids)
		if len(ids) == 0 {
			continue
		}
		n := deleteWithFiles(db, chatDir, "id IN ?", ids)
		if n > 0 {
			removed += n
			log.Printf("chat: LRU trimmed %d message(s) from %s (%s)", n, conv.Label, conv.Args)
		}
	}
	return removed
}

// Conversation identifies one message thread for LRU processing.
type Conversation struct {
	Label string
	Where string
	Args  []interface{}
}

// AllConversations lists every existing thread using an order-independent
// representation: group_key for the hospital group, peer_key for private pairs.
func AllConversations(db *gorm.DB) []Conversation {
	out := make([]Conversation, 0)

	var groups []string
	db.Model(&models.ChatMessage{}).
		Where("scope = ?", "group").
		Distinct().Pluck("group_key", &groups)
	for _, g := range groups {
		if g == "" {
			continue
		}
		out = append(out, Conversation{
			Label: "group:" + g,
			Where: "scope = ? AND group_key = ?",
			Args:  []interface{}{"group", g},
		})
	}

	var peers []string
	db.Model(&models.ChatMessage{}).
		Where("scope = ? AND peer_key <> ''", "direct").
		Distinct().Pluck("peer_key", &peers)
	for _, p := range peers {
		out = append(out, Conversation{
			Label: "direct:" + p,
			Where: "scope = ? AND peer_key = ?",
			Args:  []interface{}{"direct", p},
		})
	}
	return out
}

// deleteWithFiles removes rows matching the condition together with their image
// attachments. condition/args follow gorm.Delete semantics.
func deleteWithFiles(db *gorm.DB, chatDir, condition string, args ...interface{}) int64 {
	var victims []models.ChatMessage
	if err := db.Where(condition, args...).Find(&victims).Error; err != nil {
		log.Printf("chat: retention scan failed: %v", err)
		return 0
	}
	if len(victims) == 0 {
		return 0
	}
	for _, m := range victims {
		if m.ContentType == "image" && m.FileName != "" {
			// Guard against traversal: only remove files inside the chat dir.
			p := filepath.Join(chatDir, filepath.Base(m.FileName))
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				log.Printf("chat: remove attachment %s failed: %v", p, err)
			}
		}
	}
	tx := db.Where(condition, args...).Delete(&models.ChatMessage{})
	if tx.Error != nil {
		log.Printf("chat: retention delete failed: %v", tx.Error)
		return 0
	}
	return tx.RowsAffected
}

// sweepOrphanFiles deletes image files that no message row references any more.
func sweepOrphanFiles(db *gorm.DB, chatDir string) int64 {
	entries, err := os.ReadDir(chatDir)
	if err != nil {
		return 0 // directory may not exist yet; nothing to sweep
	}
	var removed int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var count int64
		db.Model(&models.ChatMessage{}).Where("file_name = ?", e.Name()).Count(&count)
		if count == 0 {
			if err := os.Remove(filepath.Join(chatDir, e.Name())); err == nil {
				removed++
			}
		}
	}
	return removed
}
