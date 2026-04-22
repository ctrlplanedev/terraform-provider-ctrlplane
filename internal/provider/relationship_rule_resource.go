// Copyright IBM Corp. 2021, 2026

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &RelationshipRuleResource{}
var _ resource.ResourceWithImportState = &RelationshipRuleResource{}
var _ resource.ResourceWithConfigure = &RelationshipRuleResource{}

func NewRelationshipRuleResource() resource.Resource {
	return &RelationshipRuleResource{}
}

type RelationshipRuleResource struct {
	workspace *api.WorkspaceClient
}

type RelationshipRuleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Reference   types.String `tfsdk:"reference"`
	Description types.String `tfsdk:"description"`
	Cel         types.String `tfsdk:"matcher"`
	Metadata    types.Map    `tfsdk:"metadata"`
}

func (r *RelationshipRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_relationship_rule"
}

func (r *RelationshipRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RelationshipRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RelationshipRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a relationship rule in Ctrlplane. Relationship rules define how entities (resources, deployments, environments) are related to each other.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the relationship rule",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the relationship rule",
			},
			"reference": schema.StringAttribute{
				Required:    true,
				Description: "A unique reference identifier for the relationship rule",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the relationship rule",
			},
			"matcher": schema.StringAttribute{
				Required:    true,
				Description: "A CEL expression that defines the relationship rule",
				PlanModifiers: []planmodifier.String{
					celNormalized(),
				},
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Metadata key-value pairs for the relationship rule",
				ElementType: types.StringType,
				Default: func() defaults.Map {
					empty, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
					return mapdefault.StaticValue(empty)
				}(),
			},
		},
	}
}

func (r *RelationshipRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RelationshipRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cel := normalizeCEL(data.Cel)

	metadata := map[string]string{}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		diags := data.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	requestBody := api.CreateRelationshipRuleJSONRequestBody{
		Name:        data.Name.ValueString(),
		Reference:   data.Reference.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Cel:         cel,
		Metadata:    metadata,
	}

	createResp, err := r.workspace.Client.CreateRelationshipRuleWithResponse(
		ctx,
		r.workspace.ID.String(),
		requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create relationship rule", err.Error())
		return
	}

	if createResp.StatusCode() != http.StatusCreated {
		resp.Diagnostics.AddError("Failed to create relationship rule", formatResponseError(createResp.StatusCode(), createResp.Body))
		return
	}

	if createResp.JSON201 == nil {
		resp.Diagnostics.AddError("Failed to create relationship rule", "Empty response from server")
		return
	}

	ruleId := createResp.JSON201.Id
	data.ID = types.StringValue(ruleId)

	err = waitForResource(ctx, func() (bool, error) {
		getResp, err := r.workspace.Client.GetRelationshipRuleWithResponse(ctx, r.workspace.ID.String(), ruleId)
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
		resp.Diagnostics.AddError("Failed to create relationship rule", fmt.Sprintf("Resource not available after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RelationshipRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RelationshipRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ruleResp, err := r.workspace.Client.GetRelationshipRuleWithResponse(
		ctx,
		r.workspace.ID.String(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read relationship rule",
			fmt.Sprintf("Failed to read relationship rule with ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	switch ruleResp.StatusCode() {
	case http.StatusOK:
		if ruleResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read relationship rule", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read relationship rule", formatResponseError(ruleResp.StatusCode(), ruleResp.Body))
		return
	}

	rule := ruleResp.JSON200
	data.ID = types.StringValue(rule.Id)
	data.Name = types.StringValue(rule.Name)
	data.Reference = types.StringValue(rule.Reference)
	data.Description = descriptionValue(rule.Description)
	data.Cel = types.StringValue(rule.Cel)
	data.Metadata = stringMapValue(&rule.Metadata)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RelationshipRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RelationshipRuleResourceModel
	var state RelationshipRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID

	cel := normalizeCEL(data.Cel)

	metadata := map[string]string{}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		diags := data.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	requestBody := api.RequestRelationshipRuleUpsertJSONRequestBody{
		Name:        data.Name.ValueString(),
		Reference:   data.Reference.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Cel:         cel,
		Metadata:    metadata,
	}

	upsertResp, err := r.workspace.Client.RequestRelationshipRuleUpsertWithResponse(
		ctx,
		r.workspace.ID.String(),
		data.ID.ValueString(),
		requestBody,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update relationship rule", err.Error())
		return
	}

	switch upsertResp.StatusCode() {
	case http.StatusOK, http.StatusAccepted:
	default:
		resp.Diagnostics.AddError("Failed to update relationship rule", formatResponseError(upsertResp.StatusCode(), upsertResp.Body))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RelationshipRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RelationshipRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteResp, err := r.workspace.Client.DeleteRelationshipWithResponse(
		ctx,
		r.workspace.ID.String(),
		data.ID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete relationship rule", err.Error())
		return
	}

	switch deleteResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusNotFound:
		return
	default:
		resp.Diagnostics.AddError("Failed to delete relationship rule", formatResponseError(deleteResp.StatusCode(), deleteResp.Body))
		return
	}
}
