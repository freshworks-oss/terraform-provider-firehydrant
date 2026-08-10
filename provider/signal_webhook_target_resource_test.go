package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// This tests the resource with a configuration that only has the required
// attributes specified.
func TestAccSignalWebhookTargetResource_basic(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckSignalWebhookTargetResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccSignalWebhookTargetResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSignalWebhookTargetResourceExistsWithAttributes_basic("firehydrant_signal_webhook_target.test_signal_webhook_target"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_webhook_target.test_signal_webhook_target", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "name", fmt.Sprintf("test-signal-webhook-target-%s", rName)),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "url", "https://example.com/webhook"),
				),
			},
		},
	})
}

// This tests the resources ability to update and remove attributes
// with a configuration that has all attributes specified.
func TestAccSignalWebhookTargetResource_update(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rNameUpdated := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckSignalWebhookTargetResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccSignalWebhookTargetResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSignalWebhookTargetResourceExistsWithAttributes_basic("firehydrant_signal_webhook_target.test_signal_webhook_target"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_webhook_target.test_signal_webhook_target", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "name", fmt.Sprintf("test-signal-webhook-target-%s", rName)),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "url", "https://example.com/webhook"),
				),
			},
			{
				Config: testAccSignalWebhookTargetResourceConfig_update(rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSignalWebhookTargetResourceExistsWithAttributes_update("firehydrant_signal_webhook_target.test_signal_webhook_target"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_webhook_target.test_signal_webhook_target", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "name", fmt.Sprintf("test-signal-webhook-target-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "url", "https://example.com/webhook-updated"),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "description", fmt.Sprintf("test-description-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "signing_key", fmt.Sprintf("test-signing-key-%s", rNameUpdated)),
				),
			},
			{
				Config: testAccSignalWebhookTargetResourceConfig_basic(rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSignalWebhookTargetResourceExistsWithAttributes_basic("firehydrant_signal_webhook_target.test_signal_webhook_target"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_webhook_target.test_signal_webhook_target", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "name", fmt.Sprintf("test-signal-webhook-target-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_signal_webhook_target.test_signal_webhook_target", "url", "https://example.com/webhook"),
				),
			},
		},
	})
}

// This tests the resource's ability to import with a configuration that
// only has the required attributes specified.
func TestAccSignalWebhookTargetResourceImport_basic(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckSignalWebhookTargetResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccSignalWebhookTargetResourceConfig_basic(rName),
			},
			{
				ResourceName:      "firehydrant_signal_webhook_target.test_signal_webhook_target",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// This tests the resource's ability to import with a configuration that
// has all attributes specified.
func TestAccSignalWebhookTargetResourceImport_allAttributes(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckSignalWebhookTargetResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccSignalWebhookTargetResourceConfig_update(rName),
			},
			{
				ResourceName:      "firehydrant_signal_webhook_target.test_signal_webhook_target",
				ImportState:       true,
				ImportStateVerify: true,
				// The API never returns signing_key once it has been set, so an
				// imported webhook target cannot have one in state.
				ImportStateVerifyIgnore: []string{"signing_key"},
			},
		},
	})
}

// This test does a more in-depth check to test that what was in the
// configuration matches the information we get back from the API
// with a configuration that only has the required attributes specified.
func testAccCheckSignalWebhookTargetResourceExistsWithAttributes_basic(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		webhookTargetResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if webhookTargetResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		webhookTargetResponse, err := client.Sdk.Signals.GetSignalsWebhookTarget(context.TODO(), webhookTargetResource.Primary.ID)
		if err != nil {
			return err
		}

		if webhookTargetResponse.Name == nil {
			return fmt.Errorf("Unexpected name. Expected a name, got: nil")
		}
		expected, got := webhookTargetResource.Primary.Attributes["name"], *webhookTargetResponse.Name
		if expected != got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, got)
		}

		if webhookTargetResponse.URL == nil {
			return fmt.Errorf("Unexpected url. Expected a url, got: nil")
		}
		expected, got = webhookTargetResource.Primary.Attributes["url"], *webhookTargetResponse.URL
		if expected != got {
			return fmt.Errorf("Unexpected url. Expected: %s, got: %s", expected, got)
		}

		if webhookTargetResponse.Description != nil && *webhookTargetResponse.Description != "" {
			return fmt.Errorf("Unexpected description. Expected no description, got: %s", *webhookTargetResponse.Description)
		}

		return nil
	}
}

// This test does a more in-depth check to test that what was in the
// configuration matches the information we get back from the API
// with a configuration that has all attributes specified.
func testAccCheckSignalWebhookTargetResourceExistsWithAttributes_update(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		webhookTargetResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if webhookTargetResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		webhookTargetResponse, err := client.Sdk.Signals.GetSignalsWebhookTarget(context.TODO(), webhookTargetResource.Primary.ID)
		if err != nil {
			return err
		}

		if webhookTargetResponse.Name == nil {
			return fmt.Errorf("Unexpected name. Expected a name, got: nil")
		}
		expected, got := webhookTargetResource.Primary.Attributes["name"], *webhookTargetResponse.Name
		if expected != got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, got)
		}

		if webhookTargetResponse.URL == nil {
			return fmt.Errorf("Unexpected url. Expected a url, got: nil")
		}
		expected, got = webhookTargetResource.Primary.Attributes["url"], *webhookTargetResponse.URL
		if expected != got {
			return fmt.Errorf("Unexpected url. Expected: %s, got: %s", expected, got)
		}

		if webhookTargetResponse.Description == nil {
			return fmt.Errorf("Unexpected description. Expected a description, got: nil")
		}
		expected, got = webhookTargetResource.Primary.Attributes["description"], *webhookTargetResponse.Description
		if expected != got {
			return fmt.Errorf("Unexpected description. Expected: %s, got: %s", expected, got)
		}

		return nil
	}
}

// This tests that the resource gets destroyed properly
func testAccCheckSignalWebhookTargetResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_signal_webhook_target" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			_, err := client.Sdk.Signals.GetSignalsWebhookTarget(context.TODO(), stateResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Signal webhook target %s still exists", stateResource.Primary.ID)
			}
		}

		return nil
	}
}

func testAccSignalWebhookTargetResourceConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_signal_webhook_target" "test_signal_webhook_target" {
  name = "test-signal-webhook-target-%s"
  url  = "https://example.com/webhook"
}`, rName)
}

func testAccSignalWebhookTargetResourceConfig_update(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_signal_webhook_target" "test_signal_webhook_target" {
  name        = "test-signal-webhook-target-%s"
  url         = "https://example.com/webhook-updated"
  description = "test-description-%s"
  signing_key = "test-signing-key-%s"
}`, rName, rName, rName)
}

// This tests the delete path against a server that answers the way FireHydrant
// does. The Signals delete operations in the SDK treat 204 as the only success and
// report every other status, success included, as an unknown status code, so a
// delete that worked can arrive as an error carrying a 2xx status.
func TestDeleteResourceFireHydrantSignalWebhookTarget(t *testing.T) {
	const webhookTargetID = "00000000-0000-4000-8000-000000000000"

	for _, tc := range []struct {
		name        string
		status      int
		body        string
		expectError bool
		expectID    string
	}{
		{
			name:     "200 with the deleted webhook target",
			status:   http.StatusOK,
			body:     `{"id":"` + webhookTargetID + `","name":"Example","url":"https://example.com/webhook"}`,
			expectID: "",
		},
		{
			name:     "204 with no body",
			status:   http.StatusNoContent,
			expectID: "",
		},
		{
			name:     "404 when it is already gone",
			status:   http.StatusNotFound,
			body:     `{"error":"not found"}`,
			expectID: "",
		},
		{
			name:        "500 is still a failure",
			status:      http.StatusInternalServerError,
			body:        `{"error":"boom"}`,
			expectError: true,
			expectID:    webhookTargetID,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				gotMethod, gotPath = req.Method, req.URL.Path
				w.WriteHeader(tc.status)
				if tc.body != "" {
					w.Write([]byte(tc.body))
				}
			}))
			defer ts.Close()

			client, err := firehydrant.NewRestClient("test-token-very-authorized", firehydrant.WithBaseURL(ts.URL+"/v1/"))
			if err != nil {
				t.Fatalf("Received error initializing API client: %v", err)
			}

			d := schema.TestResourceDataRaw(t, resourceSignalWebhookTarget().Schema, map[string]interface{}{})
			d.SetId(webhookTargetID)

			diags := deleteResourceFireHydrantSignalWebhookTarget(context.Background(), d, client)

			if got := diags.HasError(); got != tc.expectError {
				t.Fatalf("Unexpected error state. Expected error: %v, got: %v (%v)", tc.expectError, got, diags)
			}
			if got := d.Id(); got != tc.expectID {
				t.Errorf("Unexpected ID. Expected: %q, got: %q", tc.expectID, got)
			}
			if gotMethod != http.MethodDelete {
				t.Errorf("Unexpected method. Expected: %s, got: %s", http.MethodDelete, gotMethod)
			}
			if expected := "/v1/signals/webhook_targets/" + webhookTargetID; gotPath != expected {
				t.Errorf("Unexpected path. Expected: %s, got: %s", expected, gotPath)
			}
		})
	}
}
