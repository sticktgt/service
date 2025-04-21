package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"text/template"

	"github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

type Values struct {
	Environment    string      `json:"Environment"`
	Chart          Chart       `json:"Chart"`
	DeploymentName string      `json:"DeploymentName"`
	Namespace      string      `json:"Namespace"`
	Predictors     []Predictor `json:"Predictors"`
}

type Chart struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	Description string `json:"Description"`
}

type Predictor struct {
	Name          string        `json:"Name"`
	Replicas      int           `json:"Replicas"`
	Traffic       int           `json:"Traffic"`
	SvcOrchEnv    []EnvVar      `json:"SvcOrchEnv,omitempty"`
	ComponentSpec ComponentSpec `json:"ComponentSpec"`
	Graph         GraphNode     `json:"Graph"`
}

type EnvVar struct {
	Name      string     `json:"Name"`
	Value     string     `json:"Value,omitempty"`
	ValueFrom *SecretRef `json:"ValueFrom,omitempty"`
}

type SecretRef struct {
	SecretKeyRef struct {
		Name string `json:"Name"`
		Key  string `json:"Key"`
	} `json:"SecretKeyRef"`
}

type ComponentSpec struct {
	ServiceAccountName            string               `json:"ServiceAccountName"`
	TerminationGracePeriodSeconds int                  `json:"TerminationGracePeriodSeconds"`
	Containers                    []ComponentContainer `json:"Containers"`
}

type ComponentContainer struct {
	Name      string     `json:"Name"`
	Image     string     `json:"Image,omitempty"`
	Env       []EnvVar   `json:"Env,omitempty"`
	Resources *Resources `json:"Resources,omitempty"`
	Liveness  *Probe     `json:"Liveness,omitempty"`
	Readiness *Probe     `json:"Readiness,omitempty"`
}

type Resources struct {
	Requests map[string]string `json:"Requests,omitempty"`
	Limits   map[string]string `json:"Limits,omitempty"`
}

type Probe struct {
	Path                string `json:"Path"`
	Port                string `json:"Port"`
	InitialDelaySeconds int    `json:"InitialDelaySeconds"`
	PeriodSeconds       int    `json:"PeriodSeconds"`
	FailureThreshold    int    `json:"FailureThreshold"`
	SuccessThreshold    int    `json:"SuccessThreshold"`
}

type GraphNode struct {
	Name             string      `json:"Name"`
	Type             string      `json:"Type"`
	Implementation   string      `json:"Implementation,omitempty"`
	ModelUri         string      `json:"ModelUri,omitempty"`
	EndpointType     string      `json:"EndpointType,omitempty"`
	EnvSecretRefName string      `json:"EnvSecretRefName,omitempty"`
	Parameters       []Parameter `json:"Parameters,omitempty"`
	Logger           *Logger     `json:"Logger,omitempty"`
	Children         []GraphNode `json:"Children,omitempty"`
}

type Parameter struct {
	Name  string `json:"Name"`
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

type Logger struct {
	Mode string `json:"Mode"`
}

// ChartTemplate maps the structure of seldon.meta.yaml
type ChartTemplate struct {
	ValuesSchema map[string]struct {
		Type        string `yaml:"type"`
		Description string `yaml:"description"`
		Required    bool   `yaml:"required"`
	} `yaml:"valuesSchema"`

	Files []struct {
		Path     string `yaml:"path"`
		Template bool   `yaml:"template"`
		Content  string `yaml:"content"`
	} `yaml:"files"`
}

var log = logrus.New()

// Values is your full input data structure, matching values.json
// -- Include your full Values struct here as defined previously
// For brevity, we'll assume it's defined in another file or inline above main()

func main() {
	// Step 1: Read and parse values.json into structured data
	var values Values
	log.Printf("Reading values file")
	valData, err := os.ReadFile("../config/values.json")
	check(err)
	log.Printf("Checking values file")
	check(json.Unmarshal(valData, &values))

	// Step 2: Read and parse seldon.meta.yaml
	var meta ChartTemplate
	log.Printf("Reading template file")
	tmplData, err := os.ReadFile("../config/seldon.meta.yaml")
	check(err)
	log.Printf("Checking template file")
	check(yaml.Unmarshal(tmplData, &meta))

	//log.Printf("Values: %+v", values)
	// Step 3: Render each file from template
	for _, file := range meta.Files {

		newfilepath := file.Path
		if file.Path == "values-ENV.yaml" {
			if values.Environment != "" {
				newfilepath = "values-" + values.Environment + ".yaml"
			}
		}

		outputPath := filepath.Join("output", newfilepath)
		outputDir := filepath.Dir(outputPath)
		check(os.MkdirAll(outputDir, 0755))
		//		log.Printf("outputPath: %s", outputPath)

		content := file.Content
		if file.Template {
			tmpl, err := template.New(file.Path).Parse(file.Content)
			check(err)
			var buf bytes.Buffer
			log.Printf("executing...")
			check(tmpl.Execute(&buf, values))
			content = buf.String()
		}

		check(os.WriteFile(outputPath, []byte(content), 0644))
		log.Printf("Generated: %s", outputPath)
	}
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
