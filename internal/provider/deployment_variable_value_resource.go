// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &DeploymentVariableValueResource{}
var _ resource.ResourceWithImportState = &DeploymentVariableValueResource{}
var _ resource.ResourceWithConfigure = &DeploymentVariableValueResource{}
var _ resource.ResourceWithValidateConfig = &DeploymentVariableValueResource{}

func NewDeploymentVariableValueResource() resource.Resource {
	return &DeploymentVariableValueResource{}
}

type DeploymentVariableValueResource struct {
	workspace *api.WorkspaceClient
}

type DeploymentVariableValueResourceModel struct {
	ID               types.String  `tfsdk:"id"`
	DeploymentId     types.String  `tfsdk:"deployment_id"`
	VariableId       types.String  `tfsdk:"variable_id"`
	Priority         types.Int64   `tfsdk:"priority"`
	ResourceSelector types.String  `tfsdk:"resource_selector"`
	LiteralValue     types.Dynamic `tfsdk:"literal_value"`
	ReferenceValue   types.Object  `tfsdk:"reference_value"`
}

var referenceValueAttrTypes = map[string]attr.Type{
	"reference": types.StringType,
	"path":      types.ListType{ElemType: types.StringType},
}

func (r *DeploymentVariableValueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_variable_value"
}

func (r *DeploymentVariableValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DeploymentVariableValueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentVariableValueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deployment variable value override in Ctrlplane. A variable value provides a specific value for a deployment variable, optionally scoped to resources matching a selector expression.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the deployment variable value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The deployment ID this variable value belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"variable_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The deployment variable ID this value belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"priority": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The priority of the variable value. Higher priority values take precedence when multiple values match.",
			},
			"resource_selector": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A CEL expression to select which resources this value applies to.",
			},
			"literal_value": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: "A literal value (string, number, boolean, or object). Conflicts with `reference_value`.",
			},
		},
		Blocks: map[string]schema.Block{
			"reference_value": schema.SingleNestedBlock{
				MarkdownDescription: "A reference value pointing to a property on the matched resource. Conflicts with `literal_value`.",
				Attributes: map[string]schema.Attribute{
					"reference": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "The reference key.",
					},
					"path": schema.ListAttribute{
						Required:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "The path segments to the value in the referenced resource.",
					},
				},
			},
		},
	}
}

func (r *DeploymentVariableValueResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data DeploymentVariableValueResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasLiteral := !data.LiteralValue.IsNull() && !data.LiteralValue.IsUnknown()
	hasReference := !data.ReferenceValue.IsNull() && !data.ReferenceValue.IsUnknown()

	if hasLiteral && hasReference {
		resp.Diagnostics.AddAttributeError(
			path.Root("literal_value"),
			"Conflicting value types",
			"Only one of literal_value or reference_value may be specified, not both.",
		)
	}

	if !hasLiteral && !hasReference {
		// Allow unknowns during plan - only error if both are definitively null
		if !data.LiteralValue.IsUnknown() && !data.ReferenceValue.IsUnknown() {
			resp.Diagnostics.AddError(
				"Missing value",
				"Exactly one of literal_value or reference_value must be specified.",
			)
		}
	}
}

func (r *DeploymentVariableValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentVariableValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueID := data.ID.ValueString()
	if data.ID.IsNull() || data.ID.IsUnknown() || valueID == "" {
		valueID = uuid.NewString()
		data.ID = types.StringValue(valueID)
	}

	apiValue, err := valueFromVariableValueModel(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment variable value", fmt.Sprintf("Failed to build value: %s", err.Error()))
		return
	}

	selector, err := selectorPointerFromString(data.ResourceSelector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment variable value", fmt.Sprintf("Failed to parse resource_selector: %s", err.Error()))
		return
	}

	requestBody := api.UpsertDeploymentVariableValueRequest{
		Priority:         data.Priority.ValueInt64(),
		ResourceSelector: selector,
		Value:            *apiValue,
	}

	valueResp, err := r.workspace.Client.RequestDeploymentVariableValueUpdateWithResponse(
		ctx, r.workspace.ID.String(), data.DeploymentId.ValueString(), data.VariableId.ValueString(), valueID, requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment variable value", err.Error())
		return
	}

	if valueResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create deployment variable value", formatResponseError(valueResp.StatusCode(), valueResp.Body))
		return
	}

	if valueResp.JSON202 == nil || valueResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to create deployment variable value", "Empty value ID in response")
		return
	}

	data.ID = types.StringValue(valueResp.JSON202.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentVariableValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentVariableValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueResp, err := r.workspace.Client.GetDeploymentVariableValueWithResponse(
		ctx, r.workspace.ID.String(), data.DeploymentId.ValueString(), data.VariableId.ValueString(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read deployment variable value",
			fmt.Sprintf("Failed to read deployment variable value with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	switch valueResp.StatusCode() {
	case http.StatusOK:
		if valueResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read deployment variable value", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		if valueResp.JSON400 != nil && valueResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to read deployment variable value", fmt.Sprintf("Bad request: %s", *valueResp.JSON400.Error))
			return
		}
		resp.Diagnostics.AddError("Failed to read deployment variable value", "Bad request")
		return
	}

	if valueResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read deployment variable value", formatResponseError(valueResp.StatusCode(), valueResp.Body))
		return
	}

	value := valueResp.JSON200
	data.ID = types.StringValue(value.Id)
	data.VariableId = types.StringValue(value.DeploymentVariableId)
	data.Priority = types.Int64Value(value.Priority)

	selectorStr, err := selectorStringValue(value.ResourceSelector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read deployment variable value", fmt.Sprintf("Failed to parse resource_selector: %s", err.Error()))
		return
	}
	data.ResourceSelector = selectorStr

	diags := setValueOnModel(ctx, &data, value.Value)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentVariableValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeploymentVariableValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiValue, err := valueFromVariableValueModel(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update deployment variable value", fmt.Sprintf("Failed to build value: %s", err.Error()))
		return
	}

	selector, err := selectorPointerFromString(data.ResourceSelector)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update deployment variable value", fmt.Sprintf("Failed to parse resource_selector: %s", err.Error()))
		return
	}

	requestBody := api.UpsertDeploymentVariableValueRequest{
		Priority:         data.Priority.ValueInt64(),
		ResourceSelector: selector,
		Value:            *apiValue,
	}

	valueResp, err := r.workspace.Client.RequestDeploymentVariableValueUpdateWithResponse(
		ctx, r.workspace.ID.String(), data.DeploymentId.ValueString(), data.VariableId.ValueString(), data.ID.ValueString(), requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update deployment variable value",
			fmt.Sprintf("Failed to update deployment variable value with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if valueResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update deployment variable value", formatResponseError(valueResp.StatusCode(), valueResp.Body))
		return
	}

	if valueResp.JSON202 == nil || valueResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to update deployment variable value", "Empty value ID in response")
		return
	}

	data.ID = types.StringValue(valueResp.JSON202.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentVariableValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentVariableValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueResp, err := r.workspace.Client.RequestDeploymentVariableValueDeletionWithResponse(
		ctx, r.workspace.ID.String(), data.DeploymentId.ValueString(), data.VariableId.ValueString(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete deployment variable value", fmt.Sprintf("Failed to delete deployment variable value: %s", err.Error()))
		return
	}

	switch valueResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusBadRequest:
		if valueResp.JSON400 != nil && valueResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to delete deployment variable value", fmt.Sprintf("Bad request: %s", *valueResp.JSON400.Error))
			return
		}
	case http.StatusNotFound:
		if valueResp.JSON404 != nil && valueResp.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to delete deployment variable value", fmt.Sprintf("Not found: %s", *valueResp.JSON404.Error))
			return
		}
	}

	resp.Diagnostics.AddError("Failed to delete deployment variable value", formatResponseError(valueResp.StatusCode(), valueResp.Body))
}

// valueFromVariableValueModel converts the Terraform model into the API Value union type.
func valueFromVariableValueModel(data DeploymentVariableValueResourceModel) (*api.Value, error) {
	var value api.Value

	if !data.ReferenceValue.IsNull() && !data.ReferenceValue.IsUnknown() {
		refAttrs := data.ReferenceValue.Attributes()

		referenceAttr, ok := refAttrs["reference"]
		if !ok {
			return nil, fmt.Errorf("reference_value is missing 'reference' attribute")
		}
		reference, ok := referenceAttr.(types.String)
		if !ok {
			return nil, fmt.Errorf("reference_value.reference is not a string")
		}

		pathAttr, ok := refAttrs["path"]
		if !ok {
			return nil, fmt.Errorf("reference_value is missing 'path' attribute")
		}
		pathList, ok := pathAttr.(types.List)
		if !ok {
			return nil, fmt.Errorf("reference_value.path is not a list")
		}

		var pathStrings []string
		diags := pathList.ElementsAs(context.Background(), &pathStrings, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to convert reference_value.path to []string")
		}

		if err := value.FromReferenceValue(api.ReferenceValue{
			Reference: reference.ValueString(),
			Path:      pathStrings,
		}); err != nil {
			return nil, fmt.Errorf("failed to set reference value: %w", err)
		}

		return &value, nil
	}

	if !data.LiteralValue.IsNull() && !data.LiteralValue.IsUnknown() {
		literal, err := literalValueFromDynamic(data.LiteralValue)
		if err != nil {
			return nil, fmt.Errorf("failed to convert literal value: %w", err)
		}
		if literal == nil {
			return nil, fmt.Errorf("literal_value resolved to nil")
		}

		if err := value.FromLiteralValue(*literal); err != nil {
			return nil, fmt.Errorf("failed to set literal value: %w", err)
		}

		return &value, nil
	}

	return nil, fmt.Errorf("one of literal_value or reference_value must be provided")
}

// setValueOnModel reads from the API Value union and sets the appropriate field on the model.
func setValueOnModel(_ context.Context, data *DeploymentVariableValueResourceModel, value api.Value) diag.Diagnostics {
	var diags diag.Diagnostics

	// Try reference value first
	if refVal, err := value.AsReferenceValue(); err == nil && refVal.Reference != "" {
		pathElements := make([]attr.Value, len(refVal.Path))
		for i, p := range refVal.Path {
			pathElements[i] = types.StringValue(p)
		}

		pathList, listDiags := types.ListValue(types.StringType, pathElements)
		if listDiags.HasError() {
			diags.Append(listDiags...)
			return diags
		}

		refObj, objDiags := types.ObjectValue(referenceValueAttrTypes, map[string]attr.Value{
			"reference": types.StringValue(refVal.Reference),
			"path":      pathList,
		})
		if objDiags.HasError() {
			diags.Append(objDiags...)
			return diags
		}

		data.ReferenceValue = refObj
		data.LiteralValue = types.DynamicNull()
		return diags
	}

	// Try literal value
	if litVal, err := value.AsLiteralValue(); err == nil {
		data.LiteralValue = literalValueToDynamic(&litVal)
		data.ReferenceValue = types.ObjectNull(referenceValueAttrTypes)
		return diags
	}

	// Unknown value type - set both to null
	data.LiteralValue = types.DynamicNull()
	data.ReferenceValue = types.ObjectNull(referenceValueAttrTypes)
	return diags
}
