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
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

/*
yaml:"app.dognauts/platform-generated" json:"platformGenerated"
*/
type Values struct {
	Environment          string            `json:"environment" yaml:"environment"`
	ApiVersion           string            `json:"apiVersion" yaml:"apiVersion"`
	Chart                Chart             `json:"chart" yaml:"chart"`
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
	//  Replicas & ServerType
	Replicas   *int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	ServerType string `json:"serverType,omitempty" yaml:"serverType,omitempty"`
}

type Values2 struct {
	Environment          string          `json:"environment" yaml:"environment"`
	ApiVersion           string          `json:"apiVersion" yaml:"apiVersion"`
	Chart                Chart           `json:"chart" yaml:"chart"`
	Namespace            string          `json:"namespace" yaml:"namespace"`
	Metadata             Metadata        `json:"metadata" yaml:"metadata"`
	Servers              []ServerInput   `json:"servers,omitempty" yaml:"servers,omitempty"`
	ServerConfigs        []ServerConfig  `json:"serverConfigs,omitempty" yaml:"servers,omitempty"`
	Models               []ModelInput    `json:"models,omitempty" yaml:"models,omitempty"`
	Pipelines            []Pipeline      `json:"pipelines"`
	SeldonRuntimes       []SeldonRuntime `json:"seldonRuntimes"`
	Experiments          []Experiment    `json:"experiment"`
	SubjectArea          string          `json:"subjectArea,omitempty" yaml:"subjectArea,omitempty"`
	SourceMetafileName   string          `json:"sourceMetafileName,omitempty" yaml:"sourceMetafileName,omitempty"`
	SourceMetafileRepo   string          `json:"sourceMetafileRepo,omitempty" yaml:"sourceMetafileRepo,omitempty"`
	SourceMetafileBranch string          `json:"sourceMetafileBranch,omitempty" yaml:"sourceMetafileBranch,omitempty"`
}

type Experiment struct {
	Default      *string               `json:"default,omitempty"`
	Candidates   []ExperimentCandidate `json:"candidates"`
	Mirror       *ExperimentMirror     `json:"mirror,omitempty"`
	ResourceType string                `json:"resourceType,omitempty"` // "model" or "pipeline"
}

type ExperimentCandidate struct {
	Name   string `json:"name"`
	Weight uint32 `json:"weight"`
}

type ExperimentMirror struct {
	Name    string `json:"name"`
	Percent uint32 `json:"percent"`
}

type SeldonRuntime struct {
	SeldonConfig      string              `json:"seldonConfig"`
	Overrides         []OverrideSpec      `json:"overrides,omitempty"`
	Config            SeldonConfiguration `json:"config,omitempty"`
	DisableAutoUpdate bool                `json:"disableAutoUpdate,omitempty"`
}

type OverrideSpec struct {
	Name        string         `json:"name"`
	Disable     bool           `json:"disable,omitempty"`
	Replicas    *int32         `json:"replicas,omitempty"`
	ServiceType v1.ServiceType `json:"serviceType,omitempty"`
	PodSpec     *v1.PodSpec    `json:"podSpec,omitempty"`
}

type SeldonConfiguration struct {
	TracingConfig TracingConfig      `json:"tracingConfig,omitempty"`
	KafkaConfig   KafkaConfig        `json:"kafkaConfig,omitempty"`
	AgentConfig   AgentConfiguration `json:"agentConfig,omitempty"`
	ServiceConfig ServiceConfig      `json:"serviceConfig,omitempty"`
}

type ServiceConfig struct {
	GrpcServicePrefix string         `json:"grpcServicePrefix,omitempty"`
	ServiceType       v1.ServiceType `json:"serviceType,omitempty"`
}

type KafkaConfig struct {
	BootstrapServers      string                        `json:"bootstrap.servers,omitempty"`
	ConsumerGroupIdPrefix string                        `json:"consumerGroupIdPrefix,omitempty"`
	Debug                 string                        `json:"debug,omitempty"`
	Consumer              map[string]intstr.IntOrString `json:"consumer,omitempty"`
	Producer              map[string]intstr.IntOrString `json:"producer,omitempty"`
	Streams               map[string]intstr.IntOrString `json:"streams,omitempty"`
	TopicPrefix           string                        `json:"topicPrefix,omitempty"`
}

type TracingConfig struct {
	Disable              bool   `json:"disable,omitempty"`
	OtelExporterEndpoint string `json:"otelExporterEndpoint,omitempty"`
	OtelExporterProtocol string `json:"otelExporterProtocol,omitempty"`
	Ratio                string `json:"ratio,omitempty"`
}

type AgentConfiguration struct {
	Rclone RcloneConfiguration `json:"rclone,omitempty" yaml:"rclone,omitempty"`
}

type RcloneConfiguration struct {
	ConfigSecrets []string `json:"config_secrets,omitempty" yaml:"config_secrets,omitempty"`
	Config        []string `json:"config,omitempty" yaml:"config,omitempty"`
}

type ServerConfig struct {
	PodSpec              v1.PodSpec              `json:"podSpec"`
	VolumeClaimTemplates []PersistentVolumeClaim `json:"volumeClaimTemplates"`
}

type PersistentVolumeClaim struct {
	Name string                       `json:"name"`
	Spec v1.PersistentVolumeClaimSpec `json:"spec"`
}

type Pipeline struct {
	Name            string            `json:"name"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Input           *PipelineInput    `json:"input,omitempty"`
	Steps           []PipelineStep    `json:"steps"`
	Output          *PipelineOutput   `json:"output,omitempty"`
	Dataflow        *DataflowSpec     `json:"dataflow,omitempty"`
	AllowCycles     bool              `json:"allowCycles,omitempty"`
	MaxStepRevisits uint32            `json:"maxStepRevisits,omitempty"`
}

type PipelineInput struct {
	ExternalInputs   []string          `json:"externalInputs,omitempty"`
	ExternalTriggers []string          `json:"externalTriggers,omitempty"`
	JoinWindowMs     *uint32           `json:"joinWindowMs,omitempty"`
	JoinType         *JoinType         `json:"joinType,omitempty"`
	TriggersJoinType *JoinType         `json:"triggersJoinType,omitempty"`
	TensorMap        map[string]string `json:"tensorMap,omitempty"`
}

type DataflowSpec struct {
	CleanTopicsOnDelete bool `json:"cleanTopicsOnDelete,omitempty"`
}

type JoinType string // enum: "inner", "outer", "any"

type PipelineStep struct {
	Name             string            `json:"name"`
	Inputs           []string          `json:"inputs,omitempty"`
	JoinWindowMs     *uint32           `json:"joinWindowMs,omitempty"`
	TensorMap        map[string]string `json:"tensorMap,omitempty"`
	Triggers         []string          `json:"triggers,omitempty"`
	InputsJoinType   *JoinType         `json:"inputsJoinType,omitempty"`
	TriggersJoinType *JoinType         `json:"triggersJoinType,omitempty"`
	Batch            *PipelineBatch    `json:"batch,omitempty"`
}

type PipelineBatch struct {
	Size     *uint32 `json:"size,omitempty"`
	WindowMs *uint32 `json:"windowMs,omitempty"`
	Rolling  *bool   `json:"rolling,omitempty"`
}

type PipelineOutput struct {
	Steps        []string          `json:"steps,omitempty"`
	JoinWindowMs *uint32           `json:"joinWindowMs,omitempty"`
	StepsJoin    *JoinType         `json:"stepsJoin,omitempty"`
	TensorMap    map[string]string `json:"tensorMap,omitempty"`
}

type ModelInput struct {
	Name            string            `json:"name"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	StorageUri      string            `json:"storageUri"`
	ArtifactVersion *uint32           `json:"artifactVersion,omitempty"`
	ModelType       *string           `json:"modelType,omitempty"`
	SchemaUri       *string           `json:"schemaUri,omitempty"`
	SecretName      *string           `json:"secretName,omitempty"`
	Requirements    []string          `json:"requirements,omitempty"`
	Memory          *string           `json:"memory,omitempty"`
	ScalingSpec     `yaml:",inline"`
	Server          *string         `json:"server,omitempty"`
	Preloaded       bool            `json:"preloaded,omitempty"`
	Dedicated       bool            `json:"dedicated,omitempty"`
	Logger          *LoggingSpec    `json:"logger,omitempty"`
	Explainer       *ExplainerSpec  `json:"explainer,omitempty"`
	Parameters      []ParameterSpec `json:"parameters,omitempty"`
	Llm             *LlmSpec        `json:"llm,omitempty"`
}

type LoggingSpec struct {
	Percent *uint `json:"percent,omitempty"`
}

type ExplainerSpec struct {
	Type        string  `json:"type,omitempty"`
	ModelRef    *string `json:"modelRef,omitempty"`
	PipelineRef *string `json:"pipelineRef,omitempty"`
}

type LlmSpec struct {
	ModelRef    *string `json:"modelRef,omitempty"`
	PipelineRef *string `json:"pipelineRef,omitempty"`
}

type ParameterSpec struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ServerInput struct {
	Name                                            string                                           `json:"name" yaml:"name"`
	Labels                                          map[string]string                                `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations                                     map[string]string                                `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	ServerConfig                                    string                                           `json:"serverConfig" yaml:"serverConfig"`
	ExtraCapabilities                               []string                                         `json:"extraCapabilities,omitempty" yaml:"extraCapabilities,omitempty"`
	ImageOverrides                                  *ContainerOverrideSpec                           `json:"imageOverrides,omitempty" yaml:"imageOverrides,omitempty"`
	PodSpec                                         *corev1.PodSpec                                  `json:"podSpec,omitempty" yaml:"podSpec,omitempty"`
	StatefulSetPersistentVolumeClaimRetentionPolicy *StatefulSetPersistentVolumeClaimRetentionPolicy `json:"statefulSetPersistentVolumeClaimRetentionPolicy,omitempty"`
	ScalingSpec                                     `yaml:",inline"`
	DisableAutoUpdate                               bool `json:"disableAutoUpdate,omitempty"`
}

type StatefulSetPersistentVolumeClaimRetentionPolicy struct {
	WhenDeleted string `json:"whenDeleted" yaml:"name"`
	WhenScaled  string `json:"whenScaled" yaml:"version"`
}

type ContainerOverrideSpec struct {
	Agent  *corev1.Container `json:"agent,omitempty" yaml:"agent,omitempty"`
	RClone *corev1.Container `json:"rclone,omitempty" yaml:"rclone,omitempty"`
}

type ScalingSpec struct {
	Replicas    *int32 `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	MinReplicas *int32 `json:"minReplicas,omitempty" yaml:"minReplicas,omitempty"`
	MaxReplicas *int32 `json:"maxReplicas,omitempty" yaml:"maxReplicas,omitempty"`
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
	Name      string       `json:"name" yaml:"name"`
	Value     *interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	ValueFrom *ValueFrom   `json:"valueFrom,omitempty" yaml:"valueFrom,omitempty"`
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
	Type              string             `json:"type" yaml:"type"`
	Name              string             `json:"name,omitempty" yaml:"name,omitempty"`
	UseCachedMetrics  bool               `json:"useCachedMetrics,omitempty" yaml:"useCachedMetrics,omitempty"`
	Metadata          map[string]string  `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	AuthenticationRef *AuthenticationRef `json:"authenticationRef,omitempty" yaml:"authenticationRef,omitempty"`
	MetricType        string             `json:"metricType,omitempty" yaml:"metricType,omitempty"`
}

type AuthenticationRef struct {
	Name string `json:"name" yaml:"name"`
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
}

type KedaAdvancedSpec struct {
	RestoreToOriginalReplicaCount *bool                          `json:"restoreToOriginalReplicaCount,omitempty" yaml:"restoreToOriginalReplicaCount,omitempty"`
	HorizontalPodAutoscalerConfig *HorizontalPodAutoscalerConfig `json:"horizontalPodAutoscalerConfig,omitempty" yaml:"horizontalPodAutoscalerConfig,omitempty"`
	ScalingModifiers              ScalingModifiers               `json:"scalingModifiers,omitempty" yaml:"scalingModifiers,omitempty"`
}

type HorizontalPodAutoscalerConfig struct {
	Behavior *HorizontalPodAutoscalerBehavior `json:"behavior,omitempty" yaml:"behavior,omitempty"`
	Name     string                           `json:"name,omitempty" yaml:"name,omitempty"`
}

type HorizontalPodAutoscalerBehavior struct {
	ScaleUp   *HPAScalingRules `json:"scaleUp,omitempty" yaml:"scaleUp,omitempty"`
	ScaleDown *HPAScalingRules `json:"scaleDown,omitempty" yaml:"scaleDown,omitempty"`
}

type HPAScalingRules struct {
	StabilizationWindowSeconds int                `json:"stabilizationWindowSeconds,omitempty" yaml:"stabilizationWindowSeconds,omitempty"`
	SelectPolicy               string             `json:"selectPolicy,omitempty" yaml:"selectPolicy,omitempty"`
	Policies                   []HPAScalingPolicy `json:"policies,omitempty" yaml:"policies,omitempty"`
	Tolerance                  string             `json:"tolerance,omitempty" yaml:"tolerance,omitempty"`
}

type HPAScalingPolicy struct {
	Type          string `json:"type" yaml:"type"`
	Value         int    `json:"value" yaml:"value"`
	PeriodSeconds int    `json:"periodSeconds" yaml:"periodSeconds"`
}

type ScalingModifiers struct {
	Formula          string `json:"formula,omitempty" yaml:"formula,omitempty"`
	Target           string `json:"target,omitempty" yaml:"target,omitempty"`
	ActivationTarget string `json:"activationTarget,omitempty" yaml:"activationTarget,omitempty"`
	MetricType       string `json:"metricType,omitempty" yaml:"metricType,omitempty"`
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
	Exec                          *ExecAction      `json:"exec,omitempty" yaml:"exec,omitempty"`
	HTTPGet                       *HTTPGetAction   `json:"httpGet,omitempty" yaml:"httpGet,omitempty"`
	TCPSocket                     *TCPSocketAction `json:"tcpSocket,omitempty" yaml:"tcpSocket,omitempty"`
	GRPC                          *GRPCAction      `json:"grpc,omitempty" yaml:"grpc,omitempty"`
	InitialDelaySeconds           int              `json:"initialDelaySeconds,omitempty" yaml:"initialDelaySeconds,omitempty"`
	TimeoutSeconds                int              `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
	PeriodSeconds                 int              `json:"periodSeconds,omitempty" yaml:"periodSeconds,omitempty"`
	SuccessThreshold              int              `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`
	FailureThreshold              int              `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty"`
	TerminationGracePeriodSeconds *int64           `json:"terminationGracePeriodSeconds,omitempty" yaml:"terminationGracePeriodSeconds,omitempty"`
}

type GRPCAction struct {
	Port    int    `json:"port" yaml:"port"`
	Service string `json:"service,omitempty" yaml:"service,omitempty"` // Optional
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
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	ServiceHost string `json:"service_host,omitempty" yaml:"service_host,omitempty"`
	ServicePort int32  `json:"service_port,omitempty" yaml:"service_port,omitempty"`
	HttpPort    int32  `json:"httpPort,omitempty" yaml:"httpPort,omitempty"`
	GrpcPort    int32  `json:"grpcPort,omitempty" yaml:"grpcPort,omitempty"`
}

type HPASpec struct {
	MinReplicas int                `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas int                `json:"maxReplicas" yaml:"maxReplicas"`
	MetricsV2   []HPAMetricV2      `json:"metricsv2,omitempty" yaml:"metricsv2,omitempty"`
	Metrics     []HPAMetricV2Beta1 `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

type HPAMetricTarget struct {
	Type               string `json:"type" yaml:"type"`
	Value              string `json:"value,omitempty" yaml:"value,omitempty"`
	AverageValue       string `json:"averageValue,omitempty" yaml:"averageValue,omitempty"`
	AverageUtilization *int   `json:"averageUtilization,omitempty" yaml:"averageUtilization,omitempty"`
}

type HPAMetricIdentifier struct {
	Name     string         `json:"name" yaml:"name"`
	Selector *LabelSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
}

type LabelSelector struct {
	MatchLabels      map[string]string
	MatchExpressions []LabelSelectorRequirement
}

type LabelSelectorRequirement struct {
	Key      string
	Operator string
	Values   []string
}

type HPAMetricV2 struct {
	Type     string               `json:"type" yaml:"type"`
	Resource *HPAResourceMetricV2 `json:"resource,omitempty" yaml:"resource,omitempty"`
	Pods     *HPAPodsMetricV2     `json:"pods,omitempty" yaml:"pods,omitempty"`
	Object   *HPAObjectMetricV2   `json:"object,omitempty" yaml:"object,omitempty"`
	External *HPAExternalMetricV2 `json:"external,omitempty" yaml:"external,omitempty"`
}

type HPAResourceMetricV2 struct {
	Name   string          `json:"name" yaml:"name"`
	Target HPAMetricTarget `json:"target" yaml:"target"`
}

type HPAPodsMetricV2 struct {
	Metric HPAMetricIdentifier `json:"metric" yaml:"metric"`
	Target HPAMetricTarget     `json:"target" yaml:"target"`
}

type HPAObjectMetricV2 struct {
	DescribedObject HPAObjectReference  `json:"describedObject" yaml:"describedObject"`
	Metric          HPAMetricIdentifier `json:"metric" yaml:"metric"`
	Target          HPAMetricTarget     `json:"target" yaml:"target"`
}

type HPAExternalMetricV2 struct {
	Metric HPAMetricIdentifier `json:"metric" yaml:"metric"`
	Target HPAMetricTarget     `json:"target" yaml:"target"`
}

type HPAObjectReference struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Name       string `json:"name" yaml:"name"`
}

type HPAMetricV2Beta1 struct {
	Type     string                    `json:"type" yaml:"type"`
	Resource *HPAResourceMetricV2Beta1 `json:"resource,omitempty" yaml:"resource,omitempty"`
	Pods     *HPAPodsMetricV2Beta1     `json:"pods,omitempty" yaml:"pods,omitempty"`
	Object   *HPAObjectMetricV2Beta1   `json:"object,omitempty" yaml:"object,omitempty"`
	External *HPAExternalMetricV2Beta1 `json:"external,omitempty" yaml:"external,omitempty"`
}

type HPAResourceMetricV2Beta1 struct {
	Name                     string `json:"name" yaml:"name"`
	TargetAverageUtilization *int   `json:"targetAverageUtilization,omitempty" yaml:"targetAverageUtilization,omitempty"`
	TargetAverageValue       string `json:"targetAverageValue,omitempty" yaml:"targetAverageValue,omitempty"`
}

type HPAPodsMetricV2Beta1 struct {
	MetricName         string         `json:"metricName" yaml:"metricName"`
	TargetAverageValue string         `json:"targetAverageValue" yaml:"targetAverageValue"`
	Selector           *LabelSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
}

type HPAObjectMetricV2Beta1 struct {
	Target       HPAObjectReference `json:"target" yaml:"target"`
	MetricName   string             `json:"metricName" yaml:"metricName"`
	TargetValue  string             `json:"targetValue" yaml:"targetValue"`
	Selector     *LabelSelector     `json:"selector,omitempty" yaml:"selector,omitempty"`
	AverageValue string             `json:"averageValue,omitempty" yaml:"averageValue,omitempty"`
}

type HPAExternalMetricV2Beta1 struct {
	MetricName         string         `json:"metricName" yaml:"metricName"`
	TargetValue        string         `json:"targetValue,omitempty" yaml:"targetValue,omitempty"`
	TargetAverageValue string         `json:"targetAverageValue,omitempty" yaml:"targetAverageValue,omitempty"`
	MetricSelector     *LabelSelector `json:"metricSelector,omitempty" yaml:"selector,omitempty"`
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
	Path        string       `json:"path,omitempty" yaml:"path,omitempty"`
	Port        *interface{} `json:"port" yaml:"port"`
	Host        string       `json:"host,omitempty" yaml:"host,omitempty"`
	Scheme      string       `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty" yaml:"httpHeaders,omitempty"`
}

type HTTPHeader struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
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
	var valuesCheck Values2
	log.Printf("Reading values file")
	valData, err := os.ReadFile("../config/values2.json")
	check(err)

	log.Printf("Loading schema file")
	schemaLoader := gojsonschema.NewReferenceLoader("file://../config/schema2.json")
	documentLoader := gojsonschema.NewReferenceLoader("file://../config/values2.json")
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	check(err)
	if result.Valid() {
		log.Printf("The input JSON is valid.")
	} else {
		log.Printf("The input JSON is not valid. See errors:")
		for _, desc := range result.Errors() {
			log.Printf("- %s\n", desc)
		}
	}
	check(json.Unmarshal(valData, &valuesCheck))

	var values = make(map[string]interface{})
	check(json.Unmarshal(valData, &values))

	var meta ChartTemplate
	log.Printf("Reading template file")
	tmplData, err := os.ReadFile("../config/meta-online-inference-seldon-v2.yaml")
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
