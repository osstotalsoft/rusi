package runtime

import (
	"errors"
	"os"
	"rusi/pkg/modes"
	"rusi/pkg/utils"
	"strconv"
)

const (
	defaultGRPCHost        = "localhost"
	defaultGRPCPort        = 50003
	defaultDiagnosticsPort = 8080
	defaultEnableMetrics   = true
)

type ConfigBuilder struct {
	mode                string
	rusiGRPCPort        int
	rusiGRPCHost        string
	diagnosticsPort     int
	enableMetrics       bool
	componentsPath      string
	config              string
	controlPlaneAddress string
	appID               string
}

type Config struct {
	Mode                modes.RusiMode
	RusiGRPCHost        string
	RusiGRPCPort        int
	DiagnosticsPort     int
	EnableMetrics       bool
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
	stringVar(&c.rusiGRPCHost, "rusi-grpc-host", defaultGRPCHost, "gRPC host for the Rusi API to listen on")
	intVar(&c.rusiGRPCPort, "rusi-grpc-port", defaultGRPCPort, "gRPC port for the Rusi API to listen on")
	stringVar(&c.componentsPath, "components-path", "", "Path for components directory. If empty, components will not be loaded. Self-hosted mode only")
	stringVar(&c.config, "config", "", "Path to config file, or name of a configuration object")
	stringVar(&c.controlPlaneAddress, "control-plane-address", "", "Address for Rusi control plane")
	stringVar(&c.appID, "app-id", "", "A unique ID for Rusi. Used for Service Discovery and state")
	intVar(&c.diagnosticsPort, "diagnostics-port", defaultDiagnosticsPort, "Sets the HTTP port for the diagnostics server (healthz and metrics)")
	boolVar(&c.enableMetrics, "enable-metrics", defaultEnableMetrics, "Enable prometheus metrics endpoint")
}

func (c *ConfigBuilder) Build() (Config, error) {
	err := c.validate()
	if err != nil {
		return Config{}, err
	}

	variables := map[string]string{
		utils.AppID:        c.appID,
		utils.RusiGRPCHost: c.rusiGRPCHost,
		utils.RusiGRPCPort: strconv.Itoa(c.rusiGRPCPort),
	}

	if err = setEnvVariables(variables); err != nil {
		return Config{}, err
	}

	return Config{
		Mode:                modes.RusiMode(c.mode),
		RusiGRPCHost:        c.rusiGRPCHost,
		RusiGRPCPort:        c.rusiGRPCPort,
		ComponentsPath:      c.componentsPath,
		Config:              c.config,
		AppID:               c.appID,
		DiagnosticsPort:     c.diagnosticsPort,
		EnableMetrics:       c.enableMetrics,
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
