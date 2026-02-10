// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"time"

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

const waitForResourceTimeout = 5 * time.Minute

// waitForResource polls check until it returns true or 5 minutes have elapsed.
// check should return (true, nil) when the resource exists, (false, nil) to keep
// polling, or (false, err) to abort immediately. Uses exponential backoff starting
// at 1s and capped at 10s.
func waitForResource(ctx context.Context, check func() (bool, error)) error {
	deadline := time.Now().Add(waitForResourceTimeout)
	interval := 1 * time.Second

	for {
		exists, err := check()
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("resource not found after %s", waitForResourceTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		interval = min(interval*2, 10*time.Second)
	}
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
