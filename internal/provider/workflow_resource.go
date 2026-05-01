// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &WorkflowResource{}
	_ resource.ResourceWithImportState    = &WorkflowResource{}
	_ resource.ResourceWithConfigure      = &WorkflowResource{}
	_ resource.ResourceWithValidateConfig = &WorkflowResource{}
)

func NewWorkflowResource() resource.Resource {
	return &WorkflowResource{}
}

type WorkflowResource struct {
	workspace *api.WorkspaceClient
}

type WorkflowResourceModel struct {
	ID        types.String            `tfsdk:"id"`
	Name      types.String            `tfsdk:"name"`
	Inputs    types.String            `tfsdk:"inputs"`
	JobAgents []WorkflowJobAgentModel `tfsdk:"job_agent"`
}

type WorkflowJobAgentModel struct {
	Name     types.String `tfsdk:"name"`
	Ref      types.String `tfsdk:"ref"`
	Selector types.String `tfsdk:"selector"`

	ArgoCD         *JobAgentDispatchArgoCDModel       `tfsdk:"argocd"`
	ArgoWorkflow   *JobAgentDispatchArgoWorkflowModel `tfsdk:"argo_workflow"`
	GitHub         *JobAgentDispatchGitHubModel       `tfsdk:"github"`
	TerraformCloud *JobAgentDispatchTFCModel          `tfsdk:"terraform_cloud"`
	TestRunner     *JobAgentDispatchTestRunnerModel   `tfsdk:"test_runner"`
}

func (m *WorkflowJobAgentModel) dispatchBlocks() JobAgentDispatchBlocks {
	return JobAgentDispatchBlocks{
		ArgoCD:         m.ArgoCD,
		ArgoWorkflow:   m.ArgoWorkflow,
		GitHub:         m.GitHub,
		TerraformCloud: m.TerraformCloud,
		TestRunner:     m.TestRunner,
	}
}

func (m *WorkflowJobAgentModel) setDispatchBlocks(b JobAgentDispatchBlocks) {
	m.ArgoCD = b.ArgoCD
	m.ArgoWorkflow = b.ArgoWorkflow
	m.GitHub = b.GitHub
	m.TerraformCloud = b.TerraformCloud
	m.TestRunner = b.TestRunner
}

func (r *WorkflowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *WorkflowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	workspace, ok := req.ProviderData.(*api.WorkspaceClient)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "The provider data is not a *api.WorkspaceClient")
		return
	}
	r.workspace = workspace
}

func (r *WorkflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a workflow in Ctrlplane.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the workflow.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the workflow.",
			},
			"inputs": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded array of workflow input definitions.",
			},
		},
		Blocks: map[string]schema.Block{
			"job_agent": schema.ListNestedBlock{
				Description: "Job agents to dispatch when the workflow runs.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the job agent entry.",
						},
						"ref": schema.StringAttribute{
							Required:    true,
							Description: "ID of the job agent to reference.",
						},
						"selector": schema.StringAttribute{
							Required:    true,
							Description: "CEL expression to determine if the job agent should dispatch. Use \"true\" to always dispatch.",
						},
					},
					Blocks: jobAgentDispatchConfigBlocks(),
				},
			},
		},
	}
}

func (r *WorkflowResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, agent := range data.JobAgents {
		count := dispatchBlockCount(agent.dispatchBlocks())
		if count == 0 {
			resp.Diagnostics.AddError(
				"Invalid job agent configuration",
				fmt.Sprintf("job_agent[%d] (%q) must set exactly one of argocd, argo_workflow, github, terraform_cloud, or test_runner.", i, agent.Name.ValueString()),
			)
			continue
		}
		if count > 1 {
			resp.Diagnostics.AddError(
				"Invalid job agent configuration",
				fmt.Sprintf("job_agent[%d] (%q) may set only one of argocd, argo_workflow, github, terraform_cloud, or test_runner.", i, agent.Name.ValueString()),
			)
		}
	}
}

func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inputs, err := parseWorkflowInputs(data.Inputs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid inputs", err.Error())
		return
	}

	body := api.CreateWorkflowJSONRequestBody{
		Name:      data.Name.ValueString(),
		Inputs:    inputs,
		JobAgents: workflowJobAgentsFromModel(data.JobAgents),
	}

	createResp, err := r.workspace.Client.CreateWorkflowWithResponse(ctx, r.workspace.ID.String(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create workflow", err.Error())
		return
	}

	if createResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError("Failed to create workflow", formatResponseError(createResp.StatusCode(), createResp.Body))
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create workflow", "Empty response from server")
		return
	}

	setWorkflowModelFromAPI(&data, createResp.JSON201)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.workspace.Client.GetWorkflowWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read workflow", err.Error())
		return
	}

	switch getResp.StatusCode() {
	case http.StatusOK:
		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read workflow", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read workflow", formatResponseError(getResp.StatusCode(), getResp.Body))
		return
	}

	setWorkflowModelFromAPI(&data, getResp.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inputs, err := parseWorkflowInputs(data.Inputs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid inputs", err.Error())
		return
	}

	body := api.UpdateWorkflowJSONRequestBody{
		Name:      data.Name.ValueString(),
		Inputs:    inputs,
		JobAgents: workflowJobAgentsFromModel(data.JobAgents),
	}

	updateResp, err := r.workspace.Client.UpdateWorkflowWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update workflow", err.Error())
		return
	}

	if updateResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update workflow", formatResponseError(updateResp.StatusCode(), updateResp.Body))
		return
	}

	if updateResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update workflow", "Empty response from server")
		return
	}

	setWorkflowModelFromAPI(&data, updateResp.JSON202)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.workspace.Client.DeleteWorkflowWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete workflow", err.Error())
		return
	}

	switch deleteResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusNotFound:
		return
	default:
		resp.Diagnostics.AddError("Failed to delete workflow", formatResponseError(deleteResp.StatusCode(), deleteResp.Body))
	}
}

// --- helpers ---

// normalizeInputsJSON re-marshals workflow inputs through a generic structure
// so that JSON key order is deterministic (Go sorts map keys alphabetically).
// This prevents Terraform from detecting spurious diffs due to key ordering.
func normalizeInputsJSON(inputs []api.WorkflowInput) string {
	raw, err := json.Marshal(inputs)
	if err != nil {
		return "[]"
	}

	var normalized []map[string]interface{}
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return "[]"
	}

	out, err := json.Marshal(normalized)
	if err != nil {
		return "[]"
	}

	return string(out)
}

func parseWorkflowInputs(raw types.String) ([]api.WorkflowInput, error) {
	if raw.IsNull() || raw.IsUnknown() {
		return []api.WorkflowInput{}, nil
	}
	str := raw.ValueString()
	if str == "" || str == "[]" {
		return []api.WorkflowInput{}, nil
	}
	var inputs []api.WorkflowInput
	if err := json.Unmarshal([]byte(str), &inputs); err != nil {
		return nil, fmt.Errorf("failed to parse inputs JSON: %w", err)
	}
	return inputs, nil
}

func workflowJobAgentsFromModel(agents []WorkflowJobAgentModel) []api.CreateWorkflowJobAgent {
	result := make([]api.CreateWorkflowJobAgent, len(agents))
	for i, a := range agents {
		var config map[string]interface{}
		if cfg := jobAgentDispatchConfigToMap(a.dispatchBlocks()); cfg != nil {
			config = *cfg
		} else {
			config = map[string]interface{}{}
		}
		result[i] = api.CreateWorkflowJobAgent{
			Name:     a.Name.ValueString(),
			Ref:      a.Ref.ValueString(),
			Config:   config,
			Selector: a.Selector.ValueString(),
		}
	}
	return result
}

func setWorkflowModelFromAPI(data *WorkflowResourceModel, w *api.Workflow) {
	data.ID = types.StringValue(w.Id)
	data.Name = types.StringValue(w.Name)

	data.Inputs = types.StringValue(normalizeInputsJSON(w.Inputs))

	priorByName := make(map[string]WorkflowJobAgentModel, len(data.JobAgents))
	for _, a := range data.JobAgents {
		priorByName[a.Name.ValueString()] = a
	}

	agents := make([]WorkflowJobAgentModel, len(w.JobAgents))
	for i, a := range w.JobAgents {
		agent := WorkflowJobAgentModel{
			Name:     types.StringValue(a.Name),
			Ref:      types.StringValue(a.Ref),
			Selector: types.StringValue(a.Selector),
		}
		var prior JobAgentDispatchBlocks
		if p, ok := priorByName[a.Name]; ok {
			prior = p.dispatchBlocks()
		}
		var out JobAgentDispatchBlocks
		setJobAgentDispatchBlocksFromConfig(&prior, &out, a.Config)
		agent.setDispatchBlocks(out)
		agents[i] = agent
	}
	data.JobAgents = agents
}
