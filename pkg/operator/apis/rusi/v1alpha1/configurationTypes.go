package v1alpha1

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
	// +kubebuilder:default={enabled:true}
	MetricSpec MetricSpec `json:"metric,omitempty"`
	// +optional
	MTLSSpec MTLSSpec `json:"mtls,omitempty"`
	// +optional
	AccessControlSpec AccessControlSpec `json:"accessControl,omitempty"`
	// +optional
	Features []FeatureSpec `json:"features,omitempty"`
	// +optional
	APISpec APISpec `json:"api,omitempty"`
	// +optional
	PubSubSpec PubSubSpec `json:"pubSub,omitempty"`
}

// APISpec describes the configuration for Rusi APIs.
type APISpec struct {
	Allowed []APIAccessRule `json:"allowed,omitempty"`
}

// APIAccessRule describes an access rule for allowing a Rusi API to be enabled and accessible by an app.
type APIAccessRule struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Protocol string `json:"protocol"`
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

// MTLSSpec defines mTLS configuration.
type MTLSSpec struct {
	Enabled bool `json:"enabled"`
	// +optional
	WorkloadCertTTL string `json:"workloadCertTTL"`
	// +optional
	AllowedClockSkew string `json:"allowedClockSkew"`
}

// TracingSpec defines distributed tracing configuration.
type TracingSpec struct {
	SamplingRate string     `json:"samplingRate"`
	Zipkin       ZipkinSpec `json:"zipkin"`
}

// ZipkinSpec defines Zipkin trace configurations.
type ZipkinSpec struct {
	EndpointAddresss string `json:"endpointAddress"`
}

// MetricSpec defines metrics configuration.
type MetricSpec struct {
	Enabled bool `json:"enabled"`
}

// AppPolicySpec defines the policy data structure for each app.
type AppPolicySpec struct {
	AppName string `json:"appId" yaml:"appId"`
	// +optional
	DefaultAction string `json:"defaultAction" yaml:"defaultAction"`
	// +optional
	TrustDomain string `json:"trustDomain" yaml:"trustDomain"`
	// +optional
	Namespace string `json:"namespace" yaml:"namespace"`
	// +optional
	AppOperationActions []AppOperationAction `json:"operations" yaml:"operations"`
}

// AppOperationAction defines the data structure for each app operation.
type AppOperationAction struct {
	Operation string `json:"name" yaml:"name"`
	// +optional
	HTTPVerb []string `json:"httpVerb" yaml:"httpVerb"`
	Action   string   `json:"action" yaml:"action"`
}

// AccessControlSpec is the spec object in ConfigurationSpec.
type AccessControlSpec struct {
	// +optional
	DefaultAction string `json:"defaultAction" yaml:"defaultAction"`
	// +optional
	TrustDomain string `json:"trustDomain" yaml:"trustDomain"`
	// +optional
	AppPolicies []AppPolicySpec `json:"policies" yaml:"policies"`
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
