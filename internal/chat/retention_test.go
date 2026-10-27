package chat

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/neko233-com/server-medical-system/internal/models"
)

// newTestDB spins up an in-memory database with only the chat table migrated.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.ChatMessage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// seed inserts a message with an explicit age, bypassing GORM's automatic
// timestamp so we can test the retention window deterministically.
func seed(t *testing.T, db *gorm.DB, scope, groupKey, peerKey string, age time.Duration, content string) models.ChatMessage {
	t.Helper()
	m := models.ChatMessage{
		Scope:       scope,
		GroupKey:    groupKey,
		PeerKey:     peerKey,
		SenderID:    1,
		SenderName:  "tester",
		ContentType: "text",
		Content:     content,
		CreatedAt:   time.Now().Add(-age),
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return m
}

func count(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&models.ChatMessage{}).Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// TestPurgeRemovesExpired verifies the 31-day sliding window: anything older
// than the retention window is gone, newer rows survive.
func TestPurgeRemovesExpired(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	seed(t, db, "group", GroupKeyHospital, "", 40*24*time.Hour, "ancient")
	seed(t, db, "group", GroupKeyHospital, "", 10*24*time.Hour, "recent")

	if n := count(t, db); n != 2 {
		t.Fatalf("expected 2 seeded rows, got %d", n)
	}

	purge(db, dir, DefaultRetentionDays, 0)

	if n := count(t, db); n != 1 {
		t.Fatalf("expected only the recent row to survive, got %d", n)
	}
	var remain models.ChatMessage
	db.First(&remain)
	if remain.Content != "recent" {
		t.Fatalf("wrong row survived: %q", remain.Content)
	}
}

// TestTrimKeepsNewest verifies LRU capacity: only the newest `max` messages
// remain per conversation.
func TestTrimKeepsNewest(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	for i := 0; i < 10; i++ {
		seed(t, db, "group", GroupKeyHospital, "", time.Duration(i)*time.Minute, "msg")
	}
	Trim(db, dir, 4)

	if n := count(t, db); n != 4 {
		t.Fatalf("expected 4 rows after LRU trim, got %d", n)
	}
	var oldest models.ChatMessage
	db.Order("id asc").First(&oldest)
	if oldest.ID != 7 {
		t.Fatalf("expected oldest surviving id 7, got %d", oldest.ID)
	}
}

// TestTrimCoversPrivateThreads ensures private conversations are also limited,
// not just the hospital group.
func TestTrimCoversPrivateThreads(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()

	for i := 0; i < 6; i++ {
		seed(t, db, "direct", "", PeerKeyFor(1, 2), time.Duration(i)*time.Minute, "dm")
	}
	Trim(db, dir, 2)

	var n int64
	db.Model(&models.ChatMessage{}).Where("scope = ? AND peer_key = ?", "direct", PeerKeyFor(1, 2)).Count(&n)
	if n != 2 {
		t.Fatalf("expected private thread trimmed to 2, got %d", n)
	}
}

// TestRetentionRemovesAttachments verifies image files are deleted together with
// their message rows so uploads never leak disk space.
func TestRetentionRemovesAttachments(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	chatDir := filepath.Join(dir, "chat")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fpath := filepath.Join(chatDir, "pic.png")
	if err := os.WriteFile(fpath, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := models.ChatMessage{
		Scope:       "group",
		GroupKey:    GroupKeyHospital,
		ContentType: "image",
		FileName:    "pic.png",
		CreatedAt:   time.Now().Add(-60 * 24 * time.Hour),
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}

	purge(db, chatDir, DefaultRetentionDays, 0)

	if count(t, db) != 0 {
		t.Fatalf("expected row removed")
	}
	if _, err := os.Stat(fpath); !os.IsNotExist(err) {
		t.Fatalf("attachment should have been deleted, err=%v", err)
	}
}

// TestSweepOrphanFiles verifies unreferenced files are reclaimed.
func TestSweepOrphanFiles(t *testing.T) {
	db := newTestDB(t)
	chatDir := filepath.Join(t.TempDir(), "chat")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(chatDir, "orphan.png")
	if err := os.WriteFile(orphan, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := sweepOrphanFiles(db, chatDir); n != 1 {
		t.Fatalf("expected 1 orphan removed, got %d", n)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan file still present")
	}
}

// TestPeerKeySymmetry ensures A→B and B→A map to one conversation.
func TestPeerKeySymmetry(t *testing.T) {
	if PeerKeyFor(2, 1) != PeerKeyFor(1, 2) {
		t.Fatalf("peer key must be order independent")
	}
	if PeerKeyFor(1, 2) != "1-2" {
		t.Fatalf("unexpected key: %s", PeerKeyFor(1, 2))
	}
}
