package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"agent-skills/internal/installer"

	"github.com/BurntSushi/toml"
)

type Options struct {
	CommandName string
}

func Run(args []string, opts Options) error {
	cmdName := opts.CommandName
	if cmdName == "" {
		cmdName = filepath.Base(args[0])
	}

	if len(args) > 1 && args[1] == "config" {
		return runConfigCommand(args[2:], cmdName)
	}

	fs := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var repoRoot string
	var projectPath string
	var copyMode bool
	var symlinkMode bool
	var showVersion bool
	var fromConfig bool

	fs.StringVar(&repoRoot, "repo", "", "path to skills repo (defaults to current directory)")
	fs.StringVar(&repoRoot, "r", "", "alias for --repo")
	fs.StringVar(&projectPath, "project", "", "project path for project-local installs")
	fs.StringVar(&projectPath, "p", "", "alias for --project")
	fs.BoolVar(&copyMode, "copy", false, "copy files instead of symlink")
	fs.BoolVar(&copyMode, "c", false, "alias for --copy")
	fs.BoolVar(&symlinkMode, "symlink", false, "force symlink mode")
	fs.BoolVar(&symlinkMode, "s", false, "alias for --symlink")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.BoolVar(&showVersion, "v", false, "alias for --version")
	fs.BoolVar(&fromConfig, "from-config", false, "install all skills using config defaults")
	fs.BoolVar(&fromConfig, "f", false, "alias for --from-config")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "Usage: %s [options]\n", cmdName)
		fmt.Fprintf(out, "       %s config [--init] [-e|--edit]\n\n", cmdName)
		fmt.Fprintln(out, "Run without options to open the interactive TUI installer.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Options:")
		tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "  -r, --repo\tPath to skills repo (defaults to current directory)")
		fmt.Fprintln(tw, "  -p, --project\tProject path for project-local installs")
		fmt.Fprintln(tw, "  -c, --copy\tCopy files instead of symlink")
		fmt.Fprintln(tw, "  -s, --symlink\tForce symlink mode")
		fmt.Fprintln(tw, "  -f, --from-config\tInstall all skills using config defaults")
		fmt.Fprintln(tw, "  -v, --version\tPrint version and exit")
		fmt.Fprintln(tw, "  -h, --help\tShow help")
		_ = tw.Flush()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Config command:")
		tw = tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "  --init\tCreate config file with defaults")
		fmt.Fprintln(tw, "  -e, --edit\tEdit config in $EDITOR/$VISUAL")
		fmt.Fprintln(tw, "  Config path\t~/.config/askill/config.toml")
		_ = tw.Flush()
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if showVersion {
		fmt.Printf("%s %s\n", cmdName, Version)
		return nil
	}

	maybeSuggestBrewUpgrade(Version)

	root := repoRoot
	project := projectPath
	mode := installer.ModeSymlink
	if copyMode {
		mode = installer.ModeCopy
	}
	if symlinkMode {
		mode = installer.ModeSymlink
	}

	defaultRoot, defaultRootErr := detectRepoRoot()
	cfg, cfgErr := loadConfig()
	if (len(args) == 1 || fromConfig) && cfgErr != nil {
		return cfgErr
	}

	if fromConfig {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		defaultCfg := withDefaultConfig(cfg, defaultRoot, cwd)
		resolvedRoot, cleanup, err := resolveSkillRepoPath(defaultCfg.SkillRepoPath, defaultRoot, cwd)
		if err != nil {
			return err
		}
		if cleanup != nil {
			defer cleanup()
		}
		root = resolvedRoot
		project = resolveProjectPath(defaultCfg, cwd)
		mode = resolveInstallMode(defaultCfg)
	} else if len(args) == 1 {
		upgradeBanner := maybeUpgradeBanner(Version)
		advanced, err := promptInstallFlowTUI(upgradeBanner)
		if err != nil {
			return err
		}

		if !advanced {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			defaultCfg := withDefaultConfig(cfg, defaultRoot, cwd)
			resolvedRoot, cleanup, err := resolveSkillRepoPath(defaultCfg.SkillRepoPath, defaultRoot, cwd)
			if err != nil {
				return err
			}
			if cleanup != nil {
				defer cleanup()
			}
			root = resolvedRoot
			project = resolveProjectPath(defaultCfg, cwd)
			mode = resolveInstallMode(defaultCfg)
		} else {
			selection, err := promptSourceSelectionTUI(defaultRoot, cfg)
			if err != nil {
				return err
			}
			root = selection.root
			if selection.cleanup != nil {
				defer selection.cleanup()
			}
			cfgPrompt, err := promptConfigTUI(root, cfg)
			if err != nil {
				return err
			}
			root = cfgPrompt.root
			project = cfgPrompt.project
			mode = cfgPrompt.mode
		}
	} else if cfgErr != nil {
		return cfgErr
	}

	if copyMode && symlinkMode {
		return errors.New("choose only one of --copy or --symlink")
	}

	if repoRoot != "" {
		root = repoRoot
	}
	if projectPath != "" {
		project = projectPath
	}
	if copyMode {
		mode = installer.ModeCopy
	}
	if symlinkMode {
		mode = installer.ModeSymlink
	}

	if root == "" {
		if defaultRootErr == nil && defaultRoot != "" {
			root = defaultRoot
		}
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

	var overwriteAll bool
	selectedTargets := targets
	if len(args) == 1 {
		indices, err := selectIndicesTUI("Select install targets", targetsSummary(targets), defaultSelectAll(len(targets)), false)
		if err != nil {
			if errors.Is(err, errCanceled) {
				return nil
			}
			return err
		}
		selectedTargets = filterTargets(targets, indices)
		if len(selectedTargets) == 0 {
			return errors.New("no targets selected")
		}
		overwriteAll, err = promptOverwriteTUI()
		if err != nil {
			if errors.Is(err, errCanceled) {
				return nil
			}
			return err
		}
	} else if len(targets) > 1 {
		indices := promptIndices("Select install targets (e.g. 1,3):", targetsSummary(targets))
		selectedTargets = filterTargets(targets, indices)
		if len(selectedTargets) == 0 {
			return errors.New("no targets selected")
		}
	}

	selectedSkills := skills
	var indices []int
	var skillsErr error
	if len(args) == 1 {
		indices, skillsErr = selectIndicesTUI("Select skills to install", skillsSummary(skills), defaultSelectAll(len(skills)), false)
		if skillsErr != nil {
			if errors.Is(skillsErr, errCanceled) {
				return nil
			}
			return skillsErr
		}
	} else {
		indices = promptIndices("Select skills to install (e.g. 1,2,5):", skillsSummary(skills))
	}
	selectedSkills = filterSkills(skills, indices)
	if len(selectedSkills) == 0 {
		return errors.New("no skills selected")
	}

	reader := bufio.NewReader(os.Stdin)
	for _, target := range selectedTargets {
		if err := os.MkdirAll(target.Path, 0o755); err != nil {
			return fmt.Errorf("create target %s: %w", target.Path, err)
		}
		for _, skill := range selectedSkills {
			dest := filepath.Join(target.Path, filepath.Base(skill.Path))
			if _, err := os.Stat(dest); err == nil {
				if len(args) == 1 {
					if !overwriteAll {
						fmt.Printf("Skipping %s for %s\n", skill.Name, target.Label)
						continue
					}
				} else if !confirm(reader, fmt.Sprintf("%s exists in %s. Overwrite? [y/N]: ", filepath.Base(skill.Path), target.Label)) {
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
}

type appConfig struct {
	SkillRepoPath string `toml:"skill-repo-path"`
	ProjectChoice string `toml:"project-choice"`
	ProjectPath   string `toml:"project-path"`
	InstallMode   string `toml:"install-mode"`
}

type configSelection struct {
	root    string
	cleanup func()
}

func promptConfigTUI(root string, cfg appConfig) (config, error) {
	cwd, _ := os.Getwd()
	defaultCfg := withDefaultConfig(cfg, "", cwd)
	project, err := promptProjectPathTUI(cwd, defaultCfg)
	if err != nil {
		return config{}, err
	}

	mode, err := promptInstallModeTUI(defaultCfg)
	if err != nil {
		return config{}, err
	}

	return config{
		root:    strings.TrimSpace(root),
		project: strings.TrimSpace(project),
		mode:    mode,
	}, nil
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

func detectRepoRoot() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err == nil {
		executable = resolved
	}
	exeDir := filepath.Dir(executable)
	sharedSkills := filepath.Clean(filepath.Join(exeDir, "..", "share", "askill", "skills"))
	if installer.ExistsDir(sharedSkills) {
		return filepath.Dir(sharedSkills), nil
	}
	return "", errors.New("no bundled skills path found")
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

func defaultSelectAll(count int) map[int]bool {
	selected := make(map[int]bool, count)
	for i := 0; i < count; i++ {
		selected[i] = true
	}
	return selected
}

func promptInstallFlowTUI(banner string) (bool, error) {
	items := []string{
		"Default install (bundled skills, symlink, no project path)",
		"Advanced (choose source, project path, install mode)",
	}
	idx, err := selectIndexTUI("Install mode", items, 0, banner)
	if err != nil {
		return false, err
	}
	return idx == 1, nil
}

func promptSourceSelectionTUI(defaultRoot string, cfg appConfig) (configSelection, error) {
	cwd, _ := os.Getwd()
	defaultCfg := withDefaultConfig(cfg, defaultRoot, cwd)
	root, err := promptSkillsRootTUI(defaultRoot, defaultCfg, cwd)
	if err != nil {
		return configSelection{}, err
	}
	if root == "" {
		return configSelection{}, errors.New("no skills source selected")
	}
	resolved, cleanup, err := resolveSkillRepoPath(root, defaultRoot, cwd)
	if err != nil {
		return configSelection{}, err
	}
	return configSelection{root: resolved, cleanup: cleanup}, nil
}

func resolveSkillRepoPath(value, defaultRoot, cwd string) (string, func(), error) {
	switch strings.TrimSpace(value) {
	case "", "bundled":
		if defaultRoot != "" {
			return defaultRoot, nil, nil
		}
		if cwd != "" {
			return cwd, nil, nil
		}
	case "cwd":
		if cwd != "" {
			return cwd, nil, nil
		}
	}
	if value == "" {
		return "", nil, errors.New("empty skills repo path")
	}
	if installer.ExistsDir(value) {
		return value, nil, nil
	}
	return cloneRepo(value)
}

func cloneRepo(repo string) (string, func(), error) {
	repoURL := normalizeRepoURL(repo)
	tempDir, err := os.MkdirTemp("", "askill-repo-*")
	if err != nil {
		return "", nil, err
	}
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("clone %s: %w", repoURL, err)
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	return tempDir, cleanup, nil
}

func normalizeRepoURL(repo string) string {
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "git@") {
		return repo
	}
	if strings.HasPrefix(repo, "github.com/") {
		return "https://" + repo + ".git"
	}
	if strings.Contains(repo, "/") {
		return "https://github.com/" + repo + ".git"
	}
	return repo
}

func loadConfig() (appConfig, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return appConfig{}, err
	}
	path := filepath.Join(configDir, "askill", "config.toml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return appConfig{}, nil
		}
		return appConfig{}, err
	}
	var cfg appConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return appConfig{}, err
	}
	return cfg, nil
}

func runConfigCommand(args []string, cmdName string) error {
	fs := flag.NewFlagSet(cmdName+" config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var edit bool
	var init bool
	fs.BoolVar(&edit, "edit", false, "edit config in $EDITOR/$VISUAL")
	fs.BoolVar(&edit, "e", false, "alias for --edit")
	fs.BoolVar(&init, "init", false, "create config with defaults if missing")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintf(out, "Usage: %s config [--init] [-e|--edit]\n\n", cmdName)
		fmt.Fprintln(out, "Options:")
		tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "  --init\tCreate config file with defaults")
		fmt.Fprintln(tw, "  -e, --edit\tEdit config in $EDITOR/$VISUAL")
		fmt.Fprintln(tw, "  -h, --help\tShow help")
		_ = tw.Flush()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	if init {
		if err := ensureConfigFile(configPath); err != nil {
			return err
		}
	}

	if edit {
		if err := ensureConfigFile(configPath); err != nil {
			return err
		}
		return editConfigFile(configPath)
	}

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Config not found at %s\n", configPath)
		fmt.Println("Run `askill config --init` to create it.")
		return nil
	} else if err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := printConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("\nConfig path: %s\n", configPath)
	return nil
}

func configFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "askill", "config.toml"), nil
}

func ensureConfigFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	defaultRoot, _ := detectRepoRoot()
	cwd, _ := os.Getwd()
	defaults := withDefaultConfig(appConfig{}, defaultRoot, cwd)
	var b strings.Builder
	if err := toml.NewEncoder(&b).Encode(defaults); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func editConfigFile(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func printConfig(cfg appConfig) error {
	if err := toml.NewEncoder(os.Stdout).Encode(cfg); err != nil {
		return err
	}
	return nil
}

func withDefaultConfig(cfg appConfig, defaultRoot, cwd string) appConfig {
	if strings.TrimSpace(cfg.SkillRepoPath) == "" {
		if defaultRoot != "" {
			cfg.SkillRepoPath = "bundled"
		} else {
			cfg.SkillRepoPath = "cwd"
		}
	}
	if strings.TrimSpace(cfg.ProjectChoice) == "" {
		cfg.ProjectChoice = "skip"
	}
	if strings.TrimSpace(cfg.InstallMode) == "" {
		cfg.InstallMode = "symlink"
	}
	if cfg.ProjectChoice != "custom" {
		cfg.ProjectPath = strings.TrimSpace(cfg.ProjectPath)
	}
	return cfg
}

func resolveProjectPath(cfg appConfig, cwd string) string {
	switch cfg.ProjectChoice {
	case "cwd":
		return cwd
	case "custom":
		return strings.TrimSpace(cfg.ProjectPath)
	default:
		return ""
	}
}

func resolveInstallMode(cfg appConfig) installer.Mode {
	if strings.EqualFold(cfg.InstallMode, string(installer.ModeCopy)) {
		return installer.ModeCopy
	}
	return installer.ModeSymlink
}

type brewInfo struct {
	Formulae []struct {
		Name     string `json:"name"`
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	} `json:"formulae"`
}

func maybeUpgradeBanner(current string) string {
	latest, err := brewStableVersion()
	if err != nil || latest == "" || current == "" {
		return ""
	}
	if compareVersions(current, latest) >= 0 {
		return ""
	}
	return fmt.Sprintf("A newer askill version (%s) is available. Run: brew update && brew upgrade askill", latest)
}

func brewStableVersion() (string, error) {
	cmd := exec.Command("brew", "info", "--json=v2", "askill")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var info brewInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return "", err
	}
	for _, formula := range info.Formulae {
		if formula.Name == "askill" {
			return strings.TrimSpace(strings.TrimPrefix(formula.Versions.Stable, "v")), nil
		}
	}
	return "", errors.New("askill formula not found")
}

func compareVersions(a, b string) int {
	a = strings.TrimPrefix(strings.TrimSpace(a), "v")
	b = strings.TrimPrefix(strings.TrimSpace(b), "v")
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var aVal, bVal int
		if i < len(aParts) {
			aVal = parseVersionPart(aParts[i])
		}
		if i < len(bParts) {
			bVal = parseVersionPart(bParts[i])
		}
		if aVal < bVal {
			return -1
		}
		if aVal > bVal {
			return 1
		}
	}
	return 0
}

func parseVersionPart(part string) int {
	part = strings.TrimSpace(part)
	part = strings.TrimLeft(part, "v")
	n, err := strconv.Atoi(part)
	if err != nil {
		return 0
	}
	return n
}
