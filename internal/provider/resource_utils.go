package provider

import (
	"context"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func descriptionValue(description *string) types.String {
	if description == nil {
		return types.StringNull()
	}
	return types.StringValue(*description)
}

func stringMapPointer(value types.Map) *map[string]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	var decoded map[string]string
	diags := value.ElementsAs(context.Background(), &decoded, false)
	if diags.HasError() {
		return nil
	}

	return &decoded
}

func stringMapValue(value *map[string]string) types.Map {
	if value == nil {
		return types.MapNull(types.StringType)
	}

	result, _ := types.MapValueFrom(context.Background(), types.StringType, *value)
	return result
}

func selectorPointerFromString(value types.String) (*api.Selector, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	raw := value.ValueString()
	if raw == "" {
		return nil, nil
	}

	var selector api.Selector
	if err := selector.FromCelSelector(api.CelSelector{Cel: raw}); err != nil {
		return nil, err
	}

	return &selector, nil
}

func selectorStringValue(selector *api.Selector) (types.String, error) {
	if selector == nil {
		return types.StringNull(), nil
	}

	parsed, err := selector.AsCelSelector()
	if err != nil {
		return types.StringNull(), err
	}

	if parsed.Cel == "" {
		return types.StringNull(), nil
	}

	return types.StringValue(parsed.Cel), nil
}
