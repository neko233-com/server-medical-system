package config

import "os"

// The default HTTP port for the medical system.
// Historically this was 8080, which very often collides with proxies,
// dev servers and sandbox tooling; 6970 was chosen as an uncommon,
// clearly-owned default for this product.
const DefaultPort = "6970"

// Config holds runtime configuration loaded from environment variables.
// Every field is overridable from the environment so that deployments can
// stay configuration-free (embedded SQLite + defaults) or be fully externalised.
type Config struct {
	// Port the HTTP server listens on (default: 6970).
	Port string
	// JWTSecret signs authentication tokens. MUST be overridden in production.
	JWTSecret string
	// DBDriver selects the database backend: sqlite (default) | postgres | mysql.
	DBDriver string
	// DBPath is the SQLite file location (used only when DBDriver == sqlite).
	DBPath string
	// DBDSN is the connection string for Postgres / MySQL deployments.
	DBDSN string
	// DefaultLang is the fallback UI language when no Accept-Language is sent.
	DefaultLang string
	// AllowedOrigins is a comma separated list of CORS allowed origins.
	AllowedOrigins string
	// UploadDir is the root directory for uploaded imaging files (CT / X-ray).
	UploadDir string
	// SeedDemo controls whether demo users / rooms / printers are seeded.
	SeedDemo bool
	// LogDir is where rotating application logs are written.
	LogDir string
	// LogMaxMB is the per-file size threshold that triggers rotation.
	LogMaxMB int
	// LogKeepDays is how many days of rotated logs are retained.
	LogKeepDays int
	// ChatRetentionDays is how long chat history is kept (0 = use default 31).
	ChatRetentionDays int
	// ChatMaxPerConv caps messages kept per conversation (0 = use default).
	ChatMaxPerConv int
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Dev-only default. Override in production via env / secret manager.
		secret = "server-medical-system-dev-secret-change-me"
	}
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = "sqlite"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "medical.db"
	}
	dbDSN := os.Getenv("DB_DSN")
	defaultLang := os.Getenv("DEFAULT_LANG")
	if defaultLang == "" {
		defaultLang = "zh-CN"
	}
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = "*"
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	seedDemo := os.Getenv("SEED_DEMO") != "false" // default true
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "./logs"
	}

	return &Config{
		Port:              port,
		JWTSecret:         secret,
		DBDriver:          driver,
		DBPath:            dbPath,
		DBDSN:             dbDSN,
		DefaultLang:       defaultLang,
		AllowedOrigins:    origins,
		UploadDir:         uploadDir,
		SeedDemo:          seedDemo,
		LogDir:            logDir,
		LogMaxMB:          envInt("LOG_MAX_MB", 64),
		LogKeepDays:       envInt("LOG_KEEP_DAYS", 31),
		ChatRetentionDays: envInt("CHAT_RETENTION_DAYS", 31),
		ChatMaxPerConv:    envInt("CHAT_MAX_PER_CONVERSATION", 0),
	}
}

// envInt parses a positive integer environment variable, falling back to def
// when unset or malformed. A bad number must never abort startup.
func envInt(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n < 0 {
		return def
	}
	return n
}
