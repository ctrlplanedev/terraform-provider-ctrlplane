// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acctest

import (
	"fmt"
	"os"
	"testing"
)

// Environment variables for test configuration.
const (
	APIKeyEnvVar    = "CTRLPLANE_TOKEN"
	WorkspaceEnvVar = "CTRLPLANE_WORKSPACE"
	BaseURLEnvVar   = "CTRLPLANE_BASE_URL"
)

// GetTestEnv gets a test environment variable and logs its value.
func GetTestEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	t.Logf("Environment variable %s = %s", key, value)
	return value
}

// ProviderConfig returns provider configuration for acceptance tests.
func ProviderConfig() string {
	baseURL := os.Getenv(BaseURLEnvVar)
	if baseURL == "" {
		baseURL = "https://app.ctrlplane.com" // Default value from provider schema
	}

	return fmt.Sprintf(`
terraform {
  required_providers {
    ctrlplane = {
      source = "hashicorp/ctrlplane"
      version = "0.0.1"
    }
  }
}

provider "ctrlplane" {
  token     = %q
  workspace = %q
  base_url  = %q
}
`, os.Getenv(APIKeyEnvVar), os.Getenv(WorkspaceEnvVar), baseURL)
}

// PreCheck validates the necessary test env vars exist in running acceptance tests.
func PreCheck(t *testing.T) {
	if v := os.Getenv(APIKeyEnvVar); v == "" {
		t.Fatalf("%s must be set for acceptance tests", APIKeyEnvVar)
	}
	if v := os.Getenv(WorkspaceEnvVar); v == "" {
		t.Fatalf("%s must be set for acceptance tests", WorkspaceEnvVar)
	}
}
