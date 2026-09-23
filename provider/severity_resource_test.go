package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSeverityResource_basic(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckSeverityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccSeverityResourceConfig_basic(rSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSeverityResourceExistsWithAttributes_basic("firehydrant_severity.test_severity"),
					resource.TestCheckResourceAttrSet("firehydrant_severity.test_severity", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "slug", fmt.Sprintf("TESTSEVERITY%s", rSlug)),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "type", string(firehydrant.SeverityTypeUnexpectedDowntime)),
				),
			},
		},
	})
}

func TestAccSeverityResource_update(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	rSlugUpdated := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckSeverityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccSeverityResourceConfig_basic(rSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSeverityResourceExistsWithAttributes_basic("firehydrant_severity.test_severity"),
					resource.TestCheckResourceAttrSet("firehydrant_severity.test_severity", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "slug", fmt.Sprintf("TESTSEVERITY%s", rSlug)),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "type", string(firehydrant.SeverityTypeUnexpectedDowntime)),
				),
			},
			{
				Config: testAccSeverityResourceConfig_update(rSlugUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSeverityResourceExistsWithAttributes_update("firehydrant_severity.test_severity"),
					resource.TestCheckResourceAttrSet("firehydrant_severity.test_severity", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "slug", fmt.Sprintf("TESTSEVERITY%s", rSlugUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "description", fmt.Sprintf("test-description-%s", rSlugUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "type", string(firehydrant.SeverityTypeMaintenance)),
				),
			},
			{
				Config: testAccSeverityResourceConfig_basic(rSlugUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSeverityResourceExistsWithAttributes_basic("firehydrant_severity.test_severity"),
					resource.TestCheckResourceAttrSet("firehydrant_severity.test_severity", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "slug", fmt.Sprintf("TESTSEVERITY%s", rSlugUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_severity.test_severity", "type", string(firehydrant.SeverityTypeUnexpectedDowntime)),
				),
			},
		},
	})
}

func TestAccSeverityResource_validateSchemaAttributesSlug(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccSeverityResourceConfig_slugTooLong(rSlug),
				ExpectError: regexp.MustCompile(`expected length of slug to be in the range \(0 - 23\)`),
			},
			{
				Config:      testAccSeverityResourceConfig_slugWithInvalidCharacters(rSlug),
				ExpectError: regexp.MustCompile(`invalid value for slug \(must only include letters, numbers, and hyphens\)`),
			},
		},
	})
}

func TestAccSeverityResourceImport_basic(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSeverityResourceConfig_basic(rSlug),
			},
			{
				ResourceName:      "firehydrant_severity.test_severity",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSeverityResourceImport_allAttributes(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSeverityResourceConfig_update(rSlug),
			},
			{
				ResourceName:      "firehydrant_severity.test_severity",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckSeverityResourceExistsWithAttributes_basic(resourceSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		severityResource, ok := s.RootModule().Resources[resourceSlug]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceSlug)
		}
		if severityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		severityResponse, err := client.Severities().Get(context.TODO(), severityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := severityResource.Primary.Attributes["slug"], severityResponse.Slug
		if expected != got {
			return fmt.Errorf("Unexpected slug. Expected: %s, got: %s", expected, got)
		}

		if severityResponse.Description != "" {
			return fmt.Errorf("Unexpected description. Expected no description, got: %s", severityResponse.Description)
		}

		if severityResponse.Type != string(firehydrant.SeverityTypeUnexpectedDowntime) {
			return fmt.Errorf("Unexpected type. Expected default type of %s, got: %s", string(firehydrant.SeverityTypeUnexpectedDowntime), severityResponse.Type)
		}
		expected, got = severityResource.Primary.Attributes["type"], severityResponse.Type
		if expected != got {
			return fmt.Errorf("Unexpected type. Expected: %s, got: %s", expected, got)
		}

		return nil
	}
}

func testAccCheckSeverityResourceExistsWithAttributes_update(resourceSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		severityResource, ok := s.RootModule().Resources[resourceSlug]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceSlug)
		}
		if severityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		severityResponse, err := client.Severities().Get(context.TODO(), severityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := severityResource.Primary.Attributes["slug"], severityResponse.Slug
		if expected != got {
			return fmt.Errorf("Unexpected slug. Expected: %s, got: %s", expected, got)
		}

		expected, got = severityResource.Primary.Attributes["description"], severityResponse.Description
		if expected != got {
			return fmt.Errorf("Unexpected description. Expected: %s, got: %s", expected, got)
		}

		expected, got = severityResource.Primary.Attributes["type"], severityResponse.Type
		if expected != got {
			return fmt.Errorf("Unexpected type. Expected: %s, got: %s", expected, got)
		}

		return nil
	}
}

func testAccCheckSeverityResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_severity" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			_, err := client.Severities().Get(context.TODO(), stateResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Severity %s still exists", stateResource.Primary.ID)
			}
		}

		return nil
	}
}

func testAccSeverityResourceConfig_basic(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_severity" "test_severity" {
  slug = "TESTSEVERITY%s"
}`, rSlug)
}

func testAccSeverityResourceConfig_update(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_severity" "test_severity" {
  slug        = "TESTSEVERITY%s"
  description = "test-description-%s"
  type        = "maintenance"
}`, rSlug, rSlug)
}

func testAccSeverityResourceConfig_slugTooLong(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_severity" "test_severity" {
  slug = "THISSLUGISWAYTOOLONG%s"
}`, rSlug)
}

func testAccSeverityResourceConfig_slugWithInvalidCharacters(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_severity" "test_severity" {
  slug = "INVALID_SLUG%s"
}`, rSlug)
}

func TestOfflineSeverityAndPrioritySlugValidation(t *testing.T) {
	t.Parallel()

	slugs := []struct {
		slug  string
		valid bool
	}{
		{slug: "SEV1", valid: true},
		{slug: "P1-CRITICAL", valid: true},
		{slug: "p1-critical", valid: true},
		{slug: "A-B-C", valid: true},
		{slug: strings.Repeat("A", 23), valid: true},
		{slug: "INVALID_SLUG", valid: false},
		{slug: "INVALID SLUG", valid: false},
		{slug: "INVALID/SLUG", valid: false},
		{slug: "INVALID.SLUG", valid: false},
		{slug: "", valid: false},
		{slug: strings.Repeat("A", 24), valid: false},
	}

	resources := []struct {
		name     string
		resource *schema.Resource
	}{
		{name: "firehydrant_severity", resource: resourceSeverity()},
		{name: "firehydrant_priority", resource: resourcePriority()},
	}

	for _, r := range resources {
		validate := r.resource.Schema["slug"].ValidateDiagFunc
		if validate == nil {
			t.Fatalf("%s: slug has no ValidateDiagFunc", r.name)
		}

		for _, tc := range slugs {
			t.Run(fmt.Sprintf("%s/%q", r.name, tc.slug), func(t *testing.T) {
				diags := validate(tc.slug, cty.GetAttrPath("slug"))
				if tc.valid && diags.HasError() {
					t.Errorf("expected %q to be accepted, got: %v", tc.slug, diags)
				}
				if !tc.valid && !diags.HasError() {
					t.Errorf("expected %q to be rejected, but it passed validation", tc.slug)
				}
			})
		}
	}
}
