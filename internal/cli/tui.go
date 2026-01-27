package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var errCanceled = errors.New("canceled")

type multiSelect struct {
	title    string
	items    []string
	cursor   int
	selected map[int]bool
}

func selectIndicesTUI(title string, items []string) ([]int, error) {
	if len(items) == 0 {
		return nil, errors.New("no items to select")
	}
	restore, err := enterRawMode()
	if err != nil {
		return nil, err
	}
	defer restore()

	state := multiSelect{
		title:    title,
		items:    items,
		selected: make(map[int]bool),
	}

	render := func() {
		var b strings.Builder
		b.WriteString("\033[H\033[2J")
		b.WriteString(state.title)
		b.WriteString("\n\n")
		for i, item := range state.items {
			cursor := " "
			if state.cursor == i {
				cursor = ">"
			}
			check := " "
			if state.selected[i] {
				check = "x"
			}
			b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, check, item))
		}
		b.WriteString("\n")
		b.WriteString("j/k or ↑/↓ to move, space to select, a to toggle all, enter to confirm, q to quit\n")
		fmt.Print(b.String())
	}

	render()
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf[:1])
		if err != nil || n == 0 {
			return nil, errors.New("failed to read input")
		}
		b := buf[0]
		switch b {
		case 'q':
			fmt.Print("\n")
			return nil, errCanceled
		case 'j':
			if state.cursor < len(state.items)-1 {
				state.cursor++
			}
		case 'k':
			if state.cursor > 0 {
				state.cursor--
			}
		case ' ':
			state.selected[state.cursor] = !state.selected[state.cursor]
		case 'a':
			if len(state.selected) == len(state.items) {
				state.selected = make(map[int]bool)
			} else {
				for i := range state.items {
					state.selected[i] = true
				}
			}
		case '\r', '\n':
			fmt.Print("\n")
			var out []int
			for i := range state.items {
				if state.selected[i] {
					out = append(out, i)
				}
			}
			return out, nil
		case 0x1b:
			n, _ := os.Stdin.Read(buf[:2])
			if n == 2 && buf[0] == '[' {
				switch buf[1] {
				case 'A':
					if state.cursor > 0 {
						state.cursor--
					}
				case 'B':
					if state.cursor < len(state.items)-1 {
						state.cursor++
					}
				}
			}
		}
		render()
	}
}

func enterRawMode() (func(), error) {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("capture terminal state: %w", err)
	}
	original := strings.TrimSpace(string(output))
	raw := exec.Command("stty", "raw", "-echo")
	raw.Stdin = os.Stdin
	if err := raw.Run(); err != nil {
		return nil, fmt.Errorf("enter raw mode: %w", err)
	}
	restore := func() {
		restoreCmd := exec.Command("stty", original)
		restoreCmd.Stdin = os.Stdin
		restoreCmd.Stdout = &bytes.Buffer{}
		restoreCmd.Stderr = &bytes.Buffer{}
		_ = restoreCmd.Run()
	}
	return restore, nil
}
