package loader

import (
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
func LoadStandaloneConfiguration(config string) (configuration.Spec, error) {
	spec := LoadDefaultConfiguration()

	_, err := os.Stat(config)
	if err != nil {
		return spec, err
	}

	b, err := ioutil.ReadFile(config)
	if err != nil {
		return spec, err
	}

	// Parse environment variables from yaml
	b = []byte(os.ExpandEnv(string(b)))

	cfg := configuration.Configuration{Spec: spec}
	err = yaml.Unmarshal(b, &cfg)
	return cfg.Spec, err
}
