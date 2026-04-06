// Copyright IBM Corp. 2021, 2026

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &VariableSetResource{}
var _ resource.ResourceWithImportState = &VariableSetResource{}
var _ resource.ResourceWithConfigure = &VariableSetResource{}

func NewVariableSetResource() resource.Resource {
	return &VariableSetResource{}
}

type VariableSetResource struct {
	workspace *api.WorkspaceClient
}

type VariableSetResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Selector    types.String `tfsdk:"selector"`
	Priority    types.Int64  `tfsdk:"priority"`
	Variables   types.List   `tfsdk:"variables"`
}

type VariableSetVariableModel struct {
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
	Sensitive      types.Bool   `tfsdk:"sensitive"`
	ReferenceValue types.Object `tfsdk:"reference_value"`
}

var variableSetVariableAttrTypes = map[string]attr.Type{
	"key":       types.StringType,
	"value":     types.StringType,
	"sensitive": types.BoolType,
	"reference_value": types.ObjectType{
		AttrTypes: referenceValueAttrTypes,
	},
}

func (r *VariableSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable_set"
}

func (r *VariableSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VariableSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VariableSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a variable set in Ctrlplane. Variable sets allow you to define groups of variables that are applied to release targets matching a selector expression.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the variable set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the variable set.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "A description of the variable set.",
			},
			"selector": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A CEL expression to select which release targets this variable set applies to.",
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "The priority of the variable set. Higher priority sets take precedence.",
			},
			"variables": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "The variables in this variable set.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The key of the variable.",
						},
						"value": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The literal value as a string. Numbers, booleans, and JSON objects are also accepted and will be sent with their appropriate types. Conflicts with `reference_value`.",
						},
						"sensitive": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Whether the value is sensitive. When true, the value is stored as a sensitive value. Conflicts with `reference_value`.",
						},
						"reference_value": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "A reference value pointing to a property on the matched resource. Conflicts with `value`.",
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
				},
			},
		},
	}
}

func (r *VariableSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VariableSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables, diags := vsVariablesFromModel(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	priority := int(data.Priority.ValueInt64())
	requestBody := api.CreateVariableSetJSONRequestBody{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Selector:    normalizeCEL(data.Selector),
		Priority:    &priority,
		Variables:   variables,
	}

	createResp, err := r.workspace.Client.CreateVariableSetWithResponse(
		ctx, r.workspace.ID.String(), requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create variable set", err.Error())
		return
	}

	if createResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError("Failed to create variable set", formatResponseError(createResp.StatusCode(), createResp.Body))
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create variable set", "Empty response from server")
		return
	}

	data.ID = types.StringValue(createResp.JSON201.Id.String())

	err = waitForResource(ctx, func() (bool, error) {
		getResp, err := r.workspace.Client.GetVariableSetWithResponse(ctx, r.workspace.ID.String(), createResp.JSON201.Id.String())
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
		resp.Diagnostics.AddError("Failed to create variable set", fmt.Sprintf("Resource not available after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *VariableSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VariableSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.workspace.Client.GetVariableSetWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read variable set", fmt.Sprintf("Failed to read variable set with ID '%s': %s", data.ID.ValueString(), err.Error()))
		return
	}

	switch getResp.StatusCode() {
	case http.StatusOK:
		if getResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read variable set", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		if getResp.JSON400 != nil && getResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to read variable set", fmt.Sprintf("Bad request: %s", *getResp.JSON400.Error))
			return
		}
		resp.Diagnostics.AddError("Failed to read variable set", "Bad request")
		return
	}

	if getResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read variable set", formatResponseError(getResp.StatusCode(), getResp.Body))
		return
	}

	vs := getResp.JSON200
	data.ID = types.StringValue(vs.Id.String())
	data.Name = types.StringValue(vs.Name)
	data.Description = descriptionValue(&vs.Description)
	data.Selector = types.StringValue(vs.Selector)
	data.Priority = types.Int64Value(int64(vs.Priority))

	varList, diags := vsVariablesToModel(vs.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Variables = varList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VariableSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables, diags := vsVariablesFromModel(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	priority := int(data.Priority.ValueInt64())
	selector := normalizeCEL(data.Selector)

	requestBody := api.UpdateVariableSetJSONRequestBody{
		Name:        &name,
		Description: data.Description.ValueStringPointer(),
		Selector:    &selector,
		Priority:    &priority,
		Variables:   &variables,
	}

	updateResp, err := r.workspace.Client.UpdateVariableSetWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(), requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update variable set", fmt.Sprintf("Failed to update variable set with ID '%s': %s", data.ID.ValueString(), err.Error()))
		return
	}

	if updateResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update variable set", formatResponseError(updateResp.StatusCode(), updateResp.Body))
		return
	}

	if updateResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update variable set", "Empty response from server")
		return
	}

	data.ID = types.StringValue(updateResp.JSON202.Id.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *VariableSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VariableSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.workspace.Client.DeleteVariableSetWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete variable set", fmt.Sprintf("Failed to delete variable set: %s", err.Error()))
		return
	}

	switch deleteResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusBadRequest:
		if deleteResp.JSON400 != nil && deleteResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to delete variable set", fmt.Sprintf("Bad request: %s", *deleteResp.JSON400.Error))
			return
		}
	case http.StatusNotFound:
		if deleteResp.JSON404 != nil && deleteResp.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to delete variable set", fmt.Sprintf("Not found: %s", *deleteResp.JSON404.Error))
			return
		}
	}

	resp.Diagnostics.AddError("Failed to delete variable set", formatResponseError(deleteResp.StatusCode(), deleteResp.Body))
}

// vsVariablesFromModel converts the Terraform list of variables into API VariableSetVariable slice.
func vsVariablesFromModel(ctx context.Context, data VariableSetResourceModel) ([]api.VariableSetVariable, diag.Diagnostics) {
	var diags diag.Diagnostics

	if data.Variables.IsNull() || data.Variables.IsUnknown() {
		return []api.VariableSetVariable{}, diags
	}

	var models []VariableSetVariableModel
	diags.Append(data.Variables.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	variables := make([]api.VariableSetVariable, 0, len(models))
	for _, m := range models {
		value, err := vsVariableValueFromModel(m)
		if err != nil {
			diags.AddError("Failed to convert variable value", fmt.Sprintf("Variable '%s': %s", m.Key.ValueString(), err.Error()))
			return nil, diags
		}

		variables = append(variables, api.VariableSetVariable{
			Key:   m.Key.ValueString(),
			Value: *value,
		})
	}

	return variables, diags
}

// vsVariableValueFromModel converts a single variable model into an API Value.
func vsVariableValueFromModel(m VariableSetVariableModel) (*api.Value, error) {
	var value api.Value

	if !m.ReferenceValue.IsNull() && !m.ReferenceValue.IsUnknown() {
		refAttrs := m.ReferenceValue.Attributes()

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

	if !m.Value.IsNull() && !m.Value.IsUnknown() {
		if m.Sensitive.ValueBool() {
			if err := value.FromSensitiveValue(api.SensitiveValue{}); err != nil {
				return nil, fmt.Errorf("failed to set sensitive value: %w", err)
			}
		} else {
			var literal api.LiteralValue
			if err := literal.FromStringValue(m.Value.ValueString()); err != nil {
				return nil, fmt.Errorf("failed to set string value: %w", err)
			}
			if err := value.FromLiteralValue(literal); err != nil {
				return nil, fmt.Errorf("failed to set literal value: %w", err)
			}
		}

		return &value, nil
	}

	return nil, fmt.Errorf("one of value or reference_value must be provided")
}

// vsVariablesToModel converts API variables to a Terraform list for state.
func vsVariablesToModel(variables []api.VariableSetVariable) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(variables) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: variableSetVariableAttrTypes}), diags
	}

	elems := make([]attr.Value, 0, len(variables))
	for _, v := range variables {
		strVal := types.StringNull()
		sensitiveVal := types.BoolNull()
		refVal := types.ObjectNull(referenceValueAttrTypes)

		// Try reference value first
		if ref, err := v.Value.AsReferenceValue(); err == nil && ref.Reference != "" {
			pathElements := make([]attr.Value, len(ref.Path))
			for i, p := range ref.Path {
				pathElements[i] = types.StringValue(p)
			}
			pathList, listDiags := types.ListValue(types.StringType, pathElements)
			if listDiags.HasError() {
				diags.Append(listDiags...)
				return types.ListNull(types.ObjectType{AttrTypes: variableSetVariableAttrTypes}), diags
			}
			obj, objDiags := types.ObjectValue(referenceValueAttrTypes, map[string]attr.Value{
				"reference": types.StringValue(ref.Reference),
				"path":      pathList,
			})
			if objDiags.HasError() {
				diags.Append(objDiags...)
				return types.ListNull(types.ObjectType{AttrTypes: variableSetVariableAttrTypes}), diags
			}
			refVal = obj
		} else if _, err := v.Value.AsSensitiveValue(); err == nil {
			sensitiveVal = types.BoolValue(true)
		} else if lit, err := v.Value.AsLiteralValue(); err == nil {
			if s, err := lit.AsStringValue(); err == nil {
				strVal = types.StringValue(s)
			}
		}

		obj, objDiags := types.ObjectValue(variableSetVariableAttrTypes, map[string]attr.Value{
			"key":             types.StringValue(v.Key),
			"value":           strVal,
			"sensitive":       sensitiveVal,
			"reference_value": refVal,
		})
		if objDiags.HasError() {
			diags.Append(objDiags...)
			return types.ListNull(types.ObjectType{AttrTypes: variableSetVariableAttrTypes}), diags
		}
		elems = append(elems, obj)
	}

	list, listDiags := types.ListValue(types.ObjectType{AttrTypes: variableSetVariableAttrTypes}, elems)
	diags.Append(listDiags...)
	return list, diags
}

