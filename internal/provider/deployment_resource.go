// Copyright (c) HashiCorp, Inc.

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

var _ resource.Resource = &DeploymentResource{}
var _ resource.ResourceWithImportState = &DeploymentResource{}
var _ resource.ResourceWithConfigure = &DeploymentResource{}
var _ resource.ResourceWithValidateConfig = &DeploymentResource{}

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
			},
		},
		Blocks: map[string]schema.Block{
			"job_agent": schema.ListNestedBlock{
				Description: "Job agent configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "Job agent ID",
						},
						"priority": schema.Int64Attribute{
							Optional:    true,
							Description: "Priority of the job agent",
						},
						"selector": schema.StringAttribute{
							Optional:    true,
							Description: "CEL expression used to select resources",
						},
					},
					Blocks: map[string]schema.Block{
						"argocd": schema.SingleNestedBlock{
							Description: "ArgoCD job agent overrides",
							Attributes: map[string]schema.Attribute{
								"api_key": schema.StringAttribute{
									Optional:    true,
									Description: "ArgoCD API token",
									Sensitive:   true,
								},
								"server_url": schema.StringAttribute{
									Optional:    true,
									Description: "ArgoCD server address (host[:port] or URL)",
								},
								"template": schema.StringAttribute{
									Optional:    true,
									Description: "ArgoCD application template",
								},
							},
						},
						"github": schema.SingleNestedBlock{
							Description: "GitHub job agent overrides",
							Attributes: map[string]schema.Attribute{
								"installation_id": schema.Int64Attribute{
									Optional:    true,
									Description: "GitHub app installation ID",
								},
								"owner": schema.StringAttribute{
									Optional:    true,
									Description: "GitHub repository owner",
								},
								"ref": schema.StringAttribute{
									Optional:    true,
									Description: "Git ref to run the workflow on (defaults to \"main\" if omitted)",
								},
								"repo": schema.StringAttribute{
									Optional:    true,
									Description: "GitHub repository name",
								},
								"workflow_id": schema.Int64Attribute{
									Optional:    true,
									Description: "GitHub Actions workflow ID",
								},
							},
						},
						"terraform_cloud": schema.SingleNestedBlock{
							Description: "Terraform Cloud job agent overrides",
							Attributes: map[string]schema.Attribute{
								"address": schema.StringAttribute{
									Optional:    true,
									Description: "Terraform Cloud address (e.g. https://app.terraform.io)",
								},
								"organization": schema.StringAttribute{
									Optional:    true,
									Description: "Terraform Cloud organization name",
								},
								"template": schema.StringAttribute{
									Optional:    true,
									Description: "Terraform Cloud workspace template",
								},
								"token": schema.StringAttribute{
									Optional:    true,
									Description: "Terraform Cloud API token",
									Sensitive:   true,
								},
							},
						},
						"test_runner": schema.SingleNestedBlock{
							Description: "Test runner job agent overrides",
							Attributes: map[string]schema.Attribute{
								"delay_seconds": schema.Int64Attribute{
									Optional:    true,
									Description: "Delay in seconds before resolving the job",
								},
								"message": schema.StringAttribute{
									Optional:    true,
									Description: "Optional message to include in the job output",
								},
								"status": schema.StringAttribute{
									Optional:    true,
									Description: "Final status to set (e.g. \"successful\", \"failure\")",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *DeploymentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.JobAgent) > 1 {
		resp.Diagnostics.AddError(
			"Invalid deployment configuration",
			"Only one job_agent block can be set.",
		)
		return
	}

	if len(data.JobAgent) == 0 {
		return
	}

	ja := data.JobAgent[0]
	if ja.Id.IsNull() || (!ja.Id.IsUnknown() && ja.Id.ValueString() == "") {
		resp.Diagnostics.AddError(
			"Invalid job agent configuration",
			"job_agent.id is required when job_agent is set.",
		)
		return
	}

	if countDeploymentJobAgentBlocks(ja) > 1 {
		resp.Diagnostics.AddError(
			"Invalid job agent configuration",
			"Only one of argocd, github, terraform_cloud, or test_runner can be set.",
		)
	}
}

func (r *DeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selector, err := selectorPointerFromString(data.ResourceSelector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment", fmt.Sprintf("Invalid resource_selector CEL: %s", err.Error()))
		return
	}

	var jobAgentId *string
	var jobAgentConfig *map[string]interface{}
	if len(data.JobAgent) > 0 {
		ja := data.JobAgent[0]
		if !ja.Id.IsNull() && !ja.Id.IsUnknown() {
			id := ja.Id.ValueString()
			jobAgentId = &id
		}
		jobAgentConfig = deploymentJobAgentConfigFromModel(ja)
	}

	requestBody := api.RequestDeploymentCreationJSONRequestBody{
		Name:             data.Name.ValueString(),
		Slug:             slug.Make(data.Name.ValueString()),
		Metadata:         stringMapPointer(data.Metadata),
		ResourceSelector: selector,
		JobAgentId:       jobAgentId,
		JobAgentConfig:   jobAgentConfig,
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

	if selectorValue, err := selectorStringValue(dep.ResourceSelector); err != nil {
		resp.Diagnostics.AddError("Failed to read deployment", fmt.Sprintf("Invalid resource_selector CEL: %s", err.Error()))
		return
	} else {
		data.ResourceSelector = selectorValue
	}

	if dep.JobAgentId != nil || len(dep.JobAgentConfig) > 0 {
		jobAgent := DeploymentJobAgentModel{
			Id:             types.StringNull(),
			Priority:       types.Int64Null(),
			Selector:       types.StringNull(),
			ArgoCD:         nil,
			GitHub:         nil,
			TerraformCloud: nil,
			TestRunner:     nil,
		}
		if dep.JobAgentId != nil {
			jobAgent.Id = types.StringValue(*dep.JobAgentId)
		}
		if len(dep.JobAgentConfig) > 0 {
			setDeploymentJobAgentBlocksFromConfig(&jobAgent, dep.JobAgentConfig)
		}
		data.JobAgent = []DeploymentJobAgentModel{jobAgent}
	} else {
		data.JobAgent = nil
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selector, err := selectorPointerFromString(data.ResourceSelector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update deployment", fmt.Sprintf("Invalid resource_selector CEL: %s", err.Error()))
		return
	}

	var jobAgentId *string
	var jobAgentConfig *map[string]interface{}
	if len(data.JobAgent) > 0 {
		ja := data.JobAgent[0]
		if !ja.Id.IsNull() && !ja.Id.IsUnknown() {
			id := ja.Id.ValueString()
			jobAgentId = &id
		}
		jobAgentConfig = deploymentJobAgentConfigFromModel(ja)
	}

	requestBody := api.UpsertDeploymentRequest{
		Name:             data.Name.ValueString(),
		Slug:             slug.Make(data.Name.ValueString()),
		Metadata:         stringMapPointer(data.Metadata),
		ResourceSelector: selector,
		JobAgentId:       jobAgentId,
		JobAgentConfig:   jobAgentConfig,
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
	ID               types.String              `tfsdk:"id"`
	Name             types.String              `tfsdk:"name"`
	Metadata         types.Map                 `tfsdk:"metadata"`
	ResourceSelector types.String              `tfsdk:"resource_selector"`
	JobAgent         []DeploymentJobAgentModel `tfsdk:"job_agent"`
}

type DeploymentJobAgentModel struct {
	Id             types.String                       `tfsdk:"id"`
	Priority       types.Int64                        `tfsdk:"priority"`
	Selector       types.String                       `tfsdk:"selector"`
	ArgoCD         *DeploymentJobAgentArgoCDModel     `tfsdk:"argocd"`
	GitHub         *DeploymentJobAgentGitHubModel     `tfsdk:"github"`
	TerraformCloud *DeploymentJobAgentTFCModel        `tfsdk:"terraform_cloud"`
	TestRunner     *DeploymentJobAgentTestRunnerModel `tfsdk:"test_runner"`
}

type DeploymentJobAgentArgoCDModel struct {
	ApiKey    types.String `tfsdk:"api_key"`
	ServerUrl types.String `tfsdk:"server_url"`
	Template  types.String `tfsdk:"template"`
}

type DeploymentJobAgentGitHubModel struct {
	InstallationId types.Int64  `tfsdk:"installation_id"`
	Owner          types.String `tfsdk:"owner"`
	Ref            types.String `tfsdk:"ref"`
	Repo           types.String `tfsdk:"repo"`
	WorkflowId     types.Int64  `tfsdk:"workflow_id"`
}

type DeploymentJobAgentTFCModel struct {
	Address      types.String `tfsdk:"address"`
	Organization types.String `tfsdk:"organization"`
	Template     types.String `tfsdk:"template"`
	Token        types.String `tfsdk:"token"`
}

type DeploymentJobAgentTestRunnerModel struct {
	DelaySeconds types.Int64  `tfsdk:"delay_seconds"`
	Message      types.String `tfsdk:"message"`
	Status       types.String `tfsdk:"status"`
}

func countDeploymentJobAgentBlocks(ja DeploymentJobAgentModel) int {
	count := 0
	if ja.ArgoCD != nil {
		count++
	}
	if ja.GitHub != nil {
		count++
	}
	if ja.TerraformCloud != nil {
		count++
	}
	if ja.TestRunner != nil {
		count++
	}
	return count
}

func deploymentJobAgentConfigFromModel(ja DeploymentJobAgentModel) *map[string]interface{} {
	switch {
	case ja.ArgoCD != nil:
		cfg := map[string]any{}
		if !ja.ArgoCD.ApiKey.IsNull() && !ja.ArgoCD.ApiKey.IsUnknown() && ja.ArgoCD.ApiKey.ValueString() != "" {
			cfg["apiKey"] = ja.ArgoCD.ApiKey.ValueString()
		}
		if !ja.ArgoCD.ServerUrl.IsNull() && !ja.ArgoCD.ServerUrl.IsUnknown() && ja.ArgoCD.ServerUrl.ValueString() != "" {
			cfg["serverUrl"] = ja.ArgoCD.ServerUrl.ValueString()
		}
		if !ja.ArgoCD.Template.IsNull() && !ja.ArgoCD.Template.IsUnknown() && ja.ArgoCD.Template.ValueString() != "" {
			cfg["template"] = ja.ArgoCD.Template.ValueString()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case ja.GitHub != nil:
		cfg := map[string]any{}
		if !ja.GitHub.InstallationId.IsNull() && !ja.GitHub.InstallationId.IsUnknown() {
			cfg["installationId"] = ja.GitHub.InstallationId.ValueInt64()
		}
		if !ja.GitHub.Owner.IsNull() && !ja.GitHub.Owner.IsUnknown() && ja.GitHub.Owner.ValueString() != "" {
			cfg["owner"] = ja.GitHub.Owner.ValueString()
		}
		if !ja.GitHub.Repo.IsNull() && !ja.GitHub.Repo.IsUnknown() && ja.GitHub.Repo.ValueString() != "" {
			cfg["repo"] = ja.GitHub.Repo.ValueString()
		}
		if !ja.GitHub.WorkflowId.IsNull() && !ja.GitHub.WorkflowId.IsUnknown() {
			cfg["workflowId"] = ja.GitHub.WorkflowId.ValueInt64()
		}
		if !ja.GitHub.Ref.IsNull() && !ja.GitHub.Ref.IsUnknown() && ja.GitHub.Ref.ValueString() != "" {
			cfg["ref"] = ja.GitHub.Ref.ValueString()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case ja.TerraformCloud != nil:
		cfg := map[string]any{}
		if !ja.TerraformCloud.Address.IsNull() && !ja.TerraformCloud.Address.IsUnknown() && ja.TerraformCloud.Address.ValueString() != "" {
			cfg["address"] = ja.TerraformCloud.Address.ValueString()
		}
		if !ja.TerraformCloud.Organization.IsNull() && !ja.TerraformCloud.Organization.IsUnknown() && ja.TerraformCloud.Organization.ValueString() != "" {
			cfg["organization"] = ja.TerraformCloud.Organization.ValueString()
		}
		if !ja.TerraformCloud.Template.IsNull() && !ja.TerraformCloud.Template.IsUnknown() && ja.TerraformCloud.Template.ValueString() != "" {
			cfg["template"] = ja.TerraformCloud.Template.ValueString()
		}
		if !ja.TerraformCloud.Token.IsNull() && !ja.TerraformCloud.Token.IsUnknown() && ja.TerraformCloud.Token.ValueString() != "" {
			cfg["token"] = ja.TerraformCloud.Token.ValueString()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	case ja.TestRunner != nil:
		cfg := map[string]any{}
		if !ja.TestRunner.DelaySeconds.IsNull() && !ja.TestRunner.DelaySeconds.IsUnknown() {
			cfg["delaySeconds"] = ja.TestRunner.DelaySeconds.ValueInt64()
		}
		if !ja.TestRunner.Message.IsNull() && !ja.TestRunner.Message.IsUnknown() && ja.TestRunner.Message.ValueString() != "" {
			cfg["message"] = ja.TestRunner.Message.ValueString()
		}
		if !ja.TestRunner.Status.IsNull() && !ja.TestRunner.Status.IsUnknown() && ja.TestRunner.Status.ValueString() != "" {
			cfg["status"] = ja.TestRunner.Status.ValueString()
		}
		if len(cfg) == 0 {
			return nil
		}
		return &cfg
	default:
		return nil
	}
}

func setDeploymentJobAgentBlocksFromConfig(ja *DeploymentJobAgentModel, config map[string]interface{}) {
	ja.ArgoCD = nil
	ja.GitHub = nil
	ja.TerraformCloud = nil
	ja.TestRunner = nil

	if len(config) == 0 {
		return
	}

	if configHasAny(config, "installationId", "workflowId", "owner", "repo") {
		github := DeploymentJobAgentGitHubModel{
			InstallationId: types.Int64Null(),
			Owner:          types.StringNull(),
			Ref:            types.StringNull(),
			Repo:           types.StringNull(),
			WorkflowId:     types.Int64Null(),
		}
		if v, ok := config["installationId"]; ok && v != nil {
			github.InstallationId = types.Int64Value(toInt64(v))
		}
		if v, ok := config["owner"]; ok && v != nil && fmt.Sprint(v) != "" {
			github.Owner = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["repo"]; ok && v != nil && fmt.Sprint(v) != "" {
			github.Repo = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["workflowId"]; ok && v != nil {
			github.WorkflowId = types.Int64Value(toInt64(v))
		}
		if v, ok := config["ref"]; ok && v != nil && fmt.Sprint(v) != "" {
			github.Ref = types.StringValue(fmt.Sprint(v))
		}
		ja.GitHub = &github
		return
	}

	if configHasAny(config, "apiKey", "serverUrl", "template") {
		ja.ArgoCD = &DeploymentJobAgentArgoCDModel{
			ApiKey:    stringValueOrNull(config["apiKey"]),
			ServerUrl: stringValueOrNull(config["serverUrl"]),
			Template:  stringValueOrNull(config["template"]),
		}
		return
	}

	if configHasAny(config, "address", "organization", "template", "token") {
		ja.TerraformCloud = &DeploymentJobAgentTFCModel{
			Address:      stringValueOrNull(config["address"]),
			Organization: stringValueOrNull(config["organization"]),
			Template:     stringValueOrNull(config["template"]),
			Token:        stringValueOrNull(config["token"]),
		}
		return
	}

	if configHasAny(config, "delaySeconds", "message", "status") {
		testRunner := DeploymentJobAgentTestRunnerModel{
			DelaySeconds: types.Int64Null(),
			Message:      types.StringNull(),
			Status:       types.StringNull(),
		}
		if v, ok := config["delaySeconds"]; ok && v != nil {
			testRunner.DelaySeconds = types.Int64Value(toInt64(v))
		}
		if v, ok := config["message"]; ok && v != nil && fmt.Sprint(v) != "" {
			testRunner.Message = types.StringValue(fmt.Sprint(v))
		}
		if v, ok := config["status"]; ok && v != nil && fmt.Sprint(v) != "" {
			testRunner.Status = types.StringValue(fmt.Sprint(v))
		}
		ja.TestRunner = &testRunner
	}
}

func configHasAny(config map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := config[key]; ok {
			return true
		}
	}
	return false
}

func stringValueOrNull(value interface{}) types.String {
	if value == nil {
		return types.StringNull()
	}
	str := fmt.Sprint(value)
	if str == "" {
		return types.StringNull()
	}
	return types.StringValue(str)
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
