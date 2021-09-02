package configuration

func IsFeatureEnabled(features []FeatureSpec, target Feature) bool {
	for _, feature := range features {
		if feature.Name == target {
			return feature.Enabled
		}
	}
	return false
}
