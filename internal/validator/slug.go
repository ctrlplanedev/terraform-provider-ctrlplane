package validator

import (
	"context"

	"github.com/gosimple/slug"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &SlugValidator{}

type SlugValidator struct{}

func NewSlugValidator() validator.String {
	return &SlugValidator{}
}

// Description implements validator.String.
func (v *SlugValidator) Description(context.Context) string {
	return "must be a valid slug"
}

// MarkdownDescription implements validator.String.
func (v *SlugValidator) MarkdownDescription(context.Context) string {
	return "must be a valid slug"
}

func (v *SlugValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		return
	}

	if req.ConfigValue.IsUnknown() {
		return
	}

	if !slug.IsSlug(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddError("Invalid slug", "must be a valid slug")
	}
}
