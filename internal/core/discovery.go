package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// NearestPubspec walks up from the current directory to find the nearest pubspec.yaml
// This mirrors the shell script's behavior of detecting nested directory projects
func NearestPubspec(startDir string) (*Project, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	current := startDir
	root := filepath.VolumeName(current) + string(filepath.Separator)

	for {
		pubspecPath := filepath.Join(current, "pubspec.yaml")
		if _, err := os.Stat(pubspecPath); err == nil {
			// Found pubspec.yaml
			project := &Project{
				Path:        current,
				PubspecPath: pubspecPath,
			}

			// Try to extract project name from pubspec.yaml
			if name, err := extractProjectName(pubspecPath); err == nil {
				project.Name = name
			}

			return project, nil
		}

		// Move up one directory
		parent := filepath.Dir(current)
		if parent == current || parent == root {
			break // Reached the root
		}
		current = parent
	}

	return nil, fmt.Errorf("no pubspec.yaml found in %s or parent directories", startDir)
}

// ScanCommonRoots scans common development directories for Flutter projects
// This mirrors the shell script's local project discovery with concurrent scanning and proper cleanup
func ScanCommonRoots() ([]Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return ScanCommonRootsWithContext(ctx)
}

// ScanCommonRootsWithContext scans with context for cancellation and timeout
func ScanCommonRootsWithContext(ctx context.Context) ([]Project, error) {
	roots := CommonRoots()
	numWorkers := runtime.NumCPU() // Use all available CPU cores
	if numWorkers > len(roots) {
		numWorkers = len(roots) // Don't use more workers than roots
	}

	// Create channels for work distribution
	rootChan := make(chan string, len(roots))
	resultChan := make(chan []Project, len(roots))
	errorChan := make(chan error, len(roots))

	// Start workers with context cancellation
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return // Exit gracefully on context cancellation
				case root, ok := <-rootChan:
					if !ok {
						return // Channel closed
					}

					if _, err := os.Stat(root); os.IsNotExist(err) {
						select {
						case resultChan <- []Project{}: // Skip non-existent directories
						case <-ctx.Done():
							return
						}
						continue
					}

					rootProjects, err := scanDirectoryForProjectsWithContext(ctx, root, 3)
					if err != nil {
						select {
						case errorChan <- err:
						case <-ctx.Done():
							return
						}
						select {
						case resultChan <- []Project{}: // Continue with empty result
						case <-ctx.Done():
							return
						}
						continue
					}

					select {
					case resultChan <- rootProjects:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Send work to workers with context awareness
	go func() {
		defer close(rootChan)
		for _, root := range roots {
			select {
			case rootChan <- root:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Close channels when workers are done
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results with timeout protection
	var projects []Project
	resultsReceived := 0

	for resultsReceived < len(roots) {
		select {
		case <-ctx.Done():
			return projects, ctx.Err()
		case result, ok := <-resultChan:
			if !ok {
				break // Channel closed
			}
			projects = append(projects, result...)
			resultsReceived++
		case <-errorChan:
			// Log error but continue with other roots
			resultsReceived++
		}
	}

	return projects, nil
}

// CommonRoots returns the common development directory paths to scan
// This matches the shell script's search directories
func CommonRoots() []string {
	homeDir, _ := os.UserHomeDir()

	roots := []string{
		filepath.Join(homeDir, "Development"),
		filepath.Join(homeDir, "Projects"),
		filepath.Join(homeDir, "dev"),
		filepath.Join(homeDir, "Documents", "Development"),
		filepath.Join(homeDir, "Documents", "Projects"),
	}

	// Add current directory as well
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}

	return roots
}

// scanDirectoryForProjects recursively scans a directory for Flutter projects with optimized I/O
func scanDirectoryForProjects(dir string, maxDepth int) ([]Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return scanDirectoryForProjectsWithContext(ctx, dir, maxDepth)
}

// scanDirectoryForProjectsWithContext recursively scans with context cancellation
func scanDirectoryForProjectsWithContext(ctx context.Context, dir string, maxDepth int) ([]Project, error) {
	var projects []Project

	if maxDepth <= 0 {
		return projects, nil
	}

	// Use ReadDir for better performance than Stat + ReadDir separately
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	// First check if this directory itself is a Flutter project
	pubspecPath := filepath.Join(dir, "pubspec.yaml")
	if _, err := os.Stat(pubspecPath); err == nil {
		project := Project{
			Path:        dir,
			PubspecPath: pubspecPath,
		}

		// Only extract project name if we find pubspec.yaml
		if name, err := extractProjectNameOptimized(pubspecPath); err == nil {
			project.Name = name
		}

		projects = append(projects, project)
		return projects, nil // Don't scan subdirectories if this is already a project
	}

	// Pre-filter directories to avoid unnecessary recursive calls
	var validDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories and common non-project directories
		name := entry.Name()
		if strings.HasPrefix(name, ".") ||
			name == "node_modules" ||
			name == "build" ||
			name == ".git" ||
			name == "vendor" ||
			name == ".dart_tool" ||
			name == ".vscode" ||
			name == ".idea" {
			continue
		}

		validDirs = append(validDirs, name)
	}

	// Check context before proceeding
	select {
	case <-ctx.Done():
		return projects, ctx.Err()
	default:
	}

	// Process valid directories concurrently if there are enough of them
	if len(validDirs) > 4 && maxDepth > 1 {
		return scanDirectoriesConcurrentWithContext(ctx, dir, validDirs, maxDepth-1)
	}

	// Scan subdirectories sequentially for small numbers
	for _, name := range validDirs {
		select {
		case <-ctx.Done():
			return projects, ctx.Err()
		default:
		}

		subDir := filepath.Join(dir, name)
		subProjects, err := scanDirectoryForProjectsWithContext(ctx, subDir, maxDepth-1)
		if err != nil {
			// Continue with other directories on error
			continue
		}

		projects = append(projects, subProjects...)
	}

	return projects, nil
}

// scanDirectoriesConcurrent scans multiple directories concurrently for better performance
func scanDirectoriesConcurrent(baseDir string, dirNames []string, maxDepth int) ([]Project, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return scanDirectoriesConcurrentWithContext(ctx, baseDir, dirNames, maxDepth)
}

// scanDirectoriesConcurrentWithContext scans with proper context handling
func scanDirectoriesConcurrentWithContext(ctx context.Context, baseDir string, dirNames []string, maxDepth int) ([]Project, error) {
	type result struct {
		projects []Project
		err      error
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > len(dirNames) {
		numWorkers = len(dirNames)
	}

	dirChan := make(chan string, len(dirNames))
	resultChan := make(chan result, len(dirNames))

	// Start workers with context cancellation
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case dirName, ok := <-dirChan:
					if !ok {
						return
					}
					subDir := filepath.Join(baseDir, dirName)
					subProjects, err := scanDirectoryForProjectsWithContext(ctx, subDir, maxDepth)

					select {
					case resultChan <- result{projects: subProjects, err: err}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Send work with context awareness
	go func() {
		defer close(dirChan)
		for _, dirName := range dirNames {
			select {
			case dirChan <- dirName:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Close result channel when workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results with timeout protection
	var allProjects []Project
	resultsReceived := 0

	for resultsReceived < len(dirNames) {
		select {
		case <-ctx.Done():
			return allProjects, ctx.Err()
		case res, ok := <-resultChan:
			if !ok {
				break
			}
			if res.err == nil {
				allProjects = append(allProjects, res.projects...)
			}
			resultsReceived++
			// Ignore errors and continue - same behavior as sequential version
		}
	}

	return allProjects, nil
}

// extractProjectName extracts the project name from pubspec.yaml
func extractProjectName(pubspecPath string) (string, error) {
	return extractProjectNameOptimized(pubspecPath)
}

// extractProjectNameOptimized extracts the project name with optimized reading
func extractProjectNameOptimized(pubspecPath string) (string, error) {
	content, err := os.ReadFile(pubspecPath)
	if err != nil {
		return "", fmt.Errorf("failed to read pubspec.yaml: %w", err)
	}

	// Use string search for quick name extraction
	contentStr := string(content)
	namePrefix := "name:"
	nameIndex := strings.Index(contentStr, namePrefix)
	if nameIndex == -1 {
		return "", fmt.Errorf("no name field found in pubspec.yaml")
	}

	// Find the end of the line
	lineEnd := strings.Index(contentStr[nameIndex:], "\n")
	if lineEnd == -1 {
		lineEnd = len(contentStr)
	} else {
		lineEnd += nameIndex
	}

	// Extract the line and parse it
	line := contentStr[nameIndex:lineEnd]
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		name := strings.TrimSpace(parts[1])
		// Remove quotes if present
		name = strings.Trim(name, "\"'")
		return name, nil
	}

	return "", fmt.Errorf("no name field found in pubspec.yaml")
}

// ValidateProject performs basic validation on a Flutter project
// This mirrors the shell script's project validation logic
func ValidateProject(projectPath string, autoFix bool) ([]string, error) {
	var messages []string

	// Check if pubspec.yaml exists
	pubspecPath := filepath.Join(projectPath, "pubspec.yaml")
	if _, err := os.Stat(pubspecPath); os.IsNotExist(err) {
		return messages, fmt.Errorf("pubspec.yaml not found in %s", projectPath)
	}

	// Check if lib directory exists
	libPath := filepath.Join(projectPath, "lib")
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		messages = append(messages, "lib directory not found")
		if autoFix {
			if err := os.MkdirAll(libPath, 0755); err == nil {
				messages = append(messages, "created lib directory")
			}
		}
	}

	// Check if main.dart exists
	mainPath := filepath.Join(projectPath, "lib", "main.dart")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		messages = append(messages, "lib/main.dart not found")
		if autoFix {
			mainContent := `
import 'package:flutter/material.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Demo',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
        useMaterial3: true,
      ),
      home: const MyHomePage(title: 'Flutter Demo Home Page'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  int _counter = 0;

  void _incrementCounter() {
    setState(() {
      _counter++;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: Text(widget.title),
      ),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            const Text(
              'You have pushed the button this many times:',
            ),
            Text(
              '$_counter',
              style: Theme.of(context).textTheme.headlineMedium,
            ),
          ],
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _incrementCounter,
        tooltip: 'Increment',
        child: const Icon(Icons.add),
      ),
    );
  }
}
`
			if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err == nil {
				messages = append(messages, "created lib/main.dart")
			}
		}
	}

	// Check if it's a Git repository
	gitPath := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		messages = append(messages, "not a Git repository")
		if autoFix {
			if err := runGitInit(projectPath); err == nil {
				messages = append(messages, "initialized Git repository")
			}
		}
	}

	return messages, nil
}

// runGitInit initializes a Git repository
func runGitInit(projectPath string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	return cmd.Run()
}
