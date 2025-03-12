// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"

	rsschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// BuildResourceFilterNestedAttributes builds a nested attribute map for resource filters.
func BuildResourceFilterNestedAttributes(depth int) map[string]rsschema.Attribute {
	if depth <= 0 {
		return map[string]rsschema.Attribute{
			"type": rsschema.StringAttribute{
				Description: "Condition type.",
				Required:    true,
			},
			"key": rsschema.StringAttribute{
				Description: "Key to filter on.",
				Optional:    true,
			},
			"operator": rsschema.StringAttribute{
				Description: "Filter operator.",
				Required:    true,
			},
			"value": rsschema.StringAttribute{
				Description: "Filter value.",
				Optional:    true,
			},
			"not": rsschema.BoolAttribute{
				Description: "Negate condition.",
				Optional:    true,
			},
		}
	}
	return map[string]rsschema.Attribute{
		"type": rsschema.StringAttribute{
			Description: "Condition type.",
			Required:    true,
		},
		"key": rsschema.StringAttribute{
			Description: "Key to filter on.",
			Optional:    true,
		},
		"operator": rsschema.StringAttribute{
			Description: "Filter operator.",
			Required:    true,
		},
		"value": rsschema.StringAttribute{
			Description: "Filter value.",
			Optional:    true,
		},
		"not": rsschema.BoolAttribute{
			Description: "Negate condition.",
			Optional:    true,
		},
		"conditions": rsschema.ListNestedAttribute{
			Description: "Nested conditions.",
			Optional:    true,
			NestedObject: rsschema.NestedAttributeObject{
				Attributes: BuildResourceFilterNestedAttributes(depth - 1),
			},
		},
	}
}

// ResourceFilterModel represents the filtering criteria.
type ResourceFilterModel struct {
	Type       types.String          `tfsdk:"type"`
	Key        types.String          `tfsdk:"key"`
	Operator   types.String          `tfsdk:"operator"`
	Value      types.String          `tfsdk:"value"`
	Not        types.Bool            `tfsdk:"not"`
	Conditions []ResourceFilterModel `tfsdk:"conditions"`
}

// Constants for filter types.
const (
	FilterTypeComparison = "comparison"
	FilterTypeSelector   = "selector"
)

// conditionHash generates a deterministic hash for a ResourceFilterModel based on its content.
func conditionHash(c ResourceFilterModel) string {
	h := sha256.New()
	h.Write([]byte(c.Type.ValueString()))
	h.Write([]byte(c.Key.ValueString()))
	h.Write([]byte(c.Operator.ValueString()))
	h.Write([]byte(c.Value.ValueString()))
	if !c.Not.IsNull() {
		h.Write([]byte(fmt.Sprintf("%v", c.Not.ValueBool())))
	} else {
		h.Write([]byte("false"))
	}
	for _, sub := range c.Conditions {
		h.Write([]byte(conditionHash(sub)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[0:8]
}

// InitConditions recursively initializes the Conditions slice for comparison filters.
func (r *ResourceFilterModel) InitConditions() {
	if r == nil {
		return
	}
	if !r.Type.IsNull() && r.Type.ValueString() == FilterTypeComparison {
		if r.Conditions == nil {
			r.Conditions = []ResourceFilterModel{}
		}
		for i := range r.Conditions {
			r.Conditions[i].InitConditions()
		}
	}
}

// ToAPIFilter converts the ResourceFilterModel into a map format for API calls.
func (r *ResourceFilterModel) ToAPIFilter(ctx context.Context) (map[string]interface{}, error) {
	tflog.Debug(ctx, "Converting ResourceFilterModel to API format", map[string]interface{}{
		"has_type":         !r.Type.IsNull(),
		"type":             r.Type.ValueString(),
		"conditions_count": len(r.Conditions),
	})
	apiFilter := map[string]interface{}{
		"type": r.Type.ValueString(),
	}
	if !r.Operator.IsNull() {
		apiFilter["operator"] = r.Operator.ValueString()
	}
	if !r.Key.IsNull() {
		apiFilter["key"] = r.Key.ValueString()
	}
	if !r.Value.IsNull() {
		apiFilter["value"] = r.Value.ValueString()
	}
	if !r.Not.IsNull() {
		apiFilter["not"] = r.Not.ValueBool()
	}
	if len(r.Conditions) > 0 {
		conds, err := convertConditionsToAPIFilter(ctx, r.Conditions)
		if err != nil {
			return nil, err
		}
		apiFilter["conditions"] = conds
	}
	return apiFilter, nil
}

// ValidateAPIFilter ensures that the filter payload conforms to the expected API format.
func ValidateAPIFilter(filter map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range filter {
		result[k] = v
	}
	if _, hasType := result["type"]; !hasType {
		result["type"] = "kind"
	}
	return result
}

// convertConditionsToAPIFilter recursively converts nested conditions.
func convertConditionsToAPIFilter(ctx context.Context, conditions []ResourceFilterModel) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	for i, cond := range conditions {
		if cond.Type.ValueString() == FilterTypeComparison && len(cond.Conditions) > 0 &&
			(cond.Value.IsNull() || cond.Value.IsUnknown() || cond.Value.ValueString() == "") {
			cond.Value = types.StringValue("")
			tflog.Info(ctx, "TO_API: Setting empty string for nested comparison", map[string]interface{}{
				"index": i,
			})
		}
		condMap, err := cond.ToAPIFilter(ctx)
		if err != nil {
			return nil, fmt.Errorf("error converting condition %d: %w", i, err)
		}
		if condMap["type"] == FilterTypeComparison && condMap["conditions"] != nil {
			if _, hasValue := condMap["value"]; !hasValue {
				condMap["value"] = ""
				tflog.Info(ctx, "TO_API: Added empty string value to API condition", map[string]interface{}{
					"index": i,
				})
			}
		}
		result = append(result, condMap)
		tflog.Debug(ctx, "Added condition to API format", map[string]interface{}{
			"index":     i,
			"type":      condMap["type"],
			"has_value": condMap["value"] != nil,
		})
	}
	return result, nil
}

// FromAPIFilter converts an API filter (map) into a ResourceFilterModel.
func (m *ResourceFilterModel) FromAPIFilter(ctx context.Context, filter map[string]interface{}) error {
	m.Type = types.StringNull()
	m.Key = types.StringValue("")
	m.Operator = types.StringValue("")
	m.Value = types.StringNull()
	m.Not = types.BoolValue(false)
	m.Conditions = nil

	if filterType, ok := filter["type"].(string); ok && filterType != "" {
		m.Type = types.StringValue(filterType)
		if filterType == FilterTypeComparison {
			m.Conditions = []ResourceFilterModel{}
			if _, hasConditions := filter["conditions"]; hasConditions {
				m.Value = types.StringValue("")
				tflog.Info(ctx, "FORCE: Setting empty string for comparison with conditions", map[string]interface{}{
					"filter_type": filterType,
				})
			}
		}
		if filterType != FilterTypeComparison && filterType != FilterTypeSelector {
			if operator, ok := filter["operator"].(string); ok && operator != "" {
				m.Operator = types.StringValue(operator)
			}
			if value, ok := filter["value"].(string); ok {
				m.Value = types.StringValue(value)
			}
		}
	} else {
		return fmt.Errorf("missing or invalid filter type")
	}

	if key, ok := filter["key"].(string); ok {
		m.Key = types.StringValue(key)
	}
	if operator, ok := filter["operator"].(string); ok && operator != "" {
		m.Operator = types.StringValue(operator)
	}
	if value, ok := filter["value"].(string); ok {
		m.Value = types.StringValue(value)
	} else if m.Value.IsNull() {
		m.Value = types.StringValue("")
	}
	if notVal, ok := filter["not"].(bool); ok {
		m.Not = types.BoolValue(notVal)
	}
	if !m.Type.IsNull() && m.Type.ValueString() == FilterTypeComparison {
		if conditions, ok := filter["conditions"].([]interface{}); ok && len(conditions) > 0 {
			tflog.Debug(ctx, "Processing nested conditions", map[string]interface{}{
				"conditions_count": len(conditions),
			})
			for i, condition := range conditions {
				if condMap, ok := condition.(map[string]interface{}); ok {
					var nestedCond ResourceFilterModel
					if err := nestedCond.FromAPIFilter(ctx, condMap); err != nil {
						return fmt.Errorf("error parsing condition %d: %w", i, err)
					}
					if nestedCond.Type.ValueString() == FilterTypeComparison && len(nestedCond.Conditions) > 0 {
						nestedCond.Value = types.StringValue("")
						tflog.Info(ctx, "Nested fix: Force empty string for nested comparison", map[string]interface{}{
							"index":          i,
							"has_conditions": len(nestedCond.Conditions),
						})
					}
					m.Conditions = append(m.Conditions, nestedCond)
					tflog.Debug(ctx, "Added nested condition", map[string]interface{}{
						"index": i,
						"type":  condMap["type"],
					})
				} else {
					return fmt.Errorf("condition %d is not a valid map", i)
				}
			}
		}
	} else if !m.Type.IsNull() && m.Type.ValueString() == FilterTypeSelector {
		if key, ok := filter["key"].(string); ok && key != "" {
			m.Key = types.StringValue(key)
		}
		if operator, ok := filter["operator"].(string); ok && operator != "" {
			m.Operator = types.StringValue(operator)
		}
		if value, ok := filter["value"].(string); ok {
			m.Value = types.StringValue(value)
		}
	}
	return nil
}

// GetValueOrDefaultForType returns an empty string for comparison filters with nested conditions.
func (r *ResourceFilterModel) GetValueOrDefaultForType() types.String {
	if !r.Type.IsNull() && r.Type.ValueString() == FilterTypeComparison && len(r.Conditions) > 0 {
		return types.StringValue("")
	}
	return r.Value
}

// PostProcessResourceFilter recursively processes nested conditions for consistency.
func PostProcessResourceFilter(ctx context.Context, filter *ResourceFilterModel) {
	if filter == nil {
		return
	}
	for i := range filter.Conditions {
		cond := &filter.Conditions[i]
		if cond.Conditions != nil {
			PostProcessResourceFilter(ctx, cond)
		}
	}
}

// DeepFixResourceFilter recursively ensures that for comparison filters with nested conditions the value is an empty string.
func DeepFixResourceFilter(filter *ResourceFilterModel) {
	if filter == nil {
		return
	}
	if filter.Conditions != nil {
		for i := range filter.Conditions {
			DeepFixResourceFilter(&filter.Conditions[i])
		}
		if filter.Type.ValueString() == FilterTypeComparison && len(filter.Conditions) > 0 {
			filter.Value = types.StringValue("")
		}
	}
}
