package loader

import (
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"rusi/pkg/custom-resource/configuration"
	"testing"
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
      enabled: false`)

	writeFile("env_variables_config.yaml", `
kind: Configuration
metadata:
  name: secretappconfig
spec:
  tracing:
    zipkin:
      endpointAddress: "${RUSI_ZIPKIN_ENDPOINT}" `)

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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadStandaloneConfiguration(tc.path)
			if tc.errorExpected {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}
		})
	}

	t.Run("Parse environment variables", func(t *testing.T) {
		os.Setenv("RUSI_ZIPKIN_ENDPOINT", "http://localhost:42323")
		config, err := LoadStandaloneConfiguration("env_variables_config.yaml")
		assert.NoError(t, err, "Unexpected error")
		assert.NotNil(t, config, "Config not loaded as expected")
		assert.Equal(t, "http://localhost:42323", config.TracingSpec.Zipkin.EndpointAddresss)
	})

	t.Run("Load config file", func(t *testing.T) {
		config, err := LoadStandaloneConfiguration("config.yaml")
		assert.NoError(t, err, "Unexpected error")
		assert.NotNil(t, config, "Config not loaded as expected")
		assert.True(t, config.MetricSpec.Enabled)
		assert.True(t, config.MTLSSpec.Enabled)
		assert.Len(t, config.Features, 2)
		assert.True(t, config.Features[0].Enabled)
		assert.False(t, config.Features[1].Enabled)
		assert.Equal(t, configuration.Feature("feature1"), config.Features[0].Name)
		assert.Equal(t, configuration.Feature("feature2"), config.Features[1].Name)
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadStandaloneConfiguration(tc.confFile)
			assert.NoError(t, err)
			assert.Equal(t, tc.featureEnabled, configuration.IsFeatureEnabled(config.Features, tc.featureName))
		})
	}
}

func writeFile(path, content string) {
	_ = os.WriteFile(path, []byte(content), fs.FileMode(0644))
}
