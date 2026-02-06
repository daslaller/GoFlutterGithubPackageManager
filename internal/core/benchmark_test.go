// Package core benchmark tests - Go standard benchmark suite for performance regression detection
//
// This file provides benchmarks for performance-critical operations
// in the Flutter Package Manager. These benchmarks can be run with `go test -bench=.`
// to detect performance regressions and validate optimizations.

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

// BenchmarkGitHubAPI benchmarks GitHub API call performance
func BenchmarkGitHubAPI(b *testing.B) {
	cfg := &Config{Debug: false, Quiet: true}
	logger := NewLogger(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ListGitHubRepos(logger)
		if err != nil {
			b.Skip("GitHub API not available, skipping benchmark")
		}
	}
}

// BenchmarkGitLsRemote benchmarks Git ls-remote performance
func BenchmarkGitLsRemote(b *testing.B) {
	url := "https://github.com/flutter/flutter.git"
	ref := "main"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GitLsRemote(url, ref)
		if err != nil {
			b.Skip("Git ls-remote not available, skipping benchmark")
		}
	}
}

// BenchmarkPubspecParsing benchmarks pubspec.lock parsing performance
func BenchmarkPubspecParsing(b *testing.B) {
	cfg := &Config{Debug: false, Quiet: true}
	logger := NewLogger(cfg)

	// Find a test project
	projects, err := ScanCommonRoots()
	if err != nil || len(projects) == 0 {
		b.Skip("No Flutter projects available for testing")
	}

	projectPath := projects[0].Path

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CheckStalePrecise(logger, projectPath)
		if err != nil {
			b.Skip("Stale check not available, skipping benchmark")
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
		renderBuffer.Reset()
		menuLines = menuLines[:0]

		menuLines = append(menuLines, "🎯 Flutter Package Manager")
		menuLines = append(menuLines, "")
		menuLines = append(menuLines, "📱 Flutter Package Manager - Main Menu:")

		for j, option := range options {
			line := "  " + string(rune(j+1+'0')) + ". " + emojis[j] + " " + option
			menuLines = append(menuLines, line)
		}

		menuLines = append(menuLines, "")
		menuLines = append(menuLines, "↑/↓ navigate • enter select • q quit")

		for k, line := range menuLines {
			if k > 0 {
				renderBuffer.WriteByte('\n')
			}
			renderBuffer.WriteString(line)
		}

		_ = renderBuffer.String()
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
