package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"text/template"

	"gservice/generator"
	"gservice/utils"
	"gservice/validation"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

/*
yaml:"app.dognauts/platform-generated" json:"platformGenerated"
*/
type Values struct {
	Environment          string            `json:"environment" yaml:"environment"`
	ApiVersion           string            `json:"apiVersion" yaml:"apiVersion"`
	Chart                Chart             `json:"chart" yaml:"chart"`
	DeploymentName       string            `json:"deploymentName" yaml:"deploymentName"`
	Namespace            string            `json:"namespace" yaml:"namespace"`
	Metadata             Metadata          `json:"metadata" yaml:"metadata"`
	Predictors           []Predictor       `json:"predictors" yaml:"predictors"`
	Annotations          map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Protocol             string            `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Transport            string            `json:"transport,omitempty" yaml:"transport,omitempty"`
	SubjectArea          string            `json:"subjectArea,omitempty" yaml:"subjectArea,omitempty"`
	SourceMetafileName   string            `json:"sourceMetafileName,omitempty" yaml:"sourceMetafileName,omitempty"`
	SourceMetafileRepo   string            `json:"sourceMetafileRepo,omitempty" yaml:"sourceMetafileRepo,omitempty"`
	SourceMetafileBranch string            `json:"sourceMetafileBranch,omitempty" yaml:"sourceMetafileBranch,omitempty"`
}

type Chart struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
}

type Metadata struct {
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

type Predictor struct {
	Name                    string            `json:"name" yaml:"name"`
	Replicas                int               `json:"replicas" yaml:"replicas"`
	Traffic                 int               `json:"traffic" yaml:"traffic"`
	SvcOrchSpec             *SvcOrchSpec      `json:"svcOrchSpec,omitempty" yaml:"svcOrchSpec,omitempty"`
	Graph                   GraphNode         `json:"graph" yaml:"graph"`
	ComponentSpec           ComponentSpec     `json:"componentSpec" yaml:"componentSpec"`
	EngineResources         *Resources        `json:"engineResources,omitempty" yaml:"engineResources,omitempty"`
	Labels                  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Explainer               *Explainer        `json:"explainer,omitempty" yaml:"explainer,omitempty"`
	Shadow                  bool              `json:"shadow,omitempty" yaml:"shadow,omitempty"`
	SSL                     *SSL              `json:"ssl,omitempty" yaml:"ssl,omitempty"`
	ProgressDeadlineSeconds int               `json:"progressDeadlineSeconds,omitempty" yaml:"progressDeadlineSeconds,omitempty"`
}

type SSL struct {
	CertSecretName string `json:"certSecretName,omitempty" yaml:"certSecretName,omitempty" protobuf:"string,2,opt,name=certSecretName"`
}

type Explainer struct {
	Type                    string            `json:"type,omitempty" yaml:"type,omitempty"`
	ModelUri                string            `json:"modelUri,omitempty" yaml:"modelUri,omitempty"`
	ServiceAccountName      string            `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
	Config                  map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
	ContainerSpec           *Container        `json:"containerSpec,omitempty" yaml:"containerSpec,omitempty"`
	Endpoint                *Endpoint         `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	EnvSecretRefName        string            `json:"envSecretRefName,omitempty" yaml:"envSecretRefName,omitempty"`
	StorageInitializerImage string            `json:"storageInitializerImage,omitempty" yaml:"storageInitializerImage,omitempty"`
	Replicas                int               `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	InitParameters          string            `json:"initParameters,omitempty" yaml:"initParameters,omitempty"`
}

type EnvVar struct {
	Name      string     `json:"name" yaml:"name"`
	Value     string     `json:"value,omitempty" yaml:"value,omitempty"`
	ValueFrom *ValueFrom `json:"valueFrom,omitempty" yaml:"valueFrom,omitempty"`
}

type ValueFrom struct {
	SecretKeyRef SecretKeyRef `json:"secretKeyRef" yaml:"secretKeyRef"`
}

type SecretKeyRef struct {
	Name string `json:"name" yaml:"name"`
	Key  string `json:"key" yaml:"key"`
}

type GraphNode struct {
	Name                    string      `json:"name" yaml:"name"`
	Type                    string      `json:"type" yaml:"type"`
	Implementation          string      `json:"implementation,omitempty" yaml:"implementation,omitempty"`
	ModelUri                string      `json:"modelUri,omitempty" yaml:"modelUri,omitempty"`
	EnvSecretRefName        string      `json:"envSecretRefName,omitempty" yaml:"envSecretRefName,omitempty"`
	Logger                  *Logger     `json:"logger,omitempty" yaml:"logger,omitempty"`
	Endpoint                *Endpoint   `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Parameters              []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Children                []GraphNode `json:"children,omitempty" yaml:"children,omitempty"`
	Methods                 []string    `json:"methods,omitempty" yaml:"methods,omitempty"`
	ServiceAccountName      string      `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
	StorageInitializerImage string      `json:"storageInitializerImage,omitempty" yaml:"storageInitializerImage,omitempty"`
}

type Logger struct {
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Parameter struct {
	Name  string `json:"name" yaml:"name"`
	Type  string `json:"type" yaml:"type"`
	Value string `json:"value" yaml:"value"`
}

type ComponentSpec struct {
	ServiceAccountName            string      `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
	TerminationGracePeriodSeconds int         `json:"terminationGracePeriodSeconds,omitempty" yaml:"terminationGracePeriodSeconds,omitempty"`
	Containers                    []Container `json:"containers,omitempty" yaml:"containers,omitempty"`
	Volumes                       []Volume    `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	InitContainers                []Container `json:"initContainers,omitempty" yaml:"initContainers,omitempty"`
	HPASpec                       *HPASpec    `json:"hpaSpec,omitempty" yaml:"hpaSpec,omitempty"`
	KedaSpec                      *KedaSpec   `json:"kedaSpec,omitempty" yaml:"kedaSpec,omitempty"`
	PdbSpec                       *PdbSpec    `json:"pdbSpec,omitempty" yaml:"pdbSpec,omitempty"`
}

type KedaSpec struct {
	MinReplicaCount *int32            `json:"minReplicaCount,omitempty" yaml:"minReplicaCount,omitempty"`
	MaxReplicaCount *int32            `json:"maxReplicaCount,omitempty" yaml:"maxReplicaCount,omitempty"`
	CooldownPeriod  *int32            `json:"cooldownPeriod,omitempty" yaml:"cooldownPeriod,omitempty"`
	PollingInterval *int32            `json:"pollingInterval,omitempty" yaml:"pollingInterval,omitempty"`
	Triggers        []KedaTrigger     `json:"triggers,omitempty" yaml:"triggers,omitempty"`
	Advanced        *KedaAdvancedSpec `json:"advanced,omitempty" yaml:"advanced,omitempty"`
}

type KedaTrigger struct {
	Type              string            `json:"type" yaml:"type"`
	Metadata          map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	AuthenticationRef *struct {
		Name string `json:"name" yaml:"name"`
	} `json:"authenticationRef,omitempty" yaml:"authenticationRef,omitempty"`
}

type KedaAdvancedSpec struct {
	RestoreToOriginalReplicaCount *bool `json:"restoreToOriginalReplicaCount,omitempty" yaml:"restoreToOriginalReplicaCount,omitempty"`
	HorizontalPodAutoscalerConfig *struct {
		Behavior map[string]interface{} `json:"behavior,omitempty" yaml:"behavior,omitempty"`
	} `json:"horizontalPodAutoscalerConfig,omitempty" yaml:"horizontalPodAutoscalerConfig,omitempty"`
}

type PdbSpec struct {
	MinAvailable   string `json:"minAvailable,omitempty" yaml:"minAvailable,omitempty"`
	MaxUnavailable string `json:"maxUnavailable,omitempty" yaml:"maxUnavailable,omitempty"`
}

type Container struct {
	Name            string        `json:"name" yaml:"name"`
	Image           string        `json:"image,omitempty" yaml:"image,omitempty"`
	ImagePullPolicy string        `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`
	Args            []string      `json:"args,omitempty" yaml:"args,omitempty"`
	Env             []EnvVar      `json:"env,omitempty" yaml:"env,omitempty"`
	EnvFrom         []EnvFromItem `json:"envFrom,omitempty" yaml:"envFrom,omitempty"`
	VolumeMounts    []VolumeMount `json:"volumeMounts,omitempty" yaml:"volumeMounts,omitempty"`
	Resources       *Resources    `json:"resources,omitempty" yaml:"resources,omitempty"`
	Liveness        *Probe        `json:"liveness,omitempty" yaml:"liveness,omitempty"`
	Readiness       *Probe        `json:"readiness,omitempty" yaml:"readiness,omitempty"`
	Lifecycle       *Lifecycle    `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`
}

type ResourceQuantities struct {
	CPU    string `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
}

type Resources struct {
	Requests *ResourceQuantities `json:"requests,omitempty" yaml:"requests,omitempty"`
	Limits   *ResourceQuantities `json:"limits,omitempty" yaml:"limits,omitempty"`
}

type VolumeMount struct {
	Name      string `json:"name" yaml:"name"`
	MountPath string `json:"mountPath" yaml:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
}

type Volume struct {
	Name     string              `json:"name" yaml:"name"`
	EmptyDir *EmptyDirVolume     `json:"emptyDir,omitempty" yaml:"emptyDir,omitempty"`
	Secret   *SecretVolumeSource `json:"secret,omitempty" yaml:"secret,omitempty"`
}

type EmptyDirVolume struct {
	Medium    string `json:"medium,omitempty" yaml:"medium,omitempty"`
	SizeLimit string `json:"sizeLimit,omitempty" yaml:"sizeLimit,omitempty"`
}

type SecretVolumeSource struct {
	SecretName string `json:"secretName" yaml:"secretName"`
}

type Probe struct {
	Path                string `json:"path" yaml:"path"`
	Port                string `json:"port" yaml:"port"`
	InitialDelaySeconds int    `json:"initialDelaySeconds" yaml:"initialDelaySeconds"`
	PeriodSeconds       int    `json:"periodSeconds" yaml:"periodSeconds"`
	FailureThreshold    int    `json:"failureThreshold" yaml:"failureThreshold"`
	SuccessThreshold    int    `json:"successThreshold" yaml:"successThreshold"`
	Scheme              string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

type InitContainer struct {
	Name            string        `json:"name" yaml:"name"`
	Image           string        `json:"image" yaml:"image"`
	ImagePullPolicy string        `json:"imagePullPolicy" yaml:"imagePullPolicy"`
	Args            []string      `json:"args,omitempty" yaml:"args,omitempty"`
	Env             []EnvVar      `json:"env,omitempty" yaml:"env,omitempty"`
	EnvFrom         []EnvFromItem `json:"envFrom,omitempty" yaml:"envFrom,omitempty"`
	VolumeMounts    []VolumeMount `json:"volumeMounts,omitempty" yaml:"volumeMounts,omitempty"`
	Resources       *Resources    `json:"resources,omitempty" yaml:"resources,omitempty"`
	Liveness        *Probe        `json:"liveness,omitempty" yaml:"liveness,omitempty"`
	Readiness       *Probe        `json:"readiness,omitempty" yaml:"readiness,omitempty"`
	Lifecycle       *Lifecycle    `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`
}

type EnvFromItem struct {
	SecretRef    *SecretRef    `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty" yaml:"configMapRef,omitempty"`
}

type SecretRef struct {
	Name string `json:"name" yaml:"name"`
}

type ConfigMapRef struct {
	Name string `json:"name" yaml:"name"`
}

type Endpoint struct {
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
}

type HPATarget struct {
	Type               string `json:"type" yaml:"type"`
	AverageUtilization int    `json:"averageUtilization" yaml:"averageUtilization"`
}

type HPAResourceMetric struct {
	Name   string    `json:"name" yaml:"name"`
	Target HPATarget `json:"target" yaml:"target"`
}

type HPAMetric struct {
	Type     string            `json:"type" yaml:"type"`
	Resource HPAResourceMetric `json:"resource" yaml:"resource"`
}

type HPASpec struct {
	MinReplicas int         `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas int         `json:"maxReplicas" yaml:"maxReplicas"`
	MetricsV2   []HPAMetric `json:"metricsv2,omitempty" yaml:"metricsv2,omitempty"`
}

type SvcOrchSpec struct {
	Resources *Resources `json:"resources,omitempty" yaml:"resources,omitempty"`
	Env       []EnvVar   `json:"env,omitempty" yaml:"env,omitempty"`
}

type Lifecycle struct {
	PostStart *LifecycleHandler `json:"postStart,omitempty" yaml:"postStart,omitempty"`
	PreStop   *LifecycleHandler `json:"preStop,omitempty" yaml:"preStop,omitempty"`
}

type LifecycleHandler struct {
	Exec      *ExecAction      `json:"exec,omitempty" yaml:"exec,omitempty"`
	HTTPGet   *HTTPGetAction   `json:"httpGet,omitempty" yaml:"httpGet,omitempty"`
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty" yaml:"tcpSocket,omitempty"`
}

type ExecAction struct {
	Command []string `json:"command" yaml:"command"`
}

type HTTPGetAction struct {
	Path        string            `json:"path,omitempty" yaml:"path,omitempty"`
	Port        string            `json:"port" yaml:"port"`
	Host        string            `json:"host,omitempty" yaml:"host,omitempty"`
	Scheme      string            `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	HTTPHeaders map[string]string `json:"httpHeaders,omitempty" yaml:"httpHeaders,omitempty"`
}

type TCPSocketAction struct {
	Port string `json:"port" yaml:"port"`
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
}

type ChartTemplate struct {
	Files []struct {
		Path     string `jsonl:"path" yaml:"path"`
		Template bool   `json:"path" yaml:"template"`
		Content  string `json:"content" yaml:"content"`
	} `yaml:"files"`
}

var log = logrus.New()

func main() {
	var valuesCheck Values
	log.Printf("Reading values file")
	valData, err := os.ReadFile("../config/values.json")
	check(err)

	check(json.Unmarshal(valData, &valuesCheck))

	var values = make(map[string]interface{})
	check(json.Unmarshal(valData, &values))

	var meta ChartTemplate
	log.Printf("Reading template file")
	tmplData, err := os.ReadFile("../config/seldon.meta.yaml")
	check(err)
	check(yaml.Unmarshal(tmplData, &meta))

	/*
		valuesStr, err := json.Marshal(values)
		check(err)
		log.Printf("Generated: %s", valuesStr)
	*/
	i := 1
	if i == 1 {

		params := struct {
			MergedSetupValues  map[string]interface{}
			Environment        string
			ProcessID          string
			CreateNewEnvValues bool
		}{
			MergedSetupValues:  values,
			Environment:        valuesCheck.Environment,
			ProcessID:          "12345", // Example process ID
			CreateNewEnvValues: false,   // Example value
		}
		//func GenerateChart(meta utils.MetaStructure, setupValues map[string]interface{}, outputDir string, cliEnv string, processID string, createEnvValues bool) error {
		log.Printf("ValidateMetafile")
		err = validation.ValidateMetafile(tmplData, params.ProcessID)
		check(err)

		repoProjectSubfolder := "output" // Example subfolder path
		log.Printf("LoadMeta")
		meta, err := utils.LoadMeta(tmplData, params.ProcessID)
		check(err)
		log.Printf("GenerateChart")
		if err := generator.GenerateChart(meta, params.MergedSetupValues, repoProjectSubfolder, params.Environment, params.ProcessID, params.CreateNewEnvValues); err != nil {
			check(err)
		}
		log.WithField("processID", params.ProcessID).Info("✅ Chart files generated")
		return
	}

	for _, file := range meta.Files {
		newfilepath := file.Path
		if file.Path == "values-ENV.yaml" && valuesCheck.Environment != "" {
			newfilepath = "values-" + valuesCheck.Environment + ".yaml"
		}

		outputPath := filepath.Join("output", newfilepath)
		outputDir := filepath.Dir(outputPath)
		check(os.MkdirAll(outputDir, 0755))

		content := file.Content
		if file.Template {
			tmpl, err := template.New(file.Path).Parse(file.Content)
			check(err)
			var buf bytes.Buffer

			tplData := map[string]interface{}{
				"Values": values,
			}

			log.Printf("Execute: %s", file.Path)
			check(tmpl.Execute(&buf, tplData))
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
