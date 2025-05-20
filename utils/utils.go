package utils

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type ValueSpec struct {
	Type        string      `yaml:"type"`
	Default     interface{} `yaml:"default"`
	Description string      `yaml:"description"`
	Required    bool        `yaml:"required"`
	Options     []string    `yaml:"options"`
}

type MetaStructure struct {
	Chart        ChartInfo            `yaml:"chart"`
	Files        []File               `yaml:"files"`
	ValuesSchema map[string]ValueSpec `yaml:"valuesSchema"`
}

type ChartInfo struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

type File struct {
	Path           string `yaml:"path"`
	Content        string `yaml:"content"`
	Template       bool   `yaml:"template"`
	OutputFilename string `yaml:"outputFilename"`
}

var log = logrus.New()

func LoadMeta(fileData []byte, processID string) (MetaStructure, error) {
	var meta MetaStructure
	log.WithField("processID", processID).Infof("Unmarshalling meta")

	if err := yaml.Unmarshal(fileData, &meta); err != nil {
		return meta, err
	}
	log.WithField("processID", processID).Infof("Meta unmarshaled")

	return meta, nil
}

func MergeValuesFromFiles(files []string, processID string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	for _, file := range files {
		log.WithField("processID", processID).Infof("Merging file %s", file)
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var current map[string]interface{}
		if err := yaml.Unmarshal(data, &current); err != nil {
			return nil, err
		}

		for k, v := range current {
			values[k] = v
		}
	}

	return values, nil
}

func MergeValuesFromRequest(setupValuesMap map[string]interface{}, processID string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	log.WithField("processID", processID).Infof("Merging setup values from REST")
	toYAML, err := ConvertToYAML(setupValuesMap)
	if err != nil {
		return nil, err
	}
	data := []byte(toYAML)

	var current map[string]interface{}
	if err := yaml.Unmarshal(data, &current); err != nil {
		return nil, err
	}

	for k, v := range current {
		values[k] = v
	}

	return values, nil
}

func ConvertToYAML(data map[string]interface{}) (string, error) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal setup-values to YAML: %v", err)
	}
	return string(yamlData), nil
}

func LogWithProcessID(message string, processID string) {
	log.Printf("[%s] %s", processID, message)
}

func LogWithContext(ctx context.Context, message string) {
	processID := ctx.Value("processID").(string)
	log.Printf("[%s] %s", processID, message)
}

func EraseFolder(dirPath string, processID string) (err error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.WithField("processID", processID).Error("❌ Could not find the folder: ", dirPath)
		return err
	} else {
		err := os.RemoveAll(dirPath)
		if err != nil {
			log.WithField("processID", processID).Error("❌ The error occurred during erase", dirPath)
			return err
		} else {
			log.WithField("processID", processID).Info("✅ The temporary local chart folder is deleted: ", dirPath)
			return nil
		}
	}
}

func PrepareProjectFolderName(projectRepoName string, projectSubfolder string) string {
	return path.Join(projectRepoName, projectSubfolder)
}

func PrepareProjectRepoName(repo string, branch string, processID string) string {
	httpRegex := regexp.MustCompile(`(?i)https?://`)
	repo = httpRegex.ReplaceAllString(repo, "")

	replacer := strings.NewReplacer("/", "_", ".", "_", ":", "_")
	sanitizedRepo := replacer.Replace(repo + "-" + branch)

	return path.Join(".", sanitizedRepo+"_"+processID)
}

// ExistsDir проверяет, существует ли указанная папка на диске
func ExistsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetNestedValue Извлекает вложенное значение из map
func GetNestedValue(m map[string]interface{}, key string) (interface{}, bool) {
	keys := strings.Split(key, ".")
	var current interface{} = m

	for _, k := range keys {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}

		val, exists := currentMap[k]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

// SetNestedValue Записывает вложенное значние в map по композитному ключу
func SetNestedValue(m map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")

	for i := 0; i < len(keys)-1; i++ {
		currentKey := keys[i]
		if _, ok := m[currentKey]; !ok {
			m[currentKey] = make(map[string]interface{})
		}
		nextMap, ok := m[currentKey].(map[string]interface{})
		if !ok {
			newMap := make(map[string]interface{})
			m[currentKey] = newMap
			nextMap = newMap
		}
		m = nextMap
	}
	lastKey := keys[len(keys)-1]
	m[lastKey] = value
}
