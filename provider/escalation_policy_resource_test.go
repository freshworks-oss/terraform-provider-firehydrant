package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// escalationPolicySettleDelay is a deliberate pause before the test
// framework runs `terraform destroy`. The backend occasionally returns
// 500s when delete arrives faster than its workers can finish processing
// the preceding create/update. Used as a Check on the final step so the
// sleep happens AFTER asserts and BEFORE destroy.
const escalationPolicySettleDelay = 3 * time.Second

func sleepBeforeDestroy(d time.Duration) resource.TestCheckFunc {
	return func(*terraform.State) error {
		time.Sleep(d)
		return nil
	}
}

func TestAccEscalationPolicyResource_basic(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	sharedScheduleID := getSharedOnCallScheduleID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckEscalationPolicyResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccEscalationPolicyConfig_basic(rName, sharedTeamID, sharedScheduleID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "id"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "name", fmt.Sprintf("test-escalation-policy-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "description", fmt.Sprintf("test-description-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step_strategy", "static"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step.0.timeout", "PT1M"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step.0.targets.0.type", "OnCallSchedule"),
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "step.0.targets.0.id"),
					sleepBeforeDestroy(escalationPolicySettleDelay),
				),
			},
		},
	})
}

func TestAccEscalationPolicyResource_dynamicWithPriorityPolicies(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	sharedScheduleID := getSharedOnCallScheduleID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckEscalationPolicyResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccEscalationPolicyConfig_dynamicPriority(rName, sharedTeamID, sharedScheduleID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "id"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "name", fmt.Sprintf("test-escalation-policy-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "description", fmt.Sprintf("test-description-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step_strategy", "dynamic_by_priority"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "repetitions", "1"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step.0.timeout", "PT1M"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step.0.targets.0.type", "OnCallSchedule"),
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "step.0.targets.0.id"),
					// Test notification priority policies
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.priority", "HIGH"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.repetitions", "2"),
				),
			},
			{
				Config: testAccEscalationPolicyConfig_dynamicPriorityUpdated(rName, sharedTeamID, sharedScheduleID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "id"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "name", fmt.Sprintf("test-escalation-policy-updated-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "description", fmt.Sprintf("test-description-updated-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step_strategy", "dynamic_by_priority"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "repetitions", "1"),
					// Test multiple priority levels
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.priority", "HIGH"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.repetitions", "3"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.1.priority", "MEDIUM"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.1.repetitions", "1"),
					sleepBeforeDestroy(escalationPolicySettleDelay),
				),
			},
		},
	})
}

func TestAccEscalationPolicyResource_dynamicWithHandoffSteps(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	sharedScheduleID := getSharedOnCallScheduleID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckEscalationPolicyResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccEscalationPolicyConfig_dynamicWithHandoffSteps(rName, sharedTeamID, sharedScheduleID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "id"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "name", fmt.Sprintf("test-escalation-policy-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "step_strategy", "dynamic_by_priority"),
					// Test notification priority policies with handoff steps
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.priority", "HIGH"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.repetitions", "2"),
					resource.TestCheckResourceAttr("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.handoff_step.0.target_type", "Team"),
					resource.TestCheckResourceAttrSet("firehydrant_escalation_policy.test_escalation_policy", "notification_priority_policies.0.handoff_step.0.target_id"),
					sleepBeforeDestroy(escalationPolicySettleDelay),
				),
			},
		},
	})
}

func testAccEscalationPolicyConfig_basic(rName, sharedTeamID, sharedScheduleID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_escalation_policy" "test_escalation_policy" {
		team_id = "%s"
		name = "test-escalation-policy-%s"
		description = "test-description-%s"
		repetitions = 1
		step_strategy = "static"
		
		step {
			timeout     = "PT1M"

			targets {
				type = "OnCallSchedule"
				id   = "%s"
			}
		}

		handoff_step {
			target_type = "Team"
			target_id   = "%s"
		}
	}
	`, sharedTeamID, rName, rName, sharedScheduleID, sharedTeamID)
}

func testAccEscalationPolicyConfig_dynamicPriority(rName, sharedTeamID, sharedScheduleID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_escalation_policy" "test_escalation_policy" {
		team_id = "%s"
		name = "test-escalation-policy-%s"
		description = "test-description-%s"
		repetitions = 1
		step_strategy = "dynamic_by_priority"

		step {
			timeout     = "PT1M"
			priorities  = ["HIGH"]

			targets {
				type = "OnCallSchedule"
				id   = "%s"
			}
		}

		step {
			timeout     = "PT2M"
			priorities  = ["LOW"]

			targets {
				type = "OnCallSchedule"
				id   = "%s"
			}
		}

		notification_priority_policies {
			priority = "HIGH"
			repetitions = 2
		}

		notification_priority_policies {
			priority = "LOW"
			repetitions = 1
		}
	}
	`, sharedTeamID, rName, rName, sharedScheduleID, sharedScheduleID)
}

func testAccEscalationPolicyConfig_dynamicPriorityUpdated(rName, sharedTeamID, sharedScheduleID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_escalation_policy" "test_escalation_policy" {
		team_id = "%s"
		name = "test-escalation-policy-updated-%s"
		description = "test-description-updated-%s"
		repetitions = 1
		step_strategy = "dynamic_by_priority"

		step {
			timeout     = "PT1M"
			priorities  = ["HIGH", "MEDIUM"]

			targets {
				type = "OnCallSchedule"
				id   = "%s"
			}
		}

		notification_priority_policies {
			priority = "HIGH"
			repetitions = 3
		}

		notification_priority_policies {
			priority = "MEDIUM"
			repetitions = 1
		}
	}
	`, sharedTeamID, rName, rName, sharedScheduleID)
}

func testAccEscalationPolicyConfig_dynamicWithHandoffSteps(rName, sharedTeamID, sharedScheduleID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_escalation_policy" "test_escalation_policy" {
		team_id = "%s"
		name = "test-escalation-policy-%s"
		description = "test-description-%s"
		repetitions = 1
		step_strategy = "dynamic_by_priority"

		step {
			timeout     = "PT1M"
			priorities  = ["HIGH"]

			targets {
				type = "OnCallSchedule"
				id   = "%s"
			}
		}

		notification_priority_policies {
			priority = "HIGH"
			repetitions = 2
			
			handoff_step {
				target_type = "Team"
				target_id   = "%s"
			}
		}
	}
	`, sharedTeamID, rName, rName, sharedScheduleID, sharedTeamID)
}

func testAccCheckEscalationPolicyResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_escalation_policy" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			// Check if the escalation policy still exists
			_, err := client.Sdk.Signals.GetTeamEscalationPolicy(context.TODO(), stateResource.Primary.Attributes["team_id"], stateResource.Primary.ID)
			if err == nil {
				return fmt.Errorf("Escalation policy %s still exists", stateResource.Primary.ID)
			}
			errStr := err.Error()
			if !strings.Contains(errStr, "404") && !strings.Contains(errStr, "record not found") {
				return fmt.Errorf("Error checking escalation policy %s: %v", stateResource.Primary.ID, err)
			}
		}

		return nil
	}
}

// Unit tests: import ID parsing and the read-path nil guards. These run
// offline (no FIREHYDRANT_API_KEY) since escalation policies are scoped to a
// team, so a plain resource ID is not enough to import or refresh one.

func TestResourceFireHydrantEscalationPolicyParseId(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantTeamID string
		wantID     string
		wantErr    bool
	}{
		{name: "valid", id: "team-1:policy-1", wantTeamID: "team-1", wantID: "policy-1"},
		{name: "id contains a colon", id: "team-1:policy:1", wantTeamID: "team-1", wantID: "policy:1"},
		{name: "missing separator", id: "policy-1", wantErr: true},
		{name: "empty team", id: ":policy-1", wantErr: true},
		{name: "empty policy id", id: "team-1:", wantErr: true},
		{name: "empty string", id: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			teamID, id, err := resourceFireHydrantEscalationPolicyParseId(tc.id)
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

func offlineEscalationPolicyMockServer(payload string) *httptest.Server {
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

func offlineEscalationPolicyClient(ts *httptest.Server) *firehydrant.APIClient {
	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)
	return client
}

func TestOfflineEscalationPolicyImportSetsTeamID(t *testing.T) {
	ts := offlineEscalationPolicyMockServer(`{"id":"policy-1","name":"Primary","default":true,"repetitions":1,"steps":[]}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, resourceEscalationPolicy().Schema, map[string]interface{}{})
	r.SetId("team-1:policy-1")

	results, err := importResourceFireHydrantEscalationPolicy(context.Background(), r, offlineEscalationPolicyClient(ts))
	if err != nil {
		t.Fatalf("unexpected error importing escalation policy: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	imported := results[0]
	if imported.Id() != "policy-1" {
		t.Errorf("expected id policy-1, got %q", imported.Id())
	}
	if got := imported.Get("team_id").(string); got != "team-1" {
		t.Errorf("expected team_id team-1, got %q", got)
	}
}

// Regression test: reading an escalation policy whose API response omits
// optional pointer fields (description, step_strategy, step targets) used to
// panic. Every field the provider previously dereferenced unconditionally is
// exercised here as absent.
func TestOfflineEscalationPolicyReadHandlesMissingFields(t *testing.T) {
	ts := offlineEscalationPolicyMockServer(`{"id":"policy-1","name":"Primary","steps":[{"timeout":"PT5M","targets":[{}]}]}`)
	defer ts.Close()

	r := schema.TestResourceDataRaw(t, resourceEscalationPolicy().Schema, map[string]interface{}{
		"team_id": "team-1",
	})
	r.SetId("policy-1")

	// Panicked before the fix: default, repetitions, and step target type/id
	// were all dereferenced without a nil check.
	diags := readResourceFireHydrantEscalationPolicy(context.Background(), r, offlineEscalationPolicyClient(ts))
	if diags.HasError() {
		t.Fatalf("unexpected error reading escalation policy: %v", diags)
	}

	if got := r.Get("name").(string); got != "Primary" {
		t.Errorf("expected name Primary, got %q", got)
	}
	if got := r.Get("description").(string); got != "" {
		t.Errorf("expected empty description, got %q", got)
	}
	if got := r.Get("step.0.targets.0.type").(string); got != "" {
		t.Errorf("expected empty target type, got %q", got)
	}
}
