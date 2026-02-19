// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ resource.Resource = &DeploymentVariableResource{}
var _ resource.ResourceWithImportState = &DeploymentVariableResource{}
var _ resource.ResourceWithConfigure = &DeploymentVariableResource{}

func NewDeploymentVariableResource() resource.Resource {
	return &DeploymentVariableResource{}
}

type DeploymentVariableResource struct {
	workspace *api.WorkspaceClient
}

func (r *DeploymentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_variable"
}

func (r *DeploymentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DeploymentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DeploymentVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the deployment variable",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Required:    true,
				Description: "The deployment ID this variable belongs to",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The variable key",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The variable description",
			},
			"default_value": schema.DynamicAttribute{
				Optional:    true,
				Description: "The default value for the variable",
			},
		},
	}
}

func (r *DeploymentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeploymentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variableID := data.ID.ValueString()
	if data.ID.IsNull() || data.ID.IsUnknown() || variableID == "" {
		variableID = uuid.NewString()
		data.ID = types.StringValue(variableID)
	}

	defaultValue, err := literalValueFromDynamic(data.DefaultValue)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment variable", err.Error())
		return
	}

	requestBody := api.RequestDeploymentVariableUpdateJSONRequestBody{
		DeploymentId: data.DeploymentId.ValueString(),
		Key:          data.Key.ValueString(),
		Description:  data.Description.ValueStringPointer(),
		DefaultValue: defaultValue,
	}

	variableResp, err := r.workspace.Client.RequestDeploymentVariableUpdateWithResponse(
		ctx, r.workspace.ID.String(), variableID, requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create deployment variable", err.Error())
		return
	}

	if variableResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create deployment variable", formatResponseError(variableResp.StatusCode(), variableResp.Body))
		return
	}

	if variableResp.JSON202 == nil || variableResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to create deployment variable", "Empty deployment variable ID in response")
		return
	}

	varId := variableResp.JSON202.Id
	data.ID = types.StringValue(varId)

	err = waitForResource(ctx, func() (bool, error) {
		getResp, err := r.workspace.Client.GetDeploymentVariableWithResponse(
			ctx, r.workspace.ID.String(), varId,
		)
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
		resp.Diagnostics.AddError("Failed to create deployment variable", fmt.Sprintf("Resource not available after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeploymentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variableResp, err := r.workspace.Client.GetDeploymentVariableWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read deployment variable",
			fmt.Sprintf("Failed to read deployment variable with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	switch variableResp.StatusCode() {
	case http.StatusOK:
		if variableResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read deployment variable", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	case http.StatusBadRequest:
		if variableResp.JSON400 != nil && variableResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to read deployment variable", fmt.Sprintf("Bad request: %s", *variableResp.JSON400.Error))
			return
		}
		resp.Diagnostics.AddError("Failed to read deployment variable", "Bad request")
		return
	}

	if variableResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError("Failed to read deployment variable", formatResponseError(variableResp.StatusCode(), variableResp.Body))
		return
	}

	variable := variableResp.JSON200.Variable
	data.ID = types.StringValue(variable.Id)
	data.DeploymentId = types.StringValue(variable.DeploymentId)
	data.Key = types.StringValue(variable.Key)
	data.Description = descriptionValue(variable.Description)
	data.DefaultValue = literalValueToDynamic(variable.DefaultValue)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeploymentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeploymentVariableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultValue, err := literalValueFromDynamic(data.DefaultValue)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update deployment variable", err.Error())
		return
	}

	requestBody := api.RequestDeploymentVariableUpdateJSONRequestBody{
		DeploymentId: data.DeploymentId.ValueString(),
		Key:          data.Key.ValueString(),
		Description:  data.Description.ValueStringPointer(),
		DefaultValue: defaultValue,
	}

	variableResp, err := r.workspace.Client.RequestDeploymentVariableUpdateWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(), requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update deployment variable",
			fmt.Sprintf("Failed to update deployment variable with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if variableResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update deployment variable", formatResponseError(variableResp.StatusCode(), variableResp.Body))
		return
	}

	if variableResp.JSON202 == nil || variableResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to update deployment variable", "Empty deployment variable ID in response")
		return
	}

	data.ID = types.StringValue(variableResp.JSON202.Id)
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *DeploymentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeploymentVariableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variableResp, err := r.workspace.Client.RequestDeploymentVariableDeletionWithResponse(
		ctx, r.workspace.ID.String(), data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete deployment variable", fmt.Sprintf("Failed to delete deployment variable: %s", err.Error()))
		return
	}

	switch variableResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusBadRequest:
		if variableResp.JSON400 != nil && variableResp.JSON400.Error != nil {
			resp.Diagnostics.AddError("Failed to delete deployment variable", fmt.Sprintf("Bad request: %s", *variableResp.JSON400.Error))
			return
		}
	case http.StatusNotFound:
		if variableResp.JSON404 != nil && variableResp.JSON404.Error != nil {
			resp.Diagnostics.AddError("Failed to delete deployment variable", fmt.Sprintf("Not found: %s", *variableResp.JSON404.Error))
			return
		}
	}

	resp.Diagnostics.AddError("Failed to delete deployment variable", formatResponseError(variableResp.StatusCode(), variableResp.Body))
}

type DeploymentVariableResourceModel struct {
	ID           types.String  `tfsdk:"id"`
	DeploymentId types.String  `tfsdk:"deployment_id"`
	Key          types.String  `tfsdk:"key"`
	Description  types.String  `tfsdk:"description"`
	DefaultValue types.Dynamic `tfsdk:"default_value"`
}

func literalValueFromDynamic(value types.Dynamic) (*api.LiteralValue, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	tfValue, err := value.ToTerraformValue(context.Background())
	if err != nil {
		return nil, err
	}

	decoded, err := terraformValueToInterface(tfValue)
	if err != nil {
		return nil, err
	}

	return literalValueFromInterface(decoded)
}

func literalValueFromInterface(value interface{}) (*api.LiteralValue, error) {
	var literal api.LiteralValue

	switch v := value.(type) {
	case nil:
		if err := literal.FromNullValue(true); err != nil {
			return nil, err
		}
		return &literal, nil
	case bool:
		if err := literal.FromBooleanValue(v); err != nil {
			return nil, err
		}
	case string:
		if err := literal.FromStringValue(v); err != nil {
			return nil, err
		}
	case int:
		if err := literal.FromIntegerValue(v); err != nil {
			return nil, err
		}
	case int32:
		if err := literal.FromIntegerValue(api.IntegerValue(v)); err != nil {
			return nil, err
		}
	case int64:
		if err := literal.FromIntegerValue(api.IntegerValue(v)); err != nil {
			return nil, err
		}
	case float32:
		if err := literal.FromNumberValue(api.NumberValue(v)); err != nil {
			return nil, err
		}
	case float64:
		if math.Trunc(v) == v {
			if err := literal.FromIntegerValue(api.IntegerValue(int64(v))); err != nil {
				return nil, err
			}
		} else {
			if err := literal.FromNumberValue(api.NumberValue(v)); err != nil {
				return nil, err
			}
		}
	case map[string]interface{}:
		if err := literal.FromObjectValue(api.ObjectValue{Object: v}); err != nil {
			return nil, err
		}
	case []interface{}:
		return nil, fmt.Errorf("unsupported default_value type []interface{}")
	default:
		return nil, fmt.Errorf("unsupported default_value type %T", value)
	}

	return &literal, nil
}

func literalValueToDynamic(value *api.LiteralValue) types.Dynamic {
	if value == nil {
		return types.DynamicNull()
	}

	if v, err := value.AsBooleanValue(); err == nil {
		return types.DynamicValue(types.BoolValue(v))
	}
	if v, err := value.AsIntegerValue(); err == nil {
		return types.DynamicValue(types.Int64Value(int64(v)))
	}
	if v, err := value.AsNumberValue(); err == nil {
		return types.DynamicValue(types.Float64Value(float64(v)))
	}
	if v, err := value.AsStringValue(); err == nil {
		return types.DynamicValue(types.StringValue(v))
	}
	if v, err := value.AsObjectValue(); err == nil {
		if attrValue, _, err := attrValueFromInterface(v.Object); err == nil {
			return types.DynamicValue(attrValue)
		}
	}
	if _, err := value.AsNullValue(); err == nil {
		return types.DynamicNull()
	}

	return types.DynamicNull()
}

func attrValueFromInterface(value interface{}) (attr.Value, attr.Type, error) {
	switch v := value.(type) {
	case nil:
		return types.DynamicNull(), types.DynamicType, nil
	case bool:
		return types.BoolValue(v), types.BoolType, nil
	case string:
		return types.StringValue(v), types.StringType, nil
	case int:
		return types.Int64Value(int64(v)), types.Int64Type, nil
	case int32:
		return types.Int64Value(int64(v)), types.Int64Type, nil
	case int64:
		return types.Int64Value(v), types.Int64Type, nil
	case float32:
		return types.Float64Value(float64(v)), types.Float64Type, nil
	case float64:
		if math.Trunc(v) == v {
			return types.Int64Value(int64(v)), types.Int64Type, nil
		}
		return types.Float64Value(v), types.Float64Type, nil
	case map[string]any:
		attrTypes := make(map[string]attr.Type, len(v))
		attrValues := make(map[string]attr.Value, len(v))
		for key, raw := range v {
			convertedValue, convertedType, err := attrValueFromInterface(raw)
			if err != nil {
				return nil, nil, err
			}
			attrTypes[key] = convertedType
			attrValues[key] = convertedValue
		}
		obj, diags := types.ObjectValue(attrTypes, attrValues)
		if diags.HasError() {
			return nil, nil, fmt.Errorf("failed to build object value")
		}
		return obj, obj.Type(context.Background()), nil
	case []interface{}:
		return nil, nil, fmt.Errorf("unsupported value type []interface{}")
	default:
		return nil, nil, fmt.Errorf("unsupported value type %T", value)
	}
}

func terraformValueToInterface(value tftypes.Value) (interface{}, error) {
	if !value.IsKnown() {
		return nil, nil
	}
	if value.IsNull() {
		return nil, nil
	}

	if tftypes.String.Equal(value.Type()) {
		var decoded string
		if err := value.As(&decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}
	if tftypes.Bool.Equal(value.Type()) {
		var decoded bool
		if err := value.As(&decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}
	if tftypes.Number.Equal(value.Type()) {
		var decoded *big.Float
		if err := value.As(&decoded); err != nil {
			return nil, err
		}
		if decoded == nil {
			return nil, nil
		}
		if decoded.IsInt() {
			integer, _ := decoded.Int64()
			return integer, nil
		}
		floatVal, _ := decoded.Float64()
		return floatVal, nil
	}

	switch value.Type().(type) {
	case tftypes.Object:
		var decoded map[string]tftypes.Value
		if err := value.As(&decoded); err != nil {
			return nil, err
		}
		result := make(map[string]interface{}, len(decoded))
		for key, raw := range decoded {
			converted, err := terraformValueToInterface(raw)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case tftypes.Map:
		var decoded map[string]tftypes.Value
		if err := value.As(&decoded); err != nil {
			return nil, err
		}
		result := make(map[string]interface{}, len(decoded))
		for key, raw := range decoded {
			converted, err := terraformValueToInterface(raw)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case tftypes.List, tftypes.Tuple, tftypes.Set:
		var decoded []tftypes.Value
		if err := value.As(&decoded); err != nil {
			return nil, err
		}
		result := make([]interface{}, 0, len(decoded))
		for _, raw := range decoded {
			converted, err := terraformValueToInterface(raw)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported terraform value type %s", value.Type().String())
	}
}
