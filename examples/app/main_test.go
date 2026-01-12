package main

import (
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"reflect"
	"testing"
)

func TestDynamicMethodCaller(t *testing.T) {
	person := &Person{Name: "TestPerson", Age: 25}

	results := DynamicMethodCaller(person, "GetName", nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].String() != "TestPerson" {
		t.Errorf("expected TestPerson, got %s", results[0].String())
	}

	DynamicMethodCaller(person, "SetAge", []interface{}{30})
	if person.Age != 30 {
		t.Errorf("expected age 30, got %d", person.Age)
	}
}

func TestReflectMethodByName(t *testing.T) {
	person := &Person{Name: "ReflectTest", Age: 40}

	val := reflect.ValueOf(person)
	method := val.MethodByName("GetName")
	if !method.IsValid() {
		t.Fatal("GetName method should be valid")
	}

	results := method.Call(nil)
	if len(results) != 1 || results[0].String() != "ReflectTest" {
		t.Error("unexpected result from reflected method call")
	}
}

func TestKubeConfigValidation(t *testing.T) {
	config := &KubeConfig{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata:   Metadata{Name: "test-config"},
	}

	val := reflect.ValueOf(config)
	method := val.MethodByName("Validate")
	if !method.IsValid() {
		t.Fatal("Validate method should be valid")
	}

	results := method.Call(nil)
	if len(results) != 1 {
		t.Fatal("expected 1 result from Validate")
	}

	if !results[0].IsNil() {
		t.Errorf("expected nil error, got %v", results[0].Interface())
	}
}

func TestCryptoOperations(t *testing.T) {
	t.Run("MD5", func(t *testing.T) {
		sum := md5.Sum([]byte("test data"))
		if len(sum) != 16 {
			t.Error("md5 sum should be 16 bytes")
		}
	})

	t.Run("SHA256", func(t *testing.T) {
		sum := sha256.Sum256([]byte("test data"))
		if len(sum) != 32 {
			t.Error("sha256 sum should be 32 bytes")
		}
	})

	t.Run("AES", func(t *testing.T) {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		cipher, err := aes.NewCipher(key)
		if err != nil {
			t.Fatalf("failed to create cipher: %v", err)
		}

		if cipher.BlockSize() != 16 {
			t.Error("AES block size should be 16")
		}
	})
}
