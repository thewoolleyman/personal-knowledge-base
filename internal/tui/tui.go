package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
)

// SearchFunc is the function signature for performing a search.
type SearchFunc func(ctx context.Context, query string) ([]connectors.Result, error)

type state int

const (
	stateInput   state = iota
	stateLoading
	stateResults
)

// searchResultMsg is sent when search results arrive.
type searchResultMsg struct {
	results []connectors.Result
	err     error
}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	searchInput textinput.Model
	searchFn    SearchFunc
	results     []connectors.Result
	cursor      int
	state       state
	err         error
}

// NewModel creates a new TUI model with the given search function.
func NewModel(searchFn SearchFunc) Model {
	ti := textinput.New()
	ti.Placeholder = "Search your knowledge base..."
	ti.Focus()
	ti.Width = 60

	return Model{
		searchInput: ti,
		searchFn:    searchFn,
		state:       stateInput,
	}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return textinput.Blink() }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case searchResultMsg:
		return m.handleSearchResult(msg)
	}

	// Pass other messages to the text input
	if m.state == stateInput {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEscape:
		if m.state != stateInput {
			m.state = stateInput
			m.searchInput.Focus()
			return m, nil
		}
		return m, tea.Quit

	case tea.KeyEnter:
		if m.state == stateInput {
			query := m.searchInput.Value()
			if query == "" {
				return m, nil
			}
			m.state = stateLoading
			return m, m.doSearch(query)
		}

	case tea.KeyUp:
		if m.state == stateResults && m.cursor > 0 {
			m.cursor--
		}

	case tea.KeyDown:
		if m.state == stateResults && m.cursor < len(m.results)-1 {
			m.cursor++
		}
	}

	// Forward to text input when in input state
	if m.state == stateInput {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleSearchResult(msg searchResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = stateInput
		m.searchInput.Focus()
		return m, nil
	}

	m.results = msg.results
	m.cursor = 0
	m.state = stateResults
	return m, nil
}

func (m Model) doSearch(query string) tea.Cmd {
	return func() tea.Msg {
		results, err := m.searchFn(context.Background(), query)
		return searchResultMsg{results: results, err: err}
	}
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	urlStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	sourceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
)

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  Search your knowledge base"))
	b.WriteString("\n\n")
	b.WriteString("  " + m.searchInput.View())
	b.WriteString("\n\n")

	switch m.state {
	case stateLoading:
		b.WriteString("  Searching...\n")

	case stateResults:
		if len(m.results) == 0 {
			b.WriteString("  No results found.\n")
		} else {
			b.WriteString(fmt.Sprintf("  %d results:\n\n", len(m.results)))
			for i, r := range m.results {
				cursor := "  "
				title := titleStyle.Render(r.Title)
				if i == m.cursor {
					cursor = "> "
					title = selectedStyle.Render(r.Title)
				}
				b.WriteString(fmt.Sprintf("  %s%s\n", cursor, title))
				b.WriteString(fmt.Sprintf("     %s\n", urlStyle.Render(r.URL)))
				b.WriteString(fmt.Sprintf("     %s\n\n", sourceStyle.Render("["+r.Source+"]")))
			}
		}
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %s\n", m.err))
	}

	b.WriteString("\n  esc: back • ctrl+c: quit")
	if m.state == stateResults {
		b.WriteString(" • ↑/↓: navigate • enter: open")
	}
	b.WriteString("\n")

	return b.String()
}
