module rusi

go 1.21

//https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies

require (
	github.com/google/uuid v1.3.1
	github.com/json-iterator/go v1.1.12
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/nats-io/nats.go v1.31.0
	github.com/nats-io/stan.go v0.10.4
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/contrib/propagators/jaeger v1.20.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.19.0
	go.opentelemetry.io/otel/exporters/prometheus v0.42.0
	go.opentelemetry.io/otel/metric v1.19.0
	go.opentelemetry.io/otel/sdk v1.19.0
	go.opentelemetry.io/otel/sdk/metric v1.19.0
	go.opentelemetry.io/otel/trace v1.19.0
	golang.org/x/sync v0.4.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.28.3
	k8s.io/apiextensions-apiserver v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	k8s.io/klog/v2 v2.100.1
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b
)