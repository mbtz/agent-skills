package cli

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var errCanceled = errors.New("canceled")

func selectIndicesTUI(title string, items []string) ([]int, error) {
	if len(items) == 0 {
		return nil, errors.New("no items to select")
	}
	model := newMultiSelectModel(title, items)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}
	finalState, ok := finalModel.(multiSelectModel)
	if !ok {
		return nil, errors.New("unexpected TUI state")
	}
	if finalState.canceled {
		return nil, errCanceled
	}
	return finalState.selectedIndices(), nil
}

type multiSelectModel struct {
	title     string
	items     []string
	cursor    int
	selected  map[int]bool
	canceled  bool
	confirmed bool
}

func newMultiSelectModel(title string, items []string) multiSelectModel {
	return multiSelectModel{
		title:    title,
		items:    items,
		selected: make(map[int]bool),
	}
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			if len(m.selected) == len(m.items) {
				m.selected = make(map[int]bool)
			} else {
				for i := range m.items {
					m.selected[i] = true
				}
			}
		}
	}
	return m, nil
}

func (m multiSelectModel) View() string {
	var b strings.Builder
	b.WriteString(m.title)
	b.WriteString("\n\n")
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		check := " "
		if m.selected[i] {
			check = "x"
		}
		b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, check, item))
	}
	b.WriteString("\n")
	b.WriteString("j/k or ↑/↓ to move, space to select, a to toggle all, enter to confirm, q to quit\n")
	return b.String()
}

func (m multiSelectModel) selectedIndices() []int {
	if len(m.selected) == 0 {
		return nil
	}
	out := make([]int, 0, len(m.selected))
	for i := range m.items {
		if m.selected[i] {
			out = append(out, i)
		}
	}
	return out
}
