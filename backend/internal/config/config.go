// Package config loads runtime configuration from a .env file and/or
// environment variables. Values already present in the process environment
// always take precedence over the .env file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime settings for the API server.
type Config struct {
	ServerHost  string
	ServerPort  int
	CORSOrigins []string

	// APIKey protects write (POST/PUT/DELETE) endpoints. When empty, the API
	// key check is disabled (intended for local development only).
	APIKey string

	// EnableEPUBUpload toggles the EPUB upload endpoint. Disabled by default.
	EnableEPUBUpload bool

	DatabaseURL string
	DBMaxConns  int32
	DBMinConns  int32

	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UsePathStyle    bool
	S3PublicBaseURL   string

	MaxEPUBUploadBytes     int64
	MaxArticlePayloadBytes int64

	// JMDictPath is the path to a jmdict-simplified JSON dictionary archive,
	// used by the import-jmdict command when -file is not supplied.
	JMDictPath string
}

// Load reads configuration from the environment, applying sane defaults.
// It first attempts to load a .env file (without overriding variables that are
// already set in the environment), searching the current directory and its
// parents.
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		ServerHost:  env("SERVER_HOST", "0.0.0.0"),
		ServerPort:  envInt("SERVER_PORT", 8080),
		CORSOrigins: splitCSV(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173,chrome-extension://*")),

		APIKey:           env("API_KEY", ""),
		EnableEPUBUpload: envBool("ENABLE_EPUB_UPLOAD", false),

		DatabaseURL: env("DATABASE_URL", ""),
		DBMaxConns:  int32(envInt("DB_MAX_CONNS", 20)),
		DBMinConns:  int32(envInt("DB_MIN_CONNS", 2)),

		S3Endpoint:        env("S3_ENDPOINT", ""),
		S3Region:          env("S3_REGION", "us-east-1"),
		S3Bucket:          env("S3_BUCKET", "jp-scrapper"),
		S3AccessKeyID:     env("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey: env("S3_SECRET_ACCESS_KEY", ""),
		S3UsePathStyle:    envBool("S3_USE_PATH_STYLE", false),
		S3PublicBaseURL:   env("S3_PUBLIC_BASE_URL", ""),

		MaxEPUBUploadBytes:     int64(envInt("MAX_EPUB_UPLOAD_BYTES", 50*1024*1024)),
		MaxArticlePayloadBytes: int64(envInt("MAX_ARTICLE_PAYLOAD_BYTES", 5*1024*1024)),

		JMDictPath: env("JMDICT_PATH", ""),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set (via .env or environment)")
	}
	return cfg, nil
}

// loadDotEnv searches for a .env file starting at the current working directory
// and walking up to the filesystem root. If found, its values are loaded
// without overriding any variables already set in the environment.
func loadDotEnv() {
	explicit := os.Getenv("ENV_FILE")
	if explicit != "" {
		_ = godotenv.Load(explicit)
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		candidate := filepath.Join(dir, ".env")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			_ = godotenv.Load(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return // reached filesystem root
		}
		dir = parent
	}
}

// Addr returns the host:port the HTTP server should bind to.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return b
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
