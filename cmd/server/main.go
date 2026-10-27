package main

import (
	"io/fs"
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
	"github.com/neko233-com/server-medical-system/web"
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
			// 门诊全流程：预约挂号、病区床位、收费结算。
			handlers.RegisterAppointments(authed, database)
			handlers.RegisterBeds(authed, database)
			handlers.RegisterBilling(authed, database)
		}

		// 院内即时通讯（浮窗聊天）：WS 自鉴权，故注册在 authed 之外。
		handlers.RegisterChat(api, database, cfg, hub)
	}

	// ---- 静态前端：优先使用编译进二进制的 SPA，其次回退到磁盘上的 web/dist ----
	// 生产部署只安装二进制，因此嵌入是主路径；磁盘回退服务于开发期与热更新场景。
	registerSPA(r)

	addr := ":" + cfg.Port
	log.Printf("server-medical-system listening on %s (db=%s)", addr, cfg.DBDriver)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

// registerSPA mounts the single-page application.
//
// Resolution order:
//  1. SPA embedded in the binary (production single-binary deployments)
//  2. ./web/dist on disk (development / after a frontend rebuild)
//
// If neither is available the API still runs and returns a helpful message on
// non-/api routes instead of a bare 404.
func registerSPA(r *gin.Engine) {
	// --- 1. embedded SPA ---
	if spaFS, ok := web.SPA(); ok {
		fileServer := http.FileServer(http.FS(spaFS))
		// Hashed asset filenames live under /assets; serve them from the embedded FS.
		r.GET("/assets/*filepath", gin.WrapH(fileServer))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			// Serve the SPA shell for every client-side route.
			serveIndex(c, spaFS)
		})
		log.Printf("spa: serving embedded frontend")
		return
	}

	// --- 2. on-disk SPA ---
	distDir := "./web/dist"
	if info, statErr := os.Stat(distDir); statErr == nil && info.IsDir() {
		// Only treat the directory as a SPA when it actually holds the entry
		// point — an empty dist/ (e.g. a placeholder-only checkout) must fall
		// through to the API-only branch instead of returning bare 404s.
		if _, idxErr := os.Stat(filepath.Join(distDir, "index.html")); idxErr == nil {
			r.Static("/assets", filepath.Join(distDir, "assets"))
			r.NoRoute(func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.Path, "/api") {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				c.File(filepath.Join(distDir, "index.html"))
			})
			log.Printf("spa: serving frontend from %s", distDir)
			return
		}
	}

	// --- 3. API only ---
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "frontend not bundled",
			"message": "build the frontend first: cd web && npm install && npm run build",
		})
	})
	log.Printf("spa: no frontend found (API only)")
}

// serveIndex writes the SPA entry point (index.html) from an fs.FS.
func serveIndex(c *gin.Context, spaFS fs.FS) {
	data, err := fs.ReadFile(spaFS, "index.html")
	if err != nil {
		c.String(http.StatusServiceUnavailable, "frontend not bundled")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
