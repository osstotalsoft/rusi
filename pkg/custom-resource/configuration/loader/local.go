package loader

import (
	"context"
	"io/ioutil"
	"os"
	"rusi/pkg/custom-resource/configuration"

	yaml "gopkg.in/yaml.v2"
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
		PubSubSpec: configuration.PubSubSpec{
			Name: "",
		},
	}
}

// LoadStandaloneConfiguration gets the path to a config file and loads it into a configuration.
func LoadStandaloneConfiguration(config string) func(ctx context.Context) (<-chan configuration.Spec, error) {
	return func(ctx context.Context) (<-chan configuration.Spec, error) {
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
}
