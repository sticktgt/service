package validation

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Chart структура для секции chart
type Chart struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// File структура для секции files
type File struct {
	Path           string `yaml:"path"`
	Template       bool   `yaml:"template,omitempty"`
	OutputFilename string `yaml:"outputFilename,omitempty"`
	Content        string `yaml:"content"`
}

// ValuesSchemaItem описывает схему значений
type ValuesSchemaItem struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

// ValuesSchema ключи схемы значений
type ValuesSchema map[string]ValuesSchemaItem

// Root структура корневого YAML
type Root struct {
	Chart        Chart        `yaml:"chart"`
	Files        []File       `yaml:"files"`
	ValuesSchema ValuesSchema `yaml:"valuesSchema"`
}

// validateRequiredFields проверяет, что все обязательные поля присутствуют
func validateRequiredFields(data Root) error {
	var missingFields []string

	requiredFields := map[string]string{
		"chart.name":        data.Chart.Name,
		"chart.version":     data.Chart.Version,
		"chart.description": data.Chart.Description,
	}

	// Проверяем обязательные поля в valuesSchema
	for key := range requiredFields {
		if val, exists := requiredFields[key]; exists && val == "" {
			missingFields = append(missingFields, key)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("there are errors during meta validation. The fields are missing: %v", missingFields)
	}

	return nil
}

// ExtractValuesKeys ищет в строке все вхождения {{ .Values.<key> }} и возвращает список ключей
func ExtractValuesKeys(content string) []string {
	re := regexp.MustCompile(`{{\s*\.Values\.([a-zA-Z0-9_.]+)\s*}}`)
	matches := re.FindAllStringSubmatch(content, -1)

	var keys []string
	for _, match := range matches {
		if len(match) > 1 {
			keys = append(keys, match[1])
		}
	}
	return keys
}

// validateTemplates проверяет, что все ключи {{ .Values.<key> }} описаны в valuesSchema
func validateTemplates(data Root) ([]string, error) {
	var missingDescriptions []string
	var valuesKeys []string

	for _, file := range data.Files {
		keys := ExtractValuesKeys(file.Content)
		for _, key := range keys {
			if _, exists := data.ValuesSchema[key]; !exists {
				missingDescriptions = append(missingDescriptions, key)
			}
		}
	}

	if len(missingDescriptions) > 0 {
		return nil, fmt.Errorf("found values that aren't described in valuesSchema section: %v", missingDescriptions)
	}

	return valuesKeys, nil
}

// validateSchemaTypes проверяет, что все типы данных в valuesSchema корректные
func validateSchemaTypes(data Root) error {
	validTypes := map[string]bool{
		"string":  true,
		"integer": true,
		"boolean": true,
		"float":   true,
		"list":    true,
		"map":     true,
	}

	var invalidTypes []string
	for key, item := range data.ValuesSchema {
		if _, ok := validTypes[item.Type]; !ok {
			invalidTypes = append(invalidTypes, fmt.Sprintf("%s (unknown data type: %s)", key, item.Type))
		}
	}

	if len(invalidTypes) > 0 {
		return fmt.Errorf("data types errors: %v", invalidTypes)
	}
	return nil
}

func ValidateMetafile(metafile []byte, processID string) error {
	var data Root
	err := yaml.Unmarshal(metafile, &data)
	if err != nil {
		log.WithField("processID", processID).Errorf("Can't parse meta YAML: %v", err)
		return err
	}

	if err := validateRequiredFields(data); err != nil {
		log.WithField("processID", processID).Errorf("Meta validation error: mandatory fields: %v", err)
		return err
	}

	if _, err = validateTemplates(data); err != nil {
		log.WithField("processID", processID).Errorf("Meta validation error: templates: %v", err)
		return err
	}

	if err := validateSchemaTypes(data); err != nil {
		log.WithField("processID", processID).Errorf("Meta validation error: data types: %v", err)
		return err
	}

	log.WithField("processID", processID).Infof("✅ Meta validation completed successfully!")

	return nil
}
