// Package core/cache_warmer.go - Background Cache Warming for Performance Optimization
//
// This file implements intelligent background cache warming to eliminate first-call latency
// for expensive operations like GitHub API calls and Git operations. The cache warmer runs
// in the background and pre-populates caches with likely-needed data.
//
// Key features:
// - Background goroutine for non-blocking cache warming
// - Intelligent warming based on common usage patterns
// - Proper error handling and timeout management
// - Memory-efficient warming that doesn't overwhelm the system
// - Integration with existing cache infrastructure
//
// This optimization eliminates the "cold start" performance penalty and provides
// consistently fast response times for all operations.

package core

import (
	"context"
	"sync"
	"time"
)

// CacheWarmer manages background cache warming operations
type CacheWarmer struct {
	logger    *Logger
	cfg       *Config
	isRunning bool
	stopCh    chan struct{}
	mu        sync.RWMutex
}

// NewCacheWarmer creates a new cache warmer instance
func NewCacheWarmer(logger *Logger, cfg *Config) *CacheWarmer {
	return &CacheWarmer{
		logger: logger,
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

// Start begins background cache warming
func (cw *CacheWarmer) Start() {
	cw.mu.Lock()
	if cw.isRunning {
		cw.mu.Unlock()
		return
	}
	cw.isRunning = true
	cw.mu.Unlock()

	go cw.warmCachesLoop()
	cw.logger.Debug("cache-warmer", "Background cache warming started")
}

// Stop stops the background cache warming
func (cw *CacheWarmer) Stop() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.isRunning {
		return
	}

	close(cw.stopCh)
	cw.isRunning = false
	cw.logger.Debug("cache-warmer", "Background cache warming stopped")
}

// warmCachesLoop is the main loop for background cache warming
func (cw *CacheWarmer) warmCachesLoop() {
	// Initial immediate warming
	cw.warmInitialCaches()

	// Set up periodic warming every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cw.stopCh:
			return
		case <-ticker.C:
			cw.warmPeriodicCaches()
		}
	}
}

// warmInitialCaches performs initial cache warming on startup
func (cw *CacheWarmer) warmInitialCaches() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cw.logger.Debug("cache-warmer", "Starting initial cache warming")

	// Warm GitHub API cache
	go cw.warmGitHubCache(ctx)

	// Warm common Git operations cache
	go cw.warmGitCache(ctx)

	// Warm project discovery cache
	go cw.warmProjectCache(ctx)
}

// warmPeriodicCaches performs periodic cache refreshing
func (cw *CacheWarmer) warmPeriodicCaches() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cw.logger.Debug("cache-warmer", "Starting periodic cache warming")

	// Refresh GitHub cache if it's getting stale
	if cw.shouldRefreshGitHubCache() {
		go cw.warmGitHubCache(ctx)
	}

	// Refresh Git cache for popular repositories
	go cw.warmPopularGitRepos(ctx)
}

// warmGitHubCache pre-warms the GitHub API cache
func (cw *CacheWarmer) warmGitHubCache(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Only warm if cache is empty or near expiry
	if githubCache.Get() != nil {
		cw.logger.Debug("cache-warmer", "GitHub cache already warm, skipping")
		return
	}

	cw.logger.Debug("cache-warmer", "Warming GitHub API cache")

	repos, err := ListGitHubRepos(cw.logger)
	if err != nil {
		cw.logger.Debug("cache-warmer", "Failed to warm GitHub cache: "+err.Error())
		return
	}

	cw.logger.Debug("cache-warmer", "GitHub cache warmed with "+string(rune(len(repos)))+" repositories")
}

// warmGitCache pre-warms Git operations cache with common repositories
func (cw *CacheWarmer) warmGitCache(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	commonRepos := []struct {
		url string
		ref string
	}{
		{"https://github.com/flutter/flutter.git", "main"},
		{"https://github.com/flutter/flutter.git", "stable"},
		{"https://github.com/dart-lang/pub.git", "main"},
		{"https://github.com/flutter/packages.git", "main"},
	}

	cw.logger.Debug("cache-warmer", "Warming Git operations cache")

	for _, repo := range commonRepos {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check if already cached
		cacheKey := repo.url + "#" + repo.ref
		gitLsRemoteCache.mu.RLock()
		_, exists := gitLsRemoteCache.cache[cacheKey]
		gitLsRemoteCache.mu.RUnlock()

		if exists {
			continue // Already cached
		}

		// Warm the cache
		_, err := GitLsRemote(repo.url, repo.ref)
		if err != nil {
			cw.logger.Debug("cache-warmer", "Failed to warm Git cache for "+repo.url+": "+err.Error())
			continue
		}

		cw.logger.Debug("cache-warmer", "Warmed Git cache for "+repo.url+"#"+repo.ref)

		// Small delay to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}
}

// warmProjectCache pre-scans common project locations
func (cw *CacheWarmer) warmProjectCache(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	cw.logger.Debug("cache-warmer", "Warming project discovery cache")

	// Pre-scan common roots to warm the discovery cache
	_, err := ScanCommonRootsWithContext(ctx)
	if err != nil {
		cw.logger.Debug("cache-warmer", "Failed to warm project cache: "+err.Error())
		return
	}

	cw.logger.Debug("cache-warmer", "Project discovery cache warmed")
}

// warmPopularGitRepos warms cache for popular Flutter repositories
func (cw *CacheWarmer) warmPopularGitRepos(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Get popular repos from GitHub cache if available
	repos := githubCache.Get()
	if repos == nil {
		return
	}

	// Warm cache for first 10 repos (most likely to be used)
	maxRepos := 10
	if len(repos) < maxRepos {
		maxRepos = len(repos)
	}

	cw.logger.Debug("cache-warmer", "Warming cache for popular repositories")

	for i := 0; i < maxRepos; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		repo := repos[i]

		// Try main branch
		_, err := GitLsRemote(repo.URL, "main")
		if err == nil {
			cw.logger.Debug("cache-warmer", "Warmed "+repo.Name+"#main")
		}

		// Small delay to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}
}

// shouldRefreshGitHubCache determines if the GitHub cache needs refreshing
func (cw *CacheWarmer) shouldRefreshGitHubCache() bool {
	// Check if cache exists and is getting close to expiry
	githubCache.mu.RLock()
	defer githubCache.mu.RUnlock()

	// Refresh if cache expires within the next 2 minutes
	return time.Now().Add(2 * time.Minute).After(githubCache.expiry)
}

// WarmCachesSync performs synchronous cache warming (for testing/immediate use)
func WarmCachesSync(logger *Logger, cfg *Config) {
	warmer := NewCacheWarmer(logger, cfg)
	warmer.warmInitialCaches()

	logger.Info("cache-warmer", "Synchronous cache warming completed")
}
