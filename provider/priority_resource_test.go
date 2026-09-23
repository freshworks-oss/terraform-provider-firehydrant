package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPriorityResource_basic(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckPriorityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityResourceConfig_basic(rSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPriorityResourceExistsWithAttributes_basic("firehydrant_priority.test_priority"),
					resource.TestCheckResourceAttrSet("firehydrant_priority.test_priority", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "slug", fmt.Sprintf("TESTPRIORITY%s", rSlug)),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "default", "false"),
				),
			},
		},
	})
}

func TestAccPriorityResource_update(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	rSlugUpdated := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckPriorityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityResourceConfig_basic(rSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPriorityResourceExistsWithAttributes_basic("firehydrant_priority.test_priority"),
					resource.TestCheckResourceAttrSet("firehydrant_priority.test_priority", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "slug", fmt.Sprintf("TESTPRIORITY%s", rSlug)),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "default", "false"),
				),
			},
			{
				Config: testAccPriorityResourceConfig_update(rSlugUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPriorityResourceExistsWithAttributes_update("firehydrant_priority.test_priority"),
					resource.TestCheckResourceAttrSet("firehydrant_priority.test_priority", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "slug", fmt.Sprintf("TESTPRIORITY%s", rSlugUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "description", fmt.Sprintf("test-description-%s", rSlugUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "default", "false"),
				),
			},
			{
				Config: testAccPriorityResourceConfig_basic(rSlugUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPriorityResourceExistsWithAttributes_basic("firehydrant_priority.test_priority"),
					resource.TestCheckResourceAttrSet("firehydrant_priority.test_priority", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "slug", fmt.Sprintf("TESTPRIORITY%s", rSlugUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_priority.test_priority", "default", "false"),
				),
			},
		},
	})
}

func TestAccPriorityResource_validateSchemaAttributesSlug(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityResourceConfig_slugTooLong(rSlug),
				ExpectError: regexp.MustCompile(`expected length of slug to be in the range \(0 - 23\)`),
			},
			{
				Config:      testAccPriorityResourceConfig_slugWithInvalidCharacters(rSlug),
				ExpectError: regexp.MustCompile(`invalid value for slug \(must only include letters, numbers, and hyphens\)`),
			},
		},
	})
}

func TestAccPriorityResourceImport_basic(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityResourceConfig_basic(rSlug),
			},
			{
				ResourceName:      "firehydrant_priority.test_priority",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPriorityResourceImport_allAttributes(t *testing.T) {
	t.Parallel()
	rSlug := strings.ToUpper(acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityResourceConfig_update(rSlug),
			},
			{
				ResourceName:      "firehydrant_priority.test_priority",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckPriorityResourceExistsWithAttributes_basic(resourceSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		priorityResource, ok := s.RootModule().Resources[resourceSlug]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceSlug)
		}
		if priorityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		priorityResponse, err := client.Sdk.IncidentSettings.GetPriority(context.TODO(), priorityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := priorityResource.Primary.Attributes["slug"], *priorityResponse.Slug
		if expected != got {
			return fmt.Errorf("Unexpected slug. Expected: %s, got: %s", expected, got)
		}

		expected, got = priorityResource.Primary.Attributes["default"], fmt.Sprintf("%t", *priorityResponse.Default)
		if expected != got {
			return fmt.Errorf("Unexpected default. Expected: %s, got: %s", expected, got)
		}

		if *priorityResponse.Description != "" {
			return fmt.Errorf("Unexpected description. Expected no description, got: %s", *priorityResponse.Description)
		}

		return nil
	}
}

func testAccCheckPriorityResourceExistsWithAttributes_update(resourceSlug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		priorityResource, ok := s.RootModule().Resources[resourceSlug]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceSlug)
		}
		if priorityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		priorityResponse, err := client.Sdk.IncidentSettings.GetPriority(context.TODO(), priorityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := priorityResource.Primary.Attributes["slug"], *priorityResponse.Slug
		if expected != got {
			return fmt.Errorf("Unexpected slug. Expected: %s, got: %s", expected, got)
		}

		expected, got = priorityResource.Primary.Attributes["default"], fmt.Sprintf("%t", *priorityResponse.Default)
		if expected != got {
			return fmt.Errorf("Unexpected default. Expected: %s, got: %s", expected, got)
		}

		expected, got = priorityResource.Primary.Attributes["description"], *priorityResponse.Description
		if expected != got {
			return fmt.Errorf("Unexpected description. Expected: %s, got: %s", expected, got)
		}

		return nil
	}
}

func testAccCheckPriorityResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_priority" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			_, err := client.Sdk.IncidentSettings.GetPriority(context.TODO(), stateResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Priority %s still exists", stateResource.Primary.ID)
			}
		}

		return nil
	}
}

func testAccPriorityResourceConfig_basic(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_priority" "test_priority" {
  slug = "TESTPRIORITY%s"
}`, rSlug)
}

func testAccPriorityResourceConfig_update(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_priority" "test_priority" {
  slug        = "TESTPRIORITY%s"
  description = "test-description-%s"
  default     = false
}`, rSlug, rSlug)
}

func testAccPriorityResourceConfig_slugTooLong(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_priority" "test_priority" {
  slug = "THISSLUGISWAYTOOLONG%s"
}`, rSlug)
}

func testAccPriorityResourceConfig_slugWithInvalidCharacters(rSlug string) string {
	return fmt.Sprintf(`
resource "firehydrant_priority" "test_priority" {
  slug = "INVALID_SLUG%s"
}`, rSlug)
}
