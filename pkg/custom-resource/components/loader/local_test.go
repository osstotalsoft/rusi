package loader

import (
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const configPrefix = "."

func writeTempConfig(path, content string) error {
	return os.WriteFile(filepath.Join(configPrefix, path), []byte(content), fs.FileMode(0644))
}

func TestLoadComponentsFromFile(t *testing.T) {

	t.Run("valid yaml content", func(t *testing.T) {
		filename := "test-component-valid.yaml"
		yaml := `
apiVersion: rusi.io/v1alpha1
kind: Component
metadata:
  name: statestore
spec:
  type: state.couchbase
  metadata:
  - name: prop1
    value: value1
  - name: prop2
    value: value2
`
		err := writeTempConfig(filename, yaml)
		defer os.Remove(filename)
		assert.Nil(t, err)
		components, err := loadComponentsFromFile(filename)
		assert.Len(t, components, 1)
		assert.Nil(t, err)
	})

	t.Run("invalid yaml head", func(t *testing.T) {
		filename := "test-component-invalid.yaml"
		yaml := `
INVALID_YAML_HERE
apiVersion: rusi.io/v1alpha1
kind: Component
metadata:
name: statestore`
		err := writeTempConfig(filename, yaml)
		defer os.Remove(filename)
		assert.Nil(t, err)
		components, err := loadComponentsFromFile(filename)
		assert.Len(t, components, 0)
		assert.Nil(t, err)
	})

	t.Run("load components file not exist", func(t *testing.T) {
		filename := "test-component-no-exist.yaml"

		components, err := loadComponentsFromFile(filename)
		assert.Len(t, components, 0)
		assert.NotNil(t, err)
	})
}

func TestIsYaml(t *testing.T) {

	assert.True(t, isYaml("test.yaml"))
	assert.True(t, isYaml("test.YAML"))
	assert.True(t, isYaml("test.yml"))
	assert.True(t, isYaml("test.YML"))
	assert.False(t, isYaml("test.md"))
	assert.False(t, isYaml("test.txt"))
	assert.False(t, isYaml("test.sh"))
}

func TestStandaloneDecodeValidYaml(t *testing.T) {
	yaml := `
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
   name: statestore
spec:
   type: state.couchbase
   metadata:
   - name: prop1
     value: value1
   - name: prop2
     value: value2
`
	components, errs := decodeYaml([]byte(yaml))
	assert.Len(t, components, 1)
	assert.Empty(t, errs)
	assert.Equal(t, "statestore", components[0].Name)
	assert.Equal(t, "state.couchbase", components[0].Type)
	assert.Len(t, components[0].Metadata, 2)
	assert.Equal(t, "value1", components[0].Metadata["prop1"])
}

func TestStandaloneDecodeInvalidComponent(t *testing.T) {
	yaml := `
apiVersion: dapr.io/v1alpha1
kind: Subscription
metadata:
   name: testsub
spec:
   metadata:
   - name: prop1
     value: value1
   - name: prop2
     value: value2
`
	components, errs := decodeYaml([]byte(yaml))
	assert.Len(t, components, 0)
	assert.Len(t, errs, 0)
}

func TestStandaloneDecodeUnsuspectingFile(t *testing.T) {
	components, errs := decodeYaml([]byte("hey there"))
	assert.Len(t, components, 0)
	assert.Len(t, errs, 1)
}

func TestStandaloneDecodeInvalidYaml(t *testing.T) {

	yaml := `
INVALID_YAML_HERE
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
name: statestore`
	components, errs := decodeYaml([]byte(yaml))
	assert.Len(t, components, 0)
	assert.Len(t, errs, 1)
}

func TestStandaloneDecodeValidMultiYaml(t *testing.T) {
	yaml := `
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: statestore1
spec:
  type: state.couchbase
  metadata:
    - name: prop1
      value: value1
    - name: prop2
      value: value2
---
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: statestore2
spec:
  type: state.redis
  metadata:
    - name: prop3
      value: value3
`
	components, errs := decodeYaml([]byte(yaml))
	assert.Len(t, components, 2)
	assert.Empty(t, errs)
	assert.Equal(t, "statestore1", components[0].Name)
	assert.Equal(t, "state.couchbase", components[0].Type)
	assert.Len(t, components[0].Metadata, 2)
	assert.Equal(t, "value1", components[0].Metadata["prop1"])
	assert.Equal(t, "value2", components[0].Metadata["prop2"])

	assert.Equal(t, "statestore2", components[1].Name)
	assert.Equal(t, "state.redis", components[1].Type)
	assert.Len(t, components[1].Metadata, 1)
	assert.Equal(t, "value3", components[1].Metadata["prop3"])
}

func TestStandaloneDecodeInValidDocInMultiYaml(t *testing.T) {

	yaml := `
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: statestore1
spec:
  type: state.couchbase
  metadata:
    - name: prop1
      value: value1
    - name: prop2
      value: value2
---
INVALID_YAML_HERE
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
name: invalidyaml
---
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
    name: statestore2
spec:
  type: state.redis
  metadata:
    - name: prop3
      value: value3
`
	components, errs := decodeYaml([]byte(yaml))
	assert.Len(t, components, 2)
	assert.Len(t, errs, 1)

	assert.Equal(t, "statestore1", components[0].Name)
	assert.Equal(t, "state.couchbase", components[0].Type)
	assert.Len(t, components[0].Metadata, 2)
	assert.Equal(t, "value1", components[0].Metadata["prop1"])
	assert.Equal(t, "value2", components[0].Metadata["prop2"])

	assert.Equal(t, "statestore2", components[1].Name)
	assert.Equal(t, "state.redis", components[1].Type)
	assert.Len(t, components[1].Metadata, 1)
	assert.Equal(t, "value3", components[1].Metadata["prop3"])
}
