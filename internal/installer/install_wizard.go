package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const InstallWizardFileName = "INSTALL_WIZARD.json"

type InstallWizard struct {
	Version     int                   `json:"version"`
	Title       string                `json:"title"`
	Description string                `json:"description,omitempty"`
	Files       []InstallWizardFile   `json:"files"`
	Actions     []InstallWizardAction `json:"actions,omitempty"`
}

type InstallWizardFile struct {
	Path   string               `json:"path"`
	Format string               `json:"format"`
	Fields []InstallWizardField `json:"fields"`
}

type InstallWizardField struct {
	Key      string `json:"key"`
	Prompt   string `json:"prompt"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type InstallWizardQuestion struct {
	ID       string
	FilePath string
	Key      string
	Prompt   string
	Default  string
	Required bool
}

type InstallWizardAction struct {
	ID             string       `json:"id"`
	Type           string       `json:"type"`
	Prompt         string       `json:"prompt,omitempty"`
	Script         string       `json:"script,omitempty"`
	Args           []string     `json:"args,omitempty"`
	DefaultEnabled bool         `json:"default_enabled,omitempty"`
	TargetTypes    []TargetType `json:"target_types,omitempty"`
}

type InstallWizardActionPrompt struct {
	ID             string
	Prompt         string
	DefaultEnabled bool
}

type InstallWizardApplyContext struct {
	TargetType TargetType
	TargetPath string
	Answers    map[string]string
}

func LoadSkillInstallWizard(skillDir string) (*InstallWizard, error) {
	specPath := filepath.Join(skillDir, InstallWizardFileName)
	payload, err := os.ReadFile(specPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read wizard spec: %w", err)
	}

	var wizard InstallWizard
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wizard); err != nil {
		return nil, fmt.Errorf("decode wizard spec: %w", err)
	}
	if err := wizard.validate(); err != nil {
		return nil, fmt.Errorf("validate wizard spec: %w", err)
	}
	return &wizard, nil
}

func BuildInstallWizardQuestions(skillDir string, wizard *InstallWizard) ([]InstallWizardQuestion, error) {
	if wizard == nil {
		return nil, nil
	}

	questions := make([]InstallWizardQuestion, 0, len(wizard.Files))
	for _, fileSpec := range wizard.Files {
		var source map[string]any
		sourcePath, err := resolveWithinSkill(skillDir, fileSpec.Path)
		if err != nil {
			return nil, err
		}
		source, err = readJSONMap(sourcePath, true)
		if err != nil {
			return nil, err
		}

		for _, field := range fileSpec.Fields {
			defaultValue := strings.TrimSpace(field.Default)
			if defaultValue == "" {
				if existing, ok := getJSONPathString(source, field.Key); ok {
					defaultValue = existing
				}
			}
			questions = append(questions, InstallWizardQuestion{
				ID:       wizardQuestionID(fileSpec.Path, field.Key),
				FilePath: fileSpec.Path,
				Key:      field.Key,
				Prompt:   field.Prompt,
				Default:  defaultValue,
				Required: field.Required,
			})
		}
	}
	return questions, nil
}

func BuildInstallWizardActionPrompts(wizard *InstallWizard, selectedTargetTypes []TargetType) []InstallWizardActionPrompt {
	if wizard == nil || len(wizard.Actions) == 0 {
		return nil
	}

	prompts := make([]InstallWizardActionPrompt, 0, len(wizard.Actions))
	for _, action := range wizard.Actions {
		if !action.appliesToAnyTarget(selectedTargetTypes) {
			continue
		}
		prompt := strings.TrimSpace(action.Prompt)
		if prompt == "" {
			prompt = fmt.Sprintf("Enable %s?", action.ID)
		}
		prompts = append(prompts, InstallWizardActionPrompt{
			ID:             action.ID,
			Prompt:         prompt,
			DefaultEnabled: action.DefaultEnabled,
		})
	}
	return prompts
}

func ApplyInstallWizardAnswers(skillDir string, wizard *InstallWizard, answers map[string]string) error {
	if wizard == nil {
		return nil
	}

	for _, fileSpec := range wizard.Files {
		targetPath, err := resolveWithinSkill(skillDir, fileSpec.Path)
		if err != nil {
			return err
		}
		doc, err := readJSONMap(targetPath, true)
		if err != nil {
			return err
		}

		changed := false
		for _, field := range fileSpec.Fields {
			questionID := wizardQuestionID(fileSpec.Path, field.Key)
			value, ok := answers[questionID]
			if !ok {
				value = strings.TrimSpace(field.Default)
			}
			if field.Required && strings.TrimSpace(value) == "" {
				return fmt.Errorf("missing required value for %s (%s)", field.Key, fileSpec.Path)
			}
			if strings.TrimSpace(value) == "" {
				continue
			}
			if err := setJSONPathString(doc, field.Key, value); err != nil {
				return err
			}
			changed = true
		}
		if !changed {
			continue
		}
		if err := writeJSONMap(targetPath, doc); err != nil {
			return err
		}
	}
	return nil
}

func ApplyInstallWizardActions(skillDir string, wizard *InstallWizard, selectedActions map[string]bool, context InstallWizardApplyContext) error {
	if wizard == nil || len(wizard.Actions) == 0 {
		return nil
	}

	for _, action := range wizard.Actions {
		if !action.appliesToTargetType(context.TargetType) {
			continue
		}
		enabled := action.DefaultEnabled
		if value, ok := selectedActions[action.ID]; ok {
			enabled = value
		}
		if !enabled {
			continue
		}

		switch action.Type {
		case "run-script":
			if err := runInstallWizardScriptAction(skillDir, action, context); err != nil {
				return fmt.Errorf("run action %s: %w", action.ID, err)
			}
		default:
			return fmt.Errorf("unsupported action type %q", action.Type)
		}
	}

	return nil
}

func (wizard *InstallWizard) validate() error {
	if wizard.Version == 0 {
		wizard.Version = 1
	}
	if wizard.Version != 1 {
		return fmt.Errorf("unsupported version %d", wizard.Version)
	}
	wizard.Title = strings.TrimSpace(wizard.Title)
	wizard.Description = strings.TrimSpace(wizard.Description)
	if len(wizard.Files) == 0 {
		return errors.New("files must contain at least one entry")
	}

	for i := range wizard.Files {
		fileSpec := &wizard.Files[i]
		cleanPath, err := cleanRelativePath(fileSpec.Path)
		if err != nil {
			return fmt.Errorf("files[%d].path: %w", i, err)
		}
		fileSpec.Path = cleanPath

		format := strings.ToLower(strings.TrimSpace(fileSpec.Format))
		if format == "" {
			format = "json"
		}
		if format != "json" {
			return fmt.Errorf("files[%d].format %q is not supported", i, fileSpec.Format)
		}
		fileSpec.Format = format

		if len(fileSpec.Fields) == 0 {
			return fmt.Errorf("files[%d].fields must contain at least one entry", i)
		}
		for j := range fileSpec.Fields {
			field := &fileSpec.Fields[j]
			field.Key = strings.TrimSpace(field.Key)
			field.Prompt = strings.TrimSpace(field.Prompt)
			field.Default = strings.TrimSpace(field.Default)
			if field.Key == "" {
				return fmt.Errorf("files[%d].fields[%d].key is required", i, j)
			}
			if field.Prompt == "" {
				return fmt.Errorf("files[%d].fields[%d].prompt is required", i, j)
			}
			for _, segment := range strings.Split(field.Key, ".") {
				if strings.TrimSpace(segment) == "" {
					return fmt.Errorf("files[%d].fields[%d].key %q is invalid", i, j, field.Key)
				}
			}
		}
	}

	actionIDs := map[string]struct{}{}
	for i := range wizard.Actions {
		action := &wizard.Actions[i]
		action.ID = strings.TrimSpace(action.ID)
		if action.ID == "" {
			return fmt.Errorf("actions[%d].id is required", i)
		}
		if _, exists := actionIDs[action.ID]; exists {
			return fmt.Errorf("actions[%d].id %q is duplicated", i, action.ID)
		}
		actionIDs[action.ID] = struct{}{}

		action.Type = strings.ToLower(strings.TrimSpace(action.Type))
		if action.Type == "" {
			action.Type = "run-script"
		}

		action.Prompt = strings.TrimSpace(action.Prompt)

		switch action.Type {
		case "run-script":
			scriptPath, err := cleanRelativePath(action.Script)
			if err != nil {
				return fmt.Errorf("actions[%d].script: %w", i, err)
			}
			action.Script = scriptPath
			for argIndex, arg := range action.Args {
				action.Args[argIndex] = strings.TrimSpace(arg)
			}
		default:
			return fmt.Errorf("actions[%d].type %q is not supported", i, action.Type)
		}

		for targetIndex, targetType := range action.TargetTypes {
			trimmed := TargetType(strings.TrimSpace(string(targetType)))
			if trimmed == "" {
				return fmt.Errorf("actions[%d].target_types[%d] is empty", i, targetIndex)
			}
			if !isSupportedTargetType(trimmed) {
				return fmt.Errorf("actions[%d].target_types[%d] %q is not supported", i, targetIndex, trimmed)
			}
			action.TargetTypes[targetIndex] = trimmed
		}
	}

	return nil
}

func readJSONMap(path string, allowMissing bool) (map[string]any, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if allowMissing && errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return map[string]any{}, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil, fmt.Errorf("parse json %s: %w", path, err)
	}
	if doc == nil {
		return map[string]any{}, nil
	}
	return doc, nil
}

func writeJSONMap(path string, doc map[string]any) error {
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json %s: %w", path, err)
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func getJSONPathString(doc map[string]any, key string) (string, bool) {
	if len(doc) == 0 {
		return "", false
	}
	parts := strings.Split(key, ".")
	var current any = doc
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		value, ok := object[part]
		if !ok {
			return "", false
		}
		current = value
	}
	switch value := current.(type) {
	case string:
		return value, true
	case float64:
		return fmt.Sprintf("%v", value), true
	case bool:
		return fmt.Sprintf("%v", value), true
	default:
		return "", false
	}
}

func setJSONPathString(doc map[string]any, key, value string) error {
	parts := strings.Split(key, ".")
	if len(parts) == 0 {
		return errors.New("json key is empty")
	}
	current := doc
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, exists := current[part]
		if !exists {
			node := map[string]any{}
			current[part] = node
			current = node
			continue
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			node := map[string]any{}
			current[part] = node
			current = node
			continue
		}
		current = nextMap
	}
	current[parts[len(parts)-1]] = value
	return nil
}

func wizardQuestionID(filePath, key string) string {
	return filePath + "::" + key
}

func resolveWithinSkill(skillDir, relativePath string) (string, error) {
	cleaned, err := cleanRelativePath(relativePath)
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(skillDir, cleaned)
	rel, err := filepath.Rel(skillDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", relativePath, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s points outside skill root", relativePath)
	}
	return fullPath, nil
}

func cleanRelativePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("path is required")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("path must be relative: %s", value)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s points outside skill root", value)
	}
	return cleaned, nil
}

func isSupportedTargetType(targetType TargetType) bool {
	switch targetType {
	case TargetCodexGlobal, TargetClaudeGlobal, TargetClaudeProject, TargetCursorGlobal, TargetCursorProject:
		return true
	default:
		return false
	}
}

func (action InstallWizardAction) appliesToAnyTarget(selectedTargetTypes []TargetType) bool {
	if len(action.TargetTypes) == 0 {
		return true
	}
	if len(selectedTargetTypes) == 0 {
		return false
	}
	for _, selected := range selectedTargetTypes {
		if action.appliesToTargetType(selected) {
			return true
		}
	}
	return false
}

func (action InstallWizardAction) appliesToTargetType(targetType TargetType) bool {
	if len(action.TargetTypes) == 0 {
		return true
	}
	for _, allowed := range action.TargetTypes {
		if allowed == targetType {
			return true
		}
	}
	return false
}

func runInstallWizardScriptAction(skillDir string, action InstallWizardAction, context InstallWizardApplyContext) error {
	scriptPath, err := resolveWithinSkill(skillDir, action.Script)
	if err != nil {
		return err
	}

	args := make([]string, 0, len(action.Args))
	for _, arg := range action.Args {
		args = append(args, interpolateInstallWizardActionArg(arg, skillDir, context))
	}

	command := []string{}
	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".py":
		command = append(command, "python3", scriptPath)
	case ".sh":
		command = append(command, "bash", scriptPath)
	default:
		command = append(command, scriptPath)
	}
	command = append(command, args...)

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = skillDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := append(os.Environ(),
		"ASKILL_SKILL_DIR="+skillDir,
		"ASKILL_TARGET_TYPE="+string(context.TargetType),
		"ASKILL_TARGET_PATH="+context.TargetPath,
	)
	if len(context.Answers) > 0 {
		payload, err := json.Marshal(context.Answers)
		if err == nil {
			env = append(env, "ASKILL_WIZARD_ANSWERS_JSON="+string(payload))
		}
	}
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func interpolateInstallWizardActionArg(raw string, skillDir string, context InstallWizardApplyContext) string {
	homeDir, _ := os.UserHomeDir()
	replacer := strings.NewReplacer(
		"{{skill_dir}}", skillDir,
		"{{target_type}}", string(context.TargetType),
		"{{target_path}}", context.TargetPath,
		"{{home_dir}}", homeDir,
	)
	return replacer.Replace(raw)
}
