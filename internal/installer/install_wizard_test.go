package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
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
