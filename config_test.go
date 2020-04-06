package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestConfigYAML(t *testing.T) {
	f, err := os.Open("./appendix/example.config.yaml")
	if err != nil {
		t.Fatalf("failed to open test config yaml: %v", err)
	}
	defer f.Close()
	c := Config{}
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("config validate failed: %v", err)
	}
}
