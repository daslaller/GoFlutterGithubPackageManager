package core

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// BenchmarkResult represents the result of a performance benchmark
type BenchmarkResult struct {
	Operation  string        `json:"operation"`
	Duration   time.Duration `json:"duration"`
	MemoryUsed int64         `json:"memory_used"`
	AllocCount int64         `json:"alloc_count"`
	GCCount    int64         `json:"gc_count"`
	CPUTime    time.Duration `json:"cpu_time"`
	Throughput float64       `json:"throughput,omitempty"` // Operations per second
	ErrorMsg   string        `json:"error,omitempty"`
}

// PerformanceBenchmark provides performance testing capabilities
type PerformanceBenchmark struct {
	logger  *Logger
	results []BenchmarkResult
}

// NewPerformanceBenchmark creates a new benchmark instance
func NewPerformanceBenchmark(logger *Logger) *PerformanceBenchmark {
	return &PerformanceBenchmark{
		logger:  logger,
		results: make([]BenchmarkResult, 0),
	}
}

// BenchmarkProjectDiscovery benchmarks the project discovery optimization
func (pb *PerformanceBenchmark) BenchmarkProjectDiscovery() BenchmarkResult {
	pb.logger.Info("benchmark", "Starting project discovery benchmark")

	start := time.Now()
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	gcBefore := memBefore.NumGC

	// Run the optimized project discovery
	projects, err := ScanCommonRoots()

	duration := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	gcAfter := memAfter.NumGC

	result := BenchmarkResult{
		Operation:  "ProjectDiscovery",
		Duration:   duration,
		MemoryUsed: int64(memAfter.Alloc - memBefore.Alloc),
		AllocCount: int64(memAfter.Mallocs - memBefore.Mallocs),
		GCCount:    int64(gcAfter - gcBefore),
		CPUTime:    duration, // Approximation for single-threaded workload
	}

	if err != nil {
		result.ErrorMsg = err.Error()
	} else {
		// Calculate throughput (projects found per second)
		if duration.Seconds() > 0 {
			result.Throughput = float64(len(projects)) / duration.Seconds()
		}
		pb.logger.Info("benchmark", fmt.Sprintf("Found %d projects in %v", len(projects), duration))
	}

	pb.results = append(pb.results, result)
	return result
}

// BenchmarkGitHubAPI benchmarks the GitHub API caching
func (pb *PerformanceBenchmark) BenchmarkGitHubAPI() BenchmarkResult {
	pb.logger.Info("benchmark", "Starting GitHub API benchmark")

	start := time.Now()
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	gcBefore := memBefore.NumGC

	// Clear cache first to ensure fair test
	githubCache.InvalidateCache()

	// First call (should hit GitHub API)
	repos1, err1 := ListGitHubRepos(pb.logger)
	middleTime := time.Since(start)

	// Second call (should hit cache)
	repos2, err2 := ListGitHubRepos(pb.logger)

	duration := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	gcAfter := memAfter.NumGC

	result := BenchmarkResult{
		Operation:  "GitHubAPI",
		Duration:   duration,
		MemoryUsed: int64(memAfter.Alloc - memBefore.Alloc),
		AllocCount: int64(memAfter.Mallocs - memBefore.Mallocs),
		GCCount:    int64(gcAfter - gcBefore),
		CPUTime:    duration,
	}

	if err1 != nil || err2 != nil {
		if err1 != nil {
			result.ErrorMsg = fmt.Sprintf("First call error: %v", err1)
		} else {
			result.ErrorMsg = fmt.Sprintf("Second call error: %v", err2)
		}
	} else {
		// Calculate cache effectiveness
		cacheSpeedup := float64(middleTime.Nanoseconds()) / float64((duration - middleTime).Nanoseconds())
		pb.logger.Info("benchmark", fmt.Sprintf("GitHub API: %d repos, cache speedup: %.2fx", len(repos1), cacheSpeedup))

		if duration.Seconds() > 0 {
			result.Throughput = float64(len(repos1)+len(repos2)) / duration.Seconds()
		}
	}

	pb.results = append(pb.results, result)
	return result
}

// BenchmarkPubspecParsing benchmarks the regex-optimized pubspec parsing
func (pb *PerformanceBenchmark) BenchmarkPubspecParsing(projectPath string) BenchmarkResult {
	pb.logger.Info("benchmark", "Starting pubspec parsing benchmark")

	start := time.Now()
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	gcBefore := memBefore.NumGC

	// Run multiple iterations to get meaningful measurements
	iterations := 100
	var totalDeps int

	for i := 0; i < iterations; i++ {
		deps, err := ListGitDependencies(projectPath)
		if err != nil {
			result := BenchmarkResult{
				Operation: "PubspecParsing",
				Duration:  time.Since(start),
				ErrorMsg:  err.Error(),
			}
			pb.results = append(pb.results, result)
			return result
		}
		totalDeps = len(deps)
	}

	duration := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	gcAfter := memAfter.NumGC

	result := BenchmarkResult{
		Operation:  "PubspecParsing",
		Duration:   duration,
		MemoryUsed: int64(memAfter.Alloc - memBefore.Alloc),
		AllocCount: int64(memAfter.Mallocs - memBefore.Mallocs),
		GCCount:    int64(gcAfter - gcBefore),
		CPUTime:    duration,
	}

	// Calculate throughput (parsing operations per second)
	if duration.Seconds() > 0 {
		result.Throughput = float64(iterations) / duration.Seconds()
	}

	pb.logger.Info("benchmark", fmt.Sprintf("Pubspec parsing: %d iterations, %d deps found, %v total", iterations, totalDeps, duration))

	pb.results = append(pb.results, result)
	return result
}

// BenchmarkStaleCheck benchmarks the cached stale dependency checking
func (pb *PerformanceBenchmark) BenchmarkStaleCheck(projectPath string) BenchmarkResult {
	pb.logger.Info("benchmark", "Starting stale check benchmark")

	start := time.Now()
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	gcBefore := memBefore.NumGC

	// Clear cache first
	staleCache.InvalidateProject(projectPath)

	// First call (should do full check)
	info1, err1 := CheckStalePrecise(pb.logger, projectPath)
	middleTime := time.Since(start)

	// Second call (should hit cache)
	info2, err2 := CheckStalePrecise(pb.logger, projectPath)

	duration := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	gcAfter := memAfter.NumGC

	result := BenchmarkResult{
		Operation:  "StaleCheck",
		Duration:   duration,
		MemoryUsed: int64(memAfter.Alloc - memBefore.Alloc),
		AllocCount: int64(memAfter.Mallocs - memBefore.Mallocs),
		GCCount:    int64(gcAfter - gcBefore),
		CPUTime:    duration,
	}

	if err1 != nil || err2 != nil {
		if err1 != nil {
			result.ErrorMsg = fmt.Sprintf("First call error: %v", err1)
		} else {
			result.ErrorMsg = fmt.Sprintf("Second call error: %v", err2)
		}
	} else {
		// Calculate cache effectiveness
		if middleTime > 0 && (duration-middleTime) > 0 {
			cacheSpeedup := float64(middleTime.Nanoseconds()) / float64((duration - middleTime).Nanoseconds())
			pb.logger.Info("benchmark", fmt.Sprintf("Stale check: %d packages, cache speedup: %.2fx", len(info1), cacheSpeedup))
		}

		if duration.Seconds() > 0 {
			result.Throughput = float64(len(info1)+len(info2)) / duration.Seconds()
		}
	}

	pb.results = append(pb.results, result)
	return result
}

// RunFullBenchmark runs all benchmarks and returns a summary
func (pb *PerformanceBenchmark) RunFullBenchmark(projectPath string) []BenchmarkResult {
	pb.logger.Info("benchmark", "Starting full performance benchmark suite")

	// Force garbage collection before starting
	runtime.GC()
	debug.FreeOSMemory()

	var results []BenchmarkResult

	// Run benchmarks
	results = append(results, pb.BenchmarkProjectDiscovery())
	runtime.GC() // Clean up between benchmarks

	results = append(results, pb.BenchmarkGitHubAPI())
	runtime.GC()

	if projectPath != "" {
		results = append(results, pb.BenchmarkPubspecParsing(projectPath))
		runtime.GC()

		results = append(results, pb.BenchmarkStaleCheck(projectPath))
		runtime.GC()
	}

	// Calculate total performance metrics
	var totalDuration time.Duration
	var totalMemory int64
	var totalAllocs int64

	for _, result := range results {
		totalDuration += result.Duration
		totalMemory += result.MemoryUsed
		totalAllocs += result.AllocCount
	}

	pb.logger.Info("benchmark", fmt.Sprintf("Benchmark completed: %d operations, %v total duration, %d bytes allocated",
		len(results), totalDuration, totalMemory))

	return results
}

// GetResults returns all benchmark results
func (pb *PerformanceBenchmark) GetResults() []BenchmarkResult {
	return pb.results
}

// FormatResults returns a human-readable summary of benchmark results
func (pb *PerformanceBenchmark) FormatResults() string {
	if len(pb.results) == 0 {
		return "No benchmark results available"
	}

	var summary strings.Builder
	summary.WriteString("\n=== Performance Benchmark Results ===\n\n")

	for _, result := range pb.results {
		summary.WriteString(fmt.Sprintf("Operation: %s\n", result.Operation))
		summary.WriteString(fmt.Sprintf("  Duration: %v\n", result.Duration))
		summary.WriteString(fmt.Sprintf("  Memory Used: %d bytes\n", result.MemoryUsed))
		summary.WriteString(fmt.Sprintf("  Allocations: %d\n", result.AllocCount))
		summary.WriteString(fmt.Sprintf("  GC Count: %d\n", result.GCCount))

		if result.Throughput > 0 {
			summary.WriteString(fmt.Sprintf("  Throughput: %.2f ops/sec\n", result.Throughput))
		}

		if result.ErrorMsg != "" {
			summary.WriteString(fmt.Sprintf("  Error: %s\n", result.ErrorMsg))
		}

		summary.WriteString("\n")
	}

	return summary.String()
}
