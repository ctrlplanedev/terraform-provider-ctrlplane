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
				Description: "Input definitions for the workflow template",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the input",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of the input (string, number, or boolean)",
						},
						"default_string": schema.StringAttribute{
							Optional:    true,
							Description: "Default value for a string input",
						},
						"default_number": schema.Float64Attribute{
							Optional:    true,
							Description: "Default value for a number input",
						},
						"default_boolean": schema.BoolAttribute{
							Optional:    true,
							Description: "Default value for a boolean input",
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
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the job",
						},
						"ref": schema.StringAttribute{
							Required:    true,
							Description: "Reference to the job agent",
						},
						"config": schema.MapAttribute{
							Required:    true,
							Description: "Configuration for the job agent",
							ElementType: types.StringType,
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
	Name           types.String  `tfsdk:"name"`
	Type           types.String  `tfsdk:"type"`
	DefaultString  types.String  `tfsdk:"default_string"`
	DefaultNumber  types.Float64 `tfsdk:"default_number"`
	DefaultBoolean types.Bool    `tfsdk:"default_boolean"`
}

type WorkflowTemplateJobTemplateModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Ref    types.String `tfsdk:"ref"`
	Config types.Map    `tfsdk:"config"`
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

	jobs := workflowJobsFromModel(data.Jobs)

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

	setWorkflowTemplateModelFromAPI(&data, wt)
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

	setWorkflowTemplateModelFromAPI(&data, getResp.JSON200)
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

	jobs := workflowJobsFromModel(data.Jobs)

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

	setWorkflowTemplateModelFromAPI(&data, updateResp.JSON202)
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
		switch input.Type.ValueString() {
		case "string":
			si := api.WorkflowStringInput{
				Name: input.Name.ValueString(),
				Type: api.String,
			}
			if !input.DefaultString.IsNull() && !input.DefaultString.IsUnknown() {
				v := input.DefaultString.ValueString()
				si.Default = &v
			}
			if err := wi.FromWorkflowStringInput(si); err != nil {
				return nil, fmt.Errorf("failed to build string input '%s': %w", input.Name.ValueString(), err)
			}
		case "number":
			ni := api.WorkflowNumberInput{
				Name: input.Name.ValueString(),
				Type: api.Number,
			}
			if !input.DefaultNumber.IsNull() && !input.DefaultNumber.IsUnknown() {
				v := float32(input.DefaultNumber.ValueFloat64())
				ni.Default = &v
			}
			if err := wi.FromWorkflowNumberInput(ni); err != nil {
				return nil, fmt.Errorf("failed to build number input '%s': %w", input.Name.ValueString(), err)
			}
		case "boolean":
			bi := api.WorkflowBooleanInput{
				Name: input.Name.ValueString(),
				Type: api.Boolean,
			}
			if !input.DefaultBoolean.IsNull() && !input.DefaultBoolean.IsUnknown() {
				v := input.DefaultBoolean.ValueBool()
				bi.Default = &v
			}
			if err := wi.FromWorkflowBooleanInput(bi); err != nil {
				return nil, fmt.Errorf("failed to build boolean input '%s': %w", input.Name.ValueString(), err)
			}
		default:
			return nil, fmt.Errorf("unsupported input type '%s' for input '%s'", input.Type.ValueString(), input.Name.ValueString())
		}
		result = append(result, wi)
	}
	return result, nil
}

func workflowJobsFromModel(jobs []WorkflowTemplateJobTemplateModel) []api.CreateWorkflowJobTemplate {
	result := make([]api.CreateWorkflowJobTemplate, 0, len(jobs))
	for _, job := range jobs {
		config := make(map[string]interface{})
		if !job.Config.IsNull() && !job.Config.IsUnknown() {
			var decoded map[string]string
			diags := job.Config.ElementsAs(context.Background(), &decoded, false)
			if !diags.HasError() {
				for k, v := range decoded {
					config[k] = v
				}
			}
		}
		result = append(result, api.CreateWorkflowJobTemplate{
			Name:   job.Name.ValueString(),
			Ref:    job.Ref.ValueString(),
			Config: config,
		})
	}
	return result
}

func setWorkflowTemplateModelFromAPI(data *WorkflowTemplateResourceModel, wt *api.WorkflowTemplate) {
	data.ID = types.StringValue(wt.Id)
	data.Name = types.StringValue(wt.Name)

	inputs := make([]WorkflowTemplateInputModel, 0, len(wt.Inputs))
	for _, input := range wt.Inputs {
		m := WorkflowTemplateInputModel{
			DefaultString:  types.StringNull(),
			DefaultNumber:  types.Float64Null(),
			DefaultBoolean: types.BoolNull(),
		}

		if si, err := input.AsWorkflowStringInput(); err == nil && si.Type == api.String {
			m.Name = types.StringValue(si.Name)
			m.Type = types.StringValue("string")
			if si.Default != nil {
				m.DefaultString = types.StringValue(*si.Default)
			}
			inputs = append(inputs, m)
			continue
		}

		if ni, err := input.AsWorkflowNumberInput(); err == nil && ni.Type == api.Number {
			m.Name = types.StringValue(ni.Name)
			m.Type = types.StringValue("number")
			if ni.Default != nil {
				m.DefaultNumber = types.Float64Value(float64(*ni.Default))
			}
			inputs = append(inputs, m)
			continue
		}

		if bi, err := input.AsWorkflowBooleanInput(); err == nil && bi.Type == api.Boolean {
			m.Name = types.StringValue(bi.Name)
			m.Type = types.StringValue("boolean")
			if bi.Default != nil {
				m.DefaultBoolean = types.BoolValue(*bi.Default)
			}
			inputs = append(inputs, m)
			continue
		}
	}
	data.Inputs = inputs

	jobs := make([]WorkflowTemplateJobTemplateModel, 0, len(wt.Jobs))
	for _, job := range wt.Jobs {
		configMap := make(map[string]string, len(job.Config))
		for k, v := range job.Config {
			configMap[k] = fmt.Sprint(v)
		}
		tfConfig, _ := types.MapValueFrom(context.Background(), types.StringType, configMap)

		jobs = append(jobs, WorkflowTemplateJobTemplateModel{
			ID:     types.StringValue(job.Id),
			Name:   types.StringValue(job.Name),
			Ref:    types.StringValue(job.Ref),
			Config: tfConfig,
		})
	}
	data.Jobs = jobs
}
