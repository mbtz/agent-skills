package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkillsPreservesRelativePath(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "productivity", "grill-me")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: grill-me
description: Stress-test plans.
---
`), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	skills, err := DiscoverSkills(root)
	if err != nil {
		t.Fatalf("discover skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if got, want := skills[0].Name, "grill-me"; got != want {
		t.Fatalf("unexpected skill name: got %q want %q", got, want)
	}
	if got, want := skills[0].RelativePath, filepath.Join("productivity", "grill-me"); got != want {
		t.Fatalf("unexpected relative path: got %q want %q", got, want)
	}
}

func TestInstallSkillCanUseNestedDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target", "productivity", "grill-me")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	if err := InstallSkill(source, target, ModeCopy); err != nil {
		t.Fatalf("install skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Fatalf("expected nested installed skill: %v", err)
	}
}
