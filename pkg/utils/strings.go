package utils

import (
	corev1 "k8s.io/api/core/v1"
	"strings"
)

// add env-vars from annotations.
// see https://github.com/dapr/dapr/issues/2508.
func ParseEnvString(envStr string) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0)
	envPairs := strings.Split(envStr, ",")

	for _, value := range envPairs {
		pair := strings.Split(strings.TrimSpace(value), "=")

		if len(pair) != 2 {
			continue
		}

		envVars = append(envVars, corev1.EnvVar{
			Name:  pair[0],
			Value: pair[1],
		})
	}

	return envVars
}

// StringSliceContains return true if an array containe the "str" string.
func StringSliceContains(needle string, haystack []string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}
