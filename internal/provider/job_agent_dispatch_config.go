// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type JobAgentDispatchArgoCDModel struct {
	ApiKey    types.String `tfsdk:"api_key"`
	ServerUrl types.String `tfsdk:"server_url"`
	Template  types.String `tfsdk:"template"`
}

type JobAgentDispatchArgoWorkflowModel struct {
	ApiKey        types.String `tfsdk:"api_key"`
	WebhookSecret types.String `tfsdk:"webhook_secret"`
	ServerUrl     types.String `tfsdk:"server_url"`
	Template      types.String `tfsdk:"template"`
	Name          types.String `tfsdk:"name"`
	HttpInsecure  types.Bool   `tfsdk:"http_insecure"`
}

type JobAgentDispatchGitHubModel struct {
	InstallationId types.Int64  `tfsdk:"installation_id"`
	Owner          types.String `tfsdk:"owner"`
	Ref            types.String `tfsdk:"ref"`
	Repo           types.String `tfsdk:"repo"`
	WorkflowId     types.Int64  `tfsdk:"workflow_id"`
}

type JobAgentDispatchTFCModel struct {
	Address            types.String `tfsdk:"address"`
	Organization       types.String `tfsdk:"organization"`
	Template           types.String `tfsdk:"template"`
	Token              types.String `tfsdk:"token"`
	TriggerRunOnChange types.Bool   `tfsdk:"trigger_run_on_change"`
}

type JobAgentDispatchTestRunnerModel struct {
	DelaySeconds types.Int64  `tfsdk:"delay_seconds"`
	Message      types.String `tfsdk:"message"`
	Status       types.String `tfsdk:"status"`
}

// JobAgentDispatchBlocks groups the typed job-agent dispatch config blocks
// shared by deployment and workflow job_agent entries.
type JobAgentDispatchBlocks struct {
	ArgoCD         *JobAgentDispatchArgoCDModel
	ArgoWorkflow   *JobAgentDispatchArgoWorkflowModel
	GitHub         *JobAgentDispatchGitHubModel
	TerraformCloud *JobAgentDispatchTFCModel
	TestRunner     *JobAgentDispatchTestRunnerModel
}

// jobAgentDispatchConfigBlocks returns the schema block definitions for the
// typed job-agent dispatch config (argocd, argo_workflow, github,
// terraform_cloud, test_runner). These are shared between the deployment
// resource and the workflow resource's job_agent entries.
func jobAgentDispatchConfigBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"argocd": schema.SingleNestedBlock{
			Description: "ArgoCD job agent configuration",
			Attributes: map[string]schema.Attribute{
				"api_key":    schema.StringAttribute{Optional: true, Sensitive: true, Description: "ArgoCD API token"},
				"server_url": schema.StringAttribute{Optional: true, Description: "ArgoCD server address"},
				"template":   schema.StringAttribute{Optional: true, Description: "ArgoCD application template"},
			},
		},
		"argo_workflow": schema.SingleNestedBlock{
			Description: "Argo Workflow job agent configuration",
			Attributes: map[string]schema.Attribute{
				"api_key":        schema.StringAttribute{Optional: true, Sensitive: true, Description: "Argo Workflow API token"},
				"server_url":     schema.StringAttribute{Optional: true, Description: "Argo Workflow server address"},
				"template":       schema.StringAttribute{Optional: true, Description: "Argo Workflow application template"},
				"name":           schema.StringAttribute{Optional: true, Description: "The name of the argo template to call"},
				"webhook_secret": schema.StringAttribute{Optional: true, Sensitive: true, Description: "ArgoEvents webhook secret"},
				"http_insecure":  schema.BoolAttribute{Optional: true, Computed: true, Description: "Allow insecure HTTP connections", Default: booldefault.StaticBool(false)},
			},
		},
		"github": schema.SingleNestedBlock{
			Description: "GitHub job agent configuration",
			Attributes: map[string]schema.Attribute{
				"installation_id": schema.Int64Attribute{Optional: true, Description: "GitHub app installation ID"},
				"owner":           schema.StringAttribute{Optional: true, Description: "GitHub repository owner"},
				"ref":             schema.StringAttribute{Optional: true, Description: "Git ref to run the workflow on"},
				"repo":            schema.StringAttribute{Optional: true, Description: "GitHub repository name"},
				"workflow_id":     schema.Int64Attribute{Optional: true, Description: "GitHub Actions workflow ID"},
			},
		},
		"terraform_cloud": schema.SingleNestedBlock{
			Description: "Terraform Cloud job agent configuration",
			Attributes: map[string]schema.Attribute{
				"address":               schema.StringAttribute{Optional: true, Description: "Terraform Cloud address"},
				"organization":          schema.StringAttribute{Optional: true, Description: "Terraform Cloud organization name"},
				"template":              schema.StringAttribute{Optional: true, Description: "Terraform Cloud workspace template"},
				"token":                 schema.StringAttribute{Optional: true, Sensitive: true, Description: "Terraform Cloud API token"},
				"trigger_run_on_change": schema.BoolAttribute{Optional: true, Description: "Whether to create a TFC run on dispatch"},
			},
		},
		"test_runner": schema.SingleNestedBlock{
			Description: "Test runner job agent configuration",
			Attributes: map[string]schema.Attribute{
				"delay_seconds": schema.Int64Attribute{Optional: true, Description: "Delay in seconds before resolving the job"},
				"message":       schema.StringAttribute{Optional: true, Description: "Optional message to include in the job output"},
				"status":        schema.StringAttribute{Optional: true, Description: "Final status to set"},
			},
		},
	}
}

// jobAgentDispatchConfigToMap converts the typed dispatch blocks into the
// map[string]interface{} payload expected by the API. Returns nil when no
// block is set or when the selected block contributes no fields.
func jobAgentDispatchConfigToMap(b JobAgentDispatchBlocks) *map[string]interface{} {
	switch {
	case b.ArgoCD != nil:
		cfg := map[string]any{}
		setStringIfSet(cfg, "apiKey", b.ArgoCD.ApiKey)
		setStringIfSet(cfg, "serverUrl", b.ArgoCD.ServerUrl)
		setStringIfSet(cfg, "template", b.ArgoCD.Template)
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case b.ArgoWorkflow != nil:
		cfg := map[string]any{}
		setStringIfSet(cfg, "apiKey", b.ArgoWorkflow.ApiKey)
		setStringIfSet(cfg, "webhookSecret", b.ArgoWorkflow.WebhookSecret)
		setStringIfSet(cfg, "serverUrl", b.ArgoWorkflow.ServerUrl)
		setStringIfSet(cfg, "template", b.ArgoWorkflow.Template)
		setStringIfSet(cfg, "name", b.ArgoWorkflow.Name)
		if !b.ArgoWorkflow.HttpInsecure.IsNull() && !b.ArgoWorkflow.HttpInsecure.IsUnknown() {
			cfg["httpInsecure"] = b.ArgoWorkflow.HttpInsecure.ValueBool()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case b.GitHub != nil:
		cfg := map[string]any{}
		if !b.GitHub.InstallationId.IsNull() && !b.GitHub.InstallationId.IsUnknown() {
			cfg["installationId"] = b.GitHub.InstallationId.ValueInt64()
		}
		setStringIfSet(cfg, "owner", b.GitHub.Owner)
		setStringIfSet(cfg, "repo", b.GitHub.Repo)
		setStringIfSet(cfg, "ref", b.GitHub.Ref)
		if !b.GitHub.WorkflowId.IsNull() && !b.GitHub.WorkflowId.IsUnknown() {
			cfg["workflowId"] = b.GitHub.WorkflowId.ValueInt64()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case b.TerraformCloud != nil:
		cfg := map[string]any{}
		setStringIfSet(cfg, "address", b.TerraformCloud.Address)
		setStringIfSet(cfg, "organization", b.TerraformCloud.Organization)
		setStringIfSet(cfg, "template", b.TerraformCloud.Template)
		setStringIfSet(cfg, "token", b.TerraformCloud.Token)
		if !b.TerraformCloud.TriggerRunOnChange.IsNull() && !b.TerraformCloud.TriggerRunOnChange.IsUnknown() {
			cfg["triggerRunOnChange"] = b.TerraformCloud.TriggerRunOnChange.ValueBool()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case b.TestRunner != nil:
		cfg := map[string]any{}
		if !b.TestRunner.DelaySeconds.IsNull() && !b.TestRunner.DelaySeconds.IsUnknown() {
			cfg["delaySeconds"] = b.TestRunner.DelaySeconds.ValueInt64()
		}
		setStringIfSet(cfg, "message", b.TestRunner.Message)
		setStringIfSet(cfg, "status", b.TestRunner.Status)
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	default:
		return nil
	}
}

// dispatchBlockType returns the active block kind, or "" if no block is set.
func dispatchBlockType(b JobAgentDispatchBlocks) string {
	switch {
	case b.ArgoCD != nil:
		return "argocd"
	case b.ArgoWorkflow != nil:
		return "argo_workflow"
	case b.GitHub != nil:
		return "github"
	case b.TerraformCloud != nil:
		return "terraform_cloud"
	case b.TestRunner != nil:
		return "test_runner"
	default:
		return ""
	}
}

// dispatchBlockCount returns how many typed blocks are set.
func dispatchBlockCount(b JobAgentDispatchBlocks) int {
	count := 0
	if b.ArgoCD != nil {
		count++
	}
	if b.ArgoWorkflow != nil {
		count++
	}
	if b.GitHub != nil {
		count++
	}
	if b.TerraformCloud != nil {
		count++
	}
	if b.TestRunner != nil {
		count++
	}
	return count
}

// setJobAgentDispatchBlocksFromConfig populates the typed dispatch blocks on
// the model from the API's job-agent config map. The prior block values are
// preserved for sensitive fields (API tokens, secrets) since the API does not
// return them. The block kind is selected from `prior` when set, otherwise
// inferred from the config payload.
func setJobAgentDispatchBlocksFromConfig(prior, out *JobAgentDispatchBlocks, config map[string]interface{}) {
	priorArgoCD := prior.ArgoCD
	priorArgoWorkflow := prior.ArgoWorkflow
	priorTFC := prior.TerraformCloud

	out.ArgoCD = nil
	out.ArgoWorkflow = nil
	out.GitHub = nil
	out.TerraformCloud = nil
	out.TestRunner = nil

	if len(config) == 0 {
		return
	}

	blockType := dispatchBlockType(*prior)
	if blockType == "" {
		blockType = inferDispatchBlockType(config)
	}
	if blockType == "" {
		return
	}

	switch blockType {
	case "argocd":
		out.ArgoCD = &JobAgentDispatchArgoCDModel{
			ApiKey:    stringValueOrNull(config["apiKey"]),
			ServerUrl: stringValueOrNull(config["serverUrl"]),
			Template:  stringValueOrNull(config["template"]),
		}
		if out.ArgoCD.ApiKey.IsNull() && priorArgoCD != nil && !priorArgoCD.ApiKey.IsNull() {
			out.ArgoCD.ApiKey = priorArgoCD.ApiKey
		}
	case "argo_workflow":
		out.ArgoWorkflow = &JobAgentDispatchArgoWorkflowModel{
			ApiKey:        stringValueOrNull(config["apiKey"]),
			WebhookSecret: stringValueOrNull(config["webhookSecret"]),
			ServerUrl:     stringValueOrNull(config["serverUrl"]),
			Template:      stringValueOrNull(config["template"]),
			Name:          stringValueOrNull(config["name"]),
			HttpInsecure:  boolValueOrNull(config["httpInsecure"]),
		}
		if out.ArgoWorkflow.ApiKey.IsNull() && priorArgoWorkflow != nil && !priorArgoWorkflow.ApiKey.IsNull() {
			out.ArgoWorkflow.ApiKey = priorArgoWorkflow.ApiKey
		}
		if out.ArgoWorkflow.WebhookSecret.IsNull() && priorArgoWorkflow != nil && !priorArgoWorkflow.WebhookSecret.IsNull() {
			out.ArgoWorkflow.WebhookSecret = priorArgoWorkflow.WebhookSecret
		}
	case "github":
		gh := JobAgentDispatchGitHubModel{
			InstallationId: types.Int64Null(),
			Owner:          types.StringNull(),
			Ref:            types.StringNull(),
			Repo:           types.StringNull(),
			WorkflowId:     types.Int64Null(),
		}
		if v, ok := config["installationId"]; ok && v != nil {
			gh.InstallationId = types.Int64Value(toInt64(v))
		}
		if v, ok := config["owner"]; ok && v != nil && fmt.Sprint(v) != "" {
			gh.Owner = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["repo"]; ok && v != nil && fmt.Sprint(v) != "" {
			gh.Repo = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["ref"]; ok && v != nil && fmt.Sprint(v) != "" {
			gh.Ref = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["workflowId"]; ok && v != nil {
			gh.WorkflowId = types.Int64Value(toInt64(v))
		}
		out.GitHub = &gh
	case "terraform_cloud":
		out.TerraformCloud = &JobAgentDispatchTFCModel{
			Address:            stringValueOrNull(config["address"]),
			Organization:       stringValueOrNull(config["organization"]),
			Template:           stringValueOrNull(config["template"]),
			Token:              stringValueOrNull(config["token"]),
			TriggerRunOnChange: boolValueOrNull(config["triggerRunOnChange"]),
		}
		if out.TerraformCloud.Token.IsNull() && priorTFC != nil && !priorTFC.Token.IsNull() {
			out.TerraformCloud.Token = priorTFC.Token
		}
	case "test_runner":
		tr := JobAgentDispatchTestRunnerModel{
			DelaySeconds: types.Int64Null(),
			Message:      types.StringNull(),
			Status:       types.StringNull(),
		}
		if v, ok := config["delaySeconds"]; ok && v != nil {
			tr.DelaySeconds = types.Int64Value(toInt64(v))
		}
		if v, ok := config["message"]; ok && v != nil && fmt.Sprint(v) != "" {
			tr.Message = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["status"]; ok && v != nil && fmt.Sprint(v) != "" {
			tr.Status = types.StringValue(fmt.Sprint(v))
		}
		out.TestRunner = &tr
	}
}

type argoCDConfig struct {
	ApiKey    string `json:"apiKey"`
	ServerUrl string `json:"serverUrl"`
	Template  string `json:"template"`
}

type argoWorkflowConfig struct {
	ApiKey        string `json:"apiKey"`
	WebhookSecret string `json:"webhookSecret"`
	ServerUrl     string `json:"serverUrl"`
	Template      string `json:"template"`
	Name          string `json:"name"`
	HttpInsecure  *bool  `json:"httpInsecure"`
}

type githubConfig struct {
	InstallationId *int64 `json:"installationId"`
	Owner          string `json:"owner"`
	Ref            string `json:"ref"`
	Repo           string `json:"repo"`
	WorkflowId     *int64 `json:"workflowId"`
}

type terraformCloudConfig struct {
	Address            string `json:"address"`
	Organization       string `json:"organization"`
	Template           string `json:"template"`
	Token              string `json:"token"`
	TriggerRunOnChange *bool  `json:"triggerRunOnChange"`
}

type testRunnerConfig struct {
	DelaySeconds *int64 `json:"delaySeconds"`
	Message      string `json:"message"`
	Status       string `json:"status"`
}

func inferDispatchBlockType(config map[string]interface{}) string {
	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}

	var gh githubConfig
	_ = json.Unmarshal(data, &gh)
	if gh.Owner != "" || gh.Repo != "" || gh.InstallationId != nil || gh.WorkflowId != nil {
		return "github"
	}

	var tfc terraformCloudConfig
	_ = json.Unmarshal(data, &tfc)
	if tfc.Organization != "" || tfc.Address != "" || tfc.TriggerRunOnChange != nil {
		return "terraform_cloud"
	}

	var tr testRunnerConfig
	_ = json.Unmarshal(data, &tr)
	if tr.DelaySeconds != nil || tr.Status != "" {
		return "test_runner"
	}

	var aw argoWorkflowConfig
	_ = json.Unmarshal(data, &aw)
	if aw.WebhookSecret != "" || aw.HttpInsecure != nil || aw.Name != "" {
		return "argo_workflow"
	}

	var ac argoCDConfig
	_ = json.Unmarshal(data, &ac)
	if ac.ServerUrl != "" || ac.Template != "" || ac.ApiKey != "" {
		return "argocd"
	}

	return ""
}
