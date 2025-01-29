package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration describes an Rusi configuration setting.
type Configuration struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ConfigurationSpec `json:"spec,omitempty"`
}

// ConfigurationSpec is the spec for an configuration.
type ConfigurationSpec struct {
	// +optional
	SubscriberPipelineSpec PipelineSpec `json:"subscriberPipeline,omitempty"`
	// +optional
	PublisherPipelineSpec PipelineSpec `json:"publisherPipeline,omitempty"`
	// +optional
	TracingSpec TracingSpec `json:"tracing,omitempty"`
	// +optional
	Features []FeatureSpec `json:"features,omitempty"`
	// +optional
	PubSubSpec PubSubSpec `json:"pubSub,omitempty"`
	// +optional
	MinRuntimeVersion string `json:"minRuntimeVersion,omitempty"`
}

// PipelineSpec defines the middleware pipeline.
type PipelineSpec struct {
	Handlers []HandlerSpec `json:"handlers"`
}

// HandlerSpec defines a request handlers.
type HandlerSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// TracingSpec defines distributed tracing configuration.
type TracingSpec struct {
	Jaeger JaegerSpec `json:"jaeger"`
}

// JaegerSpec defines Jaeger trace configurations.
type JaegerSpec struct {
	UseAgent                 bool   `json:"useAgent"`
	CollectorEndpointAddress string `json:"collectorEndpointAddress"`
}

// FeatureSpec defines the features that are enabled/disabled.
type FeatureSpec struct {
	Name    string `json:"name" yaml:"name"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

// PubSubSpec defines default pubSub configuration.
type PubSubSpec struct {
	Name string `json:"name" yaml:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigurationList is a list of Rusi event sources.
type ConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Configuration `json:"items"`
}
