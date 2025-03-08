// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// stringToPtr returns a pointer to the given string.
func stringToPtr(s string) *string {
	return &s
}

// Slugify converts a string to a valid slug format.
// It replaces spaces with hyphens, collapses multiple spaces to a single hyphen,
// and converts to lowercase.
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	
	// Replace common symbols with word equivalents without adding spaces
	replacements := []struct {
		symbol string
		word   string
	}{
		{"&", "and"},
		{"|", "or"},
		{"+", "+"},
		{"@", "@"},
		{"#", ""},
		{"$", "dollar"},
		{"%", "percent"},
		{"*", "*"},
		{"=", ""},
		{"!", "!"},
		{"?", ""},
	}
	
	for _, replacement := range replacements {
		s = strings.ReplaceAll(s, replacement.symbol, replacement.word)
	}
	
	// Replace spaces with hyphens
	spaceReg := regexp.MustCompile(`\s+`)
	s = spaceReg.ReplaceAllString(s, "-")
	
	// Replace consecutive hyphens with a single hyphen
	reg := regexp.MustCompile(`-+`)
	s = reg.ReplaceAllString(s, "-")
	
	// Remove leading and trailing hyphens
	s = strings.Trim(s, "-")
	
	// Return the result, even if empty
	return s
}

// MaxSlugLength defines the maximum allowed length for slugs
const MaxSlugLength = 30

// ValidateSlugLength checks if a slug exceeds the maximum allowed length
// Returns an error if the slug is too long, nil otherwise
func ValidateSlugLength(value string) error {
	if len(value) > MaxSlugLength {
		return fmt.Errorf("Slug must not exceed %d characters, got: %d characters", MaxSlugLength, len(value))
	}
	return nil
}

// SlugValidator validates that a slug is in the correct format
type SlugValidator struct{}

// Description returns a plain text description of the validator's behavior
func (v SlugValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Validates that the slug is in the correct format (lowercase alphanumeric characters and hyphens, starting and ending with alphanumeric characters) and does not exceed %d characters.", MaxSlugLength)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior
func (v SlugValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Validates that the slug is in the correct format (lowercase alphanumeric characters and hyphens, starting and ending with alphanumeric characters) and does not exceed %d characters.", MaxSlugLength)
}

// ValidateString performs the validation
func (v SlugValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	
	// Check if the slug exceeds the maximum length
	if err := ValidateSlugLength(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Slug Length",
			err.Error(),
		)
		return
	}

	// Check if the slug matches the required pattern
	pattern := `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	regex := regexp.MustCompile(pattern)

	if !regex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Slug Format",
			"Slug must contain only lowercase alphanumeric characters and hyphens, and must start and end with an alphanumeric character.",
		)
	}
}