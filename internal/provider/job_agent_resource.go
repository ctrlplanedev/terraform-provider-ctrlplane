package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &JobAgentResource{}
var _ resource.ResourceWithImportState = &JobAgentResource{}
var _ resource.ResourceWithConfigure = &JobAgentResource{}
var _ resource.ResourceWithValidateConfig = &JobAgentResource{}

func NewJobAgentResource() resource.Resource {
	return &JobAgentResource{}
}

type JobAgentResource struct {
	workspace *api.WorkspaceClient
}

func (r *JobAgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_agent"
}

func (r *JobAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *JobAgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *JobAgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the job agent",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the job agent",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The metadata of the job agent",
				ElementType: types.StringType,
				Default: func() defaults.Map {
					empty, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
					return mapdefault.StaticValue(empty)
				}(),
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"custom": schema.ListNestedBlock{
				Description: "Custom job agent configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Job agent type",
						},
						"config": schema.MapAttribute{
							Required:    true,
							Description: "Job agent configuration",
							ElementType: types.StringType,
						},
					},
				},
			},
			"argocd": schema.ListNestedBlock{
				Description: "ArgoCD job agent configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_key": schema.StringAttribute{
							Required:    true,
							Description: "ArgoCD API token",
							Sensitive:   true,
						},
						"server_url": schema.StringAttribute{
							Required:    true,
							Description: "ArgoCD server address (host[:port] or URL)",
						},
						"template": schema.StringAttribute{
							Required:    true,
							Description: "ArgoCD application template",
						},
					},
				},
			},
			"github": schema.ListNestedBlock{
				Description: "GitHub job agent configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"installation_id": schema.Int64Attribute{
							Required:    true,
							Description: "GitHub app installation ID",
						},
						"owner": schema.StringAttribute{
							Required:    true,
							Description: "GitHub repository owner",
						},
						"ref": schema.StringAttribute{
							Optional:    true,
							Description: "Git ref to run the workflow on (defaults to \"main\" if omitted)",
						},
						"repo": schema.StringAttribute{
							Required:    true,
							Description: "GitHub repository name",
						},
						"workflow_id": schema.Int64Attribute{
							Required:    true,
							Description: "GitHub Actions workflow ID",
						},
					},
				},
			},
			"terraform_cloud": schema.ListNestedBlock{
				Description: "Terraform Cloud job agent configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"address": schema.StringAttribute{
							Required:    true,
							Description: "Terraform Cloud address (e.g. https://app.terraform.io)",
						},
						"organization": schema.StringAttribute{
							Required:    true,
							Description: "Terraform Cloud organization name",
						},
						"template": schema.StringAttribute{
							Required:    true,
							Description: "Terraform Cloud workspace template",
						},
						"token": schema.StringAttribute{
							Required:    true,
							Description: "Terraform Cloud API token",
							Sensitive:   true,
						},
					},
				},
			},
			"test_runner": schema.ListNestedBlock{
				Description: "Test runner job agent configuration",
				NestedObject: schema.NestedBlockObject{
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
	}
}

func (r *JobAgentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data JobAgentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	count := countJobAgentConfigs(data)
	if count == 0 {
		resp.Diagnostics.AddError(
			"Invalid job agent configuration",
			"Exactly one of custom, argocd, github, terraform_cloud, or test_runner must be set.",
		)
		return
	}
	if count > 1 {
		resp.Diagnostics.AddError(
			"Invalid job agent configuration",
			"Only one of custom, argocd, github, terraform_cloud, or test_runner can be set.",
		)
	}
}

func (r *JobAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data JobAgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobAgentType, config, configErr := jobAgentConfigFromModel(data)
	if configErr != nil {
		resp.Diagnostics.AddError("Failed to create job agent", configErr.Error())
		return
	}
	if config == nil {
		resp.Diagnostics.AddError("Failed to create job agent", "Exactly one job agent type must be configured")
		return
	}

	jobAgentId := data.ID.ValueString()
	if data.ID.IsNull() || data.ID.IsUnknown() || jobAgentId == "" {
		jobAgentId = uuid.NewString()
		data.ID = types.StringValue(jobAgentId)
	}

	requestBody := api.RequestJobAgentUpdateJSONRequestBody{
		Config:   *config,
		Metadata: stringMapPointer(data.Metadata),
		Name:     data.Name.ValueString(),
		Type:     jobAgentType,
	}

	jobAgentResp, err := r.workspace.Client.RequestJobAgentUpdateWithResponse(
		ctx, r.workspace.ID.String(), jobAgentId, requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create job agent", err.Error())
		return
	}

	if jobAgentResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create job agent", formatResponseError(jobAgentResp.StatusCode(), jobAgentResp.Body))
		return
	}

	if jobAgentResp.JSON202 == nil || jobAgentResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to create job agent", "Empty job agent ID in response")
		return
	}

	data.ID = types.StringValue(jobAgentResp.JSON202.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *JobAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data JobAgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobAgentResp, err := r.workspace.Client.GetJobAgentWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read job agent",
			fmt.Sprintf("Failed to read job agent with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	switch jobAgentResp.StatusCode() {
	case http.StatusOK:
		if jobAgentResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read job agent", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		resp.Diagnostics.AddError("Failed to read job agent", "Bad request")
		return
	}

	if jobAgentResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read job agent", formatResponseError(jobAgentResp.StatusCode(), jobAgentResp.Body))
		return
	}

	jobAgent := jobAgentResp.JSON200
	data.ID = types.StringValue(jobAgent.Id)
	data.Name = types.StringValue(jobAgent.Name)
	if jobAgent.Metadata == nil {
		empty, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
		data.Metadata = empty
	} else {
		data.Metadata = stringMapValue(&jobAgent.Metadata)
	}

	setJobAgentBlocksFromAPI(&data, jobAgent.Type, jobAgent.Config)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *JobAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data JobAgentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobAgentType, config, configErr := jobAgentConfigFromModel(data)
	if configErr != nil {
		resp.Diagnostics.AddError("Failed to update job agent", configErr.Error())
		return
	}
	if config == nil {
		resp.Diagnostics.AddError("Failed to update job agent", "Exactly one job agent type must be configured")
		return
	}

	requestBody := api.RequestJobAgentUpdateJSONRequestBody{
		Config:   *config,
		Metadata: stringMapPointer(data.Metadata),
		Name:     data.Name.ValueString(),
		Type:     jobAgentType,
	}

	jobAgentResp, err := r.workspace.Client.RequestJobAgentUpdateWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(), requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update job agent",
			fmt.Sprintf("Failed to update job agent with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if jobAgentResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update job agent", formatResponseError(jobAgentResp.StatusCode(), jobAgentResp.Body))
		return
	}

	if jobAgentResp.JSON202 == nil || jobAgentResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to update job agent", "Empty job agent ID in response")
		return
	}

	data.ID = types.StringValue(jobAgentResp.JSON202.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *JobAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data JobAgentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobAgentResp, err := r.workspace.Client.RequestJobAgentDeletionWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete job agent", fmt.Sprintf("Failed to delete job agent: %s", err.Error()))
		return
	}

	switch jobAgentResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusBadRequest:
		if jobAgentResp.JSON400 != nil && jobAgentResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to delete job agent", fmt.Sprintf("Bad request: %s", *jobAgentResp.JSON400.Error))
			return
		}
	case http.StatusNotFound:
		if jobAgentResp.JSON404 != nil && jobAgentResp.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to delete job agent", fmt.Sprintf("Not found: %s", *jobAgentResp.JSON404.Error))
			return
		}
	}

	resp.Diagnostics.AddError("Failed to delete job agent", formatResponseError(jobAgentResp.StatusCode(), jobAgentResp.Body))
}

type JobAgentResourceModel struct {
	ID             types.String              `tfsdk:"id"`
	Name           types.String              `tfsdk:"name"`
	Metadata       types.Map                 `tfsdk:"metadata"`
	Custom         []JobAgentCustomModel     `tfsdk:"custom"`
	ArgoCD         []JobAgentArgoCDModel     `tfsdk:"argocd"`
	GitHub         []JobAgentGitHubModel     `tfsdk:"github"`
	TerraformCloud []JobAgentTFCModel        `tfsdk:"terraform_cloud"`
	TestRunner     []JobAgentTestRunnerModel `tfsdk:"test_runner"`
}

type JobAgentCustomModel struct {
	Type   types.String `tfsdk:"type"`
	Config types.Map    `tfsdk:"config"`
}

type JobAgentArgoCDModel struct {
	ApiKey    types.String `tfsdk:"api_key"`
	ServerUrl types.String `tfsdk:"server_url"`
	Template  types.String `tfsdk:"template"`
}

type JobAgentGitHubModel struct {
	InstallationId types.Int64  `tfsdk:"installation_id"`
	Owner          types.String `tfsdk:"owner"`
	Ref            types.String `tfsdk:"ref"`
	Repo           types.String `tfsdk:"repo"`
	WorkflowId     types.Int64  `tfsdk:"workflow_id"`
}

type JobAgentTFCModel struct {
	Address      types.String `tfsdk:"address"`
	Organization types.String `tfsdk:"organization"`
	Template     types.String `tfsdk:"template"`
	Token        types.String `tfsdk:"token"`
}

type JobAgentTestRunnerModel struct {
	DelaySeconds types.Int64  `tfsdk:"delay_seconds"`
	Message      types.String `tfsdk:"message"`
	Status       types.String `tfsdk:"status"`
}

func countJobAgentConfigs(data JobAgentResourceModel) int {
	count := 0
	if len(data.Custom) > 0 {
		count++
	}
	if len(data.ArgoCD) > 0 {
		count++
	}
	if len(data.GitHub) > 0 {
		count++
	}
	if len(data.TerraformCloud) > 0 {
		count++
	}
	if len(data.TestRunner) > 0 {
		count++
	}
	return count
}

func jobAgentConfigFromModel(data JobAgentResourceModel) (string, *map[string]interface{}, error) {
	switch {
	case len(data.Custom) > 0:
		custom := data.Custom[0]
		customType := custom.Type.ValueString()
		if custom.Type.IsNull() || custom.Type.IsUnknown() || customType == "" {
			return "", nil, fmt.Errorf("custom.type is required")
		}
		config := stringInterfaceMapPointer(custom.Config)
		if config == nil {
			return "", nil, fmt.Errorf("custom.config must be a non-empty map")
		}
		return customType, config, nil
	case len(data.ArgoCD) > 0:
		argocd := data.ArgoCD[0]
		cfg := map[string]interface{}{
			"apiKey":    argocd.ApiKey.ValueString(),
			"serverUrl": argocd.ServerUrl.ValueString(),
			"template":  argocd.Template.ValueString(),
		}
		return "argocd", &cfg, nil
	case len(data.GitHub) > 0:
		github := data.GitHub[0]
		cfg := map[string]interface{}{
			"installationId": github.InstallationId.ValueInt64(),
			"owner":          github.Owner.ValueString(),
			"repo":           github.Repo.ValueString(),
			"workflowId":     github.WorkflowId.ValueInt64(),
		}
		if !github.Ref.IsNull() && !github.Ref.IsUnknown() && github.Ref.ValueString() != "" {
			cfg["ref"] = github.Ref.ValueString()
		}
		return "github", &cfg, nil
	case len(data.TerraformCloud) > 0:
		tfc := data.TerraformCloud[0]
		cfg := map[string]interface{}{
			"address":      tfc.Address.ValueString(),
			"organization": tfc.Organization.ValueString(),
			"template":     tfc.Template.ValueString(),
			"token":        tfc.Token.ValueString(),
		}
		return "terraformcloud", &cfg, nil
	case len(data.TestRunner) > 0:
		testRunner := data.TestRunner[0]
		cfg := map[string]interface{}{}
		if !testRunner.DelaySeconds.IsNull() && !testRunner.DelaySeconds.IsUnknown() {
			cfg["delaySeconds"] = testRunner.DelaySeconds.ValueInt64()
		}
		if !testRunner.Message.IsNull() && !testRunner.Message.IsUnknown() && testRunner.Message.ValueString() != "" {
			cfg["message"] = testRunner.Message.ValueString()
		}
		if !testRunner.Status.IsNull() && !testRunner.Status.IsUnknown() && testRunner.Status.ValueString() != "" {
			cfg["status"] = testRunner.Status.ValueString()
		}
		return "testrunner", &cfg, nil
	default:
		return "", nil, nil
	}
}

func setJobAgentBlocksFromAPI(data *JobAgentResourceModel, jobType string, config map[string]interface{}) {
	data.ArgoCD = nil
	data.GitHub = nil
	data.TerraformCloud = nil
	data.TestRunner = nil
	data.Custom = nil

	switch jobType {
	case "argocd":
		data.ArgoCD = []JobAgentArgoCDModel{
			{
				ApiKey:    types.StringValue(fmt.Sprint(config["apiKey"])),
				ServerUrl: types.StringValue(fmt.Sprint(config["serverUrl"])),
				Template:  types.StringValue(fmt.Sprint(config["template"])),
			},
		}
	case "github":
		github := JobAgentGitHubModel{
			InstallationId: types.Int64Value(toInt64(config["installationId"])),
			Owner:          types.StringValue(fmt.Sprint(config["owner"])),
			Repo:           types.StringValue(fmt.Sprint(config["repo"])),
			WorkflowId:     types.Int64Value(toInt64(config["workflowId"])),
			Ref:            types.StringNull(),
		}
		if ref, ok := config["ref"]; ok && ref != nil && fmt.Sprint(ref) != "" {
			github.Ref = types.StringValue(fmt.Sprint(ref))
		}
		data.GitHub = []JobAgentGitHubModel{github}
	case "terraformcloud":
		data.TerraformCloud = []JobAgentTFCModel{
			{
				Address:      types.StringValue(fmt.Sprint(config["address"])),
				Organization: types.StringValue(fmt.Sprint(config["organization"])),
				Template:     types.StringValue(fmt.Sprint(config["template"])),
				Token:        types.StringValue(fmt.Sprint(config["token"])),
			},
		}
	case "testrunner":
		testRunner := JobAgentTestRunnerModel{
			DelaySeconds: types.Int64Null(),
			Message:      types.StringNull(),
			Status:       types.StringNull(),
		}
		if delay, ok := config["delaySeconds"]; ok && delay != nil {
			testRunner.DelaySeconds = types.Int64Value(toInt64(delay))
		}
		if msg, ok := config["message"]; ok && msg != nil && fmt.Sprint(msg) != "" {
			testRunner.Message = types.StringValue(fmt.Sprint(msg))
		}
		if status, ok := config["status"]; ok && status != nil && fmt.Sprint(status) != "" {
			testRunner.Status = types.StringValue(fmt.Sprint(status))
		}
		data.TestRunner = []JobAgentTestRunnerModel{testRunner}
	default:
		data.Custom = []JobAgentCustomModel{
			{
				Type:   types.StringValue(jobType),
				Config: interfaceMapStringValue(config),
			},
		}
	}
}

func toInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

