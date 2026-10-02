package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNotificationPolicyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: defaultProviderFactories(),
		CheckDestroy:      testAccCheckNotificationPolicyResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPolicyResourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNotificationPolicyResourceExistsWithAttributes_basic("firehydrant_notification_policy.test_policy"),
					resource.TestCheckResourceAttrSet("firehydrant_notification_policy.test_policy", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "notification_group_method", "any"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "max_delay", "PT5M"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "priority", "HIGH"),
				),
			},
		},
	})
}

func TestAccNotificationPolicyResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: defaultProviderFactories(),
		CheckDestroy:      testAccCheckNotificationPolicyResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPolicyResourceConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNotificationPolicyResourceExistsWithAttributes_basic("firehydrant_notification_policy.test_policy"),
					resource.TestCheckResourceAttrSet("firehydrant_notification_policy.test_policy", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "notification_group_method", "any"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "max_delay", "PT5M"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "priority", "HIGH"),
				),
			},
			{
				Config: testAccNotificationPolicyResourceConfig_update(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckNotificationPolicyResourceExistsWithAttributes_basic("firehydrant_notification_policy.test_policy"),
					resource.TestCheckResourceAttrSet("firehydrant_notification_policy.test_policy", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "notification_group_method", "chat"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "max_delay", "PT3M"),
					resource.TestCheckResourceAttr(
						"firehydrant_notification_policy.test_policy", "priority", "LOW"),
				),
			},
		},
	})
}

func testAccCheckNotificationPolicyResourceExistsWithAttributes_basic(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		notificationPolicyResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if notificationPolicyResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := firehydrant.NewRestClient(os.Getenv("FIREHYDRANT_API_KEY"))
		if err != nil {
			return err
		}

		response, err := client.Sdk.Signals.GetNotificationPolicy(context.TODO(), notificationPolicyResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := notificationPolicyResource.Primary.Attributes["notification_group_method"], string(*response.NotificationGroupMethod)
		if expected != got {
			return fmt.Errorf("Unexpected notification_group_method. Expected: %s, got: %s", expected, got)
		}

		expected, got = notificationPolicyResource.Primary.Attributes["max_delay"], *response.MaxDelay
		if expected != got {
			return fmt.Errorf("Unexpected max_delay. Expected: %s, got: %s", expected, got)
		}

		expected, got = notificationPolicyResource.Primary.Attributes["priority"], string(*response.Priority)
		if expected != got {
			return fmt.Errorf("Unexpected priority. Expected: %s, got: %s", expected, got)
		}

		return nil
	}
}

func testAccCheckNotificationPolicyResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := firehydrant.NewRestClient(os.Getenv("FIREHYDRANT_API_KEY"))
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_notification_policy" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			_, err := client.Sdk.Signals.GetNotificationPolicy(context.TODO(), stateResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Notification Policy %s still exists", stateResource.Primary.ID)
			}
		}

		return nil
	}
}

func testAccNotificationPolicyResourceConfig_basic() string {
	return `
resource "firehydrant_notification_policy" "test_policy" {
  notification_group_method = "any"
	max_delay = "PT5M"
	priority = "HIGH"
}`
}

func testAccNotificationPolicyResourceConfig_update() string {
	return `
resource "firehydrant_notification_policy" "test_policy" {
  notification_group_method = "chat"
	max_delay = "PT3M"
	priority = "Low"
}`
}
