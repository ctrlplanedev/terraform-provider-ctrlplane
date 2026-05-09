// Copyright IBM Corp. 2021, 2026

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/gosimple/slug"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &DeploymentResource{}
	_ resource.ResourceWithImportState    = &DeploymentResource{}
	_ resource.ResourceWithConfigure      = &DeploymentResource{}
	_ resource.ResourceWithValidateConfig = &DeploymentResource{}
)

func NewDeploymentResource() resource.Resource {
	return &DeploymentResource{}
}

type DeploymentResource struct {
	workspace *api.WorkspaceClient
}

func (r *DeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

func (r *DeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the deployment",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the deployment",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The metadata of the deployment",
				ElementType: types.StringType,
				Default: func() defaults.Map {
					empty, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
					return mapdefault.StaticValue(empty)
				}(),
			},
			"resource_selector": schema.StringAttribute{
				Optional:    true,
				Description: "CEL expression used to select resources",
				PlanModifiers: []planmodifier.String{
					celNormalized(),
				},
			},
			"job_agent_selector": schema.StringAttribute{
				Optional:    true,
				Description: "CEL expression to match job agents",
			},
		},
		Blocks: jobAgentDispatchConfigBlocks(),
	}
}

func (r *DeploymentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if dispatchBlockCount(data.dispatchBlocks()) > 1 {
		resp.Diagnostics.AddError(
			"Invalid job agent configuration",
			"Only one of argocd, argo_workflow, github, terraform_cloud, or test_runner can be set.",
		)
	}
}

func (r *DeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var resourceSelector *string
	if cel := normalizeCEL(data.ResourceSelector); cel != "" {
		resourceSelector = &cel
	}

	var jobAgentSelector *string
	if !data.JobAgentSelector.IsNull() && !data.JobAgentSelector.IsUnknown() {
		s := data.JobAgentSelector.ValueString()
		jobAgentSelector = &s
	}

	requestBody := api.RequestDeploymentCreationJSONRequestBody{
		Name:             data.Name.ValueString(),
		Slug:             slug.Make(data.Name.ValueString()),
		Metadata:         stringMapPointer(data.Metadata),
		ResourceSelector: resourceSelector,
		JobAgentSelector: jobAgentSelector,
		JobAgentConfig:   deploymentJobAgentConfigFromModel(&data),
	}

	deployResp, err := r.workspace.Client.RequestDeploymentCreationWithResponse(ctx, r.workspace.ID.String(), requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment", err.Error())
		return
	}

	if deployResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create deployment", formatResponseError(deployResp.StatusCode(), deployResp.Body))
		return
	}

	if deployResp.JSON202 == nil || deployResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to create deployment", "Empty deployment ID in response")
		return
	}

	deploymentId := deployResp.JSON202.Id
	data.ID = types.StringValue(deploymentId)

	err = waitForResource(ctx, func() (bool, error) {
		getResp, err := r.workspace.Client.GetDeploymentWithResponse(ctx, r.workspace.ID.String(), deploymentId)
		if err != nil {
			return false, err
		}
		switch getResp.StatusCode() {
		case http.StatusOK:
			return true, nil
		case http.StatusNotFound:
			return false, nil
		default:
			return false, fmt.Errorf("unexpected status %d", getResp.StatusCode())
		}
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment", fmt.Sprintf("Resource not available after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployResp, err := r.workspace.Client.GetDeploymentWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read deployment", fmt.Sprintf("Failed to read deployment with ID '%s': %s", data.ID.ValueString(), err.Error()))
		return
	}

	switch deployResp.StatusCode() {
	case http.StatusOK:
		if deployResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read deployment", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		if deployResp.JSON400 != nil && deployResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to read deployment", fmt.Sprintf("Bad request: %s", *deployResp.JSON400.Error))
			return
		}
		resp.Diagnostics.AddError("Failed to read deployment", "Bad request")
		return
	}

	if deployResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read deployment", formatResponseError(deployResp.StatusCode(), deployResp.Body))
		return
	}

	dep := deployResp.JSON200.Deployment
	data.ID = types.StringValue(dep.Id)
	data.Name = types.StringValue(dep.Name)
	data.Metadata = stringMapValue(dep.Metadata)

	if dep.ResourceSelector != nil && *dep.ResourceSelector != "" {
		data.ResourceSelector = types.StringValue(*dep.ResourceSelector)
	} else {
		data.ResourceSelector = types.StringNull()
	}

	if dep.JobAgentSelector != "" {
		data.JobAgentSelector = types.StringValue(dep.JobAgentSelector)
	} else {
		data.JobAgentSelector = types.StringNull()
	}

	setDeploymentBlocksFromConfig(&data, dep.JobAgentConfig)

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var resourceSelector *string
	if cel := normalizeCEL(data.ResourceSelector); cel != "" {
		resourceSelector = &cel
	}

	var jobAgentSelector *string
	if !data.JobAgentSelector.IsNull() && !data.JobAgentSelector.IsUnknown() {
		s := data.JobAgentSelector.ValueString()
		jobAgentSelector = &s
	}

	requestBody := api.UpsertDeploymentRequest{
		Name:             data.Name.ValueString(),
		Slug:             slug.Make(data.Name.ValueString()),
		Metadata:         stringMapPointer(data.Metadata),
		ResourceSelector: resourceSelector,
		JobAgentSelector: jobAgentSelector,
		JobAgentConfig:   deploymentJobAgentConfigFromModel(&data),
	}

	deployResp, err := r.workspace.Client.RequestDeploymentUpsertWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString(), requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update deployment", fmt.Sprintf("Failed to update deployment with ID '%s': %s", data.ID.ValueString(), err.Error()))
		return
	}

	if deployResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update deployment", formatResponseError(deployResp.StatusCode(), deployResp.Body))
		return
	}

	if deployResp.JSON202 == nil || deployResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to update deployment", "Empty deployment ID in response")
		return
	}

	data.ID = types.StringValue(deployResp.JSON202.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientResp, err := r.workspace.Client.RequestDeploymentDeletionWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete deployment", fmt.Sprintf("Failed to delete deployment: %s", err.Error()))
		return
	}

	switch clientResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusBadRequest:
		if clientResp.JSON400 != nil && clientResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to delete deployment", fmt.Sprintf("Bad request: %s", *clientResp.JSON400.Error))
			return
		}
	case http.StatusNotFound:
		if clientResp.JSON404 != nil && clientResp.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to delete deployment", fmt.Sprintf("Not found: %s", *clientResp.JSON404.Error))
			return
		}
	}

	resp.Diagnostics.AddError("Failed to delete deployment", formatResponseError(clientResp.StatusCode(), clientResp.Body))
}

type DeploymentResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Metadata         types.Map    `tfsdk:"metadata"`
	ResourceSelector types.String `tfsdk:"resource_selector"`
	JobAgentSelector types.String `tfsdk:"job_agent_selector"`

	ArgoCD         *JobAgentDispatchArgoCDModel       `tfsdk:"argocd"`
	ArgoWorkflow   *JobAgentDispatchArgoWorkflowModel `tfsdk:"argo_workflow"`
	GitHub         *JobAgentDispatchGitHubModel       `tfsdk:"github"`
	TerraformCloud *JobAgentDispatchTFCModel          `tfsdk:"terraform_cloud"`
	TestRunner     *JobAgentDispatchTestRunnerModel   `tfsdk:"test_runner"`
}

func (m *DeploymentResourceModel) dispatchBlocks() JobAgentDispatchBlocks {
	return JobAgentDispatchBlocks{
		ArgoCD:         m.ArgoCD,
		ArgoWorkflow:   m.ArgoWorkflow,
		GitHub:         m.GitHub,
		TerraformCloud: m.TerraformCloud,
		TestRunner:     m.TestRunner,
	}
}

func (m *DeploymentResourceModel) setDispatchBlocks(b JobAgentDispatchBlocks) {
	m.ArgoCD = b.ArgoCD
	m.ArgoWorkflow = b.ArgoWorkflow
	m.GitHub = b.GitHub
	m.TerraformCloud = b.TerraformCloud
	m.TestRunner = b.TestRunner
}

// deploymentJobAgentConfigFromModel extracts the typed block into a
// map[string]interface{} suitable for the API's JobAgentConfig field.
func deploymentJobAgentConfigFromModel(data *DeploymentResourceModel) *map[string]interface{} {
	return jobAgentDispatchConfigToMap(data.dispatchBlocks())
}

func setStringIfSet(cfg map[string]any, key string, val types.String) {
	if !val.IsNull() && !val.IsUnknown() && val.ValueString() != "" {
		cfg[key] = val.ValueString()
	}
}

// setDeploymentBlocksFromConfig populates the typed block on the model from
// the API's JobAgentConfig map. It uses the prior state block type to decide
// which block to populate so that reads are stable.
func setDeploymentBlocksFromConfig(data *DeploymentResourceModel, config map[string]interface{}) {
	prior := data.dispatchBlocks()
	var out JobAgentDispatchBlocks
	setJobAgentDispatchBlocksFromConfig(&prior, &out, config)
	data.setDispatchBlocks(out)
}

func stringValueOrNull(value interface{}) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(fmt.Sprint(value))
}

func boolValueOrNull(value interface{}) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	if b, ok := value.(bool); ok {
		return types.BoolValue(b)
	}
	return types.BoolNull()
}

func stringInterfaceMapPointer(value types.Map) *map[string]interface{} {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var decoded map[string]string
	diags := value.ElementsAs(context.Background(), &decoded, false)
	if diags.HasError() {
		return nil
	}

	result := make(map[string]interface{}, len(decoded))
	for k, v := range decoded {
		result[k] = v
	}

	return &result
}

func interfaceMapStringValue(value map[string]interface{}) types.Map {
	if value == nil {
		return types.MapNull(types.StringType)
	}

	result := make(map[string]string, len(value))
	for k, v := range value {
		result[k] = fmt.Sprint(v)
	}

	mapped, _ := types.MapValueFrom(context.Background(), types.StringType, result)
	return mapped
}
