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
	SubscriberPipelineSpec PipelineSpec  `json:"subscriberPipeline,omitempty" yaml:"subscriberPipeline,omitempty"`
	PublisherPipelineSpec  PipelineSpec  `json:"publisherPipeline,omitempty" yaml:"publisherPipeline,omitempty"`
	Telemetry              TelemetrySpec `json:"telemetry,omitempty"  yaml:"telemetry,omitempty"`
	Features               []FeatureSpec `json:"features,omitempty" yaml:"features,omitempty"`
	PubSubSpec             PubSubSpec    `json:"pubSub,omitempty" yaml:"pubSub,omitempty"`
	MinRuntimeVersion      string        `json:"minRuntimeVersion,omitempty" yaml:"minRuntimeVersion,omitempty"`
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

// Telemetry related configuration.
type TelemetrySpec struct {
	// Telemetry collector enpoint address.
	CollectorEndpoint string `json:"collectorEndpoint"  yaml:"collectorEndpoint"`
	// Tracing configuration.
	// +optional
	Tracing TracingSpec `json:"tracing,omitempty" yaml:"tracing"`
}

type TelemetryPropagator string

const (
	TelemetryPropagatorW3c    = TelemetryPropagator("w3c")
	TelemetryPropagatorJaeger = TelemetryPropagator("jaeger")
)

// TracingSpec defines distributed tracing configuration.
type TracingSpec struct {
	// Telemetry propagator. Possible values: w3c, jaeger
	Propagator TelemetryPropagator `json:"propagator" yaml:"propagator"`
}

// FeatureSpec defines the features that are enabled/disabled.
type FeatureSpec struct {
	Name    Feature `json:"name" yaml:"name"`
	Enabled bool    `json:"enabled" yaml:"enabled"`
}

type PubSubSpec struct {
	Name string `json:"name" yaml:"name"`
}
