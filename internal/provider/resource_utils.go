package provider

import (
	"context"

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
