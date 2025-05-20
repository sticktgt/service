package validation

import (
	"strings"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// checkKeys проверяет, присутствуют ли все ключи в YAML
func checkKeys(data map[string]interface{}, keys []string) []string {
	var missing []string

	for _, key := range keys {
		parts := strings.Split(key, ".")
		if !keyExists(data, parts) {
			missing = append(missing, key)
		}
	}

	return missing
}

// keyExists проверяет существование вложенного ключа
func keyExists(data interface{}, path []string) bool {
	if len(path) == 0 {
		return true
	}

	currentMap, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	val, exists := currentMap[path[0]]
	if !exists {
		return false
	}

	return keyExists(val, path[1:])
}

func ValidateValuesInjectionConsistency(setupValues map[string]interface{}, metafileKeys []string, processID string) {
	missingKeys := checkKeys(setupValues, metafileKeys)
	if len(missingKeys) > 0 {
		log.WithField("processID", processID).Warn("There are missing keys in the setup values:")
		for _, key := range missingKeys {
			log.WithField("processID", processID).Warn("--> ", key)
		}
	} else {
		log.WithField("processID", processID).Infof("There are no missing keys in the setup values")
	}
}
