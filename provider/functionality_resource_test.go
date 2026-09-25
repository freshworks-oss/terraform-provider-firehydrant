package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestFunctionalityResourceCreatePayload(t *testing.T) {
	t.Parallel()

	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/functionalities":
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode create payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/functionalities/functionality-id":
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}

		_, _ = w.Write([]byte(`{
  "id": "functionality-id",
  "name": "checkout",
  "description": "Customer checkout",
  "labels": {"lifecycle": "production"},
  "service_tier": 1,
  "alert_on_add": true,
  "auto_add_responding_team": true,
  "services": [{"id": "service-id"}],
  "environments": [{"id": "environment-id"}],
  "links": [{"href_url": "https://example.com/dashboard", "icon_url": "https://example.com/icon.png", "name": "Dashboard"}],
  "external_resources": [{"remote_id": "remote-id", "connection_type": "github"}],
  "owner": {"id": "owner-id"},
  "teams": [{"id": "team-id"}]
}`))
	}))
	defer server.Close()

	functionality := resourceFunctionality()
	data := schema.TestResourceDataRaw(t, functionality.Schema, map[string]interface{}{
		"name":                     "checkout",
		"description":              "Customer checkout",
		"alert_on_add":             true,
		"auto_add_responding_team": true,
		"environment_ids":          []interface{}{"environment-id"},
		"external_resources": []interface{}{map[string]interface{}{
			"connection_type": "github",
			"remote_id":       "remote-id",
		}},
		"labels": map[string]interface{}{"lifecycle": "production"},
		"links": []interface{}{map[string]interface{}{
			"href_url": "https://example.com/dashboard",
			"icon_url": "https://example.com/icon.png",
			"name":     "Dashboard",
		}},
		"owner_id":     "owner-id",
		"service_ids":  []interface{}{"service-id"},
		"service_tier": 1,
		"team_ids":     []interface{}{"team-id"},
	})
	client := &firehydrant.APIClient{
		Sdk: fhsdk.New(fhsdk.WithServerURL(server.URL)),
	}

	if diagnostics := createResourceFireHydrantFunctionality(context.Background(), data, client); diagnostics.HasError() {
		t.Fatalf("create functionality returned diagnostics: %v", diagnostics)
	}

	expectedPayload := map[string]interface{}{
		"name":                     "checkout",
		"description":              "Customer checkout",
		"alert_on_add":             true,
		"auto_add_responding_team": true,
		"environments": []interface{}{map[string]interface{}{
			"id": "environment-id",
		}},
		"external_resources": []interface{}{map[string]interface{}{
			"connection_type": "github",
			"remote_id":       "remote-id",
		}},
		"labels": map[string]interface{}{"lifecycle": "production"},
		"links": []interface{}{map[string]interface{}{
			"href_url": "https://example.com/dashboard",
			"icon_url": "https://example.com/icon.png",
			"name":     "Dashboard",
		}},
		"owner": map[string]interface{}{"id": "owner-id"},
		"services": []interface{}{map[string]interface{}{
			"id": "service-id",
		}},
		"service_tier": float64(1),
		"teams": []interface{}{map[string]interface{}{
			"id": "team-id",
		}},
	}
	if !reflect.DeepEqual(payload, expectedPayload) {
		t.Errorf("create payload = %#v, want %#v", payload, expectedPayload)
	}
}

func TestFunctionalityResourceUpdateClearsCollections(t *testing.T) {
	t.Parallel()

	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/functionalities/functionality-id":
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode update payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case r.Method != http.MethodGet || r.URL.Path != "/v1/functionalities/functionality-id":
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}

		_, _ = w.Write([]byte(`{"id":"functionality-id","name":"checkout","service_tier":5}`))
	}))
	defer server.Close()

	functionality := resourceFunctionality()
	data := schema.TestResourceDataRaw(t, functionality.Schema, map[string]interface{}{
		"name": "checkout",
	})
	data.SetId("functionality-id")
	client := &firehydrant.APIClient{
		Sdk: fhsdk.New(fhsdk.WithServerURL(server.URL)),
	}

	if diagnostics := updateResourceFireHydrantFunctionality(context.Background(), data, client); diagnostics.HasError() {
		t.Fatalf("update functionality returned diagnostics: %v", diagnostics)
	}

	for _, field := range []string{"environments", "external_resources", "links", "services", "teams"} {
		if value, ok := payload[field]; !ok || !reflect.DeepEqual(value, []interface{}{}) {
			t.Errorf("update payload %q = %#v, want an empty array", field, value)
		}
	}
	for _, field := range []string{
		"remove_remaining_environments",
		"remove_remaining_external_resources",
		"remove_remaining_services",
		"remove_remaining_teams",
	} {
		if value, ok := payload[field]; !ok || value != true {
			t.Errorf("update payload %q = %#v, want true", field, value)
		}
	}
}

func TestAccFunctionalityResource_basic(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFunctionalityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionalityResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckFunctionalityResourceExistsWithAttributes_basic("firehydrant_functionality.test_functionality"),
					resource.TestCheckResourceAttrSet("firehydrant_functionality.test_functionality", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "name", fmt.Sprintf("test-functionality-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "service_ids.#", "0"),
				),
			},
		},
	})
}

func TestFunctionalityResourceSchema(t *testing.T) {
	t.Parallel()

	functionality := resourceFunctionality()
	if err := functionality.InternalValidate(nil, true); err != nil {
		t.Fatalf("functionality resource schema is invalid: %v", err)
	}

	expectedFields := []string{
		"alert_on_add",
		"auto_add_responding_team",
		"description",
		"environment_ids",
		"external_resources",
		"labels",
		"links",
		"name",
		"owner_id",
		"service_ids",
		"service_tier",
		"team_ids",
	}
	for _, field := range expectedFields {
		if _, ok := functionality.Schema[field]; !ok {
			t.Errorf("functionality resource schema does not contain %q", field)
		}
	}

	serviceTier := functionality.Schema["service_tier"]
	if got, want := serviceTier.Default, 5; got != want {
		t.Errorf("service_tier default = %v, want %v", got, want)
	}
	for _, value := range []int{0, 5} {
		_, errors := serviceTier.ValidateFunc(value, "service_tier")
		if len(errors) != 0 {
			t.Errorf("service_tier rejected valid value %d: %v", value, errors)
		}
	}
	for _, value := range []int{-1, 6} {
		_, errors := serviceTier.ValidateFunc(value, "service_tier")
		if len(errors) == 0 {
			t.Errorf("service_tier accepted invalid value %d", value)
		}
	}

	links, ok := functionality.Schema["links"].Elem.(*schema.Resource)
	if !ok {
		t.Fatal("links element is not a nested resource")
	}
	for _, field := range []string{"href_url", "icon_url", "name"} {
		if _, ok := links.Schema[field]; !ok {
			t.Errorf("links schema does not contain %q", field)
		}
	}
}

func TestAccFunctionalityResource_update(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rNameUpdated := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFunctionalityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionalityResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckFunctionalityResourceExistsWithAttributes_basic("firehydrant_functionality.test_functionality"),
					resource.TestCheckResourceAttrSet("firehydrant_functionality.test_functionality", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "name", fmt.Sprintf("test-functionality-%s", rName)),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "service_ids.#", "0"),
				),
			},
			{
				Config: testAccFunctionalityResourceConfig_update(rNameUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFunctionalityResourceExistsWithAttributes_update("firehydrant_functionality.test_functionality"),
					resource.TestCheckResourceAttrSet("firehydrant_functionality.test_functionality", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "name", fmt.Sprintf("test-functionality-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "description", fmt.Sprintf("test-description-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "service_ids.#", "2"),
					resource.TestCheckResourceAttrSet("firehydrant_functionality.test_functionality", "owner_id"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "team_ids.#", "2"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "alert_on_add", "true"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "auto_add_responding_team", "true"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "environment_ids.#", "1"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "links.#", "2"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "service_tier", "1"),
				),
			},
			{
				Config: testAccFunctionalityResourceConfig_basic(rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckFunctionalityResourceExistsWithAttributes_basic("firehydrant_functionality.test_functionality"),
					resource.TestCheckResourceAttrSet("firehydrant_functionality.test_functionality", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "name", fmt.Sprintf("test-functionality-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "service_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccFunctionalityResourceImport_basic(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionalityResourceConfig_basic(rName),
			},

			{
				ResourceName:      "firehydrant_functionality.test_functionality",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFunctionalityResourceImport_allAttributes(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionalityResourceConfig_update(rName),
			},

			{
				ResourceName:      "firehydrant_functionality.test_functionality",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFunctionalityResource_withoutAutoAddRespondingTeam(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFunctionalityResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFunctionalityResourceConfig_withoutAutoAddRespondingTeam(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckFunctionalityResourceExistsWithAttributes_withoutAutoAddRespondingTeam("firehydrant_functionality.test_functionality"),
					resource.TestCheckResourceAttrSet("firehydrant_functionality.test_functionality", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_functionality.test_functionality", "name", fmt.Sprintf("test-functionality-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "auto_add_responding_team", "false"),
				),
			},
			{
				Config: testAccFunctionalityResourceConfig_withoutAutoAddRespondingTeam(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckFunctionalityResourceExistsWithAttributes_withoutAutoAddRespondingTeam("firehydrant_functionality.test_functionality"),
					resource.TestCheckResourceAttr("firehydrant_functionality.test_functionality", "auto_add_responding_team", "false"),
				),
			},
		},
	})
}

func testAccCheckFunctionalityResourceExistsWithAttributes_basic(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		functionalityResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if functionalityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		functionalityResponse, err := client.Sdk.CatalogEntries.GetFunctionality(context.TODO(), functionalityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := functionalityResource.Primary.Attributes["name"], *functionalityResponse.Name
		if expected != got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, got)
		}

		if functionalityResponse.Description != nil && *functionalityResponse.Description != "" {
			return fmt.Errorf("Unexpected description. Expected no description, got: %s", *functionalityResponse.Description)
		}

		if len(functionalityResponse.Services) != 0 {
			return fmt.Errorf("Unexpected number of service_ids. Expected no service_ids, got: %v", len(functionalityResponse.Services))
		}

		if len(functionalityResponse.Environments) != 0 {
			return fmt.Errorf("Unexpected number of environment_ids. Expected no environment_ids, got: %v", len(functionalityResponse.Environments))
		}

		if len(functionalityResponse.Links) != 0 {
			return fmt.Errorf("Unexpected number of links. Expected no links, got: %v", len(functionalityResponse.Links))
		}

		return nil
	}
}

func testAccCheckFunctionalityResourceExistsWithAttributes_update(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		functionalityResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if functionalityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		functionalityResponse, err := client.Sdk.CatalogEntries.GetFunctionality(context.TODO(), functionalityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := functionalityResource.Primary.Attributes["name"], *functionalityResponse.Name
		if expected != got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, got)
		}

		expected, got = functionalityResource.Primary.Attributes["description"], *functionalityResponse.Description
		if expected != got {
			return fmt.Errorf("Unexpected description. Expected: %s, got: %s", expected, got)
		}

		// TODO: Check the service ids
		if len(functionalityResponse.Services) != 2 {
			return fmt.Errorf("Unexpected number of service_ids. Expected: 2, got: %v", len(functionalityResponse.Services))
		}

		if len(functionalityResponse.Environments) != 1 {
			return fmt.Errorf("Unexpected number of environment_ids. Expected: 1, got: %v", len(functionalityResponse.Environments))
		}

		if len(functionalityResponse.Links) != 2 {
			return fmt.Errorf("Unexpected number of links. Expected: 2, got: %v", len(functionalityResponse.Links))
		}

		if functionalityResponse.AlertOnAdd == nil || !*functionalityResponse.AlertOnAdd {
			return fmt.Errorf("Unexpected alert_on_add. Expected: true, got: %v", functionalityResponse.AlertOnAdd)
		}

		if functionalityResponse.ServiceTier == nil || *functionalityResponse.ServiceTier != 1 {
			return fmt.Errorf("Unexpected service_tier. Expected: 1, got: %v", functionalityResponse.ServiceTier)
		}

		return nil
	}
}

func testAccCheckFunctionalityResourceExistsWithAttributes_withoutAutoAddRespondingTeam(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		functionalityResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if functionalityResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		// This read operation would have crashed before the fix if AutoAddRespondingTeam was nil
		functionalityResponse, err := client.Sdk.CatalogEntries.GetFunctionality(context.TODO(), functionalityResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := functionalityResource.Primary.Attributes["name"], *functionalityResponse.Name
		if expected != got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, got)
		}

		// Verify that auto_add_responding_team is handled correctly even if nil from API
		// The fix should default it to false when nil
		autoAddRespondingTeam := false
		if functionalityResponse.AutoAddRespondingTeam != nil {
			autoAddRespondingTeam = *functionalityResponse.AutoAddRespondingTeam
		}
		expectedBool, gotBool := functionalityResource.Primary.Attributes["auto_add_responding_team"], fmt.Sprintf("%t", autoAddRespondingTeam)
		if expectedBool != gotBool {
			return fmt.Errorf("Unexpected auto_add_responding_team. Expected: %s, got: %s", expectedBool, gotBool)
		}

		return nil
	}
}

func testAccCheckFunctionalityResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, functionalityResource := range s.RootModule().Resources {
			if functionalityResource.Type != "firehydrant_functionality" {
				continue
			}

			if functionalityResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			_, err := client.Sdk.CatalogEntries.GetFunctionality(context.TODO(), functionalityResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Functionality %s still exists", functionalityResource.Primary.ID)
			}
		}

		return nil
	}
}

func testAccFunctionalityResourceConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_functionality" "test_functionality" {
  name = "test-functionality-%s"
  labels = {
    test1 = "test-label1-foo",
  }
}`, rName)
}

func testAccFunctionalityResourceConfig_update(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_service" "test_service1" {
  name = "test-service1-%s"
}

resource "firehydrant_service" "test_service2" {
  name = "test-service2-%s"
}

resource "firehydrant_team" "test_team1" {
  name = "test-team1-%s"
}

resource "firehydrant_team" "test_team2" {
  name = "test-team2-%s"
}

resource "firehydrant_team" "test_team3" {
  name = "test-team3-%s"
}

resource "firehydrant_environment" "test_environment" {
  name = "test-environment-%s"
}

resource "firehydrant_functionality" "test_functionality" {
  name         = "test-functionality-%s"
  alert_on_add = true
  description  = "test-description-%s"
  labels = {
    test1 = "test-label1-foo",
  }

  service_ids = [
    firehydrant_service.test_service1.id,
    firehydrant_service.test_service2.id
  ]

  environment_ids = [
    firehydrant_environment.test_environment.id
  ]

  links {
    href_url = "https://example.com/test-link1-%s"
    icon_url = "https://example.com/test-icon1-%s"
    name     = "test-link1-%s"
  }

  links {
    href_url = "https://example.com/test-link2-%s"
    name     = "test-link2-%s"
  }

  owner_id = firehydrant_team.test_team1.id
  team_ids = [
    firehydrant_team.test_team2.id,
    firehydrant_team.test_team3.id
  ]
  auto_add_responding_team = true
  service_tier             = 1
}`, rName, rName, rName, rName, rName, rName, rName, rName, rName, rName, rName, rName, rName)
}

func testAccFunctionalityResourceConfig_withoutAutoAddRespondingTeam(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_functionality" "test_functionality" {
  name = "test-functionality-%s"
  labels = {
    test1 = "test-label1-foo",
  }
}`, rName)
}
