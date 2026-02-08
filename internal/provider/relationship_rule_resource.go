// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

type RelationshipRuleEndpointModel struct {
	Type     types.String `tfsdk:"type"`
	Selector types.String `tfsdk:"selector"`
}

type RelationshipRuleResourceModel struct {
	ID               types.String                   `tfsdk:"id"`
	Name             types.String                   `tfsdk:"name"`
	Reference        types.String                   `tfsdk:"reference"`
	Description      types.String                   `tfsdk:"description"`
	RelationshipType types.String                   `tfsdk:"relationship_type"`
	Matcher          types.String                   `tfsdk:"matcher"`
	From             *RelationshipRuleEndpointModel `tfsdk:"from"`
	To               *RelationshipRuleEndpointModel `tfsdk:"to"`
	Metadata         types.Map                      `tfsdk:"metadata"`
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
	entityTypeValues := []string{"deployment", "environment", "resource"}

	endpointBlock := schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The entity type (one of: deployment, environment, resource)",
				Validators: []validator.String{
					stringvalidator.OneOf(entityTypeValues...),
				},
			},
			"selector": schema.StringAttribute{
				Optional:    true,
				Description: "A CEL expression to filter entities",
			},
		},
	}

	fromBlock := endpointBlock
	fromBlock.Description = "The source side of the relationship"

	toBlock := endpointBlock
	toBlock.Description = "The target side of the relationship"

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
			"relationship_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of relationship (e.g., \"depends_on\", \"associated_with\")",
			},
			"matcher": schema.StringAttribute{
				Required:    true,
				Description: "A CEL expression used to match entities for the relationship",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Description: "Metadata key-value pairs for the relationship rule",
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"from": fromBlock,
			"to":   toBlock,
		},
	}
}

func (r *RelationshipRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RelationshipRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.From == nil {
		resp.Diagnostics.AddError("Missing required block", "The \"from\" block is required")
		return
	}
	if data.To == nil {
		resp.Diagnostics.AddError("Missing required block", "The \"to\" block is required")
		return
	}

	var matcher api.CreateRelationshipRuleRequest_Matcher
	if err := matcher.FromCelMatcher(api.CelMatcher{Cel: data.Matcher.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Invalid matcher", err.Error())
		return
	}

	metadata := map[string]string{}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		diags := data.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	requestBody := api.CreateRelationshipRuleJSONRequestBody{
		Name:             data.Name.ValueString(),
		Reference:        data.Reference.ValueString(),
		Description:      data.Description.ValueStringPointer(),
		RelationshipType: data.RelationshipType.ValueString(),
		FromType:         api.RelatableEntityType(data.From.Type.ValueString()),
		ToType:           api.RelatableEntityType(data.To.Type.ValueString()),
		Matcher:          matcher,
		Metadata:         metadata,
	}

	if selectorValueSet(data.From.Selector) {
		fromSelector, err := selectorPointerFromString(data.From.Selector)
		if err != nil {
			resp.Diagnostics.AddError("Invalid from selector", err.Error())
			return
		}
		requestBody.FromSelector = fromSelector
	}

	if selectorValueSet(data.To.Selector) {
		toSelector, err := selectorPointerFromString(data.To.Selector)
		if err != nil {
			resp.Diagnostics.AddError("Invalid to selector", err.Error())
			return
		}
		requestBody.ToSelector = toSelector
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

	data.ID = types.StringValue(createResp.JSON201.Id)
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
	data.RelationshipType = types.StringValue(rule.RelationshipType)

	celMatcher, err := rule.Matcher.AsCelMatcher()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read matcher", err.Error())
		return
	}
	data.Matcher = types.StringValue(celMatcher.Cel)

	data.From = &RelationshipRuleEndpointModel{
		Type:     types.StringValue(string(rule.FromType)),
		Selector: types.StringNull(),
	}
	if rule.FromSelector != nil {
		fromSelectorValue, err := selectorStringValue(rule.FromSelector)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read from selector", err.Error())
			return
		}
		data.From.Selector = fromSelectorValue
	}

	data.To = &RelationshipRuleEndpointModel{
		Type:     types.StringValue(string(rule.ToType)),
		Selector: types.StringNull(),
	}
	if rule.ToSelector != nil {
		toSelectorValue, err := selectorStringValue(rule.ToSelector)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read to selector", err.Error())
			return
		}
		data.To.Selector = toSelectorValue
	}

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

	if data.From == nil {
		resp.Diagnostics.AddError("Missing required block", "The \"from\" block is required")
		return
	}
	if data.To == nil {
		resp.Diagnostics.AddError("Missing required block", "The \"to\" block is required")
		return
	}

	var matcher api.UpsertRelationshipRuleRequest_Matcher
	if err := matcher.FromCelMatcher(api.CelMatcher{Cel: data.Matcher.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Invalid matcher", err.Error())
		return
	}

	metadata := map[string]string{}
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		diags := data.Metadata.ElementsAs(ctx, &metadata, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	requestBody := api.RequestRelationshipRuleUpsertJSONRequestBody{
		Name:             data.Name.ValueString(),
		Reference:        data.Reference.ValueString(),
		Description:      data.Description.ValueStringPointer(),
		RelationshipType: data.RelationshipType.ValueString(),
		FromType:         api.RelatableEntityType(data.From.Type.ValueString()),
		ToType:           api.RelatableEntityType(data.To.Type.ValueString()),
		Matcher:          matcher,
		Metadata:         metadata,
	}

	if selectorValueSet(data.From.Selector) {
		fromSelector, err := selectorPointerFromString(data.From.Selector)
		if err != nil {
			resp.Diagnostics.AddError("Invalid from selector", err.Error())
			return
		}
		requestBody.FromSelector = fromSelector
	}

	if selectorValueSet(data.To.Selector) {
		toSelector, err := selectorPointerFromString(data.To.Selector)
		if err != nil {
			resp.Diagnostics.AddError("Invalid to selector", err.Error())
			return
		}
		requestBody.ToSelector = toSelector
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

	if upsertResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update relationship rule", formatResponseError(upsertResp.StatusCode(), upsertResp.Body))
		return
	}

	if upsertResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update relationship rule", "Empty response from server")
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
