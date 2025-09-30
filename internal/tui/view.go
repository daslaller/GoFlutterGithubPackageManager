package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// viewDetectProject renders the project detection step
func (m Model) viewDetectProject() string {
	var b strings.Builder

	if m.err != nil {
		b.WriteString(errorStyle.Render("❌ Error: "+m.err.Error()) + "\\n\\n")
	}

	if len(m.projects) == 0 {
		b.WriteString("🔍 Scanning for Flutter projects...\\n")
		b.WriteString("   • Checking current directory\\n")
		b.WriteString("   • Scanning common development folders\\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Found %d Flutter project(s):\\n\\n", len(m.projects)))

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

		b.WriteString(style.Render(projectInfo) + "\\n")
	}

	return b.String()
}

// viewChooseSource renders the source selection step
func (m Model) viewChooseSource() string {
	var b strings.Builder

	project := m.projects[m.selectedProject]
	b.WriteString(fmt.Sprintf("📂 Project: %s\\n\\n", project.Path))
	b.WriteString("Choose how to find packages to add:\\n\\n")

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
		b.WriteString(style.Render(sourceText) + "\\n")
		b.WriteString(fmt.Sprintf("   %s\\n\\n", source.desc))
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

	b.WriteString(fmt.Sprintf("Source: %s\\n\\n", sourceText))

	if m.err != nil {
		b.WriteString(errorStyle.Render("❌ Error: "+m.err.Error()) + "\\n\\n")
		return b.String()
	}

	if len(m.repos) == 0 {
		b.WriteString("🔍 Loading repositories...\\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Select packages to add (found %d repositories):\\n\\n", len(m.repos)))

	// Count selected
	selectedCount := 0
	for _, selected := range m.picks {
		if selected {
			selectedCount++
		}
	}

	if selectedCount > 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("✅ Selected: %d packages", selectedCount)) + "\\n\\n")
	}

	// Show repositories
	for i, repo := range m.repos {
		prefix := "  "
		checkbox := "☐"
		style := lipgloss.NewStyle()

		if m.picks[i] {
			checkbox = "☑"
			style = successStyle
		}

		if i == m.cursor {
			prefix = "► "
			if !m.picks[i] {
				style = selectedStyle
			}
		}

		// Privacy indicator
		privacy := ""
		if repo.Privacy == "private" {
			privacy = "🔒 "
		} else {
			privacy = "🔓 "
		}

		repoText := fmt.Sprintf("%s%s %s%s/%s", prefix, checkbox, privacy, repo.Owner, repo.Name)
		b.WriteString(style.Render(repoText) + "\\n")

		if repo.Desc != "" {
			desc := repo.Desc
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			b.WriteString(fmt.Sprintf("     %s\\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(desc)))
		}
		b.WriteString("\\n")
	}

	return b.String()
}

// viewEditSpecs renders the package specification editing step
func (m Model) viewEditSpecs() string {
	var b strings.Builder

	b.WriteString("✏️ Package Specifications\\n\\n")

	if len(m.edits) == 0 {
		b.WriteString("No packages selected.\\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Preparing %d package(s) for installation:\\n\\n", len(m.edits)))

	for i, spec := range m.edits {
		b.WriteString(boxStyle.Render(fmt.Sprintf(
			"📦 Package %d/%d\\n"+
				"   Name: %s\\n"+
				"   URL:  %s\\n"+
				"   Ref:  %s",
			i+1, len(m.edits),
			spec.Name,
			spec.URL,
			spec.Ref,
		)) + "\\n\\n")
	}

	return b.String()
}

// viewConfirm renders the confirmation step
func (m Model) viewConfirm() string {
	var b strings.Builder

	b.WriteString("✅ Confirm Installation\\n\\n")

	project := m.projects[m.selectedProject]
	b.WriteString(fmt.Sprintf("Project: %s\\n\\n", project.Path))

	b.WriteString("The following packages will be added:\\n\\n")

	for _, spec := range m.edits {
		b.WriteString(fmt.Sprintf("  • %s (%s#%s)\\n", spec.Name, spec.URL, spec.Ref))
	}

	b.WriteString("\\n")
	b.WriteString(warningStyle.Render("⚠️  This will modify your pubspec.yaml file"))
	b.WriteString("\\n")
	b.WriteString("   A backup will be created automatically.\\n\\n")

	b.WriteString("Do you want to continue? (y/N)")

	return b.String()
}

// viewExecute renders the execution step
func (m Model) viewExecute() string {
	var b strings.Builder

	b.WriteString("⚡ Installing Packages\\n\\n")

	if len(m.results) == 0 {
		b.WriteString("🔄 Starting installation...\\n")
		return b.String()
	}

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

		b.WriteString(style.Render(fmt.Sprintf("%s %s", status, message)) + "\\n")

		// Show logs if available
		for _, log := range result.Logs {
			if strings.TrimSpace(log) != "" {
				b.WriteString(fmt.Sprintf("   %s\\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(log)))
			}
		}

		if i < len(m.results)-1 {
			b.WriteString("\\n")
		}
	}

	return b.String()
}

// viewSummary renders the summary and recommendations step
func (m Model) viewSummary() string {
	var b strings.Builder

	b.WriteString("✨ Installation Complete\\n\\n")

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
		b.WriteString(successStyle.Render(fmt.Sprintf("🎉 All %d packages installed successfully!", successCount)) + "\\n\\n")
	} else {
		b.WriteString(errorStyle.Render(fmt.Sprintf("⚠️  %d succeeded, %d failed", successCount, errorCount)) + "\\n\\n")
	}

	// Show recommendations
	if len(m.recos) > 0 {
		b.WriteString("💡 Recommendations:\\n\\n")

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

			b.WriteString(style.Render(fmt.Sprintf("%s %s", icon, reco.Message)) + "\\n")
			if reco.Rationale != "" {
				b.WriteString(fmt.Sprintf("   %s\\n",
					lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(reco.Rationale)))
			}
			b.WriteString("\\n")
		}
	} else {
		b.WriteString("✅ No recommendations - your project looks great!\\n\\n")
	}

	return b.String()
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
