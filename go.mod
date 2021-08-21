module rusi

go 1.16

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/json-iterator/go v1.1.11
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/nats-io/nats-server/v2 v2.3.4 // indirect
	github.com/nats-io/nats-streaming-server v0.22.1 // indirect
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/nats-io/stan.go v0.10.0
	github.com/pkg/errors v0.9.1
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/klog/v2 v2.10.0
)

exclude github.com/go-logr/logr v1.0.0
