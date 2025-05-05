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
	Environment    string            `json:"Environment"`
	ApiVersion     string            `json:"ApiVersion"`
	Chart          Chart             `json:"Chart"`
	DeploymentName string            `json:"DeploymentName"`
	Namespace      string            `json:"Namespace"`
	Metadata       Metadata          `json:"Metadata"`
	Predictors     []Predictor       `json:"Predictors"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	Protocol       string            `json:"protocol,omitempty"`
	Transport      string            `json:"transport,omitempty"`
}

type Chart struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	Description string `json:"Description"`
}

type Metadata struct {
	Labels      map[string]string `json:"Labels,omitempty"`
	Annotations map[string]string `json:"Annotations,omitempty"`
}

type Predictor struct {
	Name          string        `json:"Name"`
	Replicas      int           `json:"Replicas"`
	Traffic       int           `json:"Traffic"`
	SvcOrchSpec   *SvcOrchSpec  `json:"svcOrchSpec,omitempty"`
	Graph         GraphNode     `json:"Graph"`
	ComponentSpec ComponentSpec `json:"ComponentSpec"`
}

type EnvVar struct {
	Name      string     `json:"Name"`
	Value     string     `json:"Value,omitempty"`
	ValueFrom *ValueFrom `json:"ValueFrom,omitempty"`
}

type ValueFrom struct {
	SecretKeyRef SecretKeyRef `json:"SecretKeyRef"`
}

type SecretKeyRef struct {
	Name string `json:"Name"`
	Key  string `json:"Key"`
}

type GraphNode struct {
	Name             string      `json:"Name"`
	Type             string      `json:"Type"`
	Implementation   string      `json:"Implementation,omitempty"`
	ModelUri         string      `json:"ModelUri,omitempty"`
	EnvSecretRefName string      `json:"EnvSecretRefName,omitempty"`
	Logger           *Logger     `json:"Logger,omitempty"`
	Endpoint         *Endpoint   `json:"Endpoint,omitempty"`
	Parameters       []Parameter `json:"Parameters,omitempty"`
	Children         []GraphNode `json:"Children,omitempty"`
}

type Logger struct {
	Mode string `json:"Mode,omitempty"`
	URL  string `json:"URL,omitempty"`
}

type Parameter struct {
	Name  string `json:"Name"`
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

type ComponentSpec struct {
	ServiceAccountName            string          `json:"ServiceAccountName,omitempty"`
	TerminationGracePeriodSeconds int             `json:"TerminationGracePeriodSeconds,omitempty"`
	Containers                    []Container     `json:"Containers"`
	Volumes                       []Volume        `json:"Volumes,omitempty"`
	InitContainers                []InitContainer `json:"InitContainers,omitempty"`
	HPASpec                       *HPASpec        `json:"hpaSpec,omitempty"`
}

type Container struct {
	Name            string        `json:"Name"`
	Image           string        `json:"Image,omitempty"`
	ImagePullPolicy string        `json:"ImagePullPolicy,omitempty"`
	Env             []EnvVar      `json:"Env,omitempty"`
	VolumeMounts    []VolumeMount `json:"VolumeMounts,omitempty"`
	Resources       *Resources    `json:"Resources,omitempty"`
	Liveness        *Probe        `json:"Liveness,omitempty"`
	Readiness       *Probe        `json:"Readiness,omitempty"`
}

type VolumeMount struct {
	Name      string `json:"Name"`
	MountPath string `json:"MountPath"`
	ReadOnly  bool   `json:"ReadOnly,omitempty"`
}

type Volume struct {
	Name     string              `json:"Name"`
	EmptyDir *struct{}           `json:"EmptyDir,omitempty"`
	Secret   *SecretVolumeSource `json:"Secret,omitempty"`
}

type SecretVolumeSource struct {
	SecretName string `json:"SecretName"`
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
	Scheme              string `json:"Scheme,omitempty"`
}

type InitContainer struct {
	Name            string        `json:"Name"`
	Image           string        `json:"Image"`
	ImagePullPolicy string        `json:"ImagePullPolicy"`
	Args            []string      `json:"Args"`
	VolumeMounts    []VolumeMount `json:"VolumeMounts"`
	EnvFrom         []EnvFromItem `json:"EnvFrom,omitempty"`
}

type EnvFromItem struct {
	SecretRef *SecretRef `json:"SecretRef,omitempty"`
}

type SecretRef struct {
	Name string `json:"Name"`
}

type Endpoint struct {
	Type string `json:"Type,omitempty"`
}

type HPATarget struct {
	Type               string `json:"type"`               // e.g., "Utilization"
	AverageUtilization int    `json:"averageUtilization"` // e.g., 70
}

type HPAResourceMetric struct {
	Name   string    `json:"name"`   // e.g., "cpu"
	Target HPATarget `json:"target"` // Target threshold
}

type HPAMetric struct {
	Type     string            `json:"type"`     // e.g., "Resource"
	Resource HPAResourceMetric `json:"resource"` // Resource-based metric
}

type HPASpec struct {
	MinReplicas int         `json:"minReplicas"`
	MaxReplicas int         `json:"maxReplicas"`
	MetricsV2   []HPAMetric `json:"metricsv2,omitempty"`
}

type SvcOrchSpec struct {
	Resources *Resources `json:"resources,omitempty"`
	Env       []EnvVar   `json:"env,omitempty"`
}

type ChartTemplate struct {
	Files []struct {
		Path     string `yaml:"path"`
		Template bool   `yaml:"template"`
		Content  string `yaml:"content"`
	} `yaml:"files"`
}

var log = logrus.New()

func main() {
	var values Values
	log.Printf("Reading values file")
	valData, err := os.ReadFile("../config/values.json")
	check(err)
	check(json.Unmarshal(valData, &values))

	//valuesStr, err := json.Marshal(values)
	//check(err)
	//log.Printf("Generated: %s", valuesStr)

	var meta ChartTemplate
	log.Printf("Reading template file")
	tmplData, err := os.ReadFile("../config/seldon.meta.yaml")
	check(err)
	check(yaml.Unmarshal(tmplData, &meta))

	for _, file := range meta.Files {
		newfilepath := file.Path
		if file.Path == "values-ENV.yaml" && values.Environment != "" {
			newfilepath = "values-" + values.Environment + ".yaml"
		}

		outputPath := filepath.Join("output", newfilepath)
		outputDir := filepath.Dir(outputPath)
		check(os.MkdirAll(outputDir, 0755))

		content := file.Content
		if file.Template {
			tmpl, err := template.New(file.Path).Parse(file.Content)
			check(err)
			var buf bytes.Buffer
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
