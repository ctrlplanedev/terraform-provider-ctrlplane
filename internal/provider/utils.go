// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import "github.com/google/uuid"

// stringToPtr returns a pointer to the given string.
func stringToPtr(s string) *string {
	return &s
}

// uuidToPtr returns a pointer to the given UUID.
func uuidToPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
