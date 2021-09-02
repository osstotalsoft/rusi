package configuration

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFeatureEnabled(t *testing.T) {
	t.Run("Test feature enabled is correct", func(t *testing.T) {
		features := []FeatureSpec{
			{
				Name:    "testEnabled",
				Enabled: true,
			},
			{
				Name:    "testDisabled",
				Enabled: false,
			},
		}
		assert.True(t, IsFeatureEnabled(features, "testEnabled"))
		assert.False(t, IsFeatureEnabled(features, "testDisabled"))
		assert.False(t, IsFeatureEnabled(features, "testMissing"))
	})
}
