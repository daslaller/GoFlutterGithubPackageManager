// Package scripts/run_terminal_tests.go - Terminal Output Test Runner
//
// This script runs comprehensive terminal output tests and generates
// detailed reports showing actual terminal frames for analysis.

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestResult represents the result of a single test
type TestResult struct {
	TestName   string
	Passed     bool
	Output     string
	Screenshot string
	Duration   time.Duration
	Error      string
}

// TestRunner executes and manages terminal output tests
type TestRunner struct {
	results      []TestResult
	outputDir    string
	verbose      bool
	captureFiles bool
}

// NewTestRunner creates a new test runner
func NewTestRunner(outputDir string, verbose bool) *TestRunner {
	return &TestRunner{
		results:      make([]TestResult, 0),
		outputDir:    outputDir,
		verbose:      verbose,
		captureFiles: true,
	}
}

func main() {
	fmt.Println("🧪 Flutter Package Manager - Terminal Output Test Runner")
	fmt.Println("========================================================")

	// Create output directory for test artifacts
	outputDir := "test_outputs"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	runner := NewTestRunner(outputDir, true)

	// Run all terminal output tests
	runner.RunAllTests()

	// Generate test report
	runner.GenerateReport()

	// Show summary
	runner.ShowSummary()
}

// RunAllTests executes all terminal output tests
func (r *TestRunner) RunAllTests() {
	tests := []string{
		"TestMainMenuDisplay",
		"TestConfigureSearchOption",
		"TestSourceConfigurationFlow",
		"TestAllMenuOptions",
		"TestPackageSelectionUI",
	}

	fmt.Printf("\n🚀 Running %d terminal output tests...\n\n", len(tests))

	for _, testName := range tests {
		r.RunSingleTest(testName)
	}

	// Run the screen capture test that saves all menu outputs
	r.RunScreenCaptureTest()
}

// RunSingleTest executes a single test and captures results
func (r *TestRunner) RunSingleTest(testName string) {
	fmt.Printf("📋 Running: %s\n", testName)
	start := time.Now()

	// Execute the test using go test
	cmd := exec.Command("go", "test", "-v", "-run", testName, "./internal/tui/testing")
	cmd.Dir = "./" // Run from project root

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := TestResult{
		TestName: testName,
		Passed:   err == nil,
		Output:   string(output),
		Duration: duration,
	}

	if err != nil {
		result.Error = err.Error()
		fmt.Printf("   ❌ FAILED (%v): %s\n", duration, err.Error())
	} else {
		fmt.Printf("   ✅ PASSED (%v)\n", duration)
	}

	if r.verbose {
		fmt.Printf("   Output: %s\n", strings.TrimSpace(string(output)))
	}

	// Save test output to file
	outputFile := filepath.Join(r.outputDir, fmt.Sprintf("%s_output.txt", testName))
	if err := os.WriteFile(outputFile, output, 0644); err != nil {
		fmt.Printf("   ⚠️  Failed to save output: %v\n", err)
	}

	r.results = append(r.results, result)
	fmt.Println()
}

// RunScreenCaptureTest runs the special screen capture test that saves all menu outputs
func (r *TestRunner) RunScreenCaptureTest() {
	fmt.Println("📸 Capturing all menu screen outputs...")

	cmd := exec.Command("go", "test", "-v", "-run", "CaptureAndSaveAllMenuScreens", "./internal/tui/testing")
	cmd.Dir = "./"

	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("   ❌ Screen capture failed: %v\n", err)
		fmt.Printf("   Output: %s\n", string(output))
	} else {
		fmt.Printf("   ✅ Screen captures saved to test files\n")
	}

	// Move any generated test output files to our output directory
	r.moveTestOutputFiles()
}

// moveTestOutputFiles moves generated test output files to the output directory
func (r *TestRunner) moveTestOutputFiles() {
	files, err := filepath.Glob("test_output_option_*.txt")
	if err != nil {
		return
	}

	for _, file := range files {
		destPath := filepath.Join(r.outputDir, filepath.Base(file))
		if err := os.Rename(file, destPath); err != nil {
			fmt.Printf("   ⚠️  Failed to move %s: %v\n", file, err)
		} else {
			fmt.Printf("   📁 Moved %s to %s\n", file, destPath)
		}
	}
}

// GenerateReport creates a comprehensive test report
func (r *TestRunner) GenerateReport() {
	fmt.Println("📊 Generating comprehensive test report...")

	reportPath := filepath.Join(r.outputDir, "terminal_test_report.md")
	report := r.buildMarkdownReport()

	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		fmt.Printf("❌ Failed to write report: %v\n", err)
		return
	}

	fmt.Printf("✅ Test report saved to: %s\n", reportPath)
}

// buildMarkdownReport creates a detailed markdown report
func (r *TestRunner) buildMarkdownReport() string {
	var report strings.Builder

	report.WriteString("# Flutter Package Manager - Terminal Output Test Report\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Summary
	passed := 0
	for _, result := range r.results {
		if result.Passed {
			passed++
		}
	}

	report.WriteString("## Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Tests:** %d\n", len(r.results)))
	report.WriteString(fmt.Sprintf("- **Passed:** %d ✅\n", passed))
	report.WriteString(fmt.Sprintf("- **Failed:** %d ❌\n", len(r.results)-passed))
	report.WriteString(fmt.Sprintf("- **Success Rate:** %.1f%%\n\n", float64(passed)/float64(len(r.results))*100))

	// Individual test results
	report.WriteString("## Test Results\n\n")
	for _, result := range r.results {
		status := "✅ PASSED"
		if !result.Passed {
			status = "❌ FAILED"
		}

		report.WriteString(fmt.Sprintf("### %s %s\n\n", result.TestName, status))
		report.WriteString(fmt.Sprintf("- **Duration:** %v\n", result.Duration))

		if result.Error != "" {
			report.WriteString(fmt.Sprintf("- **Error:** %s\n", result.Error))
		}

		report.WriteString("- **Output:**\n```\n")
		report.WriteString(result.Output)
		report.WriteString("\n```\n\n")
	}

	// Analysis section
	report.WriteString("## Analysis\n\n")
	report.WriteString("### Key Findings\n\n")

	if passed == len(r.results) {
		report.WriteString("🎉 All tests passed! The terminal output is working correctly.\n\n")
	} else {
		report.WriteString("⚠️ Some tests failed. Review the failed tests above for issues.\n\n")
	}

	report.WriteString("### Recommendations\n\n")
	report.WriteString("1. Review any failed tests for UI/UX issues\n")
	report.WriteString("2. Check that menu options lead to correct screens\n")
	report.WriteString("3. Verify terminal output formatting and content\n")
	report.WriteString("4. Ensure proper navigation and selection markers\n\n")

	// Terminal screenshots section
	report.WriteString("## Terminal Screenshots\n\n")
	report.WriteString("The following files contain captured terminal output for each menu option:\n\n")
	for i := 1; i <= 4; i++ {
		report.WriteString(fmt.Sprintf("- `test_output_option_%d.txt` - Menu option %d output\n", i, i))
	}
	report.WriteString("\nReview these files to verify the actual terminal appearance.\n")

	return report.String()
}

// ShowSummary displays a final summary of test results
func (r *TestRunner) ShowSummary() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🏁 TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	passed := 0
	totalDuration := time.Duration(0)

	for _, result := range r.results {
		status := "❌"
		if result.Passed {
			status = "✅"
			passed++
		}
		totalDuration += result.Duration

		fmt.Printf("%s %-30s %8v\n", status, result.TestName, result.Duration)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total: %d/%d passed (%.1f%%) in %v\n",
		passed, len(r.results),
		float64(passed)/float64(len(r.results))*100,
		totalDuration)

	if passed == len(r.results) {
		fmt.Println("\n🎉 All tests passed! Terminal output is verified.")
	} else {
		fmt.Printf("\n⚠️  %d test(s) failed. Check the detailed report.\n", len(r.results)-passed)
	}

	fmt.Printf("\n📁 Test artifacts saved to: %s/\n", r.outputDir)
	fmt.Printf("📊 Full report: %s/terminal_test_report.md\n", r.outputDir)
}
