// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSetDeploymentJobAgentBlocksFromConfig_WithType(t *testing.T) {
	config := map[string]interface{}{
		"template":           "my-template",
		"triggerRunOnChange": true,
	}

	var model DeploymentJobAgentModel
	setDeploymentJobAgentBlocksFromConfig(&model, config, "terraform_cloud")

	if model.TerraformCloud == nil {
		t.Fatal("expected TerraformCloud block to be set")
	}
	if model.TerraformCloud.Template.ValueString() != "my-template" {
		t.Errorf("expected template 'my-template', got %q", model.TerraformCloud.Template.ValueString())
	}
	if model.TerraformCloud.TriggerRunOnChange.ValueBool() != true {
		t.Errorf("expected trigger_run_on_change true, got %v", model.TerraformCloud.TriggerRunOnChange.ValueBool())
	}
	if model.ArgoCD != nil {
		t.Error("expected ArgoCD to be nil")
	}
}

func TestDeploymentJobAgentBlockType(t *testing.T) {
	tests := []struct {
		name     string
		model    DeploymentJobAgentModel
		expected string
	}{
		{
			name:     "argocd",
			model:    DeploymentJobAgentModel{ArgoCD: &DeploymentJobAgentArgoCDModel{}},
			expected: "argocd",
		},
		{
			name:     "github",
			model:    DeploymentJobAgentModel{GitHub: &DeploymentJobAgentGitHubModel{}},
			expected: "github",
		},
		{
			name:     "terraform_cloud",
			model:    DeploymentJobAgentModel{TerraformCloud: &DeploymentJobAgentTFCModel{}},
			expected: "terraform_cloud",
		},
		{
			name:     "test_runner",
			model:    DeploymentJobAgentModel{TestRunner: &DeploymentJobAgentTestRunnerModel{}},
			expected: "test_runner",
		},
		{
			name:     "empty",
			model:    DeploymentJobAgentModel{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deploymentJobAgentBlockType(tt.model)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestDeploymentJobAgentModelsFromAPI_PreservesToken(t *testing.T) {
	// The API never returns sensitive token values. Verify that the token
	// from prior state is preserved after reading back from the API.
	apiAgents := []api.DeploymentJobAgent{
		{
			Ref:    "agent-1",
			Config: api.JobAgentConfig{"address": "https://tfe.example.com", "organization": "my-org"},
		},
	}

	priorAgents := []DeploymentJobAgentModel{
		{
			Id: types.StringValue("agent-1"),
			TerraformCloud: &DeploymentJobAgentTFCModel{
				Address:      types.StringValue("https://tfe.example.com"),
				Organization: types.StringValue("my-org"),
				Token:        types.StringValue("secret-token"),
			},
		},
	}

	result := deploymentJobAgentModelsFromAPI(apiAgents, priorAgents)

	if len(result) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(result))
	}
	if result[0].TerraformCloud == nil {
		t.Fatal("expected TerraformCloud block to be set")
	}
	if result[0].TerraformCloud.Token.ValueString() != "secret-token" {
		t.Errorf("expected token to be preserved as 'secret-token', got %q", result[0].TerraformCloud.Token.ValueString())
	}
}

func TestDeploymentJobAgentModelsFromAPI_NullTokenStaysNull(t *testing.T) {
	// When prior state has no token (null), read should also produce null.
	apiAgents := []api.DeploymentJobAgent{
		{
			Ref:    "agent-1",
			Config: api.JobAgentConfig{"address": "https://tfe.example.com"},
		},
	}

	priorAgents := []DeploymentJobAgentModel{
		{
			Id: types.StringValue("agent-1"),
			TerraformCloud: &DeploymentJobAgentTFCModel{
				Address: types.StringValue("https://tfe.example.com"),
				Token:   types.StringNull(),
			},
		},
	}

	result := deploymentJobAgentModelsFromAPI(apiAgents, priorAgents)

	if len(result) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(result))
	}
	if result[0].TerraformCloud == nil {
		t.Fatal("expected TerraformCloud block to be set")
	}
	if !result[0].TerraformCloud.Token.IsNull() {
		t.Errorf("expected token to remain null, got %q", result[0].TerraformCloud.Token.ValueString())
	}
}

// TestImportThenApplyReadCycle simulates the full terraform import lifecycle:
//
// Step 1 (import): No prior state → agentType is "" → blocks stay nil.
// Step 2 (apply):  User's HCL config writes the block to state.
// Step 3 (read):   Prior state now has the block → agentType is derived → config populates correctly.
//
// This verifies that even though import produces empty blocks, the subsequent
// apply+read cycle recovers the correct state.
func TestImportThenApplyReadCycle(t *testing.T) {
	apiConfig := map[string]interface{}{
		"template":           "my-template",
		"triggerRunOnChange": true,
	}

	// Step 1: Import — no prior state, agentType is ""
	var afterImport DeploymentJobAgentModel
	setDeploymentJobAgentBlocksFromConfig(&afterImport, apiConfig, "")

	if afterImport.TerraformCloud != nil {
		t.Fatal("step 1 (import): expected TerraformCloud to be nil with empty agentType")
	}

	// Step 2: Apply — user's HCL config writes terraform_cloud block to state.
	// (Simulated: the plan uses HCL config, not the API read.)
	afterApply := DeploymentJobAgentModel{
		Id: types.StringValue("agent-123"),
		TerraformCloud: &DeploymentJobAgentTFCModel{
			Template:           types.StringValue("my-template"),
			TriggerRunOnChange: types.BoolValue(true),
		},
	}

	// Step 3: Next read — prior state (afterApply) provides the block type.
	blockType := deploymentJobAgentBlockType(afterApply)
	if blockType != "terraform_cloud" {
		t.Fatalf("step 3 (read): expected block type 'terraform_cloud', got %q", blockType)
	}

	var afterRead DeploymentJobAgentModel
	setDeploymentJobAgentBlocksFromConfig(&afterRead, apiConfig, blockType)

	if afterRead.TerraformCloud == nil {
		t.Fatal("step 3 (read): expected TerraformCloud to be populated")
	}
	if afterRead.TerraformCloud.Template.ValueString() != "my-template" {
		t.Errorf("step 3 (read): expected template 'my-template', got %q", afterRead.TerraformCloud.Template.ValueString())
	}
	if afterRead.TerraformCloud.TriggerRunOnChange.ValueBool() != true {
		t.Errorf("step 3 (read): expected trigger_run_on_change true, got %v", afterRead.TerraformCloud.TriggerRunOnChange.ValueBool())
	}
	if afterRead.ArgoCD != nil {
		t.Error("step 3 (read): ArgoCD should be nil — template must not match argocd")
	}
}
