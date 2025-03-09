// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func addAPIKey(apiKey string) client.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("x-api-key", apiKey)
		return nil
	}
}

func getWorkspaceID(ctx context.Context, workspaceStr string, apiClient *client.ClientWithResponses) (uuid.UUID, error) {
	Logger.Debug("getting workspace ID",
		zap.String("workspace", workspaceStr))

	workspaceID, err := uuid.Parse(workspaceStr)
	if err == nil {
		Logger.Debug("using workspace ID",
			zap.String("id", workspaceID.String()))
		return workspaceID, nil
	}

	Logger.Debug("workspace value is not a UUID, trying as slug",
		zap.String("value", workspaceStr))

	workspaceResp, err := apiClient.GetWorkspaceBySlugWithResponse(ctx, workspaceStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get workspace by slug: %w", err)
	}

	if workspaceResp.JSON200 == nil {
		return uuid.Nil, fmt.Errorf("workspace not found: %s", workspaceStr)
	}

	Logger.Debug("found workspace ID",
		zap.String("id", workspaceResp.JSON200.Id.String()))
	return workspaceResp.JSON200.Id, nil
}

func createTestSystem(ctx context.Context, apiClient *client.ClientWithResponses, namePrefix string) (uuid.UUID, error) {
	workspaceStr := os.Getenv("CTRLPLANE_WORKSPACE")
	workspaceID, err := getWorkspaceID(ctx, workspaceStr, apiClient)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get workspace ID: %w", err)
	}

	shortUUID := uuid.New().String()[:6]
	systemName := fmt.Sprintf("test-%s-%s", namePrefix, shortUUID)
	if len(systemName) > 30 {
		systemName = systemName[:30]
	}
	systemSlug := strings.ToLower(strings.ReplaceAll(systemName, "-", ""))

	Logger.Debug("creating system",
		zap.String("name", systemName),
		zap.String("slug", systemSlug),
		zap.String("workspace_id", workspaceID.String()))

	systemResp, err := apiClient.CreateSystemWithResponse(ctx, client.CreateSystemJSONRequestBody{
		Name:        systemName,
		Description: nil,
		Slug:        systemSlug,
		WorkspaceId: workspaceID,
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create system: %w", err)
	}

	if systemResp.JSON201 == nil {

		Logger.Error("system creation failed",
			zap.Int("status_code", systemResp.StatusCode()),
			zap.String("response_body", string(systemResp.Body)))
		return uuid.Nil, fmt.Errorf("system creation failed with status: %d", systemResp.StatusCode())
	}

	return systemResp.JSON201.Id, nil
}

func deleteTestSystem(ctx context.Context, apiClient *client.ClientWithResponses, systemID uuid.UUID) error {
	if systemID == uuid.Nil {
		return nil
	}

	if !ShouldCleanup() {
		Logger.Info("skipping system cleanup due to test mode",
			zap.String("mode", string(GetTestMode())),
			zap.String("system_id", systemID.String()))
		return nil
	}

	Logger.Debug("deleting system",
		zap.String("system_id", systemID.String()))
	_, err := apiClient.DeleteSystemWithResponse(ctx, systemID.String())
	if err != nil {
		return fmt.Errorf("failed to delete system: %w", err)
	}

	return nil
}

func deleteTestEnvironment(ctx context.Context, apiClient *client.ClientWithResponses, environmentID uuid.UUID) error {
	if environmentID == uuid.Nil {
		return nil
	}

	if !ShouldCleanup() {
		Logger.Info("skipping environment cleanup due to test mode",
			zap.String("mode", string(GetTestMode())),
			zap.String("environment_id", environmentID.String()))
		return nil
	}

	Logger.Debug("deleting environment",
		zap.String("environment_id", environmentID.String()))
	_, err := apiClient.DeleteEnvironmentWithResponse(ctx, environmentID.String())
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	return nil
}

func newTestClient(apiKey, apiHost string) (*client.ClientWithResponses, error) {
	apiHost = strings.TrimSuffix(apiHost, "/")
	if !strings.HasSuffix(apiHost, "/api") {
		apiHost = apiHost + "/api"
	}

	Logger.Debug("creating test client", zap.String("host", apiHost))

	return client.NewClientWithResponses(
		apiHost,
		client.WithRequestEditorFn(addAPIKey(apiKey)),
	)
}

func CleanupPreviousTestResources(ctx context.Context, apiClient *client.ClientWithResponses) error {
	mode := GetTestMode()
	if mode == TestModePersist {
		Logger.Info("persist mode: skipping cleanup of test resources")
		return nil
	}

	if mode == TestModeAutoCleanup {
		Logger.Info("autocleanup mode: resources will be cleaned up after each test")
		return nil
	}

	Logger.Info("cleanup mode is no longer supported as there is no API endpoint to list systems",
		zap.String("mode", string(mode)))
	return nil
}
