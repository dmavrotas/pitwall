package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dmavrotas/pitwall/query"
)

// Colors
var (
	red       = lipgloss.Color("#E10600") // F1 red
	white     = lipgloss.Color("#FFFFFF")
	gray      = lipgloss.Color("#6B7280")
	darkGray  = lipgloss.Color("#374151")
	green     = lipgloss.Color("#10B981")
	yellow    = lipgloss.Color("#F59E0B")
	cyan      = lipgloss.Color("#06B6D4")
	dimWhite  = lipgloss.Color("#9CA3AF")
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(red).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			Italic(true)

	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(red)

	questionStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	altRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	borderStyle = lipgloss.NewStyle().
			Foreground(darkGray)

	errorStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(gray).
			Italic(true)

	countStyle = lipgloss.NewStyle().
			Foreground(green)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(dimWhite).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)
)

type model struct {
	engine     *query.Engine
	textInput  textinput.Model
	viewport   viewport.Model
	history    []historyEntry
	width      int
	height     int
	ready      bool
	statsInfo  string
}

type historyEntry struct {
	question string
	answer   string
	isError  bool
}

// New creates and returns a new bubbletea program.
func New(engine *query.Engine, statsInfo string) *tea.Program {
	ti := textinput.New()
	ti.Placeholder = "Ask me anything about F1..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80
	ti.PromptStyle = promptStyle
	ti.Prompt = "  > "
	ti.TextStyle = lipgloss.NewStyle().Foreground(white)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(gray)

	m := model{
		engine:    engine,
		textInput: ti,
		history:   []historyEntry{},
		statsInfo: statsInfo,
	}

	return tea.NewProgram(m, tea.WithAltScreen())
}

// Init implements tea.Model.
//
//nolint:gocritic // hugeParam: value receiver required by bubbletea's tea.Model interface
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
//
//nolint:gocritic // hugeParam: value receiver required by bubbletea's tea.Model interface
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}
			m.textInput.SetValue("")

			switch strings.ToLower(input) {
			case "quit", "exit", "q":
				return m, tea.Quit
			case "help", "?":
				m.history = append(m.history, historyEntry{
					question: input,
					answer:   renderHelp(),
				})
			case "clear":
				m.history = []historyEntry{}
			default:
				result, err := m.engine.Ask(input)
				if err != nil {
					m.history = append(m.history, historyEntry{
						question: input,
						answer:   err.Error(),
						isError:  true,
					})
				} else {
					m.history = append(m.history, historyEntry{
						question: input,
						answer:   renderResult(result, m.width),
					})
				}
			}
			m.viewport.SetContent(m.renderHistory())
			m.viewport.GotoBottom()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 6 // banner + spacing
		footerHeight := 4 // input + status bar + spacing

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.SetContent(m.renderHistory())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
			m.viewport.SetContent(m.renderHistory())
		}

		m.textInput.Width = msg.Width - 8
		return m, nil
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

// View implements tea.Model.
//
//nolint:gocritic // hugeParam: value receiver required by bubbletea's tea.Model interface
func (m model) View() string {
	if !m.ready {
		return "\n  Loading..."
	}

	var sb strings.Builder

	// Header banner
	banner := titleStyle.Render(" PITWALL ")
	tagline := subtitleStyle.Render(" F1 Data Analysis Engine")
	sb.WriteString("\n  " + banner + tagline + "\n\n")

	// Viewport (scrollable history)
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Input line
	sb.WriteString(m.textInput.View())
	sb.WriteString("\n")

	// Status bar
	statusLeft := m.statsInfo
	statusRight := "help · clear · quit · esc"
	padding := m.width - lipgloss.Width(statusLeft) - lipgloss.Width(statusRight) - 2
	if padding < 1 {
		padding = 1
	}
	status := statusBarStyle.Width(m.width).Render(
		statusLeft + strings.Repeat(" ", padding) + statusRight,
	)
	sb.WriteString(status)

	return sb.String()
}

//nolint:gocritic // hugeParam: value receiver required by bubbletea's tea.Model pattern
func (m model) renderHistory() string {
	if len(m.history) == 0 {
		return renderWelcome()
	}

	var sb strings.Builder
	for i, entry := range m.history {
		if i > 0 {
			sb.WriteString("\n")
		}
		// Question
		sb.WriteString("  " + questionStyle.Render("? "+entry.question) + "\n")

		// Answer
		if entry.isError {
			sb.WriteString("  " + errorStyle.Render("Error: "+entry.answer) + "\n")
			sb.WriteString("  " + hintStyle.Render("Type 'help' for example questions.") + "\n")
		} else {
			sb.WriteString(entry.answer)
		}
	}
	return sb.String()
}

func renderWelcome() string {
	var sb strings.Builder

	sb.WriteString("\n")

	welcomeLines := []string{
		"Welcome to Pitwall! Ask questions about Formula 1 in plain English.",
		"",
		"Try something like:",
	}

	examples := []string{
		"Who has the most wins?",
		"Show me the 2021 championship standings",
		"Compare Verstappen vs Hamilton",
		"How many points did Hamilton score in 2019?",
		"Fastest pit stops in 2023",
	}

	for _, line := range welcomeLines {
		sb.WriteString("  " + hintStyle.Render(line) + "\n")
	}
	sb.WriteString("\n")
	for _, ex := range examples {
		sb.WriteString("    " + lipgloss.NewStyle().Foreground(cyan).Render("  "+ex) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString("  " + hintStyle.Render("Type 'help' for more examples.") + "\n")

	return sb.String()
}

func renderHelp() string {
	var sb strings.Builder
	sb.WriteString("\n")

	categories := []struct {
		name     string
		examples []string
	}{
		{"Race Results & Wins", []string{
			"Who has the most wins?",
			"Who won the most races in 2020?",
			"Ferrari wins in 2004",
			"Podiums in 2022",
		}},
		{"Championships & Standings", []string{
			"Show me the 2021 championship standings",
			"Season overview 2010",
			"Who are the world champions?",
		}},
		{"Driver & Team Info", []string{
			"Tell me about Hamilton",
			"Who were Hamilton's teammates?",
			"How many points did Hamilton score in 2019?",
		}},
		{"Comparisons", []string{
			"Compare Verstappen vs Hamilton",
		}},
		{"Performance Data", []string{
			"Fastest pit stops in 2023",
			"Who got the most pole positions?",
			"What are the most common DNF reasons?",
			"Fastest laps at Silverstone",
		}},
		{"Circuits", []string{
			"Tell me about Monza",
			"Which circuits have hosted the most races?",
		}},
	}

	for _, cat := range categories {
		sb.WriteString("  " + descStyle.Render(cat.name) + "\n")
		for _, ex := range cat.examples {
			sb.WriteString("    " + hintStyle.Render("  "+ex) + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderResult(result *query.Result, termWidth int) string {
	var sb strings.Builder

	if len(result.Rows) == 0 {
		sb.WriteString("  " + descStyle.Render(result.Description) + "\n")
		sb.WriteString("  " + hintStyle.Render("No results found.") + "\n")
		return sb.String()
	}

	// Calculate column widths
	cols := result.Columns
	widths := make([]int, len(cols))
	for i, c := range cols {
		label := strings.ReplaceAll(strings.ToUpper(c), "_", " ")
		widths[i] = len(label)
	}
	for _, row := range result.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Cap widths
	maxCol := 40
	if termWidth > 0 {
		available := termWidth - 4 - (len(cols)-1)*3
		maxCol = available / len(cols)
		if maxCol < 10 {
			maxCol = 10
		}
		if maxCol > 40 {
			maxCol = 40
		}
	}
	for i := range widths {
		if widths[i] > maxCol {
			widths[i] = maxCol
		}
	}

	totalWidth := 0
	for _, w := range widths {
		totalWidth += w + 3
	}

	// Description
	sb.WriteString("\n  " + descStyle.Render(result.Description) + "\n")

	// Top border
	sb.WriteString("  " + borderStyle.Render(strings.Repeat("─", totalWidth)) + "\n")

	// Headers
	sb.WriteString("  ")
	for i, c := range cols {
		label := strings.ReplaceAll(strings.ToUpper(c), "_", " ")
		if len(label) > widths[i] {
			label = label[:widths[i]-1] + "…"
		}
		sb.WriteString(headerStyle.Render(fmt.Sprintf("%-*s", widths[i], label)))
		if i < len(cols)-1 {
			sb.WriteString(borderStyle.Render(" │ "))
		}
	}
	sb.WriteString("\n")

	// Header separator
	sb.WriteString("  " + borderStyle.Render(strings.Repeat("─", totalWidth)) + "\n")

	// Data rows
	for rowIdx, row := range result.Rows {
		sb.WriteString("  ")
		style := rowStyle
		if rowIdx%2 == 1 {
			style = altRowStyle
		}
		// Highlight top 3 with special treatment
		if rowIdx < 3 && len(result.Rows) > 3 {
			switch rowIdx {
			case 0:
				style = lipgloss.NewStyle().Foreground(yellow).Bold(true) // gold
			case 1:
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#C0C0C0")) // silver
			case 2:
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#CD7F32")) // bronze
			}
		}

		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if len(cell) > widths[i] {
				cell = cell[:widths[i]-1] + "…"
			}
			sb.WriteString(style.Render(fmt.Sprintf("%-*s", widths[i], cell)))
			if i < len(cols)-1 {
				sb.WriteString(borderStyle.Render(" │ "))
			}
		}
		sb.WriteString("\n")
	}

	// Bottom border
	sb.WriteString("  " + borderStyle.Render(strings.Repeat("─", totalWidth)) + "\n")

	// Row count
	sb.WriteString("  " + countStyle.Render(fmt.Sprintf("%d rows", len(result.Rows))) + "\n")

	return sb.String()
}
