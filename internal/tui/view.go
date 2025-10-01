package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Optimized spinner frames as constants for better performance
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Pre-compiled spinner style for performance
var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))

// viewMainMenu renders the main menu like PowerShell version with optimizations
func (m Model) viewMainMenu() string {
	var b strings.Builder
	b.Grow(1024) // Pre-allocate reasonable buffer size

	if m.err != nil {
		b.WriteString(errorStyle.Render("❌ Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	if m.loading {
		frame := spinnerFrames[m.spinnerIdx]

		if m.loadingText != "" {
			b.WriteString(spinnerStyle.Render(frame))
			b.WriteString(" ")
			b.WriteString(m.loadingText)
			b.WriteString("\n\n")
		}
		return b.String()
	}

	hasLocalProject := len(m.projects) > 0
	projectName := ""
	if hasLocalProject {
		projectName = m.projects[0].Name
		if projectName == "" {
			projectName = fmt.Sprintf("Project in %s", m.projects[0].Path)
		}
	}

	// Menu options (matching shell script exactly as clarified)
	options := []struct {
		icon      string
		text      string
		available bool
		isDefault bool
	}{
		{"📁", "Scan directories", true, !hasLocalProject},                                      // Option 1
		{"🐙", "GitHub repo", true, false},                                                      // Option 2
		{"⚙️", "Configure search", true, false},                                                // Option 3
		{"📦", fmt.Sprintf("Use detected: %s", projectName), hasLocalProject, hasLocalProject},  // Option 4 [DEFAULT]
		{"🚀", fmt.Sprintf("🚀 Express Git update for %s", projectName), hasLocalProject, false}, // Option 5 (check git deps later)
		{"🔄", "🔄 Check for Flutter-PM updates", true, false},                                   // Option 6
	}

	// TODO: Check for git dependencies properly
	hasGitDeps := hasLocalProject // Simplified for now

	// Update option 5 availability based on git deps
	options[4].available = hasLocalProject && hasGitDeps

	// Render menu exactly like shell script (1-6 numbering)
	for i, option := range options {
		if !option.available {
			continue
		}

		var prefix string
		var style lipgloss.Style

		// Use actual option index for cursor, not menu display index
		if i == m.cursor {
			prefix = "► "
			style = selectedStyle
		} else {
			prefix = "  "
			style = lipgloss.NewStyle()
		}

		// Build option text exactly like shell script
		var optionBuilder strings.Builder
		optionBuilder.WriteString(fmt.Sprintf("%d. %s", i+1, option.text))
		if option.isDefault {
			optionBuilder.WriteString(" [DEFAULT]")
			if i != m.cursor {
				style = successStyle
			}
		}

		b.WriteString(style.Render(prefix + optionBuilder.String()))
		b.WriteString("\n")
	}

	if hasLocalProject {
		b.WriteString(fmt.Sprintf("\n💡 Detected Flutter project: %s\n", projectName))
	} else {
		b.WriteString("\n💡 No Flutter project detected in current directory\n")
	}

	return b.String()
}

// viewDetectProject renders the project detection step (kept for scanning results)
func (m Model) viewDetectProject() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString(errorStyle.Render("❌ Error: "+m.err.Error()) + "\n\n")
		b.WriteString("💡 Press 's' to skip project detection and continue anyway.\n")
		return b.String()
	}

	if m.loading || len(m.projects) == 0 {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[m.spinnerIdx]

		if m.loadingText != "" {
			b.WriteString(fmt.Sprintf("%s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame),
				m.loadingText))
		} else {
			b.WriteString(fmt.Sprintf("%s Scanning for Flutter projects...\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame)))
		}
		b.WriteString("   • Checking current directory\n")
		b.WriteString("   • Scanning common development folders\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Found %d Flutter project(s):\n\n", len(m.projects)))

	for i, project := range m.projects {
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.selectedProject {
			prefix = "► "
			style = selectedStyle
		}

		projectInfo := fmt.Sprintf("%s%s", prefix, project.Path)
		if project.Name != "" {
			projectInfo += fmt.Sprintf(" (%s)", project.Name)
		}

		b.WriteString(style.Render(projectInfo) + "\n")
	}

	return b.String()
}

// viewChooseSource renders the source selection step
func (m Model) viewChooseSource() string {
	var b strings.Builder

	project := m.projects[m.selectedProject]
	b.WriteString(fmt.Sprintf("📂 Project: %s\n\n", project.Path))
	b.WriteString("Choose how to find packages to add:\n\n")

	sources := []struct {
		icon string
		name string
		desc string
	}{
		{"📦", "GitHub Repositories", "Browse your GitHub repositories"},
		{"🔗", "Manual URL Entry", "Enter git repository URLs manually"},
		{"📁", "Local Repositories", "Scan local directories for git repositories"},
	}

	for i, source := range sources {
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.cursor {
			prefix = "► "
			style = selectedStyle
		}

		sourceText := fmt.Sprintf("%s%s %s", prefix, source.icon, source.name)
		b.WriteString(style.Render(sourceText) + "\n")
		b.WriteString(fmt.Sprintf("   %s\n\n", source.desc))
	}

	return b.String()
}

// viewSelectGitHubProject renders the GitHub project selection step (single-select)
func (m Model) viewSelectGitHubProject() string {
	var b strings.Builder

	if m.err != nil && !strings.Contains(m.err.Error(), "not implemented yet") {
		b.WriteString(errorStyle.Render("❌ Error: "+m.err.Error()) + "\n\n")
		return b.String()
	}

	if m.loading || len(m.repos) == 0 {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[m.spinnerIdx]

		if m.loadingText != "" {
			b.WriteString(fmt.Sprintf("%s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame),
				m.loadingText))
		} else {
			b.WriteString(fmt.Sprintf("%s Loading GitHub repositories...\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame)))
		}
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Select repository to clone as project (found %d repositories):\n", len(m.repos)))
	b.WriteString("Use ↑/↓ or j/k to navigate, SPACE or ENTER to select, q to quit\n\n")

	// Calculate window bounds like shell script
	windowEnd := m.windowStart + m.windowSize
	if windowEnd > len(m.repos) {
		windowEnd = len(m.repos)
	}

	// Show 'hidden above' indicator
	if m.windowStart > 0 {
		b.WriteString(fmt.Sprintf("... (%d more above) ...\n\n", m.windowStart))
	}

	// Pre-compiled styles for better performance
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Show repositories in current window (single-select mode)
	for i := m.windowStart; i < windowEnd; i++ {
		repo := m.repos[i]
		var prefix, checkbox, privacy string
		var style lipgloss.Style

		// Single-select checkbox (radio button style)
		if m.picks[i] {
			checkbox = "[●]" // Selected radio button
			style = successStyle
		} else {
			checkbox = "[ ]" // Empty radio button
			style = lipgloss.NewStyle()
		}

		// Determine prefix and override style if cursor
		if i == m.cursor {
			prefix = "► "
			if !m.picks[i] {
				style = selectedStyle
			}
		} else {
			prefix = "  "
		}

		// Privacy indicator
		if repo.Privacy == "private" {
			privacy = "🔒 "
		} else {
			privacy = "🔓 "
		}

		// Build repo text efficiently
		var repoBuilder strings.Builder
		repoBuilder.WriteString(prefix)
		repoBuilder.WriteString(checkbox)
		repoBuilder.WriteString(" ")
		repoBuilder.WriteString(privacy)
		repoBuilder.WriteString(repo.Owner)
		repoBuilder.WriteString("/")
		repoBuilder.WriteString(repo.Name)

		b.WriteString(style.Render(repoBuilder.String()))
		b.WriteString("\n")

		// Add description if present
		if repo.Desc != "" {
			desc := repo.Desc
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			b.WriteString("     ")
			b.WriteString(descStyle.Render(desc))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Show 'hidden below' indicator
	if windowEnd < len(m.repos) {
		remaining := len(m.repos) - windowEnd
		b.WriteString(fmt.Sprintf("... (%d more below) ...\n\n", remaining))
	}

	// Show cursor position like shell script
	if m.cursor < len(m.repos) {
		currentRepo := m.repos[m.cursor]
		b.WriteString(fmt.Sprintf("Cursor: %s/%s\n", currentRepo.Owner, currentRepo.Name))
	}

	// Show selection (single-select mode)
	selectedRepo := ""
	for i, selected := range m.picks {
		if selected && i < len(m.repos) {
			repo := m.repos[i]
			selectedRepo = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
			break
		}
	}

	if selectedRepo != "" {
		b.WriteString(fmt.Sprintf("Selected: %s\n", selectedRepo))

		// Show "not implemented" message if there's an error about it
		if m.err != nil && strings.Contains(m.err.Error(), "not implemented yet") {
			b.WriteString("\n")
			b.WriteString(warningStyle.Render("⚠️  " + m.err.Error()))
			b.WriteString("\n")
		}
	} else {
		b.WriteString("Selected: none\n")
	}

	return b.String()
}

// viewListRepos renders the repository listing step
func (m Model) viewListRepos() string {
	var b strings.Builder

	sourceText := ""
	switch m.source {
	case 0:
		sourceText = "📦 GitHub Repositories"
	case 1:
		sourceText = "🔗 Manual URL Entry"
	case 2:
		sourceText = "📁 Local Repositories"
	}

	b.WriteString(fmt.Sprintf("Source: %s\n\n", sourceText))

	if m.err != nil {
		b.WriteString(errorStyle.Render("❌ Error: "+m.err.Error()) + "\n\n")
		return b.String()
	}

	if m.loading || len(m.repos) == 0 {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[m.spinnerIdx]

		if m.loadingText != "" {
			b.WriteString(fmt.Sprintf("%s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame),
				m.loadingText))
		} else {
			b.WriteString(fmt.Sprintf("%s Loading repositories...\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame)))
		}
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Select packages to add (found %d repositories):\n", len(m.repos)))
	if m.singleSelect {
		b.WriteString("Use ↑/↓ or j/k to navigate, SPACE or ENTER to select, q to quit\n\n")
	} else {
		b.WriteString("Use ↑/↓ or j/k to navigate, SPACE to select/deselect, ENTER to confirm, q to quit\n\n")
	}

	// Count selected
	selectedCount := 0
	for _, selected := range m.picks {
		if selected {
			selectedCount++
		}
	}

	// Calculate window bounds like shell script
	windowEnd := m.windowStart + m.windowSize
	if windowEnd > len(m.repos) {
		windowEnd = len(m.repos)
	}

	// Show 'hidden above' indicator
	if m.windowStart > 0 {
		b.WriteString(fmt.Sprintf("... (%d more above) ...\n\n", m.windowStart))
	}

	// Pre-compiled styles for better performance
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Show repositories in current window
	for i := m.windowStart; i < windowEnd; i++ {
		repo := m.repos[i]
		var prefix, checkbox, privacy string
		var style lipgloss.Style

		// Determine checkbox and style based on mode
		if m.singleSelect {
			// Single-select mode: radio button style
			if m.picks[i] {
				checkbox = "[●]" // Selected radio button
				style = successStyle
			} else {
				checkbox = "[ ]" // Empty radio button
				style = lipgloss.NewStyle()
			}
		} else {
			// Multi-select mode: checkbox style
			if m.picks[i] {
				checkbox = "☑"
				style = successStyle
			} else {
				checkbox = "☐"
				style = lipgloss.NewStyle()
			}
		}

		// Determine prefix and override style if cursor
		if i == m.cursor {
			prefix = "► "
			if !m.picks[i] {
				style = selectedStyle
			}
		} else {
			prefix = "  "
		}

		// Privacy indicator
		if repo.Privacy == "private" {
			privacy = "🔒 "
		} else {
			privacy = "🔓 "
		}

		// Build repo text efficiently
		var repoBuilder strings.Builder
		repoBuilder.WriteString(prefix)
		repoBuilder.WriteString(checkbox)
		repoBuilder.WriteString(" ")
		repoBuilder.WriteString(privacy)
		repoBuilder.WriteString(repo.Owner)
		repoBuilder.WriteString("/")
		repoBuilder.WriteString(repo.Name)

		b.WriteString(style.Render(repoBuilder.String()))
		b.WriteString("\n")

		// Add description if present
		if repo.Desc != "" {
			desc := repo.Desc
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			b.WriteString("     ")
			b.WriteString(descStyle.Render(desc))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Show 'hidden below' indicator
	if windowEnd < len(m.repos) {
		remaining := len(m.repos) - windowEnd
		b.WriteString(fmt.Sprintf("... (%d more below) ...\n\n", remaining))
	}

	// Show cursor position like shell script
	if m.cursor < len(m.repos) {
		currentRepo := m.repos[m.cursor]
		b.WriteString(fmt.Sprintf("Cursor: %s/%s\n", currentRepo.Owner, currentRepo.Name))
	}

	// Show selection count and selected items
	b.WriteString(fmt.Sprintf("Selected: %d items\n", selectedCount))
	if selectedCount > 0 {
		b.WriteString("\nCurrently selected:\n")
		selectedItems := make([]string, 0, selectedCount)
		for i, selected := range m.picks {
			if selected && i < len(m.repos) {
				repo := m.repos[i]
				selectedItems = append(selectedItems, fmt.Sprintf("  ✓ %s/%s", repo.Owner, repo.Name))
			}
		}
		// Limit displayed selected items to avoid overflow
		maxShow := 5
		for i, item := range selectedItems {
			if i >= maxShow {
				remaining := len(selectedItems) - maxShow
				b.WriteString(fmt.Sprintf("  ... and %d more\n", remaining))
				break
			}
			b.WriteString(item + "\n")
		}
	}

	return b.String()
}

// viewEditSpecs renders the package specification editing step
func (m Model) viewEditSpecs() string {
	var b strings.Builder

	b.WriteString("✏️ Package Specifications\n\n")

	if len(m.edits) == 0 {
		b.WriteString("No packages selected.\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Preparing %d package(s) for installation:\n\n", len(m.edits)))

	for i, spec := range m.edits {
		b.WriteString(boxStyle.Render(fmt.Sprintf(
			"📦 Package %d/%d\n"+
				"   Name: %s\n"+
				"   URL:  %s\n"+
				"   Ref:  %s",
			i+1, len(m.edits),
			spec.Name,
			spec.URL,
			spec.Ref,
		)) + "\n\n")
	}

	return b.String()
}

// viewConfirm renders the confirmation step
func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString("✅ Confirm Installation\n\n")

	project := m.projects[m.selectedProject]
	b.WriteString(fmt.Sprintf("Project: %s\n\n", project.Path))

	b.WriteString("The following packages will be added:\n\n")

	for _, spec := range m.edits {
		b.WriteString(fmt.Sprintf("  • %s (%s#%s)\n", spec.Name, spec.URL, spec.Ref))
	}

	b.WriteString("\n")
	b.WriteString(warningStyle.Render("⚠️  This will modify your pubspec.yaml file"))
	b.WriteString("\n")
	b.WriteString("   A backup will be created automatically.\n\n")

	b.WriteString("Do you want to continue? (y/N)")

	return b.String()
}

// viewExecute renders the execution step
func (m Model) viewExecute() string {
	var b strings.Builder

	b.WriteString("⚡ Installing Packages\n\n")

	if len(m.results) == 0 {
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		frame := spinner[m.spinnerIdx]
		b.WriteString(fmt.Sprintf("%s Starting installation...\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD")).Render(frame)))
		return b.String()
	}

	// Show progress bar
	totalJobs := len(m.edits) + 1 // +1 for pub get
	completedJobs := len(m.results)
	progress := float64(completedJobs) / float64(totalJobs)

	progressBar := m.renderProgressBar(progress, 40)
	b.WriteString(fmt.Sprintf("Progress: %s %d/%d\n\n", progressBar, completedJobs, totalJobs))

	for i, result := range m.results {
		status := "🔄"
		style := lipgloss.NewStyle()

		if result.OK {
			status = "✅"
			style = successStyle
		} else if result.Err != "" {
			status = "❌"
			style = errorStyle
		}

		message := result.Message
		if message == "" && result.Err != "" {
			message = result.Err
		}

		b.WriteString(style.Render(fmt.Sprintf("%s %s", status, message)) + "\n")

		// Show logs if available
		for _, log := range result.Logs {
			if strings.TrimSpace(log) != "" {
				b.WriteString(fmt.Sprintf("   %s\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(log)))
			}
		}

		if i < len(m.results)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// viewSummary renders the summary and recommendations step
func (m Model) viewSummary() string {
	var b strings.Builder

	b.WriteString("✨ Installation Complete\n\n")

	// Show results summary
	successCount := 0
	errorCount := 0

	for _, result := range m.results {
		if result.OK {
			successCount++
		} else {
			errorCount++
		}
	}

	if errorCount == 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("🎉 All %d packages installed successfully!", successCount)) + "\n\n")
	} else {
		b.WriteString(errorStyle.Render(fmt.Sprintf("⚠️  %d succeeded, %d failed", successCount, errorCount)) + "\n\n")
	}

	// Show recommendations
	if len(m.recos) > 0 {
		b.WriteString("💡 Recommendations:\n\n")

		for _, reco := range m.recos {
			icon := "ℹ️"
			style := lipgloss.NewStyle()

			switch reco.Severity {
			case "warn":
				icon = "⚠️"
				style = warningStyle
			case "error":
				icon = "❌"
				style = errorStyle
			case "info":
				icon = "💡"
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#13B9FD"))
			}

			b.WriteString(style.Render(fmt.Sprintf("%s %s", icon, reco.Message)) + "\n")
			if reco.Rationale != "" {
				b.WriteString(fmt.Sprintf("   %s\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(reco.Rationale)))
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("✅ No recommendations - your project looks great!\n\n")
	}

	return b.String()
}

// Style cache for performance optimization
type StyleCache struct {
	mu     sync.RWMutex
	cache  map[string]lipgloss.Style
	render map[string]string // Cache rendered strings
}

var (
	styleCache = &StyleCache{
		cache:  make(map[string]lipgloss.Style),
		render: make(map[string]string),
	}
)

// renderProgressBar creates a visual progress bar with caching
func (m Model) renderProgressBar(progress float64, width int) string {
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0.0 {
		progress = 0.0
	}

	// Create cache key
	cacheKey := fmt.Sprintf("progress_%.2f_%d", progress, width)

	// Check cache first
	styleCache.mu.RLock()
	if cached, exists := styleCache.render[cacheKey]; exists {
		styleCache.mu.RUnlock()
		return cached
	}
	styleCache.mu.RUnlock()

	filled := int(progress * float64(width))
	empty := width - filled

	// Use strings.Builder for efficient string concatenation
	var bar strings.Builder
	bar.Grow(width) // Pre-allocate capacity

	for i := 0; i < filled; i++ {
		bar.WriteString("█")
	}
	for i := 0; i < empty; i++ {
		bar.WriteString("░")
	}

	percentage := int(progress * 100)

	barStyle := lipgloss.NewStyle()
	if progress == 1.0 {
		barStyle = barStyle.Foreground(lipgloss.Color("#4CAF50")) // Green when complete
	} else {
		barStyle = barStyle.Foreground(lipgloss.Color("#13B9FD")) // Blue in progress
	}

	result := fmt.Sprintf("%s %d%%", barStyle.Render(bar.String()), percentage)

	// Cache the result
	styleCache.mu.Lock()
	styleCache.render[cacheKey] = result
	styleCache.mu.Unlock()

	return result
}

// Additional styles
var (
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4CAF50")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F44336")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF9800")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#E5E7EB")).
			Padding(1, 2)
)
