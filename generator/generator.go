package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/sirupsen/logrus"

	"gservice/utils"
)

type FileConfig struct {
	Path     string
	Content  string
	Template bool
}

var log = logrus.New()

func GenerateChart(meta utils.MetaStructure, setupValues map[string]interface{}, outputDir string, cliEnv string, processID string, createEnvValues bool) error {
	// В худшем случае не передали ни одного values и мы уйдем на fallback в виде default-значений
	if setupValues == nil {
		setupValues = make(map[string]interface{})
	}

	processDefaults(meta, setupValues, processID)

	if createEnvValues {
		log.WithField("processID", processID).Info("create-env-setupValues is set to true, will not create other contents:")
	}
	// Если передано --env, переопределяем setupValues.environment
	if cliEnv != "" {
		log.WithField("processID", processID).Info("Using provided input environment:", cliEnv)
		setupValues["environment"] = cliEnv
	} else if _, exists := setupValues["environment"]; exists {
		log.WithField("processID", processID).Infof("Using environment from setupValues.yaml: %s", setupValues["environment"])
	} else {
		log.WithField("processID", processID).Warn("❌ No environment specified, defaulting to empty.")
	}

	for i, file := range meta.Files {
		outputPath := file.Path

		// Если есть outputFilename, рендерим его как шаблон
		if file.OutputFilename != "" {
			tpl, err := template.New("").Parse(file.OutputFilename)
			if err != nil {
				log.WithField("processID", processID).Errorf("❌ template parse error for output filename %s: %v", file.OutputFilename, err)
				return err
			}

			var renderedOutputPath bytes.Buffer
			tplData := map[string]interface{}{
				"Values": setupValues,
			}

			if err := tpl.Execute(&renderedOutputPath, tplData); err != nil {
				log.WithField("processID", processID).Errorf("❌ template execution error for output filename %s: %v", file.OutputFilename, err)
				return err
			}

			outputPath = renderedOutputPath.String() // Теперь `outputPath` заменяет `file.Path`
		}

		// Теперь заменяем `file.Path` на `outputPath`, чтобы не дублировать файлы
		meta.Files[i].Path = outputPath

		// Генерация контента файла
		content := file.Content
		if file.Template {
			log.WithField("processID", processID).Infof("Generating: %s", outputPath)
			tpl, err := template.New("").Parse(content)
			if err != nil {
				log.WithField("processID", processID).Errorf("❌ template parse error for %s: %v", file.OutputFilename, err)
				return err
			}

			var renderedContent bytes.Buffer
			tplData := map[string]interface{}{
				"Values": setupValues,
			}

			if err := tpl.Execute(&renderedContent, tplData); err != nil {
				log.WithField("processID", processID).Errorf("❌ template execution error for %s: %v", outputPath, err)
				return err
			}
			content = renderedContent.String()
		}

		filePath := filepath.Join(outputDir, outputPath)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func processDefaults(meta utils.MetaStructure, setupValues map[string]interface{}, processID string) {
	// Применяем значения по умолчанию
	for key, schema := range meta.ValuesSchema {
		if _, exists := utils.GetNestedValue(setupValues, key); !exists {
			log.WithField("processID", processID).Warnf("-------> Parameter %s has no value in user input", key)
			if schema.Default != nil {
				utils.SetNestedValue(setupValues, key, schema.Default)
				log.WithField("processID", processID).Infof("Applied default value for %s: %v", key, schema.Default)
			} else {
				log.WithField("processID", processID).Warnf("-------> Parameter %s has no value in user input and doesn't have a default value in metafile", key)
			}
		}
	}
}

func GenerateValuesFile(meta utils.MetaStructure, setupValues map[string]interface{}, projectFolder string, environment string, processID string) error {
	log.WithField("processID", processID).Info("Generating values file for environment:", environment)

	processDefaults(meta, setupValues, processID)

	// Находим файл с шаблоном values-env.yaml
	var valuesFileTemplate utils.File
	var envFileIndex int
	for i, file := range meta.Files {
		if file.Path == "values-env.yaml" {
			valuesFileTemplate = file
			envFileIndex = i
			break
		}
	}

	// Если передано --env, переопределяем setupValues.environment
	if environment != "" {
		log.WithField("processID", processID).Info("Using provided input environment:", environment)
		setupValues["environment"] = environment
	} else if _, exists := setupValues["environment"]; exists {
		log.WithField("processID", processID).Infof("Using environment from setupValues.yaml: %s", setupValues["environment"])
	} else {
		log.WithField("processID", processID).Warn("❌ No environment specified, defaulting to empty.")
	}

	// Если есть outputFilename, рендерим его как шаблон
	outputPath := valuesFileTemplate.Path
	if valuesFileTemplate.OutputFilename != "" {
		tpl, err := template.New("").Parse(valuesFileTemplate.OutputFilename)
		if err != nil {
			log.WithField("processID", processID).Errorf("❌ template parse error for output filename %s: %v", valuesFileTemplate.OutputFilename, err)
			return err
		}

		var renderedOutputPath bytes.Buffer
		tplData := map[string]interface{}{
			"Values": setupValues,
		}

		if err := tpl.Execute(&renderedOutputPath, tplData); err != nil {
			log.WithField("processID", processID).Errorf("❌ template execution error for output filename %s: %v", valuesFileTemplate.OutputFilename, err)
			return err
		}

		outputPath = renderedOutputPath.String() // Теперь `outputPath` заменяет `file.Path`
	}

	// Теперь заменяем `file.Path` на `outputPath`, чтобы не дублировать файлы
	meta.Files[envFileIndex].Path = outputPath

	if valuesFileTemplate.Path == "" {
		log.WithField("processID", processID).Errorf("❌ Cannot find values-env.yaml template in metafile")
		return fmt.Errorf("values-env.yaml template not found in meta")
	}

	// Генерация контента файла
	content := valuesFileTemplate.Content
	if valuesFileTemplate.Template {
		tpl, err := template.New("").Parse(content)
		if err != nil {
			log.WithField("processID", processID).Errorf("❌ template parse error for values-env.yaml: %v", err)
			return err
		}

		var renderedContent bytes.Buffer
		tplData := map[string]interface{}{
			"Values": setupValues,
		}

		if err := tpl.Execute(&renderedContent, tplData); err != nil {
			log.WithField("processID", processID).Errorf("❌ template execution error for values-env.yaml: %v", err)
			return err
		}
		content = renderedContent.String()
	}

	// Генерация имени файла
	filePath := filepath.Join(projectFolder, outputPath)

	// Создание директории, если она не существует
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	// Запись файла
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}
