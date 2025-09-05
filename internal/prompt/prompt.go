package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Issue represents a doctor check issue
type Issue struct {
	Type    string // "error", "warning", "info"
	Message string
	Fix     string
}

// SummaryData represents data for summary display
type SummaryData struct {
	Title     string
	Items     []SummaryItem
	PublicKey string
	NextSteps []string
}

// SummaryItem represents an item in the summary
type SummaryItem struct {
	Label string
	Value string
	Icon  string
}

// Confirm prompts for yes/no confirmation
func Confirm(msg string) (bool, error) {
	// Check for non-interactive environment
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" {
		// In non-interactive mode, default to yes
		return true, nil
	}

	// Simple text-based confirmation for now
	fmt.Printf("%s (y/N): ", msg)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes", nil
}

// ShowSummary displays a styled summary
func ShowSummary(data SummaryData) error {
	// Check for non-interactive environment
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" {
		// Plain text output
		fmt.Printf("\n%s\n", data.Title)
		fmt.Println(strings.Repeat("=", len(data.Title)))
		for _, item := range data.Items {
			fmt.Printf("%s: %s\n", item.Label, item.Value)
		}
		if data.PublicKey != "" {
			fmt.Printf("\nPublic Key:\n%s\n", data.PublicKey)
		}
		if len(data.NextSteps) > 0 {
			fmt.Println("\nNext Steps:")
			for i, step := range data.NextSteps {
				fmt.Printf("%d. %s\n", i+1, step)
			}
		}
		return nil
	}

	// Styled output with Lip Gloss
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render(data.Title))
	content.WriteString("\n\n")

	// Items
	for _, item := range data.Items {
		icon := "✓"
		if item.Icon != "" {
			icon = item.Icon
		}
		content.WriteString(fmt.Sprintf("%s %s: %s\n",
			successStyle.Render(icon),
			keyStyle.Render(item.Label),
			item.Value))
	}

	// Public key
	if data.PublicKey != "" {
		content.WriteString("\n")
		content.WriteString(keyStyle.Render("Public Key:"))
		content.WriteString("\n")
		content.WriteString(data.PublicKey)
		content.WriteString("\n")
	}

	// Next steps
	if len(data.NextSteps) > 0 {
		content.WriteString("\n")
		content.WriteString(keyStyle.Render("Next Steps:"))
		content.WriteString("\n")
		for i, step := range data.NextSteps {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	fmt.Println(boxStyle.Render(content.String()))
	return nil
}

// ShowDoctorReport displays a styled doctor report
func ShowDoctorReport(issues []Issue) error {
	// Check for non-interactive environment
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" {
		// Plain text output
		fmt.Println("\nDoctor Report")
		fmt.Println(strings.Repeat("=", 12))
		for _, issue := range issues {
			icon := "ℹ️"
			switch issue.Type {
			case "error":
				icon = "❌"
			case "warning":
				icon = "⚠️"
			case "info":
				icon = "ℹ️"
			}
			fmt.Printf("%s %s\n", icon, issue.Message)
			if issue.Fix != "" {
				fmt.Printf("   Fix: %s\n", issue.Fix)
			}
		}
		return nil
	}

	// Styled output with Lip Gloss
	var content strings.Builder

	content.WriteString(titleStyle.Render("Doctor Report"))
	content.WriteString("\n\n")

	if len(issues) == 0 {
		content.WriteString(successStyle.Render("✓ All checks passed! No issues found."))
	} else {
		for _, issue := range issues {
			var icon, style string
			switch issue.Type {
			case "error":
				icon = "❌"
				style = errorStyle.Render(issue.Message)
			case "warning":
				icon = "⚠️"
				style = warningStyle.Render(issue.Message)
			case "info":
				icon = "ℹ️"
				style = infoStyle.Render(issue.Message)
			default:
				icon = "ℹ️"
				style = issue.Message
			}

			content.WriteString(fmt.Sprintf("%s %s\n", icon, style))
			if issue.Fix != "" {
				content.WriteString(fmt.Sprintf("   %s\n", keyStyle.Render("Fix: "+issue.Fix)))
			}
			content.WriteString("\n")
		}
	}

	fmt.Println(boxStyle.Render(content.String()))
	return nil
}

// ShowStatusTable displays a status table
func ShowStatusTable(headers []string, rows [][]string) error {
	// Check for non-interactive environment
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" {
		// Plain text output
		for i, header := range headers {
			if i > 0 {
				fmt.Print(" | ")
			}
			fmt.Print(header)
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", len(strings.Join(headers, " | "))))
		for _, row := range rows {
			for i, cell := range row {
				if i > 0 {
					fmt.Print(" | ")
				}
				fmt.Print(cell)
			}
			fmt.Println()
		}
		return nil
	}

	// Styled output with Lip Gloss
	var content strings.Builder

	content.WriteString(titleStyle.Render("Repository Status"))
	content.WriteString("\n\n")

	// Headers
	for i, header := range headers {
		if i > 0 {
			content.WriteString(" | ")
		}
		content.WriteString(keyStyle.Render(header))
	}
	content.WriteString("\n")
	content.WriteString(strings.Repeat("-", len(strings.Join(headers, " | "))))
	content.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				content.WriteString(" | ")
			}
			content.WriteString(cell)
		}
		content.WriteString("\n")
	}

	fmt.Println(boxStyle.Render(content.String()))
	return nil
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Margin(1, 0)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(1, 2)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)
)
