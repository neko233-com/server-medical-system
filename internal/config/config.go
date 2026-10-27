package config

import "os"

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port        string
	JWTSecret   string
	DBPath      string
	DefaultLang string
	// AllowedOrigins is a comma separated list of CORS allowed origins.
	AllowedOrigins string
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Dev-only default. Override in production via env / secret manager.
		secret = "server-medical-system-dev-secret-change-me"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "medical.db"
	}
	defaultLang := os.Getenv("DEFAULT_LANG")
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = "*"
	}
	return &Config{
		Port:          port,
		JWTSecret:     secret,
		DBPath:        dbPath,
		DefaultLang:   defaultLang,
		AllowedOrigins: origins,
	}
}
