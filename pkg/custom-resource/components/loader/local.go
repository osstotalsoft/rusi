package loader

import (
	"bufio"
	"bytes"
	"context"
	"github.com/google/uuid"
	yaml "gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"path/filepath"
	"rusi/pkg/custom-resource/components"
	"strings"
)

const (
	yamlSeparator = "\n---"
	componentKind = "Component"
)

type yamlComponent struct {
	Kind     string   `yaml:"kind,omitempty"`
	Spec     yamlSpec `yaml:"spec"`
	Metadata metaData `yaml:"metadata"`
	Scopes   []string `yaml:"scopes"`
}

type metaData struct {
	Name string `yaml:"name"`
}

type yamlSpec struct {
	Type     string         `yaml:"type"`
	Version  string         `yaml:"version"`
	Metadata []metadataItem `yaml:"metadata"`
}

type metadataItem struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// LoadLocalComponents loads rusi components from a given directory.
func LoadLocalComponents(componentsPath string) ComponentsLoader {
	return func(ctx context.Context) (<-chan components.Spec, error) {
		c := make(chan components.Spec)

		files, err := ioutil.ReadDir(componentsPath)
		if err != nil {
			return nil, err
		}

		var list []components.Spec

		for _, file := range files {
			if !file.IsDir() && isYaml(file.Name()) {
				path := filepath.Join(componentsPath, file.Name())
				comps, _ := loadComponentsFromFile(path)
				if len(comps) > 0 {
					list = append(list, comps...)
				}
			}
		}

		//send to channel

		go func() {
			for _, comp := range list {
				c <- comp
			}
		}()

		return c, nil
	}
}

func loadComponentsFromFile(path string) ([]components.Spec, error) {
	var errs []error
	var comps []components.Spec
	b, err := ioutil.ReadFile(path)
	if err != nil {
		klog.ErrorS(err, "load components error when reading file", "path", path)
		return comps, err
	}
	comps, errs = decodeYaml(b)
	for _, err := range errs {
		klog.ErrorS(err, "load components error when parsing components yaml resource", "path", path)
	}
	return comps, err
}

// isYaml checks whether the file is yaml or not.
func isYaml(fileName string) bool {
	extension := strings.ToLower(filepath.Ext(fileName))
	if extension == ".yaml" || extension == ".yml" {
		return true
	}
	return false
}

// decodeYaml decodes the yaml document.
func decodeYaml(b []byte) (list []components.Spec, errs []error) {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	scanner.Split(splitYamlDoc)

	for {
		var comp yamlComponent
		err := decode(scanner, &comp)
		if err == io.EOF {
			break
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}

		if comp.Kind != componentKind {
			continue
		}

		list = append(list, components.Spec{
			Name:     comp.Metadata.Name,
			Type:     comp.Spec.Type,
			Version:  comp.Spec.Version,
			Metadata: convertMetadataItemsToProperties(comp.Spec.Metadata),
			Scopes:   comp.Scopes,
		})
	}

	return
}

// decode reads the YAML resource in document.
func decode(scanner *bufio.Scanner, c interface{}) error {
	if scanner.Scan() {
		return yaml.Unmarshal(scanner.Bytes(), c)
	}

	err := scanner.Err()
	if err == nil {
		err = io.EOF
	}
	return err
}

// splitYamlDoc - splits the yaml docs.
func splitYamlDoc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		i += sep
		after := data[i:]

		if len(after) == 0 {
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func convertMetadataItemsToProperties(items []metadataItem) map[string]string {
	properties := map[string]string{}
	for _, c := range items {
		val := c.Value
		for strings.Contains(val, "{uuid}") {
			val = strings.Replace(val, "{uuid}", uuid.New().String(), 1)
		}
		properties[c.Name] = val
	}
	return properties
}
