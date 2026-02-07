package core

import (
	"testing"
	"time"
)

func TestPackageNameCacheInvalidateAllPreventsOldCleanup(t *testing.T) {
	cache := &PackageNameCache{
		cache:         make(map[string]cachedPackageName),
		timers:        make(map[string]*time.Timer),
		ttl:           100 * time.Millisecond,
		cleanupBuffer: 0,
	}

	cache.Set("key", "old")

	cache.mu.RLock()
	oldToken := cache.cache["key"].token
	oldTimer := cache.timers["key"]
	cache.mu.RUnlock()

	if oldTimer == nil {
		t.Fatalf("expected a cleanup timer to be scheduled")
	}

	cache.InvalidateAll()
	cache.Set("key", "new")
	defer cache.InvalidateAll()

	// Simulate a stale cleanup firing after the cache reset.
	cache.cleanupEntry("key", oldToken, oldTimer)

	value, ok := cache.Get("key")
	if !ok || value != "new" {
		t.Fatalf("expected new entry to remain after InvalidateAll, got ok=%v value=%q", ok, value)
	}
}
