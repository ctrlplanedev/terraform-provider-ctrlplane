// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	
	// Process the string character by character to ensure each symbol is replaced properly
	var result strings.Builder
	for _, ch := range s {
		charStr := string(ch)
		replaced := false
		
		for _, replacement := range replacements {
			if charStr == replacement.symbol {
				result.WriteString(replacement.word)
				replaced = true
				break
			}
		}
		
		if !replaced {
			result.WriteString(charStr)
		}
	}
	
	s = result.String()
	
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

// MaxSlugLength defines the maximum allowed length for slugs.
const MaxSlugLength = 30

// ValidateSlugLength checks if a slug exceeds the maximum allowed length.
// Returns an error if the slug is too long, or nil otherwise.
func ValidateSlugLength(value string) error {
	if len(value) > MaxSlugLength {
		return fmt.Errorf("slug must not exceed %d characters, got: %d characters", MaxSlugLength, len(value))
	}
	return nil
}

// SlugValidator validates that a slug is in the correct format.
type SlugValidator struct{}

func (v SlugValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Validates that the slug is in the correct format (lowercase alphanumeric characters and hyphens, starting and ending with an alphanumeric character) and does not exceed %d characters.", MaxSlugLength)
}

func (v SlugValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Validates that the slug is in the correct format (lowercase alphanumeric characters and hyphens, starting and ending with an alphanumeric character) and does not exceed %d characters.", MaxSlugLength)
}

func (v SlugValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		tflog.Debug(ctx, "Slug validation skipped for unknown or null value", map[string]interface{}{
			"is_unknown": req.ConfigValue.IsUnknown(),
			"is_null":    req.ConfigValue.IsNull(),
			"path":       req.Path.String(),
		})
		return
	}
	value := req.ConfigValue.ValueString()
	tflog.Debug(ctx, "Validating slug", map[string]interface{}{
		"value": value,
		"path":  req.Path.String(),
	})
	if err := ValidateSlugLength(value); err != nil {
		tflog.Debug(ctx, "Slug length validation failed", map[string]interface{}{
			"value":         value,
			"path":          req.Path.String(),
			"max_length":    MaxSlugLength,
			"actual_length": len(value),
		})
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Slug Length",
			err.Error(),
		)
		return
	}
	pattern := `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
	regex := regexp.MustCompile(pattern)
	if !regex.MatchString(value) {
		tflog.Debug(ctx, "Slug format validation failed", map[string]interface{}{
			"value":   value,
			"path":    req.Path.String(),
			"pattern": pattern,
		})
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Slug Format",
			"Slug must contain only lowercase alphanumeric characters and hyphens, and must start and end with an alphanumeric character.",
		)
	} else {
		tflog.Debug(ctx, "Slug validation passed", map[string]interface{}{
			"value": value,
			"path":  req.Path.String(),
		})
	}
}
