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
	defaultStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
)

func selectIndicesTUI(title string, items []string, selected map[int]bool, showDefaultLabel bool) ([]int, error) {
	if len(items) == 0 {
		return nil, errors.New("no items to select")
	}
	model := newMultiSelectModel(title, items, selected, showDefaultLabel)
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
	title            string
	items            []string
	cursor           int
	selected         map[int]bool
	defaults         map[int]bool
	showDefaultLabel bool
	canceled         bool
	confirmed        bool
}

func newMultiSelectModel(title string, items []string, selected map[int]bool, showDefaultLabel bool) multiSelectModel {
	if selected == nil {
		selected = make(map[int]bool)
	}
	defaults := make(map[int]bool, len(selected))
	for idx := range selected {
		defaults[idx] = true
	}
	return multiSelectModel{
		title:            title,
		items:            items,
		selected:         selected,
		defaults:         defaults,
		showDefaultLabel: showDefaultLabel,
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
		label := ""
		if m.showDefaultLabel && m.defaults[i] {
			label = " " + defaultStyle.Render("default")
		}
		b.WriteString(fmt.Sprintf("%s [%s] %s%s\n", cursor, check, item, label))
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

func selectIndexTUI(title string, items []string, defaultIndex int) (int, error) {
	if len(items) == 0 {
		return -1, errors.New("no items to select")
	}
	model := newSingleSelectModel(title, items, defaultIndex)
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
	defaultIndex  int
	canceled      bool
}

func newSingleSelectModel(title string, items []string, defaultIndex int) singleSelectModel {
	if defaultIndex < 0 || defaultIndex >= len(items) {
		defaultIndex = 0
	}
	return singleSelectModel{
		title:         title,
		items:         items,
		selectedIndex: -1,
		defaultIndex:  defaultIndex,
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
		label := ""
		if i == m.defaultIndex {
			label = " " + defaultStyle.Render("default")
		}
		b.WriteString(fmt.Sprintf("%s %s%s\n", cursor, item, label))
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
		paths = append(paths, "bundled")
		labels = append(labels, "bundled")
	}
	if cfg.SkillRepoPath != "" && cfg.SkillRepoPath != "bundled" && cfg.SkillRepoPath != "cwd" {
		items = append(items, fmt.Sprintf("Configured skill repo (%s)", cfg.SkillRepoPath))
		paths = append(paths, cfg.SkillRepoPath)
		labels = append(labels, "configured")
	}
	items = append(items, fmt.Sprintf("Current directory (%s)", cwd))
	paths = append(paths, "cwd")
	labels = append(labels, "cwd")
	items = append(items, "Custom GitHub repo URL")
	paths = append(paths, "")
	labels = append(labels, "repo-url-custom")
	items = append(items, "Custom path")
	paths = append(paths, "")
	labels = append(labels, "custom")

	defaultIndex := defaultSkillsSourceIndex(cfg, labels)
	idx, err := selectIndexTUI("Select skills source", items, defaultIndex)
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
		return value, nil
	case "custom":
		return textInputTUI("Skills repo path", "Enter path to repo (folder containing skills/):", cwd)
	default:
		return paths[idx], nil
	}
}

func promptProjectPathTUI(cwd string, cfg appConfig) (string, error) {
	items := []string{
		"Skip project install",
		fmt.Sprintf("Use current directory (%s)", cwd),
		"Custom project path",
	}
	idx, err := selectIndexTUI("Project path", items, defaultProjectChoiceIndex(cfg))
	if err != nil {
		return "", err
	}
	switch idx {
	case 0:
		return "", nil
	case 1:
		return cwd, nil
	default:
		defaultPath := strings.TrimSpace(cfg.ProjectPath)
		if defaultPath == "" {
			defaultPath = cwd
		}
		return textInputTUI("Project path", "Enter project path (folder containing .cursor/ or .claude/):", defaultPath)
	}
}

func promptInstallModeTUI(cfg appConfig) (installer.Mode, error) {
	items := []string{
		"Symlink (recommended)",
		"Copy files",
	}
	idx, err := selectIndexTUI("Install mode", items, defaultInstallModeIndex(cfg))
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
		"Skip existing skills",
		"Overwrite existing skills",
	}
	idx, err := selectIndexTUI("Overwrite existing skills?", items, 0)
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}

func defaultSkillsSourceIndex(cfg appConfig, labels []string) int {
	switch cfg.SkillRepoPath {
	case "bundled":
		return indexOfLabel(labels, "bundled")
	case "cwd":
		return indexOfLabel(labels, "cwd")
	case "":
		if idx := indexOfLabel(labels, "bundled"); idx >= 0 {
			return idx
		}
		return indexOfLabel(labels, "cwd")
	default:
		if idx := indexOfLabel(labels, "configured"); idx >= 0 {
			return idx
		}
	}
	if idx := indexOfLabel(labels, "bundled"); idx >= 0 {
		return idx
	}
	if idx := indexOfLabel(labels, "cwd"); idx >= 0 {
		return idx
	}
	return 0
}

func defaultProjectChoiceIndex(cfg appConfig) int {
	switch cfg.ProjectChoice {
	case "cwd":
		return 1
	case "custom":
		return 2
	default:
		return 0
	}
}

func defaultInstallModeIndex(cfg appConfig) int {
	if strings.EqualFold(cfg.InstallMode, string(installer.ModeCopy)) {
		return 1
	}
	return 0
}

func indexOfLabel(labels []string, target string) int {
	for i, label := range labels {
		if label == target {
			return i
		}
	}
	return 0
}
