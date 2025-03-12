// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GetResourceFilterSchema returns the attribute map for an inline resource filter (for resource blocks).
func GetResourceFilterSchema() map[string]resourceschema.Attribute {
	return map[string]resourceschema.Attribute{
		"type": resourceschema.StringAttribute{
			Description: "Filter type (e.g., comparison, selector).",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"key": resourceschema.StringAttribute{
			Description: "Key to compare against.",
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"operator": resourceschema.StringAttribute{
			Description: "Comparison operator.",
			Optional:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"value": resourceschema.StringAttribute{
			Description: "Value to compare. For comparison types with nested conditions, this should be an empty string.",
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"not": resourceschema.BoolAttribute{
			Description:   "Negates the condition.",
			Optional:      true,
			Computed:      true,
			Default:       booldefault.StaticBool(false),
			PlanModifiers: []planmodifier.Bool{
				// Optionally, add boolplanmodifier.RequiresReplace() if needed.
			},
		},
		"conditions": resourceschema.ListNestedAttribute{
			Description:   "Nested filter conditions.",
			Optional:      true,
			PlanModifiers: []planmodifier.List{
				// Optionally, add listplanmodifier.RequiresReplace().
			},
			NestedObject: resourceschema.NestedAttributeObject{
				Attributes: BuildResourceFilterNestedAttributesResource(2),
			},
		},
	}
}

// BuildResourceFilterNestedAttributesResource recursively builds nested attributes for resource filters (used in resource blocks).
func BuildResourceFilterNestedAttributesResource(depth int) map[string]resourceschema.Attribute {
	if depth <= 0 {
		return GetResourceFilterNestedAttributesShallowResource()
	}
	return map[string]resourceschema.Attribute{
		"type": resourceschema.StringAttribute{
			Description: "Condition type.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"key": resourceschema.StringAttribute{
			Description: "Metadata key.",
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"operator": resourceschema.StringAttribute{
			Description: "Condition operator.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"value": resourceschema.StringAttribute{
			Description: "Condition value. For comparison types with nested conditions, this should be an empty string.",
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"not": resourceschema.BoolAttribute{
			Description:   "Negate the condition.",
			Optional:      true,
			Computed:      true,
			Default:       booldefault.StaticBool(false),
			PlanModifiers: []planmodifier.Bool{
				// Optionally add boolplanmodifier.RequiresReplace().
			},
		},
		"conditions": resourceschema.ListNestedAttribute{
			Description:   "Nested conditions.",
			Optional:      true,
			PlanModifiers: []planmodifier.List{
				// Optionally add listplanmodifier.RequiresReplace().
			},
			NestedObject: resourceschema.NestedAttributeObject{
				Attributes: BuildResourceFilterNestedAttributesResource(depth - 1),
			},
		},
	}
}

// GetResourceFilterNestedAttributesShallowResource returns a shallow nested attribute map for resource filters (used in resource blocks).
func GetResourceFilterNestedAttributesShallowResource() map[string]resourceschema.Attribute {
	return map[string]resourceschema.Attribute{
		"type": resourceschema.StringAttribute{
			Description: "Condition type.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"key": resourceschema.StringAttribute{
			Description: "Metadata key.",
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"operator": resourceschema.StringAttribute{
			Description: "Condition operator.",
			Required:    true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"value": resourceschema.StringAttribute{
			Description: "Condition value.",
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString(""),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"not": resourceschema.BoolAttribute{
			Description:   "Negate condition.",
			Optional:      true,
			Computed:      true,
			Default:       booldefault.StaticBool(false),
			PlanModifiers: []planmodifier.Bool{
				// Optionally add boolplanmodifier.RequiresReplace().
			},
		},
	}
}

// BuildDataSourceFilterNestedAttributes builds nested attributes for resource filters for data sources.
func BuildDataSourceFilterNestedAttributes(depth int) map[string]dschema.Attribute {
	if depth <= 0 {
		return GetDataSourceFilterNestedAttributesShallow()
	}
	return map[string]dschema.Attribute{
		"type": dschema.StringAttribute{
			Description: "Condition type.",
			Required:    true,
		},
		"key": dschema.StringAttribute{
			Description: "Metadata key.",
			Optional:    true,
			Computed:    true,
		},
		"operator": dschema.StringAttribute{
			Description: "Condition operator.",
			Required:    true,
		},
		"value": dschema.StringAttribute{
			Description: "Condition value.",
			Optional:    true,
			Computed:    true,
		},
		"not": dschema.BoolAttribute{
			Description: "Negate condition.",
			Optional:    true,
			Computed:    true,
		},
		"conditions": dschema.ListNestedAttribute{
			Description: "Nested filter conditions.",
			Optional:    true,
			Computed:    true,
			NestedObject: dschema.NestedAttributeObject{
				Attributes: BuildDataSourceFilterNestedAttributes(depth - 1),
			},
		},
	}
}

// GetDataSourceFilterNestedAttributesShallow returns a shallow nested attribute map for resource filters in data sources.
func GetDataSourceFilterNestedAttributesShallow() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"type": dschema.StringAttribute{
			Description: "Condition type.",
			Required:    true,
		},
		"key": dschema.StringAttribute{
			Description: "Metadata key.",
			Optional:    true,
			Computed:    true,
		},
		"operator": dschema.StringAttribute{
			Description: "Condition operator.",
			Required:    true,
		},
		"value": dschema.StringAttribute{
			Description: "Condition value.",
			Optional:    true,
			Computed:    true,
		},
		"not": dschema.BoolAttribute{
			Description: "Negate condition.",
			Optional:    true,
			Computed:    true,
		},
	}
}

// GetResourceFilterAttrTypes returns attribute types for converting a resource filter to API format.
func GetResourceFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":       types.StringType,
		"key":        types.StringType,
		"operator":   types.StringType,
		"value":      types.StringType,
		"not":        types.BoolType,
		"conditions": types.ListType{ElemType: BuildConditionObjectType(2)},
	}
}

// BuildConditionObjectType recursively builds an attr.Type for a condition object.
func BuildConditionObjectType(depth int) attr.Type {
	base := map[string]attr.Type{
		"type":     types.StringType,
		"key":      types.StringType,
		"operator": types.StringType,
		"value":    types.StringType,
		"not":      types.BoolType,
	}
	if depth <= 0 {
		return types.ObjectType{AttrTypes: base}
	}
	base["conditions"] = types.ListType{ElemType: BuildConditionObjectType(depth - 1)}
	return types.ObjectType{AttrTypes: base}
}
