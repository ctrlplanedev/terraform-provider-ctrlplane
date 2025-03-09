// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package integration_test

import (
	"context"
	"net/http"
	"os"
	"terraform-provider-ctrlplane/client"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

type TestMode string

const (
	TestModePersist TestMode = "persistent"
	// TestModeCleanup TestMode = "cleanup".
	TestModeAutoCleanup TestMode = "autocleanup"
)

var Logger *zap.Logger

func GetTestMode() TestMode {
	mode := TestMode(os.Getenv("INTEGRATION_TEST_MODE"))
	switch mode {
	case TestModePersist, TestModeAutoCleanup:
		return mode
	default:
		return TestModeAutoCleanup
	}
}

func ShouldCleanup() bool {
	mode := GetTestMode()
	return mode == TestModeAutoCleanup
}

func SkipIfCredentialsMissing() (string, string, string) {
	apiKey := os.Getenv("CTRLPLANE_TOKEN")
	if apiKey == "" {
		Skip("CTRLPLANE_TOKEN environment variable not set")
	}

	workspaceStr := os.Getenv("CTRLPLANE_WORKSPACE")
	if workspaceStr == "" {
		Skip("CTRLPLANE_WORKSPACE environment variable not set")
	}

	apiHost := os.Getenv("CTRLPLANE_BASE_URL")
	if apiHost == "" {
		Skip("CTRLPLANE_BASE_URL environment variable not set")
	}

	return apiKey, apiHost, workspaceStr
}

func init() {
	var err error
	Logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	Logger.Info("starting integration test suite",
		zap.String("mode", string(GetTestMode())))

	if GetTestMode() == TestModeAutoCleanup {
		apiKey, apiHost, _ := SkipIfCredentialsMissing()
		apiClient, err := client.NewClientWithResponses(
			apiHost,
			client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
				req.Header.Add("x-api-key", apiKey)
				return nil
			}),
		)
		if err != nil {
			Logger.Error("failed to create API client for cleanup", zap.Error(err))
			return
		}

		ctx := context.Background()
		err = CleanupPreviousTestResources(ctx, apiClient)
		if err != nil {
			Logger.Error("failed to clean up previous test resources", zap.Error(err))
		}
	}
})

var _ = AfterSuite(func() {
	_ = Logger.Sync()
})
