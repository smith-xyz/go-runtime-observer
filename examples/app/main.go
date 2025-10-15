package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type KubeConfig struct {
	APIVersion string                 `yaml:"apiVersion" validate:"required"`
	Kind       string                 `yaml:"kind" validate:"required"`
	Metadata   Metadata               `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name" validate:"required"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

func (k *KubeConfig) Validate() error {
	if k.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if k.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if k.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	return nil
}

func main() {
	for i := range 5 {
		reflect.ValueOf(fmt.Sprintf("hello-%d", i))
	}

	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	data, _ := json.Marshal(people)

	var decoded []Person
	json.Unmarshal(data, &decoded)

	yamlConfig := `
apiVersion: v1
kind: Deployment
metadata:
  name: example-app
  namespace: default
  labels:
    app: example
    version: "1.0"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: example
`

	var config KubeConfig
	yaml.Unmarshal([]byte(yamlConfig), &config)

	val := reflect.ValueOf(&config)
	method := val.MethodByName("Validate")
	if method.IsValid() {
		method.Call([]reflect.Value{})
	}
}
