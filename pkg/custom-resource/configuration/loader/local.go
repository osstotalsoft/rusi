package loader

import (
	"context"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"rusi/pkg/custom-resource/configuration"
)

// LoadDefaultConfiguration returns the default config.
func LoadDefaultConfiguration() configuration.Spec {
	return configuration.Spec{
		TracingSpec: configuration.TracingSpec{
			SamplingRate: "",
		},
		MetricSpec: configuration.MetricSpec{
			Enabled: true,
		},
		AccessControlSpec: configuration.AccessControlSpec{
			DefaultAction: configuration.AllowAccess,
			TrustDomain:   configuration.DefaultTrustDomain,
		},
	}
}

// LoadStandaloneConfiguration gets the path to a config file and loads it into a configuration.
func LoadStandaloneConfiguration(ctx context.Context, config string) (<-chan configuration.Spec, error) {
	spec := LoadDefaultConfiguration()
	c := make(chan configuration.Spec)

	_, err := os.Stat(config)
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadFile(config)
	if err != nil {
		return c, err
	}

	// Parse environment variables from yaml
	b = []byte(os.ExpandEnv(string(b)))

	cfg := configuration.Configuration{Spec: spec}
	err = yaml.Unmarshal(b, &cfg)

	go func() {
		c <- cfg.Spec
	}()

	return c, err
}
