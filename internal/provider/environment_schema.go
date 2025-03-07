// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GetEnvironmentResourceSchema returns the schema for the environment resource.
func GetEnvironmentResourceSchema() resourceschema.Schema {
	return resourceschema.Schema{
		Description: "Manages a CtrlPlane environment.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Description: "Unique identifier for the environment.",
				Computed:    true,
			},
			"name": resourceschema.StringAttribute{
				Description: "Name of the environment.",
				Required:    true,
			},
			"description": resourceschema.StringAttribute{
				Description: "Description of the environment.",
				Optional:    true,
			},
			"system_id": resourceschema.StringAttribute{
				Description: "ID of the system this environment belongs to.",
				Required:    true,
			},
			"policy_id": resourceschema.StringAttribute{
				Description: "ID of the policy associated with this environment.",
				Computed:    true,
			},
			"metadata": resourceschema.MapAttribute{
				Description: "Metadata for the environment.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"resource_filter_id": resourceschema.StringAttribute{
				Description: "ID of a ctrlplane_resource_filter resource to use. Cannot be specified together with resource_filter.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_filter": resourceschema.SingleNestedAttribute{
				Description: "Inline resource filter for the environment. Cannot be specified together with resource_filter_id.",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					// Changes to an inline filter force recreation.
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: GetResourceFilterSchema(), // This function is defined in resource_filter_schema.go
			},
			"release_channels": resourceschema.ListAttribute{
				Description: "Release channels for the environment.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// GetEnvironmentDataSourceSchema returns the schema for the environment data source.
func GetEnvironmentDataSourceSchema() dschema.Schema {
	return dschema.Schema{
		Description: "Data source for a CtrlPlane environment.",
		Attributes: map[string]dschema.Attribute{
			"id": dschema.StringAttribute{
				Description: "Unique identifier for the environment.",
				Computed:    true,
			},
			"name": dschema.StringAttribute{
				Description: "Name of the environment.",
				Required:    true,
			},
			"description": dschema.StringAttribute{
				Description: "Description of the environment.",
				Computed:    true,
			},
			"system_id": dschema.StringAttribute{
				Description: "ID of the system this environment belongs to.",
				Required:    true,
			},
			"policy_id": dschema.StringAttribute{
				Description: "ID of the policy associated with this environment.",
				Computed:    true,
			},
			"metadata": dschema.MapAttribute{
				Description: "Metadata for the environment.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"resource_filter": dschema.SingleNestedAttribute{
				Description: "Resource filter for the environment.",
				Computed:    true,
				Attributes: map[string]dschema.Attribute{
					"type": dschema.StringAttribute{
						Description: "Filter type (e.g., comparison, selector).",
						Computed:    true,
					},
					"key": dschema.StringAttribute{
						Description: "Key to compare against.",
						Computed:    true,
					},
					"operator": dschema.StringAttribute{
						Description: "Comparison operator.",
						Computed:    true,
					},
					"value": dschema.StringAttribute{
						Description: "Value to compare.",
						Computed:    true,
					},
					"not": dschema.BoolAttribute{
						Description: "Negates the condition.",
						Computed:    true,
					},
					"conditions": dschema.ListNestedAttribute{
						Description: "Nested filter conditions.",
						Computed:    true,
						NestedObject: dschema.NestedAttributeObject{
							Attributes: BuildDataSourceFilterNestedAttributes(2),
						},
					},
				},
			},
			"release_channels": dschema.ListAttribute{
				Description: "Release channels for the environment.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
