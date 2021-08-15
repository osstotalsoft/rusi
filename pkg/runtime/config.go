package runtime

import "fmt"

const (
	DefaultGRPCPort    = 50001
	DefaultProfilePort = 7777
	// DefaultMetricsPort is the default port for metrics endpoints.
	DefaultMetricsPort = 9090
)

type Config struct {
	RusiGRPCPort    string
	AppPort         string
	ComponentsPath  string
	Config          string
	AppID           string
	EnableProfiling bool
}

func NewRuntimeConfig() Config {
	return Config{}
}

func (c *Config) AttachCmdFlags(
	stringVar func(p *string, name string, value string, usage string),
	boolVar func(p *bool, name string, value bool, usage string)) {

	stringVar(&c.RusiGRPCPort, "rusi-grpc-port", fmt.Sprintf("%v", DefaultGRPCPort), "gRPC port for the Rusi API to listen on")
	stringVar(&c.AppPort, "app-port", "", "The port the application is listening on")
	stringVar(&c.ComponentsPath, "components-path", "", "Path for components directory. If empty, components will not be loaded. Self-hosted mode only")
	stringVar(&c.Config, "config", "", "Path to config file, or name of a configuration object")
	stringVar(&c.AppID, "app-id", "", "A unique ID for Rusi. Used for Service Discovery and state")
	boolVar(&c.EnableProfiling, "enable-profiling", false, "Enable profiling")
}
