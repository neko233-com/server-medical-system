package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/neko233-com/server-medical-system/internal/config"
	"github.com/neko233-com/server-medical-system/internal/db"
	"github.com/neko233-com/server-medical-system/internal/handlers"
	"github.com/neko233-com/server-medical-system/internal/i18n"
	"github.com/neko233-com/server-medical-system/internal/middleware"
)

func main() {
	cfg := config.Load()

	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	db.Seed(database)

	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(i18n.Middleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "server-medical-system"})
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
		}
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
	log.Printf("server-medical-system listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
