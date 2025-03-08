package provider

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string",
			input:    "Simple Test",
			expected: "simple-test",
		},
		{
			name:     "String with special characters",
			input:    "Test & Special",
			expected: "test-and-special",
		},
		{
			name:     "String with multiple special characters",
			input:    "Begin && & || | + @ # $ % * = ! ? ---   End -",
			expected: "begin-andand-and-oror-or-+-@-dollar-percent-*-!-end",
		},
		{
			name:     "String with multiple special characters no spaces",
			input:    "Begin&&&|||+@#$%*=!?---End -",
			expected: "beginandandandororor+@dollarpercent*!-end",
		},
		{
			name:     "String with multiple special characters and hyphens",
			input:    "- Begin----&&&---|||--+-@-#-$-%-*-=-!-?---End -",
			expected: "begin-andandand-ororor-+-@-dollar-percent-*-!-end",
		},
		{
			name:     "String with consecutive spaces",
			input:    "Test   with   spaces",
			expected: "test-with-spaces",
		},
		{
			name:     "String with leading and trailing spaces",
			input:    "   Test with spaces   ",
			expected: "test-with-spaces",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "String with only special characters",
			input:    "&||+@#$%*=!?",
			expected: "andoror+@dollarpercent*!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateSlugLength(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Valid slug length",
			input:       "valid-slug",
			expectError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: false,
		},
		{
			name:        "Maximum allowed length",
			input:       "abcdefghijklmnopqrstuvwxyz1234", // 30 characters
			expectError: false,
		},
		{
			name:        "Exceeds maximum length",
			input:       "abcdefghijklmnopqrstuvwxyz12345", // 31 characters
			expectError: true,
		},
		{
			name:        "Way too long",
			input:       "this-is-a-very-long-slug-that-definitely-exceeds-the-maximum-allowed-length-for-slugs-in-the-system",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlugLength(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateSlugLength(%q) error = %v, expectError %v", tt.input, err, tt.expectError)
			}
		})
	}
} 