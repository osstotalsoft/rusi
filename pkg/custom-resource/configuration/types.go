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
	TracingSpec            TracingSpec   `json:"tracing,omitempty"  yaml:"tracing,omitempty"`
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

// TracingSpec defines distributed tracing configuration.
type TracingSpec struct {
	Zipkin ZipkinSpec `json:"zipkin" yaml:"zipkin"`
}

// ZipkinSpec defines Zipkin trace configurations.
type ZipkinSpec struct {
	EndpointAddresss string `json:"endpointAddress" yaml:"endpointAddress"`
}

// FeatureSpec defines the features that are enabled/disabled.
type FeatureSpec struct {
	Name    Feature `json:"name" yaml:"name"`
	Enabled bool    `json:"enabled" yaml:"enabled"`
}

type PubSubSpec struct {
	Name string `json:"name" yaml:"name"`
}
