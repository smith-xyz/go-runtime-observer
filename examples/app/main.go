package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"example.com/example/app/internal/security"
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

func (p *Person) GetName() string {
	return p.Name
}

func (p *Person) SetAge(age int) {
	p.Age = age
}

// DynamicMethodCaller demonstrates the call graph gap:
// This function -> reflect operations -> actual struct method
// The logs should show how to connect these dots using pointer addresses
func DynamicMethodCaller(obj *Person, methodName string, args []interface{}) []reflect.Value {
	val := reflect.ValueOf(obj)
	method := val.MethodByName(methodName)
	if !method.IsValid() {
		return nil
	}

	callArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		callArgs[i] = reflect.ValueOf(arg)
	}

	return method.Call(callArgs)
}

func main() {
	if !security.ValidateInput("test") {
		panic("validation failed")
	}

	for i := 0; i < 5; i++ {
		reflect.ValueOf(fmt.Sprintf("hello-%d", i))
	}

	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}

	data, _ := json.Marshal(people)

	var decoded []Person
	_ = json.Unmarshal(data, &decoded)

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
	_ = yaml.Unmarshal([]byte(yamlConfig), &config)

	val := reflect.ValueOf(&config)
	method := val.MethodByName("Validate")
	if method.IsValid() {
		method.Call([]reflect.Value{})
	}

	// Test call graph reconstruction with pointer tracking
	person := &Person{Name: "Charlie", Age: 35}

	// Get method via reflection
	val = reflect.ValueOf(person)
	method = val.MethodByName("GetName")
	if !method.IsValid() {
		panic("method not found")
	}

	// Temporal gap: do some unsafe operations before calling
	unsafeArr := [16]int{3: 3, 9: 9, 11: 11}
	eleSize := int(unsafe.Sizeof(unsafeArr[0]))
	p9 := &unsafeArr[9]
	up9 := unsafe.Pointer(p9)
	_ = (*int)(unsafe.Add(up9, -6*eleSize))

	// Now call the method - internal ptr should still match
	name := method.Call([]reflect.Value{})
	if len(name) > 0 {
		fmt.Printf("Got name via reflection: %s\n", name[0].String())
	}

	// Another test with SetAge - also with temporal gap
	method2 := val.MethodByName("SetAge")
	if !method2.IsValid() {
		panic("method not found")
	}

	// More temporal operations
	anotherArr := [8]int{1: 1, 5: 5}
	p5 := &anotherArr[5]
	up5 := unsafe.Pointer(p5)
	_ = (*int)(unsafe.Add(up5, -3*int(unsafe.Sizeof(anotherArr[0]))))

	// Call with gap between MethodByName and Call
	method2.Call([]reflect.Value{reflect.ValueOf(40)})

	security.UnsafeMemoryOperator(5)
}
