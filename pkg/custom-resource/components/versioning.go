package components

import "strings"

const (
	// Unstable version (v0).
	UnstableVersion = "v0"

	// First stable version (v1).
	FirstStableVersion = "v1"
)

func IsInitialVersion(version string) bool {
	v := strings.ToLower(version)
	return v == "" || v == UnstableVersion || v == FirstStableVersion
}
