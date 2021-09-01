package custom_resource

const (
	// HostAddress is the address of the instance.
	HostAddress string = "HOST_ADDRESS"
	// RusiGRPCPort is the rusi api grpc port.
	RusiGRPCPort string = "RUSI_GRPC_PORT"
	// RusiMetricsPort is the rusi metrics port.
	RusiMetricsPort string = "RUSI_METRICS_PORT"
	// RusiProfilePort is the rusi performance profiling port.
	RusiProfilePort string = "RUSI_PROFILE_PORT"
	// AppPort is the port of the application, http/grpc depending on mode.
	AppPort string = "APP_PORT"
	// AppID is the ID of the application.
	AppID string = "APP_ID"
)
