// Copyright IBM Corp. 2021, 2026

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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

func normalizeCEL(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return strings.Join(strings.Fields(value.ValueString()), " ")
}

// celNormalizedPlanModifier keeps the prior state value when the planned
// config and state differ only by CEL-equivalent whitespace. The API collapses
// whitespace on the server side, so without this, a multi-line heredoc config
// would drift from the single-line form returned by Read on every plan.
type celNormalizedPlanModifier struct{}

func (celNormalizedPlanModifier) Description(_ context.Context) string {
	return "Suppresses diffs when the planned and prior-state CEL differ only by whitespace."
}

func (m celNormalizedPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (celNormalizedPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if normalizeCEL(req.PlanValue) == normalizeCEL(req.StateValue) {
		resp.PlanValue = req.StateValue
	}
}

func celNormalized() planmodifier.String {
	return celNormalizedPlanModifier{}
}
