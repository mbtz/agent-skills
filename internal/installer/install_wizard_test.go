package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWizardBuildQuestionsAndApply(t *testing.T) {
	skillDir := t.TempDir()
	configPath := filepath.Join(skillDir, "scripts", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"vault_root":"~/existing","sessions_root":"~/.codex/sessions"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	spec := `{
  "version": 1,
  "title": "Setup",
  "files": [
    {
      "path": "scripts/config.json",
      "format": "json",
      "fields": [
        {"key": "vault_root", "prompt": "Vault root", "required": true},
        {"key": "sessions_root", "prompt": "Sessions root", "default": "~/.codex/sessions", "required": true}
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(skillDir, InstallWizardFileName), []byte(spec), 0o644); err != nil {
		t.Fatalf("write wizard: %v", err)
	}

	wizard, err := LoadSkillInstallWizard(skillDir)
	if err != nil {
		t.Fatalf("load wizard: %v", err)
	}
	if wizard == nil {
		t.Fatal("expected wizard")
	}

	questions, err := BuildInstallWizardQuestions(skillDir, wizard)
	if err != nil {
		t.Fatalf("build questions: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}

	questionDefaults := map[string]string{}
	for _, question := range questions {
		questionDefaults[question.ID] = question.Default
	}
	if got := questionDefaults[wizardQuestionID("scripts/config.json", "vault_root")]; got != "~/existing" {
		t.Fatalf("unexpected vault_root default: %q", got)
	}
	if got := questionDefaults[wizardQuestionID("scripts/config.json", "sessions_root")]; got != "~/.codex/sessions" {
		t.Fatalf("unexpected sessions_root default: %q", got)
	}

	answers := map[string]string{
		wizardQuestionID("scripts/config.json", "vault_root"):    "~/vault/new",
		wizardQuestionID("scripts/config.json", "sessions_root"): "~/sessions/new",
	}
	if err := ApplyInstallWizardAnswers(skillDir, wizard, answers); err != nil {
		t.Fatalf("apply answers: %v", err)
	}

	payload, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if got := decoded["vault_root"]; got != "~/vault/new" {
		t.Fatalf("unexpected vault_root value: %#v", got)
	}
	if got := decoded["sessions_root"]; got != "~/sessions/new" {
		t.Fatalf("unexpected sessions_root value: %#v", got)
	}
}

func TestInstallWizardRejectsPathTraversal(t *testing.T) {
	skillDir := t.TempDir()
	spec := `{
  "version": 1,
  "files": [
    {
      "path": "../outside.json",
      "format": "json",
      "fields": [
        {"key": "vault_root", "prompt": "Vault root", "required": true}
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(skillDir, InstallWizardFileName), []byte(spec), 0o644); err != nil {
		t.Fatalf("write wizard: %v", err)
	}

	if _, err := LoadSkillInstallWizard(skillDir); err == nil {
		t.Fatal("expected load error for path traversal")
	}
}

func TestInstallWizardRequiredValue(t *testing.T) {
	skillDir := t.TempDir()
	configPath := filepath.Join(skillDir, "scripts", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"vault_root":""}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	spec := `{
  "version": 1,
  "files": [
    {
      "path": "scripts/config.json",
      "format": "json",
      "fields": [
        {"key": "vault_root", "prompt": "Vault root", "required": true}
      ]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(skillDir, InstallWizardFileName), []byte(spec), 0o644); err != nil {
		t.Fatalf("write wizard: %v", err)
	}

	wizard, err := LoadSkillInstallWizard(skillDir)
	if err != nil {
		t.Fatalf("load wizard: %v", err)
	}

	if err := ApplyInstallWizardAnswers(skillDir, wizard, map[string]string{}); err == nil {
		t.Fatal("expected required value error")
	}
}

func TestInstallWizardActionPromptsAndExecution(t *testing.T) {
	skillDir := t.TempDir()
	scriptPath := filepath.Join(skillDir, "scripts", "setup.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}

	script := `#!/usr/bin/env bash
set -euo pipefail
printf "%s|%s" "${ASKILL_TARGET_TYPE}" "$1" > "${ASKILL_SKILL_DIR}/action-result.txt"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	configPath := filepath.Join(skillDir, "scripts", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"vault_root":"~/vault"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	spec := `{
  "version": 1,
  "files": [
    {
      "path": "scripts/config.json",
      "format": "json",
      "fields": [
        {"key": "vault_root", "prompt": "Vault root", "required": true}
      ]
    }
  ],
  "actions": [
    {
      "id": "install-hook",
      "type": "run-script",
      "prompt": "Install hook?",
      "script": "scripts/setup.sh",
      "args": ["{{target_path}}"],
      "target_types": ["codex-global"]
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(skillDir, InstallWizardFileName), []byte(spec), 0o644); err != nil {
		t.Fatalf("write wizard: %v", err)
	}

	wizard, err := LoadSkillInstallWizard(skillDir)
	if err != nil {
		t.Fatalf("load wizard: %v", err)
	}
	if wizard == nil {
		t.Fatal("expected wizard")
	}

	prompts := BuildInstallWizardActionPrompts(wizard, []TargetType{TargetCodexGlobal})
	if len(prompts) != 1 {
		t.Fatalf("expected 1 action prompt, got %d", len(prompts))
	}
	if prompts[0].ID != "install-hook" {
		t.Fatalf("unexpected action prompt id: %s", prompts[0].ID)
	}

	prompts = BuildInstallWizardActionPrompts(wizard, []TargetType{TargetClaudeGlobal})
	if len(prompts) != 0 {
		t.Fatalf("expected no action prompts for non-matching target, got %d", len(prompts))
	}

	if err := ApplyInstallWizardActions(skillDir, wizard, map[string]bool{"install-hook": true}, InstallWizardApplyContext{
		TargetType: TargetCodexGlobal,
		TargetPath: "/tmp/codex-target",
	}); err != nil {
		t.Fatalf("apply actions: %v", err)
	}

	resultPath := filepath.Join(skillDir, "action-result.txt")
	result, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("read action result: %v", err)
	}
	if got := strings.TrimSpace(string(result)); got != "codex-global|/tmp/codex-target" {
		t.Fatalf("unexpected action result: %q", got)
	}
}
