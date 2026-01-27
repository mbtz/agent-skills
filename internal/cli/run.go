package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"agent-skills/internal/installer"
)

type Options struct {
	CommandName string
}

func Run(args []string, opts Options) error {
	cmdName := opts.CommandName
	if cmdName == "" {
		cmdName = filepath.Base(args[0])
	}

	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	repoRoot := fs.String("repo", "", "path to skills repo (defaults to current directory)")
	projectPath := fs.String("project", "", "project path for project-local installs")
	allSkills := fs.Bool("all", false, "install all skills without prompt")
	copyMode := fs.Bool("copy", false, "copy files instead of symlink")
	symlinkMode := fs.Bool("symlink", false, "force symlink mode")
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *showVersion {
		fmt.Printf("%s %s\n", cmdName, Version)
		return nil
	}

	root := *repoRoot
	project := *projectPath
	mode := installer.ModeSymlink
	if *copyMode {
		mode = installer.ModeCopy
	}
	if *symlinkMode {
		mode = installer.ModeSymlink
	}
	all := *allSkills

	if len(args) == 1 {
		cfg, err := promptConfig()
		if err != nil {
			return err
		}
		root = cfg.root
		project = cfg.project
		mode = cfg.mode
		all = cfg.all
	}

	if *copyMode && *symlinkMode {
		return errors.New("choose only one of --copy or --symlink")
	}

	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		root = cwd
	}

	skillsRoot := filepath.Join(root, "skills")
	skills, err := installer.DiscoverSkills(skillsRoot)
	if err != nil {
		return fmt.Errorf("discover skills: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determine home directory: %w", err)
	}

	targets := installer.DiscoverTargets(homeDir, project)
	if len(targets) == 0 {
		return fmt.Errorf("no install targets found under %s. Create a harness folder or pass --project", homeDir)
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	selectedTargets := targets
	if len(targets) > 1 {
		indices := promptIndices("Select install targets (e.g. 1,3):", targetsSummary(targets))
		selectedTargets = filterTargets(targets, indices)
		if len(selectedTargets) == 0 {
			return errors.New("no targets selected")
		}
	}

	selectedSkills := skills
	if !all {
		indices := promptIndices("Select skills to install (e.g. 1,2,5):", skillsSummary(skills))
		selectedSkills = filterSkills(skills, indices)
		if len(selectedSkills) == 0 {
			return errors.New("no skills selected")
		}
	}

	reader := bufio.NewReader(os.Stdin)
	for _, target := range selectedTargets {
		if err := os.MkdirAll(target.Path, 0o755); err != nil {
			return fmt.Errorf("create target %s: %w", target.Path, err)
		}
		for _, skill := range selectedSkills {
			dest := filepath.Join(target.Path, filepath.Base(skill.Path))
			if _, err := os.Stat(dest); err == nil {
				if !confirm(reader, fmt.Sprintf("%s exists in %s. Overwrite? [y/N]: ", filepath.Base(skill.Path), target.Label)) {
					fmt.Printf("Skipping %s for %s\n", skill.Name, target.Label)
					continue
				}
				if err := os.RemoveAll(dest); err != nil {
					return fmt.Errorf("remove existing %s: %w", dest, err)
				}
			}
			if err := installer.InstallSkill(skill.Path, dest, mode); err != nil {
				return fmt.Errorf("install %s to %s: %w", skill.Name, target.Label, err)
			}
			fmt.Printf("Installed %s to %s (%s)\n", skill.Name, target.Label, mode)
		}
	}

	return nil
}

type config struct {
	root    string
	project string
	mode    installer.Mode
	all     bool
}

func promptConfig() (config, error) {
	reader := bufio.NewReader(os.Stdin)
	cwd, _ := os.Getwd()

	root, err := promptString(reader, "Skills repo path (enter for current dir): ", cwd)
	if err != nil {
		return config{}, err
	}

	project, err := promptString(reader, "Project path (enter to skip): ", "")
	if err != nil {
		return config{}, err
	}

	modeChoice := promptChoice("Install mode:", []string{"symlink", "copy"})
	mode := installer.ModeSymlink
	if modeChoice == 2 {
		mode = installer.ModeCopy
	}

	allChoice := promptChoice("Install all skills without prompt?", []string{"no", "yes"})
	all := allChoice == 2

	return config{
		root:    strings.TrimSpace(root),
		project: strings.TrimSpace(project),
		mode:    mode,
		all:     all,
	}, nil
}

func promptString(reader *bufio.Reader, prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s[%s] ", prompt, defaultValue)
	} else {
		fmt.Print(prompt)
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue, nil
	}
	return text, nil
}

func promptChoice(prompt string, items []string) int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	for i, item := range items {
		fmt.Printf("%d) %s\n", i+1, item)
	}
	fmt.Print("> ")
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	value, err := strconv.Atoi(text)
	if err != nil || value < 1 || value > len(items) {
		return 1
	}
	return value
}

func promptIndices(prompt string, items []string) []int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(prompt)
	for i, item := range items {
		fmt.Printf("%d) %s\n", i+1, item)
	}
	fmt.Print("> ")
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	raw := strings.Split(text, ",")
	var indices []int
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		indices = append(indices, value-1)
	}
	return indices
}

func confirm(reader *bufio.Reader, prompt string) bool {
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(strings.ToLower(text))
	return text == "y" || text == "yes"
}

func targetsSummary(targets []installer.Target) []string {
	items := make([]string, 0, len(targets))
	for _, target := range targets {
		status := "missing (will create)"
		if target.Exists {
			status = "found"
		}
		items = append(items, fmt.Sprintf("%s - %s (%s)", target.Label, target.Path, status))
	}
	return items
}

func skillsSummary(skills []installer.Skill) []string {
	items := make([]string, 0, len(skills))
	for _, skill := range skills {
		desc := strings.TrimSpace(skill.Description)
		if desc == "" {
			desc = "no description"
		}
		items = append(items, fmt.Sprintf("%s - %s", skill.Name, desc))
	}
	return items
}

func filterTargets(targets []installer.Target, indices []int) []installer.Target {
	if len(indices) == 0 {
		return nil
	}
	var out []installer.Target
	for _, idx := range indices {
		if idx >= 0 && idx < len(targets) {
			out = append(out, targets[idx])
		}
	}
	return out
}

func filterSkills(skills []installer.Skill, indices []int) []installer.Skill {
	if len(indices) == 0 {
		return nil
	}
	var out []installer.Skill
	for _, idx := range indices {
		if idx >= 0 && idx < len(skills) {
			out = append(out, skills[idx])
		}
	}
	return out
}
