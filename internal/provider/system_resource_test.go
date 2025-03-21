// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-ctrlplane/client"
	ctrlacctest "terraform-provider-ctrlplane/testing/acctest"
)

func testAccSystemResourceConfig(name string, slug string, description *string) string {
	configTemplate := `
resource "ctrlplane_system" "test" {
  name = %[1]q
  slug = %[2]q
  %s
}`

	descriptionBlock := ""
	if description != nil {
		descriptionBlock = fmt.Sprintf("  description = %q", *description)
	}

	resourceConfig := fmt.Sprintf(configTemplate, name, slug, descriptionBlock)

	return fmt.Sprintf("%s\n%s", ctrlacctest.ProviderConfig(), resourceConfig)
}

func testAccSystemResourceConfigMissingRequired() string {
	randomSlug := "test-slug-" + strings.ToLower(acctest.RandString(5))
	return fmt.Sprintf(`%s
resource "ctrlplane_system" "test" {
  # Missing name
  slug = %q
}`, ctrlacctest.ProviderConfig(), randomSlug)
}

func testAccSystemResourceConfigMissingSlug(name string) string {
	// Ensure the name is valid for slugification
	// This will help ensure the test passes by providing a name that can be properly slugified
	// Provide a HCL-safe string without special characters
	return fmt.Sprintf(`%s
resource "ctrlplane_system" "test" {
  name = %q
  # Slug is intentionally omitted to test automatic generation
}`, ctrlacctest.ProviderConfig(), name)
}

// Helper function to create a string pointer.
func stringPtr(s string) *string {
	return &s
}

// Helper function to create a configuration with a duplicate slug.
func testAccSystemResourceConfigDuplicateSlug(slug string) string {
	return fmt.Sprintf(`%s
resource "ctrlplane_system" "test" {
  name = %q
  slug = %q
}

resource "ctrlplane_system" "duplicate" {
  name = "Another System"
  slug = %q
}`, ctrlacctest.ProviderConfig(), slug, slug, slug)
}

// Helper function to create a configuration with an invalid slug format.
func testAccSystemResourceConfigInvalidSlug(name string) string {
	return fmt.Sprintf(`%s
resource "ctrlplane_system" "test" {
  name = %q
  slug = "invalid slug with spaces"
  # The validator should reject this invalid slug format
}`, ctrlacctest.ProviderConfig(), name)
}

// testAccCheckSystemDestroy verifies the system has been destroyed.
func testAccCheckSystemDestroy(s *terraform.State) error {
	// We can't access the provider instance directly in the test framework
	// Instead, we'll make a direct API call using the environment variables
	apiKey := os.Getenv(ctrlacctest.APIKeyEnvVar)
	baseURL := os.Getenv(ctrlacctest.BaseURLEnvVar)

	if baseURL == "" {
		baseURL = "https://app.ctrlplane.dev" // Default value
	}

	// Create a client for API calls
	clientWithResponses, err := client.NewClientWithResponses(
		baseURL,
		client.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Add("Authorization", "Bearer "+apiKey)
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("error creating client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ctrlplane_system" {
			continue
		}

		// Try to get the system
		systemID, err := uuid.Parse(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("invalid system ID format: %s", err)
		}

		resp, err := clientWithResponses.GetSystemWithResponse(context.Background(), systemID)
		if err == nil && resp.StatusCode() != 404 {
			return fmt.Errorf("system still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func TestAccSystemResource(t *testing.T) {
	// Generate a unique prefix for all resources in this test
	rPrefix := acctest.RandString(8)
	firstName := fmt.Sprintf("first-%s-%s", rPrefix, acctest.RandString(5))
	secondName := fmt.Sprintf("second-%s-%s", rPrefix, acctest.RandString(5))
	thirdName := fmt.Sprintf("third-%s-%s", rPrefix, acctest.RandString(5))
	description := fmt.Sprintf("Test description %s", rPrefix)
	updatedDescription := fmt.Sprintf("Updated description %s", rPrefix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSystemResourceConfig(firstName, firstName, &description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", firstName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", firstName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "description", description),
					// Verify ID is set and is a UUID
					resource.TestMatchResourceAttr("ctrlplane_system.test", "id", regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),
				),
			},
			// ImportState testing
			{
				ResourceName:      "ctrlplane_system.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Description won't be imported as it's not part of the ID
				ImportStateVerifyIgnore: []string{"description"},
			},
			// Update and Read testing - change name and slug
			{
				Config: testAccSystemResourceConfig(secondName, secondName, &description),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", secondName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", secondName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "description", description),
				),
			},
			// Update and Read testing - change description only
			{
				Config: testAccSystemResourceConfig(secondName, secondName, &updatedDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", secondName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", secondName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "description", updatedDescription),
				),
			},
			// Update and Read testing - remove description
			{
				Config: testAccSystemResourceConfig(secondName, secondName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", secondName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", secondName),
					resource.TestCheckNoResourceAttr("ctrlplane_system.test", "description"),
				),
			},
			// Update and Read testing - change name only
			{
				Config: testAccSystemResourceConfig(thirdName, secondName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", thirdName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", secondName),
					resource.TestCheckNoResourceAttr("ctrlplane_system.test", "description"),
				),
			},
			// Update and Read testing - change slug only
			{
				Config: testAccSystemResourceConfig(thirdName, thirdName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", thirdName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", thirdName),
					resource.TestCheckNoResourceAttr("ctrlplane_system.test", "description"),
				),
			},
			// Update and Read testing - add description
			{
				Config: testAccSystemResourceConfig(thirdName, thirdName, &updatedDescription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", thirdName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", thirdName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "description", updatedDescription),
				),
			},
		},
	})
}

func TestAccSystemResourceErrorHandling(t *testing.T) {
	// Create simple values that are guaranteed to be slug-safe
	randomName := "testsys" + strings.ToLower(acctest.RandString(8))
	randomSlug := "testslug" + strings.ToLower(acctest.RandString(8))

	t.Logf("Generated randomName: %s, randomSlug: %s", randomName, randomSlug)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			// Test with missing required fields (should fail)
			{
				Config:      testAccSystemResourceConfigMissingRequired(),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			// Test with explicit slug (should succeed)
			{
				Config: testAccSystemResourceConfig(randomName, randomSlug, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", randomName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", randomSlug),
				),
			},
			// Test provided slug is invalid (should fail)
			{
				Config:      testAccSystemResourceConfigInvalidSlug(randomName),
				ExpectError: regexp.MustCompile(`Invalid Slug Format`),
			},
			// Test with missing slug (should succeed as slug is optional)
			{
				Config: testAccSystemResourceConfigMissingSlug(randomName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", randomName),
				),
			},
			// Test with duplicate slug (should fail)
			{
				Config:      testAccSystemResourceConfigDuplicateSlug(randomSlug),
				ExpectError: regexp.MustCompile(`Error: Failed to create system`),
			},
		},
	})
}

func TestAccSystemResourceInvalidSlug(t *testing.T) {
	randomName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccSystemResourceConfigInvalidSlug(randomName),
				ExpectError: regexp.MustCompile(`Invalid Slug Format`),
			},
		},
	})
}

func TestAccSystemResourceImportErrors(t *testing.T) {
	importName := fmt.Sprintf("import-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemResourceConfig(importName, importName, nil),
			},
			{
				ResourceName:      "ctrlplane_system.test",
				ImportState:       true,
				ImportStateId:     "not-a-uuid",
				ImportStateVerify: false,
				ExpectError:       regexp.MustCompile(`Invalid System ID`),
			},
			{
				ResourceName:      "ctrlplane_system.test",
				ImportState:       true,
				ImportStateId:     "00000000-0000-0000-0000-000000000000",
				ImportStateVerify: false,
				ExpectError:       regexp.MustCompile(`Failed to read system`),
			},
		},
	})
}

func TestSetSystemResourceData(t *testing.T) {
	tests := []struct {
		name           string
		inputSystem    interface{}
		expectedResult systemResourceModel
	}{
		{
			name: "nil description",
			inputSystem: &client.System{
				Id:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "Test System",
				Slug:        "test-system",
				Description: nil,
			},
			expectedResult: systemResourceModel{
				Id:          types.StringValue("00000000-0000-0000-0000-000000000001"),
				Name:        types.StringValue("Test System"),
				Slug:        types.StringValue("test-system"),
				Description: types.StringNull(),
			},
		},
		{
			name: "empty string description",
			inputSystem: &client.System{
				Id:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Name:        "Test System 2",
				Slug:        "test-system-2",
				Description: stringPtr(""),
			},
			expectedResult: systemResourceModel{
				Id:          types.StringValue("00000000-0000-0000-0000-000000000002"),
				Name:        types.StringValue("Test System 2"),
				Slug:        types.StringValue("test-system-2"),
				Description: types.StringNull(),
			},
		},
		{
			name: "non-empty description",
			inputSystem: &client.System{
				Id:          uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				Name:        "Test System 3",
				Slug:        "test-system-3",
				Description: stringPtr("This is a description"),
			},
			expectedResult: systemResourceModel{
				Id:          types.StringValue("00000000-0000-0000-0000-000000000003"),
				Name:        types.StringValue("Test System 3"),
				Slug:        types.StringValue("test-system-3"),
				Description: types.StringValue("This is a description"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result systemResourceModel
			setSystemResourceData(&result, tt.inputSystem)
			if result.Id != tt.expectedResult.Id {
				t.Errorf("ID mismatch: got %v, want %v", result.Id, tt.expectedResult.Id)
			}
			if result.Name != tt.expectedResult.Name {
				t.Errorf("Name mismatch: got %v, want %v", result.Name, tt.expectedResult.Name)
			}
			if result.Slug != tt.expectedResult.Slug {
				t.Errorf("Slug mismatch: got %v, want %v", result.Slug, tt.expectedResult.Slug)
			}
			if result.Description.IsNull() != tt.expectedResult.Description.IsNull() {
				t.Errorf("Description nullness mismatch: got IsNull=%v, want IsNull=%v",
					result.Description.IsNull(), tt.expectedResult.Description.IsNull())
			}
			if !result.Description.IsNull() && result.Description.ValueString() != tt.expectedResult.Description.ValueString() {
				t.Errorf("Description value mismatch: got %v, want %v",
					result.Description.ValueString(), tt.expectedResult.Description.ValueString())
			}
		})
	}
}

func testAccSystemResourceConfigSameNameDifferentSlug(name string, firstSlug string, secondSlug string) string {
	return fmt.Sprintf(`%s
resource "ctrlplane_system" "test" {
  name = %q
  slug = %q
}

resource "ctrlplane_system" "second" {
  name = %q
  slug = %q
}`, ctrlacctest.ProviderConfig(), name, firstSlug, name, secondSlug)
}

func TestAccSystemResourceSameNameDifferentSlug(t *testing.T) {
	// Generate a truly unique name and slugs
	rPrefix := acctest.RandString(8)
	sharedName := fmt.Sprintf("shared-%s-%s", rPrefix, acctest.RandString(5))
	firstSlug := fmt.Sprintf("first-%s-%s", rPrefix, acctest.RandString(5))
	secondSlug := fmt.Sprintf("second-%s-%s", rPrefix, acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemResourceConfig(sharedName, firstSlug, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", sharedName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", firstSlug),
				),
			},
			{
				Config: testAccSystemResourceConfigSameNameDifferentSlug(sharedName, firstSlug, secondSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", sharedName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", firstSlug),
					resource.TestCheckResourceAttr("ctrlplane_system.second", "name", sharedName),
					resource.TestCheckResourceAttr("ctrlplane_system.second", "slug", secondSlug),
				),
			},
		},
	})
}

func TestAccSystemResourceAutoGeneratedSlug(t *testing.T) {
	randomName := acctest.RandString(5)
	complexName := "Test System " + randomName
	expectedSlug := "test-system-" + strings.ToLower(randomName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemResourceConfigMissingSlug(complexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", complexName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", expectedSlug),
				),
			},
		},
	})
}

func TestAccSystemResourceComplexNameSlugification(t *testing.T) {
	rName := acctest.RandString(5)
	complexName := fmt.Sprintf("Test %s With Spaces", rName)
	expectedSlug := fmt.Sprintf("test-%s-with-spaces", strings.ToLower(rName))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemResourceConfigMissingSlug(complexName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ctrlplane_system.test", "name", complexName),
					resource.TestCheckResourceAttr("ctrlplane_system.test", "slug", expectedSlug),
				),
			},
		},
	})
}

func TestAccSystemResourceLongSlug(t *testing.T) {
	rName := acctest.RandString(5)
	longName := fmt.Sprintf("Long test %s with a very long name that will generate a slug exceeding thirty characters", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccSystemResourceConfigMissingSlug(longName),
				ExpectError: regexp.MustCompile("slug must not exceed 30 characters"),
			},
		},
	})
}

func TestAccSystemResourceExplicitLongSlug(t *testing.T) {
	longSlug := "this-is-a-very-long-slug-that-exceeds-thirty-characters"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccSystemResourceConfigWithLongSlug("Test System", longSlug),
				ExpectError: regexp.MustCompile("slug must not exceed 30 characters"),
			},
		},
	})
}

func testAccSystemResourceConfigWithLongSlug(name string, slug string) string {
	return fmt.Sprintf(`%s
resource "ctrlplane_system" "test" {
  name = %q
  slug = %q
}`, ctrlacctest.ProviderConfig(), name, slug)
}

func TestAccSystemResourceUpdateWithLongSlug(t *testing.T) {
	resourceName := "ctrlplane_system.test"
	rName := acctest.RandString(5)
	initialName := fmt.Sprintf("Initial System %s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
	initialSlug := fmt.Sprintf("initial-system-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))
	updatedName := fmt.Sprintf("Updated System %s", rName)
	longSlug := fmt.Sprintf("very-long-slug-%s-that-exceeds-thirty-characters", rName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSystemResourceConfigWithLongSlug(initialName, initialSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", initialName),
					resource.TestCheckResourceAttr(resourceName, "slug", initialSlug),
				),
			},
			{
				Config:      testAccSystemResourceConfigWithLongSlug(updatedName, longSlug),
				ExpectError: regexp.MustCompile("slug must not exceed 30 characters"),
				PlanOnly:    true,
			},
			{
				Config: testAccSystemResourceConfigWithLongSlug(initialName, initialSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", initialName),
					resource.TestCheckResourceAttr(resourceName, "slug", initialSlug),
				),
			},
		},
	})
}

// TODO We need full CRUD on the system resource so we don't have to destroy and recreate the system to update the slug

// TestAccSystemResourceRecreateOnSlugChange verifies that updating the slug
// causes the resource to be replaced.
func TestAccSystemResourceRecreateOnSlugChange(t *testing.T) {
	resourceName := "ctrlplane_system.test"
	replaceName := fmt.Sprintf("ReplaceSystem%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum))
	firstSlug := fmt.Sprintf("first-slug-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))
	secondSlug := fmt.Sprintf("second-slug-%s", acctest.RandStringFromCharSet(5, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { ctrlacctest.PreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSystemDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create a system with the initial slug.
			{
				Config: testAccSystemResourceConfig(replaceName, firstSlug, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", replaceName),
					resource.TestCheckResourceAttr(resourceName, "slug", firstSlug),
				),
			},
			// Step 2: Update the configuration to use a different slug.
			// This change should force a resource replacement.
			{
				Config: testAccSystemResourceConfig(replaceName, secondSlug, nil),
				// We only run the plan phase here to verify that a replacement is required.
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				// Optionally, you can add checks against the plan output if your test framework supports it.
			},
		},
	})
}
