package installer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Mode string

const (
	ModeSymlink Mode = "symlink"
	ModeCopy    Mode = "copy"
)

type Skill struct {
	Name        string
	Description string
	Path        string
}

type TargetType string

const (
	TargetCodexGlobal  TargetType = "codex-global"
	TargetClaudeGlobal TargetType = "claude-global"
	TargetClaudeProject TargetType = "claude-project"
	TargetCursorGlobal TargetType = "cursor-global"
	TargetCursorProject TargetType = "cursor-project"
)

type Target struct {
	Type   TargetType
	Label  string
	Path   string
	Exists bool
}

func DiscoverSkills(skillsRoot string) ([]Skill, error) {
	rootInfo, err := os.Stat(skillsRoot)
	if err != nil {
		return nil, fmt.Errorf("skills root not found: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("skills root is not a directory: %s", skillsRoot)
	}

	var skills []Skill
	err = filepath.WalkDir(skillsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		skillFile := filepath.Join(path, "SKILL.md")
		info, err := os.Stat(skillFile)
		if err != nil || info.IsDir() {
			return nil
		}
		name, desc, err := parseSkillFrontmatter(skillFile)
		if err != nil {
			return fmt.Errorf("parse %s: %w", skillFile, err)
		}
		if name == "" {
			name = filepath.Base(path)
		}
		skills = append(skills, Skill{
			Name:        name,
			Description: desc,
			Path:        path,
		})
		return fs.SkipDir
	})
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return nil, errors.New("no skills found")
	}
	return skills, nil
}

func DiscoverTargets(homeDir, projectPath string) []Target {
	var targets []Target

	codexPath := filepath.Join(homeDir, ".codex", "skills")
	if existsDir(codexPath) {
		targets = append(targets, Target{
			Type:   TargetCodexGlobal,
			Label:  "Codex CLI (global)",
			Path:   codexPath,
			Exists: true,
		})
	}

	claudeGlobal := filepath.Join(homeDir, ".claude", "skills")
	if existsDir(claudeGlobal) {
		targets = append(targets, Target{
			Type:   TargetClaudeGlobal,
			Label:  "Claude Code (global)",
			Path:   claudeGlobal,
			Exists: true,
		})
	}

	if projectPath != "" {
		claudeProject := filepath.Join(projectPath, ".claude", "skills")
		targets = append(targets, Target{
			Type:   TargetClaudeProject,
			Label:  "Claude Code (project)",
			Path:   claudeProject,
			Exists: existsDir(claudeProject),
		})

		cursorProject := filepath.Join(projectPath, ".cursor", "skills")
		targets = append(targets, Target{
			Type:   TargetCursorProject,
			Label:  "Cursor (project)",
			Path:   cursorProject,
			Exists: existsDir(cursorProject),
		})
	}

	cursorGlobal := filepath.Join(homeDir, ".cursor", "skills")
	if existsDir(cursorGlobal) {
		targets = append(targets, Target{
			Type:   TargetCursorGlobal,
			Label:  "Cursor (global)",
			Path:   cursorGlobal,
			Exists: true,
		})
	}

	return targets
}

func InstallSkill(srcDir, destDir string, mode Mode) error {
	switch mode {
	case ModeSymlink:
		return installSymlink(srcDir, destDir)
	case ModeCopy:
		return copyDir(srcDir, destDir)
	default:
		return fmt.Errorf("unknown install mode: %s", mode)
	}
}

func installSymlink(srcDir, destDir string) error {
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("remove existing target: %w", err)
	}
	return os.Symlink(srcDir, destDir)
}

func copyDir(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destDir, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			if err := os.RemoveAll(targetPath); err != nil {
				return err
			}
			return os.Symlink(linkTarget, targetPath)
		}
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		return copyFile(path, targetPath, info.Mode())
	})
}

func copyFile(src, dest string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, mode)
}

func parseSkillFrontmatter(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	inFrontmatter := false
	var name, desc string
	for scanner.Scan() {
		line := scanner.Text()
		lineNo++
		if lineNo == 1 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			break
		}
		if !inFrontmatter {
			break
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
		}
		if strings.HasPrefix(trimmed, "description:") {
			desc = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return name, desc, nil
}

func existsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
