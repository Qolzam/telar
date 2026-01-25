package prompt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PromptVariant represents a single prompt variant for a specific model family
type PromptVariant struct {
	ID          string `yaml:"id"`
	ModelFamily string `yaml:"model_family"` // e.g. "qwen", "llama", "gpt"
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

// PromptFile represents a YAML file containing prompt variants for a task
type PromptFile struct {
	Version  string          `yaml:"version"`
	Task     string          `yaml:"task"`
	Variants []PromptVariant `yaml:"variants"`
}

// Registry manages prompt templates loaded from YAML files
type Registry struct {
	prompts map[string]*PromptFile // Key: task_name (e.g., "moderation")
	basePath string
}

// NewRegistry creates a new prompt registry and loads prompt files from the specified directory
func NewRegistry(path string) (*Registry, error) {
	r := &Registry{
		prompts:  make(map[string]*PromptFile),
		basePath: path,
	}

	// Load all YAML files in the prompts directory
	if err := r.loadDirectory(path); err != nil {
		return nil, fmt.Errorf("failed to load prompt directory: %w", err)
	}

	return r, nil
}

// loadDirectory loads all YAML files from the prompts directory
func (r *Registry) loadDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read prompts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		if err := r.loadFile(filePath); err != nil {
			return fmt.Errorf("failed to load prompt file %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// loadFile loads a single YAML prompt file
func (r *Registry) loadFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	var file PromptFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if file.Task == "" {
		return fmt.Errorf("prompt file missing 'task' field: %s", filepath)
	}

	r.prompts[file.Task] = &file
	return nil
}

// GetTemplate finds the best prompt template for the given task and model name
// It uses a matching strategy:
// 1. Explicit variant override (for A/B testing)
// 2. Exact or partial match on model family name
// 3. Fallback to first variant if no match found
func (r *Registry) GetTemplate(task string, modelName string) (string, error) {
	return r.GetTemplateWithOverride(task, modelName, "")
}

// GetTemplateWithOverride finds the best prompt template with optional variant override
// variantOverride: If specified, forces selection of this variant ID (for A/B testing)
func (r *Registry) GetTemplateWithOverride(task string, modelName string, variantOverride string) (string, error) {
	file, ok := r.prompts[task]
	if !ok {
		return "", fmt.Errorf("task '%s' not found in registry", task)
	}

	if len(file.Variants) == 0 {
		return "", fmt.Errorf("no prompt variants found for task '%s'", task)
	}

	// Strategy 0: Explicit variant override (for A/B testing)
	if variantOverride != "" {
		for _, v := range file.Variants {
			if strings.EqualFold(v.ID, variantOverride) {
				log.Printf("[PROMPT] Selected variant: %s (explicit override for model: %s)", v.ID, modelName)
				return v.Template, nil
			}
		}
		log.Printf("[PROMPT] Warning: Variant override '%s' not found, falling back to auto-selection", variantOverride)
	}

	// Normalize model name for matching
	modelNameLower := strings.ToLower(modelName)

	// Strategy 1: Exact or partial match on model family
	for _, v := range file.Variants {
		modelFamilyLower := strings.ToLower(v.ModelFamily)
		if strings.Contains(modelNameLower, modelFamilyLower) {
			log.Printf("[PROMPT] Selected variant: %s (matched model_family: %s for model: %s)", v.ID, v.ModelFamily, modelName)
			return v.Template, nil
		}
	}

	// Strategy 2: Fallback to first variant (or look for "default" ID)
	for _, v := range file.Variants {
		if v.ID == "default" {
			log.Printf("[PROMPT] Selected variant: %s (default fallback for model: %s)", v.ID, modelName)
			return v.Template, nil
		}
	}

	// Strategy 3: Return first variant as ultimate fallback
	firstVariant := file.Variants[0]
	log.Printf("[PROMPT] Selected variant: %s (first variant fallback for model: %s)", firstVariant.ID, modelName)
	return firstVariant.Template, nil
}

// GetVariantInfo returns metadata about available variants for a task
func (r *Registry) GetVariantInfo(task string) ([]PromptVariant, error) {
	file, ok := r.prompts[task]
	if !ok {
		return nil, fmt.Errorf("task '%s' not found in registry", task)
	}

	return file.Variants, nil
}
