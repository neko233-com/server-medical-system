package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neko233-com/server-medical-system/internal/chat"
	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/db"
	"github.com/neko233-com/server-medical-system/internal/handlers"
	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/logging"
	"github.com/neko233-com/server-medical-system/internal/middleware"
)

func main() {
	cfg := config.Load()

	database, err := db.Init(cfg)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	db.Seed(database, cfg.SeedDemo)

	// Ensure the uploads root exists for imaging files.
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("cannot create upload dir: %v", err)
	}

	// ---- 日志：stdout + 自动轮转文件（按大小分割 + 按天保留） ----
	rotator, err := logging.Init(cfg.LogDir, cfg.LogMaxMB, cfg.LogKeepDays)
	if err != nil {
		log.Fatalf("log init failed: %v", err)
	}
	defer rotator.Close()

	// ---- 院内聊天：WebSocket 中枢 + 31 天 LRU 自动回收 ----
	hub := chat.NewHub()
	chat.StartRetention(database, cfg.UploadDir, cfg.ChatRetentionDays, cfg.ChatMaxPerConv, time.Hour)

	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(i18n.Middleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "server-medical-system", "db_driver": cfg.DBDriver})
	})

	api := r.Group("/api")
	{
		handlers.RegisterAuth(api, database, cfg)
		api.GET("/auth/me", middleware.AuthRequired(cfg), func(c *gin.Context) {
			handlers.Me(c, database)
		})

		// Everything below requires a valid token.
		authed := api.Group("")
		authed.Use(middleware.AuthRequired(cfg))
		{
			handlers.RegisterPatients(authed, database)
			handlers.RegisterCases(authed, database)
			handlers.RegisterMedicines(authed, database, cfg)
			handlers.RegisterPrescriptions(authed, database)
			handlers.RegisterRBAC(authed, database)
			handlers.RegisterRooms(authed, database)
			handlers.RegisterPrinters(authed, database)
			handlers.RegisterSurgeries(authed, database)
			handlers.RegisterImaging(authed, database, cfg.UploadDir)
			// 医院运营分析（后台运营 / 全院核算），按 report:read 受控。
			handlers.RegisterOperations(authed, database)
		}

		// 院内即时通讯（浮窗聊天）：WS 自鉴权，故注册在 authed 之外。
		handlers.RegisterChat(api, database, cfg, hub)
	}

	// Optional: serve the built SPA from ./web/dist so the whole system runs
	// from a single binary. In dev, run the Vite server instead (it proxies /api).
	distDir := "./web/dist"
	if info, statErr := os.Stat(distDir); statErr == nil && info.IsDir() {
		r.Static("/assets", filepath.Join(distDir, "assets"))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.File(filepath.Join(distDir, "index.html"))
		})
	}

	addr := ":" + cfg.Port
	log.Printf("server-medical-system listening on %s (db=%s)", addr, cfg.DBDriver)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
