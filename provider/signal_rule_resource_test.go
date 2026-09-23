package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Signal rule tests run serially (no t.Parallel): every rule mutation
// triggers a Temporal config-sync workflow keyed by the organization ID, so
// rule tests contend org-wide regardless of which team they use. Temporal
// admits ~1 start of a given workflow ID per second, and concurrent rule
// mutations exhaust the API's retry budget and return 500s.

func TestAccFireHydrantSignalRule_basic(t *testing.T) {
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFireHydrantSignalRuleDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFireHydrantSignalRuleConfigBasic(rName, "MEDIUM", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "MEDIUM"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_name"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_is_pageable"),
				),
			},
			{
				Config: testAccFireHydrantSignalRuleConfigBasic(rName, "LOW", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "LOW"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_name"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_is_pageable"),
				),
			},
			{
				Config: testAccFireHydrantSignalRuleConfigBasic(rName, "HIGH", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "HIGH"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_name"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_is_pageable"),
				),
			},
		},
	})
}

func TestAccFireHydrantSignalRule_invalidPriority(t *testing.T) {
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFireHydrantSignalRuleDestroy(),
		Steps: []resource.TestStep{
			{
				Config:      testAccFireHydrantSignalRuleConfigBasic(rName, "INVALID", sharedTeamID),
				ExpectError: regexp.MustCompile(`expected notification_priority_override to be one of \[LOW MEDIUM HIGH\], got INVALID`),
			},
		},
	})
}

func TestAccFireHydrantSignalRule_createIncidentConditionWhen(t *testing.T) {
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFireHydrantSignalRuleDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFireHydrantSignalRuleConfigWithIncidentCondition(rName, "WHEN_ALWAYS", "PT30M", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_ALWAYS"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "deduplication_expiry", "PT30M"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "MEDIUM"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_name"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_is_pageable"),
				),
			},
		},
	})
}

func testAccFireHydrantSignalRuleConfigBasic(rName, priority, sharedTeamID string) string {
	existingUser := os.Getenv("EXISTING_USER_EMAIL")
	if existingUser == "" {
		existingUser = "local@firehydrant.io"
	}

	return fmt.Sprintf(`
	data "firehydrant_user" "test_user" {
			email = "%s"
	}

	resource "firehydrant_signal_rule" "test" {
		team_id = "%s"
		name = "test-signal-rule-%s"
		expression = "signal.summary == 'test-signal-summary-%s'"
		target_type = "User"
		target_id = data.firehydrant_user.test_user.id
		notification_priority_override = "%s"
		create_incident_condition_when = "WHEN_UNSPECIFIED"
	}
	`, existingUser, sharedTeamID, rName, rName, priority)
}

func TestAccFireHydrantSignalRule_IncidentTypeIDMissing(t *testing.T) {
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFireHydrantSignalRuleDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFireHydrantSignalRuleConfigIncidentTypeIDMissing(rName, sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "LOW"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
				),
			},
		},
	})
}

func TestAccFireHydrantSignalRule_NotificationPriorityAddRemove(t *testing.T) {
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFireHydrantSignalRuleDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFireHydrantSignalRuleConfigWithPriority(rName, "HIGH", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "HIGH"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
				),
			},
			{
				Config: testAccFireHydrantSignalRuleConfigWithPriority(rName, "MEDIUM", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "MEDIUM"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
				),
			},
			{
				Config: testAccFireHydrantSignalRuleConfigWithPriority(rName, "HIGH", sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "notification_priority_override", "HIGH"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
				),
			},
		},
	})
}

func TestAccFireHydrantSignalRule_withoutNotificationPriorityOverride(t *testing.T) {
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckFireHydrantSignalRuleDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccFireHydrantSignalRuleConfigWithoutPriority(rName, sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					// Verify notification_priority_override is not set (empty string or not present)
					resource.TestCheckNoResourceAttr("firehydrant_signal_rule.test", "notification_priority_override"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "create_incident_condition_when", "WHEN_UNSPECIFIED"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_name"),
					resource.TestCheckResourceAttrSet("firehydrant_signal_rule.test", "target_is_pageable"),
				),
			},
			{
				// Update the name to verify updates work without notification_priority_override
				Config: testAccFireHydrantSignalRuleConfigWithoutPriorityUpdated(rName, sharedTeamID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFireHydrantSignalRuleExists("firehydrant_signal_rule.test"),
					resource.TestCheckNoResourceAttr("firehydrant_signal_rule.test", "notification_priority_override"),
					resource.TestCheckResourceAttr("firehydrant_signal_rule.test", "name", fmt.Sprintf("test-signal-rule-updated-%s", rName)),
				),
			},
		},
	})
}

func testAccFireHydrantSignalRuleConfigWithPriority(rName, priority, sharedTeamID string) string {
	existingUser := os.Getenv("EXISTING_USER_EMAIL")
	if existingUser == "" {
		existingUser = "local@firehydrant.io"
	}

	return fmt.Sprintf(`
	data "firehydrant_user" "test_user" {
		email = "%s"
	}

	resource "firehydrant_signal_rule" "test" {
		team_id = "%s"
		name = "test-signal-rule-%s"
		expression = "signal.summary == 'test-signal-summary-%s'"
		target_type = "User"
		target_id = data.firehydrant_user.test_user.id
		notification_priority_override = "%s"
		create_incident_condition_when = "WHEN_UNSPECIFIED"
	}
	`, existingUser, sharedTeamID, rName, rName, priority)
}

func testAccFireHydrantSignalRuleConfigWithoutPriority(rName, sharedTeamID string) string {
	existingUser := os.Getenv("EXISTING_USER_EMAIL")
	if existingUser == "" {
		existingUser = "local@firehydrant.io"
	}

	return fmt.Sprintf(`
	data "firehydrant_user" "test_user" {
		email = "%s"
	}

	resource "firehydrant_signal_rule" "test" {
		team_id = "%s"
		name = "test-signal-rule-%s"
		expression = "signal.summary == 'test-signal-summary-%s'"
		target_type = "User"
		target_id = data.firehydrant_user.test_user.id
		create_incident_condition_when = "WHEN_UNSPECIFIED"
	}
	`, existingUser, sharedTeamID, rName, rName)
}

func testAccFireHydrantSignalRuleConfigWithoutPriorityUpdated(rName, sharedTeamID string) string {
	existingUser := os.Getenv("EXISTING_USER_EMAIL")
	if existingUser == "" {
		existingUser = "local@firehydrant.io"
	}

	return fmt.Sprintf(`
	data "firehydrant_user" "test_user" {
		email = "%s"
	}

	resource "firehydrant_signal_rule" "test" {
		team_id = "%s"
		name = "test-signal-rule-updated-%s"
		expression = "signal.summary == 'test-signal-summary-%s'"
		target_type = "User"
		target_id = data.firehydrant_user.test_user.id
		create_incident_condition_when = "WHEN_UNSPECIFIED"
	}
	`, existingUser, sharedTeamID, rName, rName)
}

func testAccFireHydrantSignalRuleConfigIncidentTypeIDMissing(rName, sharedTeamID string) string {
	existingUser := os.Getenv("EXISTING_USER_EMAIL")
	if existingUser == "" {
		existingUser = "local@firehydrant.io"
	}

	return fmt.Sprintf(`
	data "firehydrant_user" "test_user" {
		email = "%s"
	}

	resource "firehydrant_signal_rule" "test" {
		team_id = "%s"
		name = "test-signal-rule-%s"
		expression = "signal.summary == 'test-signal-summary-%s'"
		target_type = "User"
		target_id = data.firehydrant_user.test_user.id
		notification_priority_override = "LOW"
		create_incident_condition_when = "WHEN_UNSPECIFIED"
	}
	`, existingUser, sharedTeamID, rName, rName)
}

func testAccFireHydrantSignalRuleConfigWithIncidentCondition(rName, condition, expiry, sharedTeamID string) string {
	existingUser := os.Getenv("EXISTING_USER_EMAIL")
	if existingUser == "" {
		existingUser = "local@firehydrant.io"
	}

	return fmt.Sprintf(`
	data "firehydrant_user" "test_user" {
		email = "%s"
	}

	resource "firehydrant_signal_rule" "test" {
		team_id = "%s"
		name = "test-signal-rule-%s"
		expression = "signal.summary == 'test-signal-summary-%s'"
		target_type = "User"
		target_id = data.firehydrant_user.test_user.id
		create_incident_condition_when = "%s"
		deduplication_expiry = "%s"
		notification_priority_override = "MEDIUM"
	}
	`, existingUser, sharedTeamID, rName, rName, condition, expiry)
}

func testAccCheckFireHydrantSignalRuleExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no resource ID is set")
		}

		return nil
	}
}

func testAccCheckFireHydrantSignalRuleDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "firehydrant_signal_rule" {
				continue
			}

			// Check if the signal rule still exists
			_, err := client.Sdk.Signals.GetTeamSignalRule(context.TODO(), rs.Primary.Attributes["team_id"], rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("Signal rule %s still exists", rs.Primary.ID)
			}
			errStr := err.Error()
			if !strings.Contains(errStr, "404") && !strings.Contains(errStr, "record not found") {
				return fmt.Errorf("Error checking signal rule %s: %v", rs.Primary.ID, err)
			}
		}

		return nil
	}
}

// Unit tests: import ID parsing and the read-path nil guards. These run
// offline (no FIREHYDRANT_API_KEY) since signal rules are scoped to a team,
// so a plain resource ID is not enough to import or refresh one.

func TestResourceFireHydrantSignalRuleParseId(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantTeamID string
		wantID     string
		wantErr    bool
	}{
		{name: "valid", id: "team-1:rule-1", wantTeamID: "team-1", wantID: "rule-1"},
		{name: "id contains a colon", id: "team-1:rule:1", wantTeamID: "team-1", wantID: "rule:1"},
		{name: "missing separator", id: "rule-1", wantErr: true},
		{name: "empty team", id: ":rule-1", wantErr: true},
		{name: "empty rule id", id: "team-1:", wantErr: true},
		{name: "empty string", id: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			teamID, id, err := resourceFireHydrantSignalRuleParseId(tc.id)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error parsing %q, got teamID=%q id=%q", tc.id, teamID, id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error parsing %q: %v", tc.id, err)
			}
			if teamID != tc.wantTeamID || id != tc.wantID {
				t.Fatalf("parsing %q: expected teamID=%q id=%q, got teamID=%q id=%q", tc.id, tc.wantTeamID, tc.wantID, teamID, id)
			}
		})
	}
}

func offlineSignalRuleMockServer(payload string) *httptest.Server {
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

func offlineSignalRuleClient(ts *httptest.Server) *firehydrant.APIClient {
	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)
	return client
}

func TestOfflineSignalRuleImportSetsTeamID(t *testing.T) {
	ts := offlineSignalRuleMockServer(`{"id":"rule-1","name":"Route to team","expression":"true","target":{"type":"Team","id":"team-1"}}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, resourceSignalRule().Schema, map[string]interface{}{})
	r.SetId("team-1:rule-1")

	results, err := importResourceFireHydrantSignalRule(context.Background(), r, offlineSignalRuleClient(ts))
	if err != nil {
		t.Fatalf("unexpected error importing signal rule: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	imported := results[0]
	if imported.Id() != "rule-1" {
		t.Errorf("expected id rule-1, got %q", imported.Id())
	}
	if got := imported.Get("team_id").(string); got != "team-1" {
		t.Errorf("expected team_id team-1, got %q", got)
	}
}

// Regression test: reading a signal rule whose API response omits the
// target object, or fields within it, used to panic on
// signalRule.GetTarget().GetType() before the nil check below it ever ran.
func TestOfflineSignalRuleReadHandlesMissingTarget(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "target omitted", payload: `{"id":"rule-1","name":"Legacy rule","expression":"true"}`},
		{name: "target is null", payload: `{"id":"rule-1","name":"Legacy rule","expression":"true","target":null}`},
		{name: "target present but empty", payload: `{"id":"rule-1","name":"Legacy rule","expression":"true","target":{}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := offlineSignalRuleMockServer(tc.payload)
			defer ts.Close()

			r := schema.TestResourceDataRaw(t, resourceSignalRule().Schema, map[string]interface{}{
				"team_id": "team-1",
			})
			r.SetId("rule-1")

			// Panicked before the fix.
			diags := readResourceFireHydrantSignalRule(context.Background(), r, offlineSignalRuleClient(ts))
			if diags.HasError() {
				t.Fatalf("unexpected error reading signal rule: %v", diags)
			}

			if got := r.Get("name").(string); got != "Legacy rule" {
				t.Errorf("expected name %q, got %q", "Legacy rule", got)
			}
			if got := r.Get("target_type").(string); got != "" {
				t.Errorf("expected empty target_type, got %q", got)
			}
			if got := r.Get("target_id").(string); got != "" {
				t.Errorf("expected empty target_id, got %q", got)
			}
		})
	}
}
