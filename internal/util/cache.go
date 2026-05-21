package util

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// SearchContext stores the search parameters for a callback.
// FileType 不在此处保存:它由每个按钮的 callback_data 单独承载,
// 类型切换/翻页都从 CBData.FileType 读取,缓存层无需冗余存一份。
type SearchContext struct {
	Query      string
	UserID     uint
	SkipVector bool // /ss 命令置 true,翻页时跳过向量搜索层,只走 FTS+ILIKE
	ExpireAt   time.Time
}

// SearchCache stores search contexts with short IDs to work within
// Telegram's 64-byte callback_data limit.
type SearchCache struct {
	mu    sync.RWMutex
	store map[string]*SearchContext
}

// NewSearchCache creates a new SearchCache instance.
func NewSearchCache() *SearchCache {
	sc := &SearchCache{
		store: make(map[string]*SearchContext),
	}
	// Start a background cleaner
	go sc.cleanupLoop()
	return sc
}

// Set stores a search context and returns a short key.
func (sc *SearchCache) Set(ctx *SearchContext) string {
	key := generateShortID()
	ctx.ExpireAt = time.Now().Add(30 * time.Minute)
	sc.mu.Lock()
	sc.store[key] = ctx
	sc.mu.Unlock()
	return key
}

// Get retrieves a search context by key.
func (sc *SearchCache) Get(key string) *SearchContext {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	ctx, ok := sc.store[key]
	if !ok || time.Now().After(ctx.ExpireAt) {
		return nil
	}
	return ctx
}

// cleanupLoop periodically removes expired entries.
func (sc *SearchCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sc.mu.Lock()
		now := time.Now()
		for k, v := range sc.store {
			if now.After(v.ExpireAt) {
				delete(sc.store, k)
			}
		}
		sc.mu.Unlock()
	}
}

// generateShortID returns a 6-character hex string.
func generateShortID() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return hex.EncodeToString([]byte(time.Now().Format("150405")))[:6]
	}
	return hex.EncodeToString(b)
}
