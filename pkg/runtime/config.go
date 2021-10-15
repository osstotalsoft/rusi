package runtime

import (
	"errors"
	"os"
	"rusi/pkg/modes"
	"rusi/pkg/utils"
	"strconv"
)

const (
	defaultGRPCPort = 50003
	// DefaultMetricsPort is the default port for metrics endpoints.
	defaultMetricsPort = 9090
	defaultHealthzPort = 8080
)

type ConfigBuilder struct {
	mode                string
	rusiGRPCPort        int
	healthzPort         int
	componentsPath      string
	config              string
	controlPlaneAddress string
	appID               string
}

type Config struct {
	Mode                modes.RusiMode
	RusiGRPCPort        int
	HealthzPort         int
	ComponentsPath      string
	Config              string
	ControlPlaneAddress string
	AppID               string
}

func NewRuntimeConfigBuilder() ConfigBuilder {
	return ConfigBuilder{}
}

func (c *ConfigBuilder) AttachCmdFlags(
	stringVar func(p *string, name string, value string, usage string),
	boolVar func(p *bool, name string, value bool, usage string),
	intVar func(p *int, name string, value int, usage string)) {

	stringVar(&c.mode, "mode", string(modes.StandaloneMode), "Runtime mode for Rusi (kubernetes / standalone - default:standalone )")
	intVar(&c.rusiGRPCPort, "rusi-grpc-port", defaultGRPCPort, "gRPC port for the Rusi API to listen on")
	stringVar(&c.componentsPath, "components-path", "", "Path for components directory. If empty, components will not be loaded. Self-hosted mode only")
	stringVar(&c.config, "config", "", "Path to config file, or name of a configuration object")
	stringVar(&c.controlPlaneAddress, "control-plane-address", "", "Address for Rusi control plane")
	stringVar(&c.appID, "app-id", "", "A unique ID for Rusi. Used for Service Discovery and state")
	intVar(&c.healthzPort, "healthz-port", defaultHealthzPort, "Sets the HTTP port for the healthz server")
}

func (c *ConfigBuilder) Build() (Config, error) {
	err := c.validate()
	if err != nil {
		return Config{}, err
	}

	variables := map[string]string{
		utils.AppID:        c.appID,
		utils.RusiGRPCPort: strconv.Itoa(c.rusiGRPCPort),
		//utils.RusiMetricsPort: metricsExporter.Options().Port,
		//utils.RusiProfilePort: c.profilePort,
	}

	if err = setEnvVariables(variables); err != nil {
		return Config{}, err
	}

	return Config{
		Mode:                modes.RusiMode(c.mode),
		RusiGRPCPort:        c.rusiGRPCPort,
		ComponentsPath:      c.componentsPath,
		Config:              c.config,
		AppID:               c.appID,
		HealthzPort:         c.healthzPort,
		ControlPlaneAddress: c.controlPlaneAddress,
	}, nil
}

func (c *ConfigBuilder) validate() error {
	if c.appID == "" {
		return errors.New("app-id parameter cannot be empty")
	}
	if c.config == "" {
		return errors.New("config parameter cannot be empty")
	}

	if modes.RusiMode(c.mode) == modes.KubernetesMode && c.controlPlaneAddress == "" {
		return errors.New("controlPlaneAddress is mandatory in kubernetes mode")
	}
	return nil
}

func setEnvVariables(variables map[string]string) error {
	for key, value := range variables {
		err := os.Setenv(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}
