package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccIncidentTypeResource_basic(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckIncidentTypeResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccIncidentTypeResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckIncidentTypeResourceExistsWithAttributes_basic("firehydrant_incident_type.test_incident_type"),
					resource.TestCheckResourceAttrSet("firehydrant_incident_type.test_incident_type", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "name", fmt.Sprintf("test-incident-type-%s", rName)),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "description", fmt.Sprintf("test-description-%s", rName)),
				),
			},
		},
	})
}

func TestAccIncidentTypeResource_update(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rNameUpdated := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckIncidentTypeResourceDestroy(),
			testAccCheckTeamResourceDestroy(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccIncidentTypeResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckIncidentTypeResourceExistsWithAttributes_basic("firehydrant_incident_type.test_incident_type"),
					resource.TestCheckResourceAttrSet("firehydrant_incident_type.test_incident_type", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "name", fmt.Sprintf("test-incident-type-%s", rName)),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "description", fmt.Sprintf("test-description-%s", rName)),
				),
			},
			{
				Config: testAccIncidentTypeResourceConfig_basic(rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckIncidentTypeResourceExistsWithAttributes_basic("firehydrant_incident_type.test_incident_type"),
					resource.TestCheckResourceAttrSet("firehydrant_incident_type.test_incident_type", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "name", fmt.Sprintf("test-incident-type-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "description", fmt.Sprintf("test-description-%s", rNameUpdated)),
				),
			},
			{
				Config: testAccIncidentTypeResourceConfig_update(rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckIncidentTypeResourceExistsWithAttributes_update("firehydrant_incident_type.test_incident_type"),
					resource.TestCheckResourceAttrSet("firehydrant_incident_type.test_incident_type", "id"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "name", fmt.Sprintf("test-incident-type-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "description", fmt.Sprintf("test-description-%s", rNameUpdated)),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.description", "test-template-description"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.customer_impact_summary", "test-summary"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.severity_slug", "SEV1"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.priority_slug", "TESTPRIORITY"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.private_incident", "false"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.tags.0", "foo"),
					resource.TestCheckResourceAttr(
						"firehydrant_incident_type.test_incident_type", "template.0.tags.1", "bar"),
					resource.TestCheckResourceAttrSet(
						"firehydrant_incident_type.test_incident_type", "template.0.team_ids.0"),
					resource.TestCheckResourceAttrSet(
						"firehydrant_incident_type.test_incident_type", "template.0.team_ids.1"),
					resource.TestCheckResourceAttrSet(
						"firehydrant_incident_type.test_incident_type", "template.0.runbook_ids.0"),
				),
			},
		},
	})
}

func TestAccIncidentTypeResourceImport_basic(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccIncidentTypeResourceConfig_basic(rName),
			},
			{
				ResourceName:      "firehydrant_incident_type.test_incident_type",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccIncidentTypeResourceImport_allAttributes(t *testing.T) {
	t.Parallel()
	rName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccIncidentTypeResourceConfig_update(rName),
			},
			{
				ResourceName:      "firehydrant_incident_type.test_incident_type",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckIncidentTypeResourceExistsWithAttributes_basic(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		incidentTypeResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if incidentTypeResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		incidentTypeResponse, err := client.Sdk.IncidentSettings.GetIncidentType(context.TODO(), incidentTypeResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := incidentTypeResource.Primary.Attributes["name"], incidentTypeResponse.Name
		if expected != *got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, *got)
		}

		expected, got = incidentTypeResource.Primary.Attributes["description"], incidentTypeResponse.Description
		if expected != *got {
			return fmt.Errorf("Unexpected summary. Expected: %s, got: %s", expected, *got)
		}

		return nil
	}
}

func testAccCheckIncidentTypeResourceExistsWithAttributes_update(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		incidentTypeResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		if incidentTypeResource.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		incidentTypeResponse, err := client.Sdk.IncidentSettings.GetIncidentType(context.TODO(), incidentTypeResource.Primary.ID)
		if err != nil {
			return err
		}

		expected, got := incidentTypeResource.Primary.Attributes["name"], incidentTypeResponse.Name
		if expected != *got {
			return fmt.Errorf("Unexpected name. Expected: %s, got: %s", expected, *got)
		}

		expected, got = incidentTypeResource.Primary.Attributes["description"], incidentTypeResponse.Description
		if expected != *got {
			return fmt.Errorf("Unexpected description. Expected: %s, got: %s", expected, *got)
		}

		return nil
	}
}

func testAccCheckIncidentTypeResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_incident_type" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			_, err := client.Sdk.IncidentSettings.GetIncidentType(context.TODO(), stateResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Incident Type %s still exists", stateResource.Primary.ID)
			}
		}

		return nil
	}
}

func testAccIncidentTypeResourceConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_incident_type" "test_incident_type" {
  name        = "test-incident-type-%s"
  description = "test-description-%s"

	template {}
}`, rName, rName)
}

func testAccIncidentTypeResourceConfig_update(rName string) string {
	return fmt.Sprintf(`
resource "firehydrant_team" "test_team_1" {
  name = "test-team-1-%s"
}

resource "firehydrant_team" "test_team_2" {
  name = "test-team-2-%s"
}



data "firehydrant_runbook_action" "create_incident_channel" {
  slug             = "create_incident_channel"
  integration_slug = "slack"
}

resource "firehydrant_runbook" "test_runbook_1" {
  name = "test-runbook-1-%s"

  steps {
    name      = "Create Incident Channel"
    action_id = data.firehydrant_runbook_action.create_incident_channel.id

    config = jsonencode({
      channel_name_format = "-inc-{{ number }}"
    })
  }
}

resource "firehydrant_service" "test_service_1" {
  name = "test-service-1-%s"
}

resource "firehydrant_service" "test_service_2" {
  name = "test-service-2-%s"
}

resource "firehydrant_incident_type" "test_incident_type" {
  name        = "test-incident-type-%s"
  description = "test-description-%s"
	template {
	  description = "test-template-description"
		customer_impact_summary = "test-summary"
		severity_slug = "SEV1"
		priority_slug = "TESTPRIORITY"
		private_incident = false

		tags = [ "foo", "bar" ]
		# A single runbook on purpose: runbook_ids is an ordered list in the
		# provider schema but the API returns set semantics, so two or more
		# entries produce order-drift refresh plans.
		runbook_ids = [ firehydrant_runbook.test_runbook_1.id ]
		team_ids = [ firehydrant_team.test_team_1.id, firehydrant_team.test_team_2.id ]

		impacts {
			impact_id    = firehydrant_service.test_service_1.id
			condition_id = "99762c0c-1ee0-44a0-a3a7-d1316dd902ca"
		}

		impacts {
			impact_id    = firehydrant_service.test_service_2.id
			condition_id = "99762c0c-1ee0-44a0-a3a7-d1316dd902ca"
		}
	}
}`, rName, rName, rName, rName, rName, rName, rName)
}

// Regression tests: incident types created before templates gained their current
// fields come back with the template object, or fields within it, missing. The
// provider used to dereference those pointers unconditionally and panicked, which
// surfaced to practitioners as "Plugin did not respond".

func offlineIncidentTypeMockServer(payload string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	}))
}

func offlineIncidentTypeClient(ts *httptest.Server) *firehydrant.APIClient {
	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)
	return client
}

func TestOfflineIncidentTypeReadHandlesMissingTemplateFields(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "template omitted entirely",
			payload: `{"id":"it-1","name":"Legacy","description":"An old incident type"}`,
		},
		{
			name:    "template present but empty",
			payload: `{"id":"it-1","name":"Legacy","description":"An old incident type","template":{}}`,
		},
		{
			name:    "template partially populated",
			payload: `{"id":"it-1","name":"Legacy","description":"An old incident type","template":{"description":"only this one"}}`,
		},
		{
			name: "template fields explicitly null",
			payload: `{"id":"it-1","name":"Legacy","description":"An old incident type","template":{` +
				`"description":null,"customer_impact_summary":null,"severity":null,"priority":null,` +
				`"private_incident":null,"tag_list":null,"runbook_ids":null,"team_ids":null,"impacts":null}}`,
		},
		{
			name:    "top-level name and description omitted",
			payload: `{"id":"it-1","template":{}}`,
		},
		{
			name:    "template is null",
			payload: `{"id":"it-1","name":"Legacy","description":"An old incident type","template":null}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := offlineIncidentTypeMockServer(tc.payload)
			defer ts.Close()

			r := schema.TestResourceDataRaw(t, resourceIncidentType().Schema, map[string]interface{}{})
			r.SetId("it-1")

			// Panicked before the fix.
			diags := readResourceIncidentType(context.Background(), r, offlineIncidentTypeClient(ts))
			if diags.HasError() {
				t.Fatalf("unexpected error reading incident type: %v", diags)
			}

			if r.Id() != "it-1" {
				t.Errorf("expected id it-1, got %q", r.Id())
			}
			if got := r.Get("template.#").(int); got != 1 {
				t.Errorf("expected exactly 1 template block, got %d", got)
			}
			for _, listAttr := range []string{"template.0.tags", "template.0.runbook_ids", "template.0.team_ids", "template.0.impacts"} {
				if r.Get(listAttr) == nil {
					t.Errorf("expected %s to be an empty list, got nil", listAttr)
				}
			}
		})
	}
}

func TestOfflineIncidentTypeReadWarnsOnMissingTemplate(t *testing.T) {
	ts := offlineIncidentTypeMockServer(`{"id":"it-1","name":"Legacy","description":"An old incident type"}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, resourceIncidentType().Schema, map[string]interface{}{})
	r.SetId("it-1")

	diags := readResourceIncidentType(context.Background(), r, offlineIncidentTypeClient(ts))
	if diags.HasError() {
		t.Fatalf("unexpected error reading incident type: %v", diags)
	}
	if len(diags) != 1 || diags[0].Severity != diag.Warning {
		t.Fatalf("expected exactly one warning about the missing template, got %v", diags)
	}
}

func TestOfflineIncidentTypeReadSkipsIncompleteImpacts(t *testing.T) {
	ts := offlineIncidentTypeMockServer(`{"id":"it-1","name":"Legacy","description":"d","template":{"impacts":[` +
		`{"id":null,"condition_id":null},` +
		`{"id":"impact-1","condition_id":"condition-1"},` +
		`{"id":"impact-2","condition_id":null}]}}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, resourceIncidentType().Schema, map[string]interface{}{})
	r.SetId("it-1")

	diags := readResourceIncidentType(context.Background(), r, offlineIncidentTypeClient(ts))
	if diags.HasError() {
		t.Fatalf("unexpected error reading incident type: %v", diags)
	}

	if got := r.Get("template.0.impacts.#").(int); got != 1 {
		t.Fatalf("expected 1 complete impact to survive, got %d", got)
	}
	if got := r.Get("template.0.impacts.0.impact_id").(string); got != "impact-1" {
		t.Errorf("expected impact_id impact-1, got %q", got)
	}
	if got := r.Get("template.0.impacts.0.condition_id").(string); got != "condition-1" {
		t.Errorf("expected condition_id condition-1, got %q", got)
	}
	if len(diags) != 1 || diags[0].Severity != diag.Warning {
		t.Fatalf("expected exactly one warning about incomplete impacts, got %v", diags)
	}
}

// Baseline: a fully-populated payload must still map every field correctly, so the
// nil guards cannot silently swallow real values.
func TestOfflineIncidentTypeReadFullyPopulated(t *testing.T) {
	ts := offlineIncidentTypeMockServer(`{"id":"it-1","name":"Outage","description":"top desc","template":{` +
		`"description":"tmpl desc","customer_impact_summary":"cis","severity":"SEV1","priority":"P1-CRITICAL",` +
		`"private_incident":true,"tag_list":["tag-a","tag-b"],"runbook_ids":["rb-1"],"team_ids":["team-1","team-2"],` +
		`"impacts":[{"id":"impact-1","condition_id":"condition-1"}]}}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, resourceIncidentType().Schema, map[string]interface{}{})
	r.SetId("it-1")

	diags := readResourceIncidentType(context.Background(), r, offlineIncidentTypeClient(ts))
	if diags.HasError() {
		t.Fatalf("unexpected error reading incident type: %v", diags)
	}
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for a complete payload, got %v", diags)
	}

	for attr, want := range map[string]interface{}{
		"name":                               "Outage",
		"description":                        "top desc",
		"template.0.description":             "tmpl desc",
		"template.0.customer_impact_summary": "cis",
		"template.0.severity_slug":           "SEV1",
		"template.0.priority_slug":           "P1-CRITICAL",
		"template.0.private_incident":        true,
		"template.0.tags.#":                  2,
		"template.0.tags.0":                  "tag-a",
		"template.0.runbook_ids.0":           "rb-1",
		"template.0.team_ids.#":              2,
		"template.0.team_ids.1":              "team-2",
		"template.0.impacts.0.impact_id":     "impact-1",
	} {
		if got := r.Get(attr); got != want {
			t.Errorf("%s: expected %v (%T), got %v (%T)", attr, want, want, got, got)
		}
	}
}

func TestOfflineIncidentTypeDataSourceHandlesMissingTemplate(t *testing.T) {
	ts := offlineIncidentTypeMockServer(`{"id":"it-1","name":"Legacy","description":"An old incident type"}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, dataSourceIncidentType().Schema, map[string]interface{}{
		"id": "it-1",
	})

	// Panicked before the fix.
	diags := readDataIncidentType(context.Background(), r, offlineIncidentTypeClient(ts))
	if diags.HasError() {
		t.Fatalf("unexpected error reading incident type data source: %v", diags)
	}
	if r.Id() != "it-1" {
		t.Errorf("expected id it-1, got %q", r.Id())
	}
	if got := r.Get("template.#").(int); got != 1 {
		t.Errorf("expected exactly 1 template block, got %d", got)
	}
}
