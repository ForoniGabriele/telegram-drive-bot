package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Bot         BotConfig
	DB          DBConfig
	Storage     StorageConfig
	Security    SecurityConfig
	Embedding   EmbeddingConfig
	Maintenance MaintenanceConfig
}

type BotConfig struct {
	Token    string
	ProxyURL string // Optional. SOCKS5 (socks5://host:port) or HTTP (http://host:port) proxy for Telegram API.
}

// 数据库连接参数调优
// 默认参数是为supabase 免费层级的15个连接调的
type DBConfig struct {
	URL             string
	MaxOpenConns    int           // Hard cap on open connections (default: 10)
	MaxIdleConns    int           // Connections kept warm in the pool (default: 5)
	ConnMaxLifetime time.Duration // Hard recycle age; defends against stale TCP (default: 30m)
	ConnMaxIdleTime time.Duration // Close after this much idle time (default: 5m)
}

// StorageConfig covers the file-storage strategy and its tunables.
type StorageConfig struct {
	UseChannel     bool    // If true, forward files to storage channels; if false, use file_id directly.
	DeleteCopies   bool    // If true, DeleteFile also removes channel copies (best-effort, failures logged).
	AsyncForward   bool    // If true, forward to storage channels asynchronously after DB write.
	AsyncDelete    bool    // If true, delete from storage channels asynchronously after DB commit.
	ChatIDs        []int64 // Ordered list of storage channels; first is primary, rest are redundant copies.
	FailureLimit   int     // Consecutive failures before a channel is temporarily disabled.
	CooldownPeriod time.Duration
}

// SecurityConfig covers access control.
type SecurityConfig struct {
	OwnerID int64 // Telegram user_id of the initial owner/super-admin.
}

// EmbeddingConfig covers the optional vector search feature.
// When Enabled is false the remaining fields are ignored and vector search is a no-op.
type EmbeddingConfig struct {
	Enabled    bool    // VECTOR_SEARCH_ENABLED
	APIType    string  // EMBEDDING_API_TYPE — "openai" or "gemini"
	BaseURL    string  // EMBEDDING_BASE_URL — embedding API base URL
	APIKey     string  // EMBEDDING_API_KEY
	Model      string  // EMBEDDING_MODEL  — e.g. "text-embedding-3-small" or "gemini-embedding-2"
	Dimensions int     // EMBEDDING_DIMENSIONS — vector dimensionality (default 256)
	Threshold  float64 // EMBEDDING_THRESHOLD — max cosine distance for vector search
}

// MaintenanceConfig 控制后台维护任务的执行节奏
// 目前只有 cap_sync 一个任务,后续如果加 emb_sync 等定时任务再往这里扩
type MaintenanceConfig struct {
	// CapSyncInterval 控制定时 caption 同步的周期
	// 0 表示禁用后台任务(此时仍可通过 /cap_sync 手动触发)
	// 由 CAP_SYNC_INTERVAL 解析,空串视为禁用
	CapSyncInterval time.Duration
}

// Default values for tunables not exposed as env vars today.
const (
	defaultFailureLimit   = 3
	defaultCooldownPeriod = 5 * time.Minute

	defaultDBMaxOpenConns    = 10
	defaultDBMaxIdleConns    = 5
	defaultDBConnMaxLifetime = 30 * time.Minute
	defaultDBConnMaxIdleTime = 5 * time.Minute

	defaultEmbeddingDimensions = 256
	defaultEmbeddingThreshold  = 0.4
)

// 首先尝试从工作目录中加载 .env 文件
// 如果不存在 .env 文件，则必须直接设置环境变量
func Load() (*Config, error) {
	_ = godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	ownerIDStr := os.Getenv("OWNER_ID")
	ownerID, err := strconv.ParseInt(ownerIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("OWNER_ID must be a valid integer: %w", err)
	}

	useChannelStorage := true
	if v := os.Getenv("USE_CHANNEL_STORAGE"); v != "" {
		useChannelStorage = strings.EqualFold(v, "true") || v == "1"
	}

	deleteChannelCopies := true
	if v := os.Getenv("DELETE_CHANNEL_COPIES"); v != "" {
		deleteChannelCopies = strings.EqualFold(v, "true") || v == "1"
	}

	storageChatIDs, err := parseStorageChatIDs()
	if err != nil {
		return nil, err
	}
	if useChannelStorage && len(storageChatIDs) == 0 {
		return nil, fmt.Errorf("STORAGE_CHAT_IDS is required when USE_CHANNEL_STORAGE is enabled")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	failureLimit, err := parseIntEnv("STORAGE_FAILURE_LIMIT", defaultFailureLimit)
	if err != nil {
		return nil, err
	}
	cooldown, err := parseDurationEnv("STORAGE_COOLDOWN_PERIOD", defaultCooldownPeriod)
	if err != nil {
		return nil, err
	}

	asyncForward := false
	if v := os.Getenv("ASYNC_FORWARD"); v != "" {
		asyncForward = strings.EqualFold(v, "true") || v == "1"
	}
	if asyncForward && !useChannelStorage {
		asyncForward = false // only meaningful in channel mode
	}

	asyncDelete := false
	if v := os.Getenv("ASYNC_DELETE"); v != "" {
		asyncDelete = strings.EqualFold(v, "true") || v == "1"
	}
	if asyncDelete && (!useChannelStorage || !deleteChannelCopies) {
		asyncDelete = false // requires channel mode with delete-copies enabled
	}

	dbMaxOpen, err := parseIntEnv("DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns)
	if err != nil {
		return nil, err
	}
	dbMaxIdle, err := parseIntEnv("DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns)
	if err != nil {
		return nil, err
	}
	dbConnLifetime, err := parseDurationEnv("DB_CONN_MAX_LIFETIME", defaultDBConnMaxLifetime)
	if err != nil {
		return nil, err
	}
	dbConnIdleTime, err := parseDurationEnv("DB_CONN_MAX_IDLE_TIME", defaultDBConnMaxIdleTime)
	if err != nil {
		return nil, err
	}
	if dbMaxIdle > dbMaxOpen {
		return nil, fmt.Errorf("DB_MAX_IDLE_CONNS (%d) must not exceed DB_MAX_OPEN_CONNS (%d)", dbMaxIdle, dbMaxOpen)
	}

	embCfg, err := loadEmbeddingConfig()
	if err != nil {
		return nil, err
	}

	// CAP_SYNC_INTERVAL 未设 = 0 = 关闭后台任务
	// 设了正值 = 后台 ticker 周期; parseDurationEnv 已经拒绝 <= 0 的显式输入
	capSyncInterval, err := parseDurationEnv("CAP_SYNC_INTERVAL", 0)
	if err != nil {
		return nil, err
	}

	return &Config{
		Bot: BotConfig{
			Token:    botToken,
			ProxyURL: os.Getenv("PROXY_URL"),
		},
		DB: DBConfig{
			URL:             databaseURL,
			MaxOpenConns:    dbMaxOpen,
			MaxIdleConns:    dbMaxIdle,
			ConnMaxLifetime: dbConnLifetime,
			ConnMaxIdleTime: dbConnIdleTime,
		},
		Storage: StorageConfig{
			UseChannel:     useChannelStorage,
			DeleteCopies:   deleteChannelCopies,
			AsyncForward:   asyncForward,
			AsyncDelete:    asyncDelete,
			ChatIDs:        storageChatIDs,
			FailureLimit:   failureLimit,
			CooldownPeriod: cooldown,
		},
		Security: SecurityConfig{
			OwnerID: ownerID,
		},
		Embedding: embCfg,
		Maintenance: MaintenanceConfig{
			CapSyncInterval: capSyncInterval,
		},
	}, nil
}

// parseStorageChatIDs reads STORAGE_CHAT_IDS (comma-separated).
// Returns an ordered, deduplicated list. First entry is treated as the primary write target.
func parseStorageChatIDs() ([]int64, error) {
	raw := strings.TrimSpace(os.Getenv("STORAGE_CHAT_IDS"))
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	seen := make(map[int64]struct{}, len(parts))
	ids := make([]int64, 0, len(parts))
	for i, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chat_id at position %d in STORAGE_CHAT_IDS: %q: %w", i+1, s, err)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

// parseIntEnv reads an integer env var; returns fallback when unset.
func parseIntEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be > 0", key)
	}
	return n, nil
}

// parseFloatEnv reads a float env var; returns fallback when unset.
func parseFloatEnv(key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a float: %w", key, err)
	}
	return f, nil
}

// parseDurationEnv reads a Go duration (e.g. "5m", "30s"); returns fallback when unset.
func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a Go duration (e.g. 5m, 30s): %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be > 0", key)
	}
	return d, nil
}

// loadEmbeddingConfig reads the optional vector-search environment variables.
// When VECTOR_SEARCH_ENABLED is false (default), only Enabled is populated.
func loadEmbeddingConfig() (EmbeddingConfig, error) {
	enabled := false
	if v := os.Getenv("VECTOR_SEARCH_ENABLED"); v != "" {
		enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if !enabled {
		return EmbeddingConfig{}, nil
	}

	baseURL := strings.TrimSpace(os.Getenv("EMBEDDING_BASE_URL"))
	if baseURL == "" {
		return EmbeddingConfig{}, fmt.Errorf("EMBEDDING_BASE_URL is required when VECTOR_SEARCH_ENABLED=true")
	}
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		return EmbeddingConfig{}, fmt.Errorf("EMBEDDING_API_KEY is required when VECTOR_SEARCH_ENABLED=true")
	}
	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		return EmbeddingConfig{}, fmt.Errorf("EMBEDDING_MODEL is required when VECTOR_SEARCH_ENABLED=true")
	}

	apiType := strings.ToLower(strings.TrimSpace(os.Getenv("EMBEDDING_API_TYPE")))
	if apiType == "" {
		apiType = "openai" // default
	}
	if apiType != "openai" && apiType != "gemini" {
		return EmbeddingConfig{}, fmt.Errorf("EMBEDDING_API_TYPE must be \"openai\" or \"gemini\", got %q", apiType)
	}

	dimensions, err := parseIntEnv("EMBEDDING_DIMENSIONS", defaultEmbeddingDimensions)
	if err != nil {
		return EmbeddingConfig{}, err
	}

	threshold, err := parseFloatEnv("EMBEDDING_THRESHOLD", defaultEmbeddingThreshold)
	if err != nil {
		return EmbeddingConfig{}, err
	}

	return EmbeddingConfig{
		Enabled:    true,
		APIType:    apiType,
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      model,
		Dimensions: dimensions,
		Threshold:  threshold,
	}, nil
}
