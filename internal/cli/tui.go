package cli

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"agent-skills/internal/installer"
)

var errCanceled = errors.New("canceled")

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func selectIndicesTUI(title string, items []string, selected map[int]bool) ([]int, error) {
	if len(items) == 0 {
		return nil, errors.New("no items to select")
	}
	model := newMultiSelectModel(title, items, selected)
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

func newMultiSelectModel(title string, items []string, selected map[int]bool) multiSelectModel {
	if selected == nil {
		selected = make(map[int]bool)
	}
	return multiSelectModel{
		title:    title,
		items:    items,
		selected: selected,
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
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = cursorStyle.Render(">")
		}
		check := " "
		if m.selected[i] {
			check = selectedStyle.Render("x")
		}
		b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, check, item))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k or ↑/↓ to move, space to select, a to toggle all, enter to confirm, q to quit"))
	b.WriteString("\n")
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

func selectIndexTUI(title string, items []string) (int, error) {
	if len(items) == 0 {
		return -1, errors.New("no items to select")
	}
	model := newSingleSelectModel(title, items)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return -1, err
	}
	finalState, ok := finalModel.(singleSelectModel)
	if !ok {
		return -1, errors.New("unexpected TUI state")
	}
	if finalState.canceled {
		return -1, errCanceled
	}
	return finalState.selectedIndex, nil
}

type singleSelectModel struct {
	title         string
	items         []string
	cursor        int
	selectedIndex int
	canceled      bool
}

func newSingleSelectModel(title string, items []string) singleSelectModel {
	return singleSelectModel{
		title:         title,
		items:         items,
		selectedIndex: -1,
	}
}

func (m singleSelectModel) Init() tea.Cmd {
	return nil
}

func (m singleSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			m.selectedIndex = m.cursor
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m singleSelectModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")
	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = cursorStyle.Render(">")
		}
		b.WriteString(fmt.Sprintf("%s %s\n", cursor, item))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k or ↑/↓ to move, enter to confirm, q to quit"))
	b.WriteString("\n")
	return b.String()
}

func textInputTUI(title, prompt, value string) (string, error) {
	model := newTextInputModel(title, prompt, value)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}
	finalState, ok := finalModel.(textInputModel)
	if !ok {
		return "", errors.New("unexpected TUI state")
	}
	if finalState.canceled {
		return "", errCanceled
	}
	return strings.TrimSpace(finalState.value), nil
}

type textInputModel struct {
	title    string
	prompt   string
	value    string
	cursor   int
	canceled bool
}

func newTextInputModel(title, prompt, value string) textInputModel {
	return textInputModel{
		title:  title,
		prompt: prompt,
		value:  value,
		cursor: len(value),
	}
}

func (m textInputModel) Init() tea.Cmd {
	return nil
}

func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right":
			if m.cursor < len(m.value) {
				m.cursor++
			}
		case "backspace":
			if m.cursor > 0 {
				m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
				m.cursor--
			}
		case "delete":
			if m.cursor < len(m.value) {
				m.value = m.value[:m.cursor] + m.value[m.cursor+1:]
			}
		default:
			if msg.Type == tea.KeyRunes {
				insert := string(msg.Runes)
				m.value = m.value[:m.cursor] + insert + m.value[m.cursor:]
				m.cursor += len(insert)
			}
		}
	}
	return m, nil
}

func (m textInputModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")
	b.WriteString(m.prompt)
	b.WriteString("\n")
	if m.value == "" {
		b.WriteString(cursorStyle.Render("|"))
	} else {
		before := m.value[:m.cursor]
		after := m.value[m.cursor:]
		b.WriteString(before)
		b.WriteString(cursorStyle.Render("|"))
		b.WriteString(after)
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("type to edit, enter to confirm, q to quit"))
	b.WriteString("\n")
	return b.String()
}

func promptSkillsRootTUI(defaultRoot string, cfg appConfig, cwd string) (string, error) {
	items := []string{}
	paths := []string{}
	labels := []string{}

	if defaultRoot != "" {
		items = append(items, fmt.Sprintf("Bundled skills (%s)", defaultRoot))
		paths = append(paths, defaultRoot)
		labels = append(labels, "bundled")
	}
	if cfg.RepoURL != "" {
		items = append(items, fmt.Sprintf("GitHub repo (configured: %s)", cfg.RepoURL))
		paths = append(paths, "repo-url:"+cfg.RepoURL)
		labels = append(labels, "repo-url")
	}
	items = append(items, fmt.Sprintf("Current directory (%s)", cwd))
	paths = append(paths, cwd)
	labels = append(labels, "cwd")
	items = append(items, "Custom GitHub repo URL")
	paths = append(paths, "")
	labels = append(labels, "repo-url-custom")
	items = append(items, "Custom path")
	paths = append(paths, "")
	labels = append(labels, "custom")

	idx, err := selectIndexTUI("Select skills source", items)
	if err != nil {
		return "", err
	}
	if idx < 0 || idx >= len(labels) {
		return "", errors.New("invalid selection")
	}
	switch labels[idx] {
	case "repo-url-custom":
		value, err := textInputTUI("GitHub repo", "Enter repo URL or owner/name:", "")
		if err != nil {
			return "", err
		}
		return "repo-url:" + value, nil
	case "custom":
		return textInputTUI("Skills repo path", "Enter path to repo (folder containing skills/):", cwd)
	default:
		return paths[idx], nil
	}
}

func promptProjectPathTUI(cwd string) (string, error) {
	items := []string{
		"Skip project install",
		fmt.Sprintf("Use current directory (%s)", cwd),
		"Custom project path",
	}
	idx, err := selectIndexTUI("Project path", items)
	if err != nil {
		return "", err
	}
	switch idx {
	case 0:
		return "", nil
	case 1:
		return cwd, nil
	default:
		return textInputTUI("Project path", "Enter project path (folder containing .cursor/ or .claude/):", cwd)
	}
}

func promptInstallModeTUI() (installer.Mode, error) {
	items := []string{
		"Symlink (recommended)",
		"Copy files",
	}
	idx, err := selectIndexTUI("Install mode", items)
	if err != nil {
		return "", err
	}
	if idx == 1 {
		return installer.ModeCopy, nil
	}
	return installer.ModeSymlink, nil
}

func promptOverwriteTUI() (bool, error) {
	items := []string{
		"Overwrite existing skills",
		"Skip existing skills",
	}
	idx, err := selectIndexTUI("Overwrite existing skills?", items)
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}
