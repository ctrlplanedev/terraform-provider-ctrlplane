// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ResourceProviderResource{}
var _ resource.ResourceWithImportState = &ResourceProviderResource{}
var _ resource.ResourceWithConfigure = &ResourceProviderResource{}

func NewResourceProviderResource() resource.Resource {
	return &ResourceProviderResource{}
}

type ResourceProviderResource struct {
	workspace *api.WorkspaceClient
}

func (r *ResourceProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_provider"
}

func (r *ResourceProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func (r *ResourceProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a resource provider and its resources in Ctrlplane. " +
			"All resources belonging to the provider are declared as nested blocks and " +
			"sent as a complete set on every apply, avoiding race conditions that occur " +
			"when managing individual resources separately.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the resource provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the resource provider",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Metadata key-value pairs for the resource provider",
				ElementType: types.StringType,
				Default: func() defaults.Map {
					empty, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
					return mapdefault.StaticValue(empty)
				}(),
			},
		},
		Blocks: map[string]schema.Block{
			"resource": schema.ListNestedBlock{
				Description: "Resources managed by this provider",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the resource",
						},
						"identifier": schema.StringAttribute{
							Required:    true,
							Description: "The unique identifier for the resource",
						},
						"kind": schema.StringAttribute{
							Required:    true,
							Description: "The kind/type of the resource",
						},
						"version": schema.StringAttribute{
							Required:    true,
							Description: "The version of the resource",
						},
						"config": schema.StringAttribute{
							Optional:    true,
							Description: "JSON-encoded configuration for the resource. Use jsonencode() to set this value.",
						},
						"metadata": schema.MapAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Metadata key-value pairs for the resource",
							ElementType: types.StringType,
							Default: func() defaults.Map {
								empty, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
								return mapdefault.StaticValue(empty)
							}(),
						},
					},
				},
			},
		},
	}
}

func (r *ResourceProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerID := uuid.NewString()

	upsertResp, err := r.workspace.Client.RequestResourceProviderUpsertWithResponse(
		ctx,
		r.workspace.ID.String(),
		api.RequestResourceProviderUpsertJSONRequestBody{
			Id:       providerID,
			Name:     data.Name.ValueString(),
			Metadata: stringMapPointer(data.Metadata),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource provider", err.Error())
		return
	}
	if upsertResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create resource provider", formatResponseError(upsertResp.StatusCode(), upsertResp.Body))
		return
	}
	if upsertResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to create resource provider", "Empty response from server")
		return
	}

	data.ID = types.StringValue(upsertResp.JSON202.Id)

	if len(data.Resources) > 0 {
		if err := r.setResources(ctx, data.ID.ValueString(), data.Resources); err != nil {
			resp.Diagnostics.AddError("Failed to set resources", err.Error())
			return
		}

		firstIdentifier := data.Resources[0].Identifier.ValueString()
		if err := r.waitForResourceAvailable(ctx, firstIdentifier); err != nil {
			resp.Diagnostics.AddError("Failed to create resources",
				fmt.Sprintf("Resources not available after creation: %s", err.Error()))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ResourceProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerResp, err := r.workspace.Client.GetResourceProviderByNameWithResponse(
		ctx,
		r.workspace.ID.String(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource provider", err.Error())
		return
	}

	switch providerResp.StatusCode() {
	case http.StatusOK:
		if providerResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read resource provider", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read resource provider",
			formatResponseError(providerResp.StatusCode(), providerResp.Body))
		return
	}

	provider := providerResp.JSON200
	data.ID = types.StringValue(provider.Id)
	data.Name = types.StringValue(provider.Name)
	data.Metadata = stringMapValue(provider.Metadata)

	resourcesResp, err := r.workspace.Client.GetResourceProviderResourcesWithResponse(
		ctx,
		r.workspace.ID.String(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list provider resources", err.Error())
		return
	}
	if resourcesResp.StatusCode() != http.StatusOK || resourcesResp.JSON200 == nil {
		resp.Diagnostics.AddError("Failed to list provider resources",
			formatResponseError(resourcesResp.StatusCode(), resourcesResp.Body))
		return
	}

	var updatedResources []ResourceProviderResourceItemModel
	for _, apiRes := range resourcesResp.JSON200.Items {
		updatedResources = append(updatedResources, ResourceProviderResourceItemModel{
			Name:       types.StringValue(apiRes.Name),
			Identifier: types.StringValue(apiRes.Identifier),
			Kind:       types.StringValue(apiRes.Kind),
			Version:    types.StringValue(apiRes.Version),
			Config:     configToJSONString(apiRes.Config),
			Metadata:   stringMapValue(&apiRes.Metadata),
		})
	}
	data.Resources = updatedResources

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceProviderModel
	var state ResourceProviderModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID

	upsertResp, err := r.workspace.Client.RequestResourceProviderUpsertWithResponse(
		ctx,
		r.workspace.ID.String(),
		api.RequestResourceProviderUpsertJSONRequestBody{
			Id:       data.ID.ValueString(),
			Name:     data.Name.ValueString(),
			Metadata: stringMapPointer(data.Metadata),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update resource provider", err.Error())
		return
	}
	if upsertResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update resource provider",
			formatResponseError(upsertResp.StatusCode(), upsertResp.Body))
		return
	}

	// Set all current resources on the provider first so the backend is
	// never left in a partially-deleted state if the call fails.
	if len(data.Resources) > 0 {
		if err := r.setResources(ctx, data.ID.ValueString(), data.Resources); err != nil {
			resp.Diagnostics.AddError("Failed to set resources", err.Error())
			return
		}
	}

	// Delete resources that were removed from the config.
	newIdentifiers := make(map[string]bool, len(data.Resources))
	for _, res := range data.Resources {
		newIdentifiers[res.Identifier.ValueString()] = true
	}
	for _, res := range state.Resources {
		identifier := res.Identifier.ValueString()
		if newIdentifiers[identifier] {
			continue
		}
		if err := r.deleteResource(ctx, identifier); err != nil {
			resp.Diagnostics.AddError("Failed to delete resource",
				fmt.Sprintf("Failed to delete resource '%s': %s", identifier, err.Error()))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *ResourceProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, res := range data.Resources {
		identifier := res.Identifier.ValueString()
		if err := r.deleteResource(ctx, identifier); err != nil {
			resp.Diagnostics.AddError("Failed to delete resource",
				fmt.Sprintf("Failed to delete resource '%s': %s", identifier, err.Error()))
			return
		}
	}
}

// setResources calls SetResourceProviderResources with the full list.
func (r *ResourceProviderResource) setResources(ctx context.Context, providerID string, items []ResourceProviderResourceItemModel) error {
	apiResources, err := resourceItemsFromModel(items)
	if err != nil {
		return err
	}

	setResp, err := r.workspace.Client.SetResourceProviderResourcesWithResponse(
		ctx,
		r.workspace.ID.String(),
		providerID,
		api.SetResourceProviderResourcesJSONRequestBody{
			Resources: apiResources,
		},
	)
	if err != nil {
		return err
	}
	if setResp.StatusCode() != http.StatusAccepted {
		return fmt.Errorf("%s", formatResponseError(setResp.StatusCode(), setResp.Body))
	}
	return nil
}

// deleteResource deletes a single resource by identifier, ignoring 404s.
func (r *ResourceProviderResource) deleteResource(ctx context.Context, identifier string) error {
	deleteResp, err := r.workspace.Client.RequestResourceDeletionByIdentifierWithResponse(
		ctx,
		r.workspace.ID.String(),
		identifier,
	)
	if err != nil {
		return err
	}
	switch deleteResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent, http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("%s", formatResponseError(deleteResp.StatusCode(), deleteResp.Body))
	}
}

// waitForResourceAvailable polls until a resource is readable by identifier.
func (r *ResourceProviderResource) waitForResourceAvailable(ctx context.Context, identifier string) error {
	return waitForResource(ctx, func() (bool, error) {
		getResp, err := r.workspace.Client.GetResourceByIdentifierWithResponse(
			ctx, r.workspace.ID.String(), identifier,
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
}

// ResourceProviderModel describes the resource provider data model.
type ResourceProviderModel struct {
	ID        types.String                        `tfsdk:"id"`
	Name      types.String                        `tfsdk:"name"`
	Metadata  types.Map                           `tfsdk:"metadata"`
	Resources []ResourceProviderResourceItemModel `tfsdk:"resource"`
}

// ResourceProviderResourceItemModel describes an individual resource within the provider.
type ResourceProviderResourceItemModel struct {
	Name       types.String `tfsdk:"name"`
	Identifier types.String `tfsdk:"identifier"`
	Kind       types.String `tfsdk:"kind"`
	Version    types.String `tfsdk:"version"`
	Config     types.String `tfsdk:"config"`
	Metadata   types.Map    `tfsdk:"metadata"`
}

// resourceItemsFromModel converts Terraform model resource items to API request format.
func resourceItemsFromModel(items []ResourceProviderResourceItemModel) ([]api.ResourceProviderResource, error) {
	now := time.Now().UTC()
	result := make([]api.ResourceProviderResource, 0, len(items))
	for _, item := range items {
		config, err := configFromJSONString(item.Config)
		if err != nil {
			return nil, fmt.Errorf("resource '%s': %w", item.Identifier.ValueString(), err)
		}
		result = append(result, api.ResourceProviderResource{
			Name:       item.Name.ValueString(),
			Identifier: item.Identifier.ValueString(),
			Kind:       item.Kind.ValueString(),
			Version:    item.Version.ValueString(),
			Config:     config,
			Metadata:   resourceMetadataFromMap(item.Metadata),
			CreatedAt:  now,
		})
	}
	return result, nil
}

func configFromJSONString(s types.String) (map[string]interface{}, error) {
	if s.IsNull() || s.IsUnknown() || s.ValueString() == "" {
		return map[string]interface{}{}, nil
	}
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(s.ValueString()), &config); err != nil {
		return nil, fmt.Errorf("config must be a JSON object: %w", err)
	}
	return config, nil
}

func configToJSONString(m map[string]interface{}) types.String {
	if len(m) == 0 {
		return types.StringNull()
	}
	b, err := json.Marshal(m)
	if err != nil {
		return types.StringNull()
	}
	return types.StringValue(string(b))
}
