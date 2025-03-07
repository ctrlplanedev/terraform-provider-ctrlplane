// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package integration_test

import (
	"context"
	"fmt"
	"terraform-provider-ctrlplane/client"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Environment API", func() {
	var (
		apiClient *client.ClientWithResponses
		ctx       context.Context
	)

	BeforeEach(func() {
		apiKey, apiHost, _ := SkipIfCredentialsMissing()

		var err error
		apiClient, err = newTestClient(apiKey, apiHost)
		Expect(err).NotTo(HaveOccurred(), "Failed to create client")

		ctx = context.Background()
		Logger.Info("running environment tests")
	})

	AfterEach(func() {
		_ = Logger.Sync()
	})

	Context("when creating a new environment", func() {
		var (
			systemID uuid.UUID
			envName  string
		)

		BeforeEach(func() {
			var err error
			systemID, _, err = createTestSystem(ctx, apiClient, "env-no-filter")
			Expect(err).NotTo(HaveOccurred(), "Failed to create test system")

			shortUUID := uuid.New().String()[:6]
			envName = fmt.Sprintf("env-test-%s", shortUUID)
		})

		AfterEach(func() {
			err := deleteTestSystem(ctx, apiClient, systemID)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete test system")
		})

		It("should create an environment without a resource filter", func() {
			releaseChannels := []string{}
			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment without resource filter"
			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  nil,
				ReleaseChannels: &releaseChannels,
			})

			Expect(err).NotTo(HaveOccurred(), "Failed to create environment")
			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			Expect(envResp.JSON200).NotTo(BeNil(), "Environment creation response is nil")
			Expect(envResp.JSON200.Id).NotTo(Equal(uuid.Nil), "Environment ID is nil")
			Expect(envResp.JSON200.Name).To(Equal(envName), "Environment name does not match")
			Expect(envResp.JSON200.SystemId).To(Equal(systemID), "System ID does not match")
			Expect(envResp.JSON200.Description).To(Equal(&description), "Description does not match")
			Expect(envResp.JSON200.Metadata).To(Equal(&metadata), "Metadata does not match")
			Expect(envResp.JSON200.ResourceFilter).To(BeNil(), "Resource filter should be nil")

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred(), "Failed to get system")
			Logger.Debug("system response",
				zap.Int("status_code", systemResp.StatusCode()),
				zap.String("response_body", string(systemResp.Body)))
			Expect(systemResp.JSON200).NotTo(BeNil(), "System response is nil")
			Expect(systemResp.JSON200.Environments).NotTo(BeNil(), "System environments is nil")

			var found bool
			for _, env := range *systemResp.JSON200.Environments {
				if env.Id == envResp.JSON200.Id {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created environment not found in system's environments list")

			Logger.Debug("deleting environment",
				zap.String("id", envResp.JSON200.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, envResp.JSON200.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with empty resource filter", func() {
			filter := map[string]interface{}{
				"type":     "kind",
				"operator": "equals",
				"value":    "Deployment",
			}

			shortUUID := uuid.New().String()[:6]
			envName = fmt.Sprintf("env-empty-filter-%s", shortUUID)

			releaseChannels := []string{}
			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with resource filter"
			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			})

			Expect(err).NotTo(HaveOccurred(), "Failed to create environment")
			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			Expect(envResp.JSON200).NotTo(BeNil(), "Environment creation response is nil")
			Expect(envResp.JSON200.Id).NotTo(Equal(uuid.Nil), "Environment ID is nil")
			Expect(envResp.JSON200.Name).To(Equal(envName), "Environment name does not match")
			Expect(envResp.JSON200.SystemId).To(Equal(systemID), "System ID does not match")
			Expect(envResp.JSON200.Description).To(Equal(&description), "Description does not match")
			Expect(envResp.JSON200.Metadata).To(Equal(&metadata), "Metadata does not match")
			Expect(envResp.JSON200.ResourceFilter).NotTo(BeNil(), "Resource filter should not be nil")

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred(), "Failed to get system")
			Logger.Debug("system response",
				zap.Int("status_code", systemResp.StatusCode()),
				zap.String("response_body", string(systemResp.Body)))
			Expect(systemResp.JSON200).NotTo(BeNil(), "System response is nil")
			Expect(systemResp.JSON200.Environments).NotTo(BeNil(), "System environments is nil")

			var found bool
			for _, env := range *systemResp.JSON200.Environments {
				if env.Id == envResp.JSON200.Id {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created environment not found in system's environments list")

			Logger.Debug("deleting environment",
				zap.String("id", envResp.JSON200.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, envResp.JSON200.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with comparison resource filter", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-comparison")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-comparison-%s", uuid.New().String()[:6])
			filter := map[string]interface{}{
				"type":     "kind",
				"operator": "equals",
				"value":    "Deployment",
			}
			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with comparison filter"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with comparison filter",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.Name).To(Equal(envName))
			Expect(env.SystemId.String()).To(Equal(systemID.String()))
			Expect(env.Description).To(Equal(&description), "Description does not match")
			Expect(env.Metadata).To(Equal(&metadata), "Metadata does not match")
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			Expect(rf["type"]).To(Equal("kind"))
			Expect(rf["operator"]).To(Equal("equals"))
			Expect(rf["value"]).To(Equal("Deployment"))

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred(), "Failed to get system")
			Logger.Debug("system response",
				zap.Int("status_code", systemResp.StatusCode()),
				zap.String("response_body", string(systemResp.Body)))
			Expect(systemResp.JSON200).NotTo(BeNil(), "System response is nil")
			Expect(systemResp.JSON200.Environments).NotTo(BeNil(), "System environments is nil")

			var found bool
			for _, env := range *systemResp.JSON200.Environments {
				if env.Id == envResp.JSON200.Id {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created environment not found in system's environments list")

			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with metadata resource filter", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-metadata")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-metadata-%s", uuid.New().String()[:6])
			filter := map[string]interface{}{
				"type":     "metadata",
				"key":      "environment",
				"operator": "equals",
				"value":    "production",
			}
			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with metadata filter"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with metadata filter",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.Name).To(Equal(envName))
			Expect(env.SystemId.String()).To(Equal(systemID.String()))
			Expect(env.Description).To(Equal(&description), "Description does not match")
			Expect(env.Metadata).To(Equal(&metadata), "Metadata does not match")
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			Expect(rf["type"]).To(Equal("metadata"))
			Expect(rf["key"]).To(Equal("environment"))
			Expect(rf["operator"]).To(Equal("equals"))
			Expect(rf["value"]).To(Equal("production"))

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred(), "Failed to get system")
			Logger.Debug("system response",
				zap.Int("status_code", systemResp.StatusCode()),
				zap.String("response_body", string(systemResp.Body)))
			Expect(systemResp.JSON200).NotTo(BeNil(), "System response is nil")
			Expect(systemResp.JSON200.Environments).NotTo(BeNil(), "System environments is nil")

			var found bool
			for _, env := range *systemResp.JSON200.Environments {
				if env.Id == envResp.JSON200.Id {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created environment not found in system's environments list")

			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with a simple filter", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-simple-filter")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-simple-filter-%s", uuid.New().String()[:6])
			filter := map[string]interface{}{
				"type":     "metadata",
				"key":      "environment",
				"operator": "equals",
				"value":    "staging",
			}
			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with simple filter"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with simple filter",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.Name).To(Equal(envName))
			Expect(env.SystemId.String()).To(Equal(systemID.String()))
			Expect(env.Description).To(Equal(&description))
			Expect(env.Metadata).To(Equal(&metadata))
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			Expect(rf["type"]).To(Equal("metadata"))
			Expect(rf["key"]).To(Equal("environment"))
			Expect(rf["operator"]).To(Equal("equals"))
			Expect(rf["value"]).To(Equal("staging"))

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred())
			Expect(systemResp.JSON200).NotTo(BeNil())
			Expect(systemResp.JSON200.Environments).NotTo(BeNil())

			var found bool
			for _, env := range *systemResp.JSON200.Environments {
				if env.Id == envResp.JSON200.Id {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created environment not found in system's environments list")

			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create an environment with a complex filter", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-complex-filter")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-complex-filter-%s", uuid.New().String()[:6])
			filter := map[string]interface{}{
				"not":      false,
				"type":     "comparison",
				"operator": "and",
				"conditions": []map[string]interface{}{
					{
						"type":     "metadata",
						"key":      "environment",
						"operator": "equals",
						"value":    "staging",
					},
					{
						"type":     "kind",
						"operator": "equals",
						"value":    "Deployment",
					},
				},
			}
			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with complex filter"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with complex filter",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.Name).To(Equal(envName))
			Expect(env.SystemId.String()).To(Equal(systemID.String()))
			Expect(env.Description).To(Equal(&description))
			Expect(env.Metadata).To(Equal(&metadata))
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			Expect(rf["type"]).To(Equal("comparison"))
			Expect(rf["operator"]).To(Equal("and"))
			Expect(rf["not"]).To(Equal(false))

			conditions, ok := rf["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "conditions should be an array")
			Expect(conditions).To(HaveLen(2), "should have 2 conditions")

			condition1 := conditions[0].(map[string]interface{})
			Expect(condition1["type"]).To(Equal("metadata"))
			Expect(condition1["key"]).To(Equal("environment"))
			Expect(condition1["operator"]).To(Equal("equals"))
			Expect(condition1["value"]).To(Equal("staging"))

			condition2 := conditions[1].(map[string]interface{})
			Expect(condition2["type"]).To(Equal("kind"))
			Expect(condition2["operator"]).To(Equal("equals"))
			Expect(condition2["value"]).To(Equal("Deployment"))

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred())
			Expect(systemResp.JSON200).NotTo(BeNil())
			Expect(systemResp.JSON200.Environments).NotTo(BeNil())

			var found bool
			for _, env := range *systemResp.JSON200.Environments {
				if env.Id == envResp.JSON200.Id {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Created environment not found in system's environments list")

			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with date condition resource filter", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-date-filter")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-date-filter-%s", uuid.New().String()[:6])

			// Create a resource filter with a date condition
			filter := map[string]interface{}{
				"type":     "comparison",
				"operator": "and",
				"not":      true,
				"conditions": []map[string]interface{}{
					{
						"type":     "name",
						"operator": "equals",
						"value":    "web",
					},
					{
						"type":     "kind",
						"operator": "equals",
						"value":    "Deployment",
					},
					{
						"type":     "created-at",
						"operator": "before",
						"value":    "2025-03-06T22:34:11.070Z",
					},
				},
			}

			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with date condition filter"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with date condition filter",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.Name).To(Equal(envName))
			Expect(env.SystemId.String()).To(Equal(systemID.String()))
			Expect(env.Description).To(Equal(&description), "Description does not match")
			Expect(env.Metadata).To(Equal(&metadata), "Metadata does not match")
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			Expect(rf["type"]).To(Equal("comparison"))
			Expect(rf["operator"]).To(Equal("and"))
			Expect(rf["not"]).To(Equal(true))

			conditions, ok := rf["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Conditions should be an array")
			Expect(len(conditions)).To(Equal(3), "Expected 3 conditions")

			// Verify that we have a date condition
			var hasDateCondition bool
			for _, c := range conditions {
				condition, ok := c.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Condition should be a map")

				if condType, ok := condition["type"].(string); ok && condType == "created-at" {
					hasDateCondition = true
					Expect(condition["operator"]).To(Equal("before"))
					Expect(condition["value"]).To(Equal("2025-03-06T22:34:11.070Z"))
					// Notably, there should be no "conditions" field for leaf conditions
					_, hasConditions := condition["conditions"]
					Expect(hasConditions).To(BeFalse(), "Leaf condition should not have a conditions field")
				}
			}
			Expect(hasDateCondition).To(BeTrue(), "Resource filter should have a date condition")

			// Get the environment and verify the filter
			getEnvResp, err := apiClient.GetEnvironmentWithResponse(ctx, env.Id.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(getEnvResp.StatusCode()).To(Equal(200))

			getEnv := getEnvResp.JSON200
			Expect(getEnv).NotTo(BeNil())
			Expect(getEnv.ResourceFilter).NotTo(BeNil())

			getRf := *getEnv.ResourceFilter
			Expect(getRf["type"]).To(Equal("comparison"))
			Expect(getRf["operator"]).To(Equal("and"))
			Expect(getRf["not"]).To(Equal(true))

			getConditions, ok := getRf["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Conditions should be an array")
			Expect(len(getConditions)).To(Equal(3), "Expected 3 conditions")

			// Cleanup
			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should support multiple environments in the same system", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "multi-env")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			environments := []struct {
				name   string
				filter map[string]interface{}
			}{
				{
					name: fmt.Sprintf("env-prod-%s", uuid.New().String()[:6]),
					filter: map[string]interface{}{
						"type":     "metadata",
						"key":      "environment",
						"operator": "equals",
						"value":    "production",
					},
				},
				{
					name: fmt.Sprintf("env-staging-%s", uuid.New().String()[:6]),
					filter: map[string]interface{}{
						"type":     "metadata",
						"key":      "environment",
						"operator": "equals",
						"value":    "staging",
					},
				},
				{
					name: fmt.Sprintf("env-deployments-%s", uuid.New().String()[:6]),
					filter: map[string]interface{}{
						"type":     "kind",
						"operator": "equals",
						"value":    "Deployment",
					},
				},
			}

			createdEnvs := make([]uuid.UUID, 0, len(environments))
			releaseChannels := []string{}

			for _, env := range environments {
				description := fmt.Sprintf("Test environment for %s", env.name)
				metadata := map[string]string{
					"test": "true",
					"env":  "integration",
				}

				envReq := client.CreateEnvironmentJSONRequestBody{
					Name:            env.name,
					Description:     &description,
					SystemId:        systemID.String(),
					PolicyId:        nil,
					Metadata:        &metadata,
					ResourceFilter:  &env.filter,
					ReleaseChannels: &releaseChannels,
				}

				Logger.Debug("creating environment",
					zap.String("name", env.name),
					zap.Any("filter", env.filter))

				envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
				Expect(err).NotTo(HaveOccurred())
				Expect(envResp.StatusCode()).To(Equal(200))
				Expect(envResp.JSON200).NotTo(BeNil())
				createdEnvs = append(createdEnvs, envResp.JSON200.Id)
			}

			systemResp, err := apiClient.GetSystemWithResponse(ctx, systemID)
			Expect(err).NotTo(HaveOccurred())
			Expect(systemResp.JSON200).NotTo(BeNil())
			Expect(systemResp.JSON200.Environments).NotTo(BeNil())
			Expect(*systemResp.JSON200.Environments).To(HaveLen(len(environments)))

			for _, envID := range createdEnvs {
				err = deleteTestEnvironment(ctx, apiClient, envID)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should create an environment with nested comparison conditions", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-nested-comparison")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-nested-comp-%s", uuid.New().String()[:6])

			// Create a resource filter with nested comparison conditions
			filter := map[string]interface{}{
				"type":     "comparison",
				"operator": "and",
				"not":      true,
				"conditions": []map[string]interface{}{
					{
						"type":     "name",
						"operator": "equals",
						"value":    "api",
					},
					{
						"type":     "comparison",
						"operator": "or",
						"not":      false,
						"conditions": []map[string]interface{}{
							{
								"type":     "kind",
								"operator": "equals",
								"value":    "Deployment",
							},
							{
								"type":     "kind",
								"operator": "equals",
								"value":    "StatefulSet",
							},
						},
					},
				},
			}

			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with nested comparison conditions"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with nested comparison conditions",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			Expect(rf["type"]).To(Equal("comparison"))
			Expect(rf["operator"]).To(Equal("and"))
			Expect(rf["not"]).To(Equal(true))

			conditions, ok := rf["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Conditions should be an array")
			Expect(len(conditions)).To(Equal(2), "Expected 2 conditions")

			// Verify we have a nested comparison condition
			var hasNestedComparison bool
			for _, c := range conditions {
				condition, ok := c.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Condition should be a map")

				if condType, ok := condition["type"].(string); ok && condType == "comparison" {
					hasNestedComparison = true
					Expect(condition["operator"]).To(Equal("or"))
					Expect(condition["not"]).To(Equal(false))

					nestedConditions, hasNested := condition["conditions"].([]interface{})
					Expect(hasNested).To(BeTrue(), "Comparison condition should have nested conditions")
					Expect(len(nestedConditions)).To(Equal(2), "Expected 2 nested conditions")
				}
			}
			Expect(hasNestedComparison).To(BeTrue(), "Resource filter should have a nested comparison condition")

			// Cleanup
			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with mixed condition types", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-mixed-conditions")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-mixed-conditions-%s", uuid.New().String()[:6])

			// Create a resource filter with mixed condition types
			filter := map[string]interface{}{
				"type":     "comparison",
				"operator": "and",
				"not":      false,
				"conditions": []map[string]interface{}{
					{
						"type":     "name",
						"operator": "equals",
						"value":    "api",
					},
					{
						"type":     "kind",
						"operator": "equals",
						"value":    "Deployment",
					},
					{
						"type":     "metadata",
						"key":      "environment",
						"operator": "equals",
						"value":    "production",
					},
				},
			}

			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with mixed condition types"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with mixed condition types",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.ResourceFilter).NotTo(BeNil())

			rf := *env.ResourceFilter
			conditions, ok := rf["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Conditions should be an array")
			Expect(len(conditions)).To(Equal(3), "Expected 3 conditions")

			// Verify we have each condition type
			var hasName, hasKind, hasMetadata bool
			for _, c := range conditions {
				condition, ok := c.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Condition should be a map")

				condType, ok := condition["type"].(string)
				Expect(ok).To(BeTrue(), "Condition should have a type")

				switch condType {
				case "name":
					hasName = true
					Expect(condition["value"]).To(Equal("api"))
				case "kind":
					hasKind = true
					Expect(condition["value"]).To(Equal("Deployment"))
				case "metadata":
					hasMetadata = true
					Expect(condition["key"]).To(Equal("environment"))
					Expect(condition["value"]).To(Equal("production"))
				}

				// Verify no conditions array on leaf nodes
				_, hasConditions := condition["conditions"]
				Expect(hasConditions).To(BeFalse(), "Leaf condition should not have a conditions field")
			}

			Expect(hasName).To(BeTrue(), "Resource filter should have a name condition")
			Expect(hasKind).To(BeTrue(), "Resource filter should have a kind condition")
			Expect(hasMetadata).To(BeTrue(), "Resource filter should have a metadata condition")

			// Cleanup
			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})

		It("should create an environment with deeply nested conditions", func() {
			systemID, _, err := createTestSystem(ctx, apiClient, "env-deep-nesting")
			Expect(err).NotTo(HaveOccurred())
			defer deleteTestSystem(ctx, apiClient, systemID)

			releaseChannels := []string{}
			envName := fmt.Sprintf("env-deep-nesting-%s", uuid.New().String()[:6])

			// Create a resource filter with 3 levels of nesting
			filter := map[string]interface{}{
				"type":     "comparison",
				"operator": "and",
				"not":      false,
				"conditions": []map[string]interface{}{
					{
						"type":     "name",
						"operator": "equals",
						"value":    "api",
					},
					{
						"type":     "comparison",
						"operator": "or",
						"conditions": []map[string]interface{}{
							{
								"type":     "kind",
								"operator": "equals",
								"value":    "Deployment",
							},
							{
								"type":     "comparison",
								"operator": "and",
								"conditions": []map[string]interface{}{
									{
										"type":     "metadata",
										"key":      "tier",
										"operator": "equals",
										"value":    "data",
									},
									{
										"type":     "kind",
										"operator": "equals",
										"value":    "StatefulSet",
									},
								},
							},
						},
					},
				},
			}

			metadata := map[string]string{
				"test": "true",
				"env":  "integration",
			}
			description := "Test environment with deeply nested conditions"
			envReq := client.CreateEnvironmentJSONRequestBody{
				Name:            envName,
				Description:     &description,
				SystemId:        systemID.String(),
				PolicyId:        nil,
				Metadata:        &metadata,
				ResourceFilter:  &filter,
				ReleaseChannels: &releaseChannels,
			}

			Logger.Debug("creating environment with deeply nested conditions",
				zap.String("name", envName),
				zap.Any("filter", envReq.ResourceFilter))

			envResp, err := apiClient.CreateEnvironmentWithResponse(ctx, envReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(envResp.StatusCode()).To(Equal(200))

			Logger.Debug("environment creation response",
				zap.Int("status_code", envResp.StatusCode()),
				zap.String("response_body", string(envResp.Body)))

			env := envResp.JSON200
			Expect(env).NotTo(BeNil())
			Expect(env.ResourceFilter).NotTo(BeNil())

			// Get the environment and verify
			getEnvResp, err := apiClient.GetEnvironmentWithResponse(ctx, env.Id.String())
			Expect(err).NotTo(HaveOccurred())
			Expect(getEnvResp.StatusCode()).To(Equal(200))

			getEnv := getEnvResp.JSON200
			Expect(getEnv).NotTo(BeNil())
			Expect(getEnv.ResourceFilter).NotTo(BeNil())

			// Verify the structure is preserved
			getRf := *getEnv.ResourceFilter
			Expect(getRf["type"]).To(Equal("comparison"))
			Expect(getRf["operator"]).To(Equal("and"))

			topConditions, ok := getRf["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Top conditions should be an array")
			Expect(len(topConditions)).To(Equal(2), "Expected 2 top level conditions")

			// Find the nested comparison
			var level2Comparison map[string]interface{}
			for _, c := range topConditions {
				condition, ok := c.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Condition should be a map")

				if condType, ok := condition["type"].(string); ok && condType == "comparison" {
					level2Comparison = condition
					break
				}
			}

			Expect(level2Comparison).NotTo(BeNil(), "Should have a nested comparison")
			Expect(level2Comparison["operator"]).To(Equal("or"))

			level2Conditions, ok := level2Comparison["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Level 2 conditions should be an array")
			Expect(len(level2Conditions)).To(Equal(2), "Expected 2 conditions at level 2")

			// Find the level 3 comparison
			var level3Comparison map[string]interface{}
			for _, c := range level2Conditions {
				condition, ok := c.(map[string]interface{})
				Expect(ok).To(BeTrue(), "Condition should be a map")

				if condType, ok := condition["type"].(string); ok && condType == "comparison" {
					level3Comparison = condition
					break
				}
			}

			Expect(level3Comparison).NotTo(BeNil(), "Should have a level 3 comparison")
			Expect(level3Comparison["operator"]).To(Equal("and"))

			level3Conditions, ok := level3Comparison["conditions"].([]interface{})
			Expect(ok).To(BeTrue(), "Level 3 conditions should be an array")
			Expect(len(level3Conditions)).To(Equal(2), "Expected 2 conditions at level 3")

			// Cleanup
			Logger.Debug("deleting environment", zap.String("id", env.Id.String()))
			err = deleteTestEnvironment(ctx, apiClient, env.Id)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete environment")
		})
	})
})
