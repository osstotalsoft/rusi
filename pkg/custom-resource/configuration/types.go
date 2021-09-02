package configuration

const (
	AllowAccess        = "allow"
	DenyAccess         = "deny"
	DefaultTrustDomain = "public"
)

type Feature string

type Configuration struct {
	Spec Spec `json:"spec" yaml:"spec"`
}

type Spec struct {
	SubscriberPipelineSpec PipelineSpec      `json:"subscriberPipeline,omitempty" yaml:"subscriberPipeline,omitempty"`
	PublisherPipelineSpec  PipelineSpec      `json:"publisherPipeline,omitempty" yaml:"publisherPipeline,omitempty"`
	TracingSpec            TracingSpec       `json:"tracing,omitempty"  yaml:"tracing,omitempty"`
	MetricSpec             MetricSpec        `json:"metric,omitempty" yaml:"metric,omitempty"`
	MTLSSpec               MTLSSpec          `json:"mtls,omitempty" yaml:"mtls,omitempty"`
	AccessControlSpec      AccessControlSpec `json:"accessControl,omitempty" yaml:"accessControl,omitempty"`
	Features               []FeatureSpec     `json:"features,omitempty" yaml:"features,omitempty"`
	APISpec                APISpec           `json:"api,omitempty" yaml:"api,omitempty"`
}

// APISpec describes the configuration for Rusi APIs.
type APISpec struct {
	Allowed []APIAccessRule `json:"allowed,omitempty" yaml:"allowed,omitempty"`
}

// APIAccessRule describes an access rule for allowing a Rusi API to be enabled and accessible by an app.
type APIAccessRule struct {
	Name     string `json:"name" yaml:"name"`
	Version  string `json:"version" yaml:"version"`
	Protocol string `json:"protocol" yaml:"protocol"`
}

// PipelineSpec defines the middleware pipeline.
type PipelineSpec struct {
	Handlers []HandlerSpec `json:"handlers" yaml:"handlers"`
}

// HandlerSpec defines a request handlers.
type HandlerSpec struct {
	Name    string `json:"name" yaml:"name"`
	Type    string `json:"type" yaml:"type"`
	Version string `json:"version" yaml:"version"`
}

// MTLSSpec defines mTLS configuration.
type MTLSSpec struct {
	Enabled          bool   `json:"enabled" yaml:"enabled"`
	WorkloadCertTTL  string `json:"workloadCertTTL" yaml:"workloadCertTTL"`
	AllowedClockSkew string `json:"allowedClockSkew" yaml:"allowedClockSkew"`
}

// TracingSpec defines distributed tracing configuration.
type TracingSpec struct {
	SamplingRate string     `json:"samplingRate" yaml:"samplingRate"`
	Zipkin       ZipkinSpec `json:"zipkin" yaml:"zipkin"`
}

// ZipkinSpec defines Zipkin trace configurations.
type ZipkinSpec struct {
	EndpointAddresss string `json:"endpointAddress" yaml:"endpointAddress"`
}

// MetricSpec defines metrics configuration.
type MetricSpec struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// AppPolicySpec defines the policy data structure for each app.
type AppPolicySpec struct {
	AppName             string               `json:"appId" yaml:"appId"`
	DefaultAction       string               `json:"defaultAction" yaml:"defaultAction"`
	TrustDomain         string               `json:"trustDomain" yaml:"trustDomain"`
	Namespace           string               `json:"namespace" yaml:"namespace"`
	AppOperationActions []AppOperationAction `json:"operations" yaml:"operations"`
}

// AppOperationAction defines the data structure for each app operation.
type AppOperationAction struct {
	Operation string   `json:"name" yaml:"name"`
	HTTPVerb  []string `json:"httpVerb" yaml:"httpVerb"`
	Action    string   `json:"action" yaml:"action"`
}

// AccessControlSpec is the spec object in ConfigurationSpec.
type AccessControlSpec struct {
	DefaultAction string          `json:"defaultAction" yaml:"defaultAction"`
	TrustDomain   string          `json:"trustDomain" yaml:"trustDomain"`
	AppPolicies   []AppPolicySpec `json:"policies" yaml:"policies"`
}

// FeatureSpec defines the features that are enabled/disabled.
type FeatureSpec struct {
	Name    Feature `json:"name" yaml:"name"`
	Enabled bool    `json:"enabled" yaml:"enabled"`
}
