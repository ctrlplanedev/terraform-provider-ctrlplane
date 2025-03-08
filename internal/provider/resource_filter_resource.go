// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	filterRegistry      = make(map[string]ResourceFilterModel)
	registryMutex       sync.RWMutex
	registryInitialized bool = false
)

// InitFilterRegistry ensures the registry is initialized.
// This should be called during provider setup.
func InitFilterRegistry() {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if !registryInitialized {
		filterRegistry = make(map[string]ResourceFilterModel)
		registryInitialized = true
		tflog.Info(context.Background(), "Resource filter registry initialized")
	}
}

// RegisterResourceFilter adds a resource filter to the in-memory registry.
func RegisterResourceFilter(id string, filter ResourceFilterModel) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if !registryInitialized {
		InitFilterRegistry()
	}

	filterRegistry[id] = filter
	tflog.Debug(context.Background(), "Registered resource filter", map[string]interface{}{
		"id": id,
	})
}

// GetResourceFilterByID retrieves a resource filter from the in-memory registry.
// This function now retries more times with a longer delay if the filter isn't immediately found.
func GetResourceFilterByID(ctx context.Context, id string) (*ResourceFilterModel, error) {
	registryMutex.RLock()
	filter, exists := filterRegistry[id]
	registryMutex.RUnlock()

	if !exists {
		// Increase retries and delay for referenced filters.
		maxRetries := 10
		retryDelayMs := 1000 // 1 second delay between retries

		tflog.Info(ctx, "Resource filter not immediately found in registry, will retry", map[string]interface{}{
			"id":             id,
			"max_retries":    maxRetries,
			"retry_delay_ms": retryDelayMs,
			"registry_size":  len(filterRegistry),
		})

		for i := 0; i < maxRetries; i++ {
			time.Sleep(time.Duration(retryDelayMs) * time.Millisecond)

			registryMutex.RLock()
			filter, exists = filterRegistry[id]
			registryMutex.RUnlock()

			if exists {
				tflog.Info(ctx, "Resource filter found in registry after retry", map[string]interface{}{
					"id":    id,
					"retry": i + 1,
				})
				return &filter, nil
			}

			tflog.Debug(ctx, "Resource filter still not found, retrying", map[string]interface{}{
				"id":    id,
				"retry": i + 1,
			})
		}

		tflog.Warn(ctx, "Resource filter not found in registry after retries", map[string]interface{}{
			"id":            id,
			"retries":       maxRetries,
			"registry_size": len(filterRegistry),
		})
		return nil, fmt.Errorf("resource filter with ID %s not found in registry after %d retries", id, maxRetries)
	}

	tflog.Debug(ctx, "Retrieved resource filter from registry", map[string]interface{}{
		"id": id,
	})
	return &filter, nil
}

// ---------------- Resource Filter Resource Implementation ----------------

// Ensure ResourceFilterResource satisfies the resource interfaces.
var _ resource.Resource = &ResourceFilterResource{}
var _ resource.ResourceWithImportState = &ResourceFilterResource{}

// NewResourceFilterResource returns a new resource filter resource.
func NewResourceFilterResource() resource.Resource {
	return &ResourceFilterResource{}
}

// ResourceFilterResource is a state-only resource that stores a resource filter.
type ResourceFilterResource struct{}

// ResourceFilterResourceModel represents the state of the ctrlplane_resource_filter.
type ResourceFilterResourceModel struct {
	ID         types.String          `tfsdk:"id"`
	Type       types.String          `tfsdk:"type"`
	Key        types.String          `tfsdk:"key"`
	Operator   types.String          `tfsdk:"operator"`
	Value      types.String          `tfsdk:"value"`
	Not        types.Bool            `tfsdk:"not"`
	Conditions []ResourceFilterModel `tfsdk:"conditions"`
}

func (r *ResourceFilterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_filter"
}

func (r *ResourceFilterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "A state-only resource for defining reusable resource filters.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Description: "Unique identifier for the resource filter, auto-generated based on content.",
				Computed:    true,
			},
			"type": resourceschema.StringAttribute{
				Description: "Filter type (e.g., comparison, selector).",
				Required:    true,
			},
			"key": resourceschema.StringAttribute{
				Description: "Key to compare against.",
				Optional:    true,
			},
			"operator": resourceschema.StringAttribute{
				Description: "Comparison operator.",
				Optional:    true,
			},
			"value": resourceschema.StringAttribute{
				Description: "Value to compare.",
				Optional:    true,
			},
			"not": resourceschema.BoolAttribute{
				Description: "Negates the condition.",
				Optional:    true,
			},
			"conditions": resourceschema.ListNestedAttribute{
				Description: "Nested filter conditions.",
				Optional:    true,
				NestedObject: resourceschema.NestedAttributeObject{
					Attributes: BuildResourceFilterNestedAttributesResource(2),
				},
			},
		},
	}
}

func (r *ResourceFilterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// No external configuration needed.
}

func (r *ResourceFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceFilterResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required fields based on type
	if plan.Type.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The 'type' attribute is required for all filter resources.",
		)
		return
	}

	// For non-comparison types, check that operator is set
	filterType := plan.Type.ValueString()
	if filterType != FilterTypeComparison && plan.Operator.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			fmt.Sprintf("The 'operator' attribute is required for filter type '%s'.", filterType),
		)
		return
	}

	// For metadata type, key is required
	if filterType == "metadata" && plan.Key.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"The 'key' attribute is required for filter type 'metadata'.",
		)
		return
	}

	filterModel := ResourceFilterModel{
		Type:       plan.Type,
		Key:        plan.Key,
		Operator:   plan.Operator,
		Value:      plan.Value,
		Not:        plan.Not,
		Conditions: plan.Conditions,
	}
	filterModel.InitConditions()

	id := conditionHash(filterModel)
	plan.ID = types.StringValue(id)

	RegisterResourceFilter(id, filterModel)
	tflog.Info(ctx, "Created resource filter", map[string]interface{}{
		"id":                  id,
		"type":                filterModel.Type.ValueString(),
		"operator":            filterModel.Operator.ValueString(),
		"conditions_count":    len(filterModel.Conditions),
		"registry_size_after": len(filterRegistry),
	})

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ResourceFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceFilterResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.ID.IsNull() {
		filterModel := ResourceFilterModel{
			Type:       state.Type,
			Key:        state.Key,
			Operator:   state.Operator,
			Value:      state.Value,
			Not:        state.Not,
			Conditions: state.Conditions,
		}
		filterModel.InitConditions()
		id := state.ID.ValueString()
		RegisterResourceFilter(id, filterModel)
		tflog.Info(ctx, "Re-registered existing resource filter during Read", map[string]interface{}{
			"id":                  id,
			"type":                filterModel.Type.ValueString(),
			"operator":            filterModel.Operator.ValueString(),
			"conditions_count":    len(filterModel.Conditions),
			"registry_size_after": len(filterRegistry),
		})
	}
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ResourceFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceFilterResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filterModel := ResourceFilterModel{
		Type:       plan.Type,
		Key:        plan.Key,
		Operator:   plan.Operator,
		Value:      plan.Value,
		Not:        plan.Not,
		Conditions: plan.Conditions,
	}
	filterModel.InitConditions()

	id := conditionHash(filterModel)
	plan.ID = types.StringValue(id)

	RegisterResourceFilter(id, filterModel)
	tflog.Info(ctx, "Updated resource filter", map[string]interface{}{
		"id":                  id,
		"type":                filterModel.Type.ValueString(),
		"operator":            filterModel.Operator.ValueString(),
		"conditions_count":    len(filterModel.Conditions),
		"registry_size_after": len(filterRegistry),
	})

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ResourceFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceFilterResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !state.ID.IsNull() {
		registryMutex.Lock()
		delete(filterRegistry, state.ID.ValueString())
		registryMutex.Unlock()
		tflog.Info(ctx, "Deleted resource filter from registry", map[string]interface{}{
			"id": state.ID.ValueString(),
		})
	}
	tflog.Info(ctx, "Deleted resource filter from state")
}

func (r *ResourceFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import the ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	
	// Get the filter from the registry
	filterModel, err := GetResourceFilterByID(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Resource Filter",
			fmt.Sprintf("Could not find resource filter in registry with ID %s: %s", req.ID, err),
		)
		return
	}
	
	// Set all the necessary attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), filterModel.Type)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), filterModel.Key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("operator"), filterModel.Operator)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), filterModel.Value)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("not"), filterModel.Not)...)
	
	if len(filterModel.Conditions) > 0 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("conditions"), filterModel.Conditions)...)
	}
	
	tflog.Info(ctx, "Imported resource filter", map[string]interface{}{
		"id": req.ID,
	})
}
