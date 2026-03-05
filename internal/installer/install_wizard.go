package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const InstallWizardFileName = "INSTALL_WIZARD.json"

type InstallWizard struct {
	Version     int                 `json:"version"`
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Files       []InstallWizardFile `json:"files"`
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
		fileSpec.Path = strings.TrimSpace(fileSpec.Path)
		if fileSpec.Path == "" {
			return fmt.Errorf("files[%d].path is required", i)
		}
		if filepath.IsAbs(fileSpec.Path) {
			return fmt.Errorf("files[%d].path must be relative", i)
		}
		cleaned := filepath.Clean(fileSpec.Path)
		if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("files[%d].path points outside skill root", i)
		}
		fileSpec.Path = cleaned

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
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("path must be relative: %s", relativePath)
	}
	cleaned := filepath.Clean(relativePath)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s points outside skill root", relativePath)
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
