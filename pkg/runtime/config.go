package runtime

import (
	"errors"
	"fmt"
	"os"
	custom_resource "rusi/pkg/custom-resource"
	"rusi/pkg/modes"
)

const (
	DefaultGRPCPort    = 50003
	DefaultProfilePort = 7777
	// DefaultMetricsPort is the default port for metrics endpoints.
	DefaultMetricsPort = 9090
)

type ConfigBuilder struct {
	mode                string
	rusiGRPCPort        string
	componentsPath      string
	config              string
	controlPlaneAddress string
	appID               string
	enableProfiling     bool
}

type Config struct {
	Mode                modes.RusiMode
	RusiGRPCPort        string
	ComponentsPath      string
	Config              string
	ControlPlaneAddress string
	AppID               string
	EnableProfiling     bool
}

func NewRuntimeConfigBuilder() ConfigBuilder {
	return ConfigBuilder{}
}

func (c *ConfigBuilder) AttachCmdFlags(
	stringVar func(p *string, name string, value string, usage string),
	boolVar func(p *bool, name string, value bool, usage string)) {

	stringVar(&c.mode, "mode", string(modes.StandaloneMode), "Runtime mode for Rusi (kubernetes / standalone - default:standalone )")
	stringVar(&c.rusiGRPCPort, "rusi-grpc-port", fmt.Sprintf("%v", DefaultGRPCPort), "gRPC port for the Rusi API to listen on")
	stringVar(&c.componentsPath, "components-path", "", "Path for components directory. If empty, components will not be loaded. Self-hosted mode only")
	stringVar(&c.config, "config", "", "Path to config file, or name of a configuration object")
	stringVar(&c.controlPlaneAddress, "control-plane-address", "", "Address for Rusi control plane")
	stringVar(&c.appID, "app-id", "", "A unique ID for Rusi. Used for Service Discovery and state")
	boolVar(&c.enableProfiling, "enable-profiling", false, "Enable profiling")
}

func (c *ConfigBuilder) Build() (Config, error) {
	err := c.validate()
	if err != nil {
		return Config{}, err
	}

	variables := map[string]string{
		custom_resource.AppID:        c.appID,
		custom_resource.RusiGRPCPort: c.rusiGRPCPort,
		//custom_resource.RusiMetricsPort: metricsExporter.Options().Port,
		//custom_resource.RusiProfilePort: c.profilePort,
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
		EnableProfiling:     c.enableProfiling,
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
