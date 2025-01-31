package loader

import (
	"context"
	"io/fs"
	"os"
	"rusi/pkg/custom-resource/configuration"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadStandaloneConfiguration(t *testing.T) {
	writeFile("config.yaml", `
kind: Configuration
metadata:
  name: secretappconfig
spec:
  mtls:
    enabled: true
  features:
    - name: "feature1"
      enabled: true
    - name: "feature2"
      enabled: false
  pubSub:
    name: natsstreaming-pubsub`)

	writeFile("env_variables_config.yaml", `
kind: Configuration
metadata:
  name: secretappconfig
spec:
  telemetry:
    collectorEndpoint: "${RUSI_TELEMETRY_COLLECTOR_ENDPOINT}"
    tracing:
      propagator: w3c `)

	defer os.Remove("config.yaml")
	defer os.Remove("env_variables_config.yaml")

	testCases := []struct {
		name          string
		path          string
		errorExpected bool
	}{
		{
			name:          "Valid config file",
			path:          "config.yaml",
			errorExpected: false,
		},
		{
			name:          "Invalid file path",
			path:          "invalid_file.yaml",
			errorExpected: true,
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadStandaloneConfiguration(tc.path)(ctx)
			if tc.errorExpected {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}
		})
	}

	t.Run("Parse environment variables", func(t *testing.T) {
		os.Setenv("RUSI_TELEMETRY_COLLECTOR_ENDPOINT", "http://localhost:42323")
		configChan, err := LoadStandaloneConfiguration("env_variables_config.yaml")(ctx)
		config := <-configChan
		assert.NoError(t, err, "Unexpected error")
		assert.NotNil(t, config, "Config not loaded as expected")
		assert.Equal(t, "http://localhost:42323", config.Telemetry.CollectorEndpoint)
	})

	t.Run("Load config file", func(t *testing.T) {
		configChan, err := LoadStandaloneConfiguration("config.yaml")(ctx)
		config := <-configChan
		assert.NoError(t, err, "Unexpected error")
		assert.NotNil(t, config, "Config not loaded as expected")
		assert.Len(t, config.Features, 2)
		assert.True(t, config.Features[0].Enabled)
		assert.False(t, config.Features[1].Enabled)
		assert.Equal(t, configuration.Feature("feature1"), config.Features[0].Name)
		assert.Equal(t, configuration.Feature("feature2"), config.Features[1].Name)
		assert.Equal(t, config.PubSubSpec.Name, "natsstreaming-pubsub")
	})
}

func TestFeatureSpecForStandAlone(t *testing.T) {

	writeFile("feature_config.yaml", `
kind: Configuration
metadata:
  name: configName
spec:
  features:
  - name: Actor.Reentrancy
    enabled: true
  - name: Test.Feature
    enabled: false`)
	defer os.Remove("feature_config.yaml")

	testCases := []struct {
		name           string
		confFile       string
		featureName    configuration.Feature
		featureEnabled bool
	}{
		{
			name:           "Feature is enabled",
			confFile:       "feature_config.yaml",
			featureName:    configuration.Feature("Actor.Reentrancy"),
			featureEnabled: true,
		},
		{
			name:           "Feature is disabled",
			confFile:       "feature_config.yaml",
			featureName:    configuration.Feature("Test.Feature"),
			featureEnabled: false,
		},
		{
			name:           "Feature is disabled if missing",
			confFile:       "feature_config.yaml",
			featureName:    configuration.Feature("Test.Missing"),
			featureEnabled: false,
		},
	}
	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configChan, err := LoadStandaloneConfiguration(tc.confFile)(ctx)
			config := <-configChan
			assert.NoError(t, err)
			assert.Equal(t, tc.featureEnabled, configuration.IsFeatureEnabled(config.Features, tc.featureName))
		})
	}
}

func writeFile(path, content string) {
	_ = os.WriteFile(path, []byte(content), fs.FileMode(0644))
}
