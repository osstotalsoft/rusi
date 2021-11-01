package injector

import (
	"flag"
	"github.com/kelseyhightower/envconfig"
)

var defaultConfig = NewConfigWithDefaults()

// Config represents configuration options for the Rusi Sidecar Injector webhook server.
type Config struct {
	TLSCertFile            string `envconfig:"TLS_CERT_FILE" required:"true"`
	TLSKeyFile             string `envconfig:"TLS_KEY_FILE" required:"true"`
	SidecarImage           string `envconfig:"SIDECAR_IMAGE" required:"true"`
	SidecarImagePullPolicy string `envconfig:"SIDECAR_IMAGE_PULL_POLICY"`
	Namespace              string `envconfig:"NAMESPACE" required:"true"`
	KubeClusterDomain      string `envconfig:"KUBE_CLUSTER_DOMAIN"`
	ValidateServiceAccount bool
}

// NewConfigWithDefaults returns a Config object with default values already
// applied. Callers are then free to set custom values for the remaining fields
// and/or override default values.
func NewConfigWithDefaults() Config {
	return Config{
		SidecarImagePullPolicy: "Always",
		ValidateServiceAccount: true,
	}
}

func BindConfigFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.BoolVar(&defaultConfig.ValidateServiceAccount, "validate_service_account", defaultConfig.ValidateServiceAccount, "If true, injector will validate that rusi sidecars can only be created by a specified list of serviceAccounts")
}

// GetConfig returns configuration derived from environment variables.
func GetConfig() (Config, error) {
	err := envconfig.Process("", &defaultConfig)
	return defaultConfig, err
}
