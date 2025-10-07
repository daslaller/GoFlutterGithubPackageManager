// Package core benchmark tests - Go standard benchmark suite for performance regression detection
//
// This file provides comprehensive benchmarks for all performance-critical operations
// in the Flutter Package Manager. These benchmarks can be run with `go test -bench=.`
// to detect performance regressions and validate optimizations.
//
// Key benchmarks:
// - BenchmarkProjectDiscovery: Tests concurrent project scanning
// - BenchmarkGitHubAPICaching: Tests GitHub API cache effectiveness
// - BenchmarkGitLsRemoteCaching: Tests Git operation caching
// - BenchmarkPubspecParsing: Tests pubspec.yaml parsing performance
// - BenchmarkStringBuilderRendering: Tests UI rendering optimizations

package core

import (
	"context"
	"strings"
	"testing"
	"time"
)

// BenchmarkProjectDiscovery benchmarks the concurrent project discovery
func BenchmarkProjectDiscovery(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ScanCommonRootsWithContext(ctx)
		if err != nil {
			b.Fatalf("ScanCommonRootsWithContext failed: %v", err)
		}
	}
}

// BenchmarkProjectDiscoverySmall benchmarks discovery with small context timeout
func BenchmarkProjectDiscoverySmall(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_, _ = ScanCommonRootsWithContext(ctx)
		cancel()
	}
}

// BenchmarkGitHubAPICache benchmarks GitHub API caching performance
func BenchmarkGitHubAPICache(b *testing.B) {
	cfg := &Config{Debug: false, Quiet: true}
	logger := NewLogger(cfg)

	// Pre-warm cache
	_, err := ListGitHubRepos(logger)
	if err != nil {
		b.Skip("GitHub API not available, skipping benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ListGitHubRepos(logger)
		if err != nil {
			b.Fatalf("ListGitHubRepos failed: %v", err)
		}
	}
}

// BenchmarkGitHubAPICacheFirst benchmarks first GitHub API call (no cache)
func BenchmarkGitHubAPICacheFirst(b *testing.B) {
	cfg := &Config{Debug: false, Quiet: true}
	logger := NewLogger(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		githubCache.InvalidateCache()
		_, err := ListGitHubRepos(logger)
		if err != nil {
			b.Skip("GitHub API not available, skipping benchmark")
		}
	}
}

// BenchmarkGitLsRemoteCache benchmarks Git ls-remote caching
func BenchmarkGitLsRemoteCache(b *testing.B) {
	url := "https://github.com/flutter/flutter.git"
	ref := "main"

	// Pre-warm cache
	_, err := GitLsRemote(url, ref)
	if err != nil {
		b.Skip("Git ls-remote not available, skipping benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GitLsRemote(url, ref)
		if err != nil {
			b.Fatalf("GitLsRemote failed: %v", err)
		}
	}
}

// BenchmarkGitLsRemoteFirst benchmarks first Git ls-remote call (no cache)
func BenchmarkGitLsRemoteFirst(b *testing.B) {
	url := "https://github.com/flutter/flutter.git"
	ref := "main"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear cache for each iteration
		gitLsRemoteCache.mu.Lock()
		gitLsRemoteCache.cache = make(map[string]string)
		gitLsRemoteCache.mu.Unlock()

		_, err := GitLsRemote(url, ref)
		if err != nil {
			b.Skip("Git ls-remote not available, skipping benchmark")
		}
	}
}

// BenchmarkStaleCheckCache benchmarks stale dependency cache effectiveness
func BenchmarkStaleCheckCache(b *testing.B) {
	cfg := &Config{Debug: false, Quiet: true}
	logger := NewLogger(cfg)

	// Find a test project
	projects, err := ScanCommonRoots()
	if err != nil || len(projects) == 0 {
		b.Skip("No Flutter projects available for testing")
	}

	projectPath := projects[0].Path

	// Pre-warm cache
	_, err = CheckStalePrecise(logger, projectPath)
	if err != nil {
		b.Skip("Stale check not available, skipping benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CheckStalePrecise(logger, projectPath)
		if err != nil {
			b.Fatalf("CheckStalePrecise failed: %v", err)
		}
	}
}

// BenchmarkStringBuilderBasic benchmarks basic string building
func BenchmarkStringBuilderBasic(b *testing.B) {
	lines := []string{
		"Header line 1",
		"Header line 2",
		"Menu item 1",
		"Menu item 2",
		"Menu item 3",
		"Menu item 4",
		"Footer line 1",
		"Footer line 2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		for j, line := range lines {
			if j > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(line)
		}
		_ = builder.String()
	}
}

// BenchmarkStringBuilderPreallocated benchmarks pre-allocated string builder
func BenchmarkStringBuilderPreallocated(b *testing.B) {
	lines := []string{
		"Header line 1",
		"Header line 2",
		"Menu item 1",
		"Menu item 2",
		"Menu item 3",
		"Menu item 4",
		"Footer line 1",
		"Footer line 2",
	}

	var builder strings.Builder
	builder.Grow(256) // Pre-allocate capacity

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.Reset()
		for j, line := range lines {
			if j > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(line)
		}
		_ = builder.String()
	}
}

// BenchmarkStringConcatenation benchmarks naive string concatenation
func BenchmarkStringConcatenation(b *testing.B) {
	lines := []string{
		"Header line 1",
		"Header line 2",
		"Menu item 1",
		"Menu item 2",
		"Menu item 3",
		"Menu item 4",
		"Footer line 1",
		"Footer line 2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := ""
		for j, line := range lines {
			if j > 0 {
				result += "\n"
			}
			result += line
		}
		_ = result
	}
}

// BenchmarkMenuRendering benchmarks optimized menu rendering pattern
func BenchmarkMenuRendering(b *testing.B) {
	// Simulate the menu rendering pattern from MainMenuModel
	menuLines := make([]string, 0, 20)
	var renderBuffer strings.Builder
	renderBuffer.Grow(1024)

	emojis := [4]string{"📁", "🐙", "⚙️", "🔄"}
	options := []string{
		"Scan directories",
		"GitHub repo",
		"Configure search",
		"Check for updates",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset buffers (simulating optimized View() method)
		renderBuffer.Reset()
		menuLines = menuLines[:0]

		// Header
		menuLines = append(menuLines, "🎯 Flutter Package Manager")
		menuLines = append(menuLines, "")
		menuLines = append(menuLines, "📱 Flutter Package Manager - Main Menu:")

		// Menu options
		for j, option := range options {
			line := "  " + string(rune(j+1+'0')) + ". " + emojis[j] + " " + option
			menuLines = append(menuLines, line)
		}

		menuLines = append(menuLines, "")
		menuLines = append(menuLines, "↑/↓ navigate • enter select • q quit")

		// Join efficiently
		for k, line := range menuLines {
			if k > 0 {
				renderBuffer.WriteByte('\n')
			}
			renderBuffer.WriteString(line)
		}

		_ = renderBuffer.String()
	}
}

// BenchmarkCacheWarming benchmarks background cache warming
func BenchmarkCacheWarming(b *testing.B) {
	cfg := &Config{Debug: false, Quiet: true}
	logger := NewLogger(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WarmCachesSync(logger, cfg)
	}
}

// BenchmarkMemoryPoolUsage benchmarks sync.Pool usage for readers
func BenchmarkMemoryPoolUsage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := readerPool.Get()
		readerPool.Put(reader)
	}
}
