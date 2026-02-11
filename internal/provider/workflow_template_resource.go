// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &WorkflowTemplateResource{}
var _ resource.ResourceWithImportState = &WorkflowTemplateResource{}
var _ resource.ResourceWithConfigure = &WorkflowTemplateResource{}

func NewWorkflowTemplateResource() resource.Resource {
	return &WorkflowTemplateResource{}
}

type WorkflowTemplateResource struct {
	workspace *api.WorkspaceClient
}

func (r *WorkflowTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_template"
}

func (r *WorkflowTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *WorkflowTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkflowTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a workflow template in Ctrlplane.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the workflow template",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the workflow template",
			},
		},
		Blocks: map[string]schema.Block{
			"input": schema.ListNestedBlock{
				Description: "Input definitions for the workflow template. Each input must define exactly one type block: string, number, or boolean.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The key of the input",
						},
					},
					Blocks: map[string]schema.Block{
						"string": schema.SingleNestedBlock{
							Description: "Defines a string type input. Mutually exclusive with number and boolean.",
							Attributes: map[string]schema.Attribute{
								"default": schema.StringAttribute{
									Optional:    true,
									Description: "Default value for the string input",
								},
							},
						},
						"number": schema.SingleNestedBlock{
							Description: "Defines a number type input. Mutually exclusive with string and boolean.",
							Attributes: map[string]schema.Attribute{
								"default": schema.Float64Attribute{
									Optional:    true,
									Description: "Default value for the number input",
								},
							},
						},
						"boolean": schema.SingleNestedBlock{
							Description: "Defines a boolean type input. Mutually exclusive with string and number.",
							Attributes: map[string]schema.Attribute{
								"default": schema.BoolAttribute{
									Optional:    true,
									Description: "Default value for the boolean input",
								},
							},
						},
					},
				},
			},
			"job": schema.ListNestedBlock{
				Description: "Job definitions for the workflow template",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the job template (assigned by the server)",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The key of the job",
						},
						"if": schema.StringAttribute{
							Optional:    true,
							Description: "CEL expression to determine if the job should run",
						},
					},
					Blocks: map[string]schema.Block{
						"agent": schema.SingleNestedBlock{
							Description: "Job agent configuration. Specifies which agent runs the job and its configuration.",
							Attributes: map[string]schema.Attribute{
								"ref": schema.StringAttribute{
									Required:    true,
									Description: "Reference to the job agent",
								},
								"config": schema.MapAttribute{
									Optional:    true,
									Description: "Generic configuration map for the job agent. Mutually exclusive with typed blocks (argocd, github, terraform_cloud, test_runner).",
									ElementType: types.StringType,
								},
							},
							Blocks: map[string]schema.Block{
								"argocd": schema.SingleNestedBlock{
									Description: "ArgoCD job agent configuration.",
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
									Description: "GitHub job agent configuration.",
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
											Description: "Git ref to run the workflow on",
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
									Description: "Terraform Cloud job agent configuration.",
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
									Description: "Test runner job agent configuration.",
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
			},
		},
	}
}

type WorkflowTemplateResourceModel struct {
	ID     types.String                       `tfsdk:"id"`
	Name   types.String                       `tfsdk:"name"`
	Inputs []WorkflowTemplateInputModel       `tfsdk:"input"`
	Jobs   []WorkflowTemplateJobTemplateModel `tfsdk:"job"`
}

type WorkflowTemplateInputModel struct {
	Key     types.String                       `tfsdk:"key"`
	String  *WorkflowTemplateInputStringModel  `tfsdk:"string"`
	Number  *WorkflowTemplateInputNumberModel  `tfsdk:"number"`
	Boolean *WorkflowTemplateInputBooleanModel `tfsdk:"boolean"`
}

type WorkflowTemplateInputStringModel struct {
	Default types.String `tfsdk:"default"`
}

type WorkflowTemplateInputNumberModel struct {
	Default types.Float64 `tfsdk:"default"`
}

type WorkflowTemplateInputBooleanModel struct {
	Default types.Bool `tfsdk:"default"`
}

type WorkflowTemplateJobTemplateModel struct {
	ID    types.String            `tfsdk:"id"`
	Key   types.String            `tfsdk:"key"`
	If    types.String            `tfsdk:"if"`
	Agent *WorkflowJobAgentModel  `tfsdk:"agent"`
}

type WorkflowJobAgentModel struct {
	Ref            types.String                         `tfsdk:"ref"`
	Config         types.Map                            `tfsdk:"config"`
	ArgoCD         *WorkflowJobAgentArgoCDModel         `tfsdk:"argocd"`
	GitHub         *WorkflowJobAgentGitHubModel         `tfsdk:"github"`
	TerraformCloud *WorkflowJobAgentTerraformCloudModel `tfsdk:"terraform_cloud"`
	TestRunner     *WorkflowJobAgentTestRunnerModel     `tfsdk:"test_runner"`
}

type WorkflowJobAgentArgoCDModel struct {
	ApiKey    types.String `tfsdk:"api_key"`
	ServerUrl types.String `tfsdk:"server_url"`
	Template  types.String `tfsdk:"template"`
}

type WorkflowJobAgentGitHubModel struct {
	InstallationId types.Int64  `tfsdk:"installation_id"`
	Owner          types.String `tfsdk:"owner"`
	Ref            types.String `tfsdk:"ref"`
	Repo           types.String `tfsdk:"repo"`
	WorkflowId     types.Int64  `tfsdk:"workflow_id"`
}

type WorkflowJobAgentTerraformCloudModel struct {
	Address      types.String `tfsdk:"address"`
	Organization types.String `tfsdk:"organization"`
	Template     types.String `tfsdk:"template"`
	Token        types.String `tfsdk:"token"`
}

type WorkflowJobAgentTestRunnerModel struct {
	DelaySeconds types.Int64  `tfsdk:"delay_seconds"`
	Message      types.String `tfsdk:"message"`
	Status       types.String `tfsdk:"status"`
}

func (r *WorkflowTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkflowTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inputs, err := workflowInputsFromModel(data.Inputs)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build workflow inputs", err.Error())
		return
	}

	jobs, jobDiags := workflowJobsFromModel(ctx, data.Jobs)
	resp.Diagnostics.Append(jobDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := api.CreateWorkflowTemplateJSONRequestBody{
		Name:   data.Name.ValueString(),
		Inputs: inputs,
		Jobs:   jobs,
	}

	createResp, err := r.workspace.Client.CreateWorkflowTemplateWithResponse(
		ctx, r.workspace.ID.String(), body,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create workflow template", err.Error())
		return
	}

	if createResp.StatusCode() != http.StatusAccepted && createResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError("Failed to create workflow template", formatResponseError(createResp.StatusCode(), createResp.Body))
		return
	}

	var wt *api.WorkflowTemplate
	if createResp.JSON202 != nil {
		wt = createResp.JSON202
	}
	if wt == nil {
		// Try to parse the response body directly (e.g. 201)
		var parsed api.WorkflowTemplate
		if jsonErr := json.Unmarshal(createResp.Body, &parsed); jsonErr != nil {
			resp.Diagnostics.AddError("Failed to create workflow template", "Empty response from server")
			return
		}
		wt = &parsed
	}

	resp.Diagnostics.Append(setWorkflowTemplateModelFromAPI(ctx, &data, wt)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkflowTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.workspace.Client.GetWorkflowTemplateWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read workflow template",
			fmt.Sprintf("Failed to read workflow template with ID '%s': %s", data.ID.ValueString(), err.Error()))
		return
	}

	switch getResp.StatusCode() {
	case http.StatusOK:
		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read workflow template", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read workflow template", formatResponseError(getResp.StatusCode(), getResp.Body))
		return
	}

	resp.Diagnostics.Append(setWorkflowTemplateModelFromAPI(ctx, &data, getResp.JSON200)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkflowTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inputs, err := workflowInputsFromModel(data.Inputs)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build workflow inputs", err.Error())
		return
	}

	jobs, jobDiags := workflowJobsFromModel(ctx, data.Jobs)
	resp.Diagnostics.Append(jobDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := api.UpdateWorkflowTemplateJSONRequestBody{
		Name:   data.Name.ValueString(),
		Inputs: inputs,
		Jobs:   jobs,
	}

	updateResp, err := r.workspace.Client.UpdateWorkflowTemplateWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(), body,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update workflow template",
			fmt.Sprintf("Failed to update workflow template with ID '%s': %s", data.ID.ValueString(), err.Error()))
		return
	}

	if updateResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update workflow template", formatResponseError(updateResp.StatusCode(), updateResp.Body))
		return
	}

	if updateResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update workflow template", "Empty response from server")
		return
	}

	resp.Diagnostics.Append(setWorkflowTemplateModelFromAPI(ctx, &data, updateResp.JSON202)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkflowTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.workspace.Client.DeleteWorkflowTemplateWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete workflow template",
			fmt.Sprintf("Failed to delete workflow template: %s", err.Error()))
		return
	}

	switch deleteResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusNotFound:
		return
	default:
		resp.Diagnostics.AddError("Failed to delete workflow template", formatResponseError(deleteResp.StatusCode(), deleteResp.Body))
	}
}

func workflowInputsFromModel(inputs []WorkflowTemplateInputModel) ([]api.WorkflowInput, error) {
	result := make([]api.WorkflowInput, 0, len(inputs))
	for _, input := range inputs {
		var wi api.WorkflowInput
		key := input.Key.ValueString()

		count := 0
		if input.String != nil {
			count++
		}
		if input.Number != nil {
			count++
		}
		if input.Boolean != nil {
			count++
		}

		if count == 0 {
			return nil, fmt.Errorf("input '%s' must define exactly one of: string, number, or boolean block", key)
		}
		if count > 1 {
			return nil, fmt.Errorf("input '%s' must define exactly one of: string, number, or boolean block, but %d were defined", key, count)
		}

		switch {
		case input.String != nil:
			si := api.WorkflowStringInput{
				Name: key,
				Type: api.String,
			}
			if !input.String.Default.IsNull() && !input.String.Default.IsUnknown() {
				v := input.String.Default.ValueString()
				si.Default = &v
			}
			if err := wi.FromWorkflowStringInput(si); err != nil {
				return nil, fmt.Errorf("failed to build string input '%s': %w", key, err)
			}
		case input.Number != nil:
			ni := api.WorkflowNumberInput{
				Name: key,
				Type: api.Number,
			}
			if !input.Number.Default.IsNull() && !input.Number.Default.IsUnknown() {
				v := float32(input.Number.Default.ValueFloat64())
				ni.Default = &v
			}
			if err := wi.FromWorkflowNumberInput(ni); err != nil {
				return nil, fmt.Errorf("failed to build number input '%s': %w", key, err)
			}
		case input.Boolean != nil:
			bi := api.WorkflowBooleanInput{
				Name: key,
				Type: api.Boolean,
			}
			if !input.Boolean.Default.IsNull() && !input.Boolean.Default.IsUnknown() {
				v := input.Boolean.Default.ValueBool()
				bi.Default = &v
			}
			if err := wi.FromWorkflowBooleanInput(bi); err != nil {
				return nil, fmt.Errorf("failed to build boolean input '%s': %w", key, err)
			}
		}

		result = append(result, wi)
	}
	return result, nil
}

func workflowJobAgentConfigFromModel(ctx context.Context, agent *WorkflowJobAgentModel) (map[string]interface{}, diag.Diagnostics) {
	if agent == nil {
		return make(map[string]interface{}), nil
	}

	// Count how many config sources are set
	count := 0
	if !agent.Config.IsNull() && !agent.Config.IsUnknown() {
		count++
	}
	if agent.ArgoCD != nil {
		count++
	}
	if agent.GitHub != nil {
		count++
	}
	if agent.TerraformCloud != nil {
		count++
	}
	if agent.TestRunner != nil {
		count++
	}

	if count > 1 {
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic(
				"Invalid agent configuration",
				"Agent must use at most one of: config, argocd, github, terraform_cloud, or test_runner.",
			),
		}
	}

	config := make(map[string]interface{})

	switch {
	case agent.ArgoCD != nil:
		if !agent.ArgoCD.ApiKey.IsNull() && !agent.ArgoCD.ApiKey.IsUnknown() {
			config["apiKey"] = agent.ArgoCD.ApiKey.ValueString()
		}
		if !agent.ArgoCD.ServerUrl.IsNull() && !agent.ArgoCD.ServerUrl.IsUnknown() {
			config["serverUrl"] = agent.ArgoCD.ServerUrl.ValueString()
		}
		if !agent.ArgoCD.Template.IsNull() && !agent.ArgoCD.Template.IsUnknown() {
			config["template"] = agent.ArgoCD.Template.ValueString()
		}
	case agent.GitHub != nil:
		if !agent.GitHub.InstallationId.IsNull() && !agent.GitHub.InstallationId.IsUnknown() {
			config["installationId"] = agent.GitHub.InstallationId.ValueInt64()
		}
		if !agent.GitHub.Owner.IsNull() && !agent.GitHub.Owner.IsUnknown() {
			config["owner"] = agent.GitHub.Owner.ValueString()
		}
		if !agent.GitHub.Repo.IsNull() && !agent.GitHub.Repo.IsUnknown() {
			config["repo"] = agent.GitHub.Repo.ValueString()
		}
		if !agent.GitHub.WorkflowId.IsNull() && !agent.GitHub.WorkflowId.IsUnknown() {
			config["workflowId"] = agent.GitHub.WorkflowId.ValueInt64()
		}
		if !agent.GitHub.Ref.IsNull() && !agent.GitHub.Ref.IsUnknown() {
			config["ref"] = agent.GitHub.Ref.ValueString()
		}
	case agent.TerraformCloud != nil:
		if !agent.TerraformCloud.Address.IsNull() && !agent.TerraformCloud.Address.IsUnknown() {
			config["address"] = agent.TerraformCloud.Address.ValueString()
		}
		if !agent.TerraformCloud.Organization.IsNull() && !agent.TerraformCloud.Organization.IsUnknown() {
			config["organization"] = agent.TerraformCloud.Organization.ValueString()
		}
		if !agent.TerraformCloud.Template.IsNull() && !agent.TerraformCloud.Template.IsUnknown() {
			config["template"] = agent.TerraformCloud.Template.ValueString()
		}
		if !agent.TerraformCloud.Token.IsNull() && !agent.TerraformCloud.Token.IsUnknown() {
			config["token"] = agent.TerraformCloud.Token.ValueString()
		}
	case agent.TestRunner != nil:
		if !agent.TestRunner.DelaySeconds.IsNull() && !agent.TestRunner.DelaySeconds.IsUnknown() {
			config["delaySeconds"] = agent.TestRunner.DelaySeconds.ValueInt64()
		}
		if !agent.TestRunner.Message.IsNull() && !agent.TestRunner.Message.IsUnknown() {
			config["message"] = agent.TestRunner.Message.ValueString()
		}
		if !agent.TestRunner.Status.IsNull() && !agent.TestRunner.Status.IsUnknown() {
			config["status"] = agent.TestRunner.Status.ValueString()
		}
	default:
		// Generic config map fallback
		if !agent.Config.IsNull() && !agent.Config.IsUnknown() {
			var decoded map[string]string
			diags := agent.Config.ElementsAs(ctx, &decoded, false)
			if diags.HasError() {
				return nil, diags
			}
			for k, v := range decoded {
				config[k] = v
			}
		}
	}

	return config, nil
}

func workflowJobsFromModel(ctx context.Context, jobs []WorkflowTemplateJobTemplateModel) ([]api.CreateWorkflowJobTemplate, diag.Diagnostics) {
	result := make([]api.CreateWorkflowJobTemplate, 0, len(jobs))
	for _, job := range jobs {
		config, diags := workflowJobAgentConfigFromModel(ctx, job.Agent)
		if diags.HasError() {
			return nil, diags
		}

		ref := ""
		if job.Agent != nil {
			ref = job.Agent.Ref.ValueString()
		}

		jt := api.CreateWorkflowJobTemplate{
			Name:   job.Key.ValueString(),
			Ref:    ref,
			Config: config,
		}
		if !job.If.IsNull() && !job.If.IsUnknown() {
			v := job.If.ValueString()
			jt.If = &v
		}
		result = append(result, jt)
	}
	return result, nil
}

func setWorkflowTemplateModelFromAPI(ctx context.Context, data *WorkflowTemplateResourceModel, wt *api.WorkflowTemplate) diag.Diagnostics {
	data.ID = types.StringValue(wt.Id)
	data.Name = types.StringValue(wt.Name)

	inputs := make([]WorkflowTemplateInputModel, 0, len(wt.Inputs))
	for _, input := range wt.Inputs {
		if si, err := input.AsWorkflowStringInput(); err == nil && si.Type == api.String {
			m := WorkflowTemplateInputModel{
				Key:    types.StringValue(si.Name),
				String: &WorkflowTemplateInputStringModel{Default: types.StringNull()},
			}
			if si.Default != nil {
				m.String.Default = types.StringValue(*si.Default)
			}
			inputs = append(inputs, m)
			continue
		}

		if ni, err := input.AsWorkflowNumberInput(); err == nil && ni.Type == api.Number {
			m := WorkflowTemplateInputModel{
				Key:    types.StringValue(ni.Name),
				Number: &WorkflowTemplateInputNumberModel{Default: types.Float64Null()},
			}
			if ni.Default != nil {
				m.Number.Default = types.Float64Value(float64(*ni.Default))
			}
			inputs = append(inputs, m)
			continue
		}

		if bi, err := input.AsWorkflowBooleanInput(); err == nil && bi.Type == api.Boolean {
			m := WorkflowTemplateInputModel{
				Key:     types.StringValue(bi.Name),
				Boolean: &WorkflowTemplateInputBooleanModel{Default: types.BoolNull()},
			}
			if bi.Default != nil {
				m.Boolean.Default = types.BoolValue(*bi.Default)
			}
			inputs = append(inputs, m)
			continue
		}

		// Unrecognized input type — surface as a warning so the caller
		// knows data was skipped rather than silently losing it.
		inputJSON, _ := input.MarshalJSON()
		return diag.Diagnostics{
			diag.NewWarningDiagnostic(
				"Unknown workflow input type",
				fmt.Sprintf("Could not determine the type for input at index %d; raw value: %s. "+
					"This input will be omitted from state.", len(inputs), string(inputJSON)),
			),
		}
	}
	data.Inputs = inputs

	jobs := make([]WorkflowTemplateJobTemplateModel, 0, len(wt.Jobs))
	for _, job := range wt.Jobs {
		ifVal := types.StringNull()
		if job.If != nil {
			ifVal = types.StringValue(*job.If)
		}

		agent := &WorkflowJobAgentModel{
			Ref:    types.StringValue(job.Ref),
			Config: types.MapNull(types.StringType),
		}
		setWorkflowJobAgentBlocksFromConfig(agent, job.Config)

		jm := WorkflowTemplateJobTemplateModel{
			ID:    types.StringValue(job.Id),
			Key:   types.StringValue(job.Name),
			If:    ifVal,
			Agent: agent,
		}

		jobs = append(jobs, jm)
	}
	data.Jobs = jobs

	return nil
}

func setWorkflowJobAgentBlocksFromConfig(agent *WorkflowJobAgentModel, config map[string]interface{}) {
	agent.ArgoCD = nil
	agent.GitHub = nil
	agent.TerraformCloud = nil
	agent.TestRunner = nil

	if len(config) == 0 {
		return
	}

	// Detect GitHub by known keys
	if configHasAny(config, "installationId", "workflowId", "owner", "repo") {
		gh := WorkflowJobAgentGitHubModel{
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
		if v, ok := config["workflowId"]; ok && v != nil {
			gh.WorkflowId = types.Int64Value(toInt64(v))
		}
		if v, ok := config["ref"]; ok && v != nil && fmt.Sprint(v) != "" {
			gh.Ref = types.StringValue(fmt.Sprint(v))
		}
		agent.GitHub = &gh
		return
	}

	// Detect ArgoCD by known keys
	if configHasAny(config, "apiKey", "serverUrl") {
		agent.ArgoCD = &WorkflowJobAgentArgoCDModel{
			ApiKey:    stringValueOrNull(config["apiKey"]),
			ServerUrl: stringValueOrNull(config["serverUrl"]),
			Template:  stringValueOrNull(config["template"]),
		}
		return
	}

	// Detect Terraform Cloud by known keys
	if configHasAny(config, "address", "organization", "token") {
		agent.TerraformCloud = &WorkflowJobAgentTerraformCloudModel{
			Address:      stringValueOrNull(config["address"]),
			Organization: stringValueOrNull(config["organization"]),
			Template:     stringValueOrNull(config["template"]),
			Token:        stringValueOrNull(config["token"]),
		}
		return
	}

	// Detect Test Runner by known keys
	if configHasAny(config, "delaySeconds", "message", "status") {
		tr := WorkflowJobAgentTestRunnerModel{
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
		agent.TestRunner = &tr
		return
	}

	// Fallback: generic config map
	configMap := make(map[string]string, len(config))
	for k, v := range config {
		if s, ok := v.(string); ok {
			configMap[k] = s
			continue
		}
		b, _ := json.Marshal(v)
		configMap[k] = string(b)
	}
	tfConfig, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)
	agent.Config = tfConfig
}
