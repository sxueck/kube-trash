package config

import (
	"os"
	"testing"
)

func TestArgsEnv_WithValidStructPointer_SetsValuesFromEnvironment(t *testing.T) {
	os.Setenv("API_SERVER", "https://example.com")
	os.Setenv("KUBE_CONFIG", "kubeconfig")

	c := &Cfg{}
	ArgsEnv(c)

	if c.APIServer != "https://example.com" {
		t.Errorf("Expected APIServer to be 'https://example.com', got '%s'", c.APIServer)
	}
	if c.KubeConfig != "kubeconfig" {
		t.Errorf("Expected KubeConfig to be 'kubeconfig', got '%s'", c.KubeConfig)
	}

	// Cleanup
	os.Unsetenv("API_SERVER")
	os.Unsetenv("KUBE_CONFIG")
}

func TestArgsEnv_WithPointerToNonStruct_ReportsError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected function to panic when pointer to non-struct is passed")
		}
	}()

	var nonStructPointer *int
	ArgsEnv(nonStructPointer)
}

func TestArgsEnv_WithEnvVariablesAndDefaults_SetsValuesFromEnvironment(t *testing.T) {
	os.Setenv("API_SERVER", "")
	os.Setenv("KUBE_CONFIG", "")

	c := &Cfg{}
	ArgsEnv(c)

	if c.APIServer != "" {
		t.Errorf("Expected APIServer to be '', got '%s'", c.APIServer)
	}
	if c.KubeConfig != "" {
		t.Errorf("Expected KubeConfig to be '', got '%s'", c.KubeConfig)
	}

	// Cleanup
	os.Unsetenv("API_SERVER")
	os.Unsetenv("KUBE_CONFIG")
}
