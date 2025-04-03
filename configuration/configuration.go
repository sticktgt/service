package configuration

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	//	"os"
	"sync"
)

var (
	once   sync.Once
	Config *viper.Viper
	log    = logrus.New()
)

func initConfig() {
	v := viper.New()
	v.SetConfigName("application")
	v.SetConfigType("yaml")
	v.AddConfigPath("../config")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Unable to load configuration file: %v", err)
	}

	/*	requiredEnvs := []string{
			v.GetString("env.metaRepositoryTokenEnv"),
			v.GetString("env.chartRepositoryTokenEnv"),
			v.GetString("env.ociHelmRepositoryTokenEnv"),
		}
		for _, key := range requiredEnvs {
			if _, exists := os.LookupEnv(key); !exists {
				log.Fatalf("The environment variable %s is not set!", key)
			}
		}
	*/
	requiredKeys := []string{
		"repository1.path",
		"repository1.branch",
		"repository1.username",
		"repository2.path",
		"repository2.branch",
		"repository2.username",
	}

	Config = v
	jsonConfig, err := json.MarshalIndent(v.AllSettings(), "", " ")
	if err != nil {
		log.Fatalf("Unable to convert config to JSON: %v", err)
	}

	for _, key := range requiredKeys {
		if !Config.IsSet(key) {
			log.Fatalf("The environment variable %s is not set!", key)
		}
	}

	log.Info("Application config loaded")
	log.Infof("Config arguments: %v", string(jsonConfig))
}

func Init() {
	once.Do(initConfig)
}
