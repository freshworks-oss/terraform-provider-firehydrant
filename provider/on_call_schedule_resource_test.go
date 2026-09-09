package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
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

// Regression test: the API accepts start_time for every strategy type (it
// seeds the initial rotation's first shift), but the provider used to discard
// it for non-custom strategies, silently starting weekly rotations at the
// beginning of the week. The API never echoes start_time back, so the drop
// was invisible in state diffs.
func TestOfflineOnCallScheduleCreate_sendsStartTimeForWeeklyStrategy(t *testing.T) {
	var createBody map[string]interface{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.Method == "POST" {
			if err := json.NewDecoder(req.Body).Decode(&createBody); err != nil {
				t.Errorf("failed to decode create request body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		}
		w.Write([]byte(`{
  "id": "schedule-id",
  "name": "test-schedule",
  "description": "test-description",
  "time_zone": "America/New_York",
  "members": [],
  "strategy": {"type": "weekly", "handoff_time": "10:00:00", "handoff_day": "thursday"},
  "restrictions": [],
  "rotations": []
}`))
	}))
	defer ts.Close()

	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)

	startTime := "2026-06-15T09:00:00Z"
	r := schema.TestResourceDataRaw(t, resourceOnCallSchedule().Schema, map[string]interface{}{
		"team_id":     "team-1",
		"name":        "test-schedule",
		"description": "test-description",
		"time_zone":   "America/New_York",
		"start_time":  startTime,
		"strategy": []interface{}{
			map[string]interface{}{
				"type":         "weekly",
				"handoff_time": "10:00:00",
				"handoff_day":  "thursday",
			},
		},
	})

	d := createResourceFireHydrantOnCallSchedule(context.Background(), r, client)
	if d.HasError() {
		t.Fatalf("error creating on-call schedule: %v", d)
	}

	if createBody == nil {
		t.Fatal("create request body was never captured")
	}

	got, ok := createBody["start_time"]
	if !ok {
		t.Fatalf("start_time missing from create request body; body keys: %v", createBody)
	}
	if got != startTime {
		t.Fatalf("expected start_time %q in create request body, got %q", startTime, got)
	}
}

func TestAccOnCallScheduleResource_basic(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckOnCallScheduleResourceDestroy(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallScheduleConfig_basic(rName, sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_on_call_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule", "name", fmt.Sprintf("test-on-call-schedule-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule", "description", fmt.Sprintf("test-description-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule", "time_zone", "America/New_York"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule", "strategy.0.type", "weekly"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule", "strategy.0.handoff_time", "10:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule", "strategy.0.handoff_day", "thursday"),
				),
			},
			{
				Config: testAccOnCallScheduleConfig_restrictions(rName, sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "name", fmt.Sprintf("test-on-call-schedule-restrictions-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "strategy.0.type", "weekly"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "strategy.0.handoff_time", "10:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "strategy.0.handoff_day", "thursday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.0.start_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.0.start_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.0.end_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.0.end_time", "14:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.1.start_day", "tuesday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.1.start_time", "12:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.1.end_day", "tuesday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_on_call_schedule_with_restrictions", "restrictions.1.end_time", "18:00:00"),
				),
			},
			{
				Config: testAccOnCallScheduleConfig_customStrategy(rName, sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_custom_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_custom_schedule", "name", fmt.Sprintf("test-custom-schedule-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_custom_schedule", "strategy.0.type", "custom"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_custom_schedule", "strategy.0.shift_duration", "PT8H"),
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_custom_schedule", "start_time"),
				),
			},
		},
	})
}

func testAccOnCallScheduleConfig_basic(rName, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_on_call_schedule" {
		team_id = "%s"
		name = "test-on-call-schedule-%s"
		description = "test-description-%s"
		time_zone = "America/New_York"

		strategy {
			type         = "weekly"
			handoff_time = "10:00:00"
			handoff_day  = "thursday"
		}
	}
	`, sharedTeamID, rName, rName)
}

func testAccOnCallScheduleConfig_customStrategy(rName, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_custom_schedule" {
		team_id = "%s"
		name = "test-custom-schedule-%s"
		description = "test-description-%s"
		time_zone = "America/New_York"
		start_time = "%s"

		strategy {
			type           = "custom"
			shift_duration = "PT8H"
		}
	}
	`, sharedTeamID, rName, rName, time.Now().Add(24*time.Hour).Format(time.RFC3339))
}

func testAccOnCallScheduleConfig_restrictions(rName, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_on_call_schedule_with_restrictions" {
		team_id = "%s"
		name = "test-on-call-schedule-restrictions-%s"
		description = "test-description-%s"
		time_zone = "America/New_York"
		slack_user_group_id = "test-group-1"

		strategy {
			type         = "weekly"
			handoff_time = "10:00:00"
			handoff_day  = "thursday"
		}

		restrictions {
			start_day = "monday"
			start_time = "09:00:00"
			end_day = "monday"
			end_time = "14:00:00"
		}

		restrictions {
			start_day = "tuesday"
			start_time = "12:00:00"
			end_day = "tuesday"
			end_time = "18:00:00"
		}
	}
	`, sharedTeamID, rName, rName)
}

func TestAccOnCallScheduleResource_rotationName(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	var initialRotationID string

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckOnCallScheduleResourceDestroy(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallScheduleConfig_rotationName(rName, sharedTeamID, "Primary", "first rotation under this schedule"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_rotation_name", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_rotation_name", "name", fmt.Sprintf("test-rotation-name-schedule-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_rotation_name", "rotation_name", "Primary"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_rotation_name", "rotation_description", "first rotation under this schedule"),
					testAccCheckPrimaryRotation("firehydrant_on_call_schedule.test_rotation_name", "Primary", "first rotation under this schedule", &initialRotationID),
				),
			},
			{
				// Mutate rotation_name/rotation_description in place. Asserts no replacement (same rotation ID) and the new values landed.
				Config: testAccOnCallScheduleConfig_rotationName(rName, sharedTeamID, "Secondary", "renamed in place"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_rotation_name", "rotation_name", "Secondary"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_rotation_name", "rotation_description", "renamed in place"),
					testAccCheckPrimaryRotation("firehydrant_on_call_schedule.test_rotation_name", "Secondary", "renamed in place", &initialRotationID),
				),
			},
		},
	})
}

func testAccOnCallScheduleConfig_rotationName(rName, sharedTeamID, rotationName, rotationDescription string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_rotation_name" {
		team_id              = "%s"
		name                 = "test-rotation-name-schedule-%s"
		description          = "test-description-%s"
		time_zone            = "America/New_York"
		rotation_name        = "%s"
		rotation_description = "%s"

		strategy {
			type         = "weekly"
			handoff_time = "10:00:00"
			handoff_day  = "thursday"
		}
	}
	`, sharedTeamID, rName, rName, rotationName, rotationDescription)
}

// testAccCheckPrimaryRotation verifies the schedule has exactly one rotation whose name and description match the expected values.
// On the first invocation (when *capturedID is empty) it records the rotation ID; on subsequent invocations it asserts the ID
// hasn't changed, which proves the rotation was updated in place rather than destroyed and recreated.
func testAccCheckPrimaryRotation(resourceName, expectedName, expectedDescription string, capturedID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		stateResource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		scheduleID := stateResource.Primary.ID
		teamID := stateResource.Primary.Attributes["team_id"]
		if scheduleID == "" || teamID == "" {
			return fmt.Errorf("missing schedule_id or team_id for %s", resourceName)
		}

		client, err := getAccTestClient()
		if err != nil {
			return err
		}
		schedule, err := client.Sdk.Signals.GetTeamOnCallSchedule(context.Background(), teamID, scheduleID, nil, nil)
		if err != nil {
			return fmt.Errorf("fetching schedule %s: %w", scheduleID, err)
		}
		// FireHydrant always creates exactly one rotation alongside a schedule on create.
		rotations := schedule.GetRotations()
		if len(rotations) != 1 {
			return fmt.Errorf("expected exactly 1 rotation for schedule %s, got %d", scheduleID, len(rotations))
		}

		gotID := ""
		if rotations[0].ID != nil {
			gotID = *rotations[0].ID
		}
		gotName := ""
		if rotations[0].Name != nil {
			gotName = *rotations[0].Name
		}
		gotDescription := ""
		if rotations[0].Description != nil {
			gotDescription = *rotations[0].Description
		}

		if gotName != expectedName {
			return fmt.Errorf("rotation name mismatch for schedule %s: want %q, got %q", scheduleID, expectedName, gotName)
		}
		if gotDescription != expectedDescription {
			return fmt.Errorf("rotation description mismatch for schedule %s: want %q, got %q", scheduleID, expectedDescription, gotDescription)
		}

		if capturedID != nil {
			if *capturedID == "" {
				*capturedID = gotID
			} else if *capturedID != gotID {
				return fmt.Errorf("rotation ID changed for schedule %s: want %q (in-place update), got %q (replacement)", scheduleID, *capturedID, gotID)
			}
		}
		return nil
	}
}

func testAccCheckOnCallScheduleResourceDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client, err := getAccTestClient()
		if err != nil {
			return err
		}

		for _, stateResource := range s.RootModule().Resources {
			if stateResource.Type != "firehydrant_on_call_schedule" {
				continue
			}

			if stateResource.Primary.ID == "" {
				return fmt.Errorf("No instance ID is set")
			}

			// Check if the on-call schedule still exists
			_, err := client.Sdk.Signals.GetTeamOnCallSchedule(context.TODO(), stateResource.Primary.Attributes["team_id"], stateResource.Primary.ID, nil, nil)
			if err == nil {
				return fmt.Errorf("On-call schedule %s still exists", stateResource.Primary.ID)
			}
			errStr := err.Error()
			if !strings.Contains(errStr, "404") && !strings.Contains(errStr, "record not found") {
				return fmt.Errorf("Error checking on-call schedule %s: %v", stateResource.Primary.ID, err)
			}
		}

		return nil
	}
}

func offlineOnCallScheduleMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle different endpoints
		if req.Method == "GET" {
			// GET request for reading
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
  "id": "schedule-id",
  "name": "A pleasant on-call schedule",
  "description": "Managed by Terraform. Contact @platform-eng for changes.",
  "members": [
    {
      "id": null,
      "name": null
    },
    {
      "id": "member-1",
      "name": "Frederick Graff"
    }
  ],
  "team": {
    "id": "team-1",
    "name": "Philadelphia"
  },
  "time_zone": "America/New_York"
}`))
		} else if req.Method == "POST" {
			// POST request for creating
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
  "id": "schedule-id",
  "name": "A pleasant on-call schedule",
  "description": "Managed by Terraform. Contact @platform-eng for changes.",
  "members": [
    {
      "id": "member-1",
      "name": "Frederick Graff"
    }
  ],
  "team": {
    "id": "team-1",
    "name": "Philadelphia"
  },
  "time_zone": "America/New_York"
}`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func TestOfflineOnCallScheduleReadSkipsNilMemberID(t *testing.T) {
	ts := offlineOnCallScheduleMockServer()
	defer ts.Close()

	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)

	r := schema.TestResourceDataRaw(t, resourceOnCallSchedule().Schema, map[string]interface{}{
		"team_id":     "team-1",
		"name":        "test-on-call-schedule",
		"description": "test-description",
		"time_zone":   "America/New_York",
		"member_ids":  []interface{}{"member-1"},
	})

	d := readResourceFireHydrantOnCallSchedule(context.Background(), r, client)
	if d.HasError() {
		t.Fatalf("error reading on-call schedule: %v", d)
	}

	memberIDs := r.Get("member_ids").([]interface{})
	if len(memberIDs) != 1 {
		t.Fatalf("expected 1 member ID, got %d", len(memberIDs))
	}

	memberID, ok := memberIDs[0].(string)
	if !ok {
		t.Fatalf("expected member ID to be a string, got %T: %v", memberIDs[0], memberIDs[0])
	}
	if memberID != "member-1" {
		t.Fatalf("expected member ID to be member-1, got %s", memberIDs[0].(string))
	}
}

func TestOfflineOnCallScheduleCreate(t *testing.T) {
	ts := offlineOnCallScheduleMockServer()
	defer ts.Close()

	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)

	r := schema.TestResourceDataRaw(t, resourceOnCallSchedule().Schema, map[string]interface{}{
		"team_id":     "team-1",
		"name":        "test-on-call-schedule",
		"description": "test-description",
		"time_zone":   "America/New_York",
		"member_ids":  []interface{}{"member-1"},
	})

	d := createResourceFireHydrantOnCallSchedule(context.Background(), r, client)
	if d.HasError() {
		t.Fatalf("error creating on-call schedule: %v", d)
	}

	memberIDs := r.Get("member_ids").([]interface{})
	if len(memberIDs) != 1 {
		t.Fatalf("expected 1 member ID, got %d", len(memberIDs))
	}

	memberID, ok := memberIDs[0].(string)
	if !ok {
		t.Fatalf("expected member ID to be a string, got %T: %v", memberIDs[0], memberIDs[0])
	}
	if memberID != "member-1" {
		t.Fatalf("expected member ID to be member-1, got %s", memberIDs[0].(string))
	}
}

// Deprecated, but ensure it still works until we officially remove support.
func TestOfflineOnCallScheduleCreateDeprecated(t *testing.T) {
	ts := offlineOnCallScheduleMockServer()
	defer ts.Close()

	client := &firehydrant.APIClient{}
	client.Sdk = fhsdk.New(
		fhsdk.WithServerURL(ts.URL),
		fhsdk.WithSecurity(components.Security{
			APIKey: "test-token-very-authorized",
		}),
	)

	r := schema.TestResourceDataRaw(t, resourceOnCallSchedule().Schema, map[string]interface{}{
		"team_id":     "team-1",
		"name":        "test-on-call-schedule",
		"description": "test-description",
		"time_zone":   "America/New_York",
		"members":     []interface{}{"member-1"},
	})

	d := createResourceFireHydrantOnCallSchedule(context.Background(), r, client)
	if d.HasError() {
		t.Fatalf("error creating on-call schedule: %v", d)
	}

	memberIDs := r.Get("member_ids").([]interface{})
	if len(memberIDs) != 1 {
		t.Fatalf("expected 1 member ID, got %d", len(memberIDs))
	}

	memberID, ok := memberIDs[0].(string)
	if !ok {
		t.Fatalf("expected member ID to be a string, got %T: %v", memberIDs[0], memberIDs[0])
	}
	if memberID != "member-1" {
		t.Fatalf("expected member ID to be member-1, got %s", memberIDs[0].(string))
	}
}

// FH-3378 regression: updating a custom-strategy schedule (e.g. reordering member_ids) re-sends
// the strategy block. handoff_day/handoff_time do not apply to custom rotations, and the API
// rejects an irrelevant handoff_day with "strategy[handoff_day] does not have a valid value"
// (HTTP 400), so the update payload must drop them.
func TestBuildUpdateStrategy_discardsHandoffForCustomStrategy(t *testing.T) {
	s := buildUpdateStrategy(map[string]interface{}{
		"type":           "custom",
		"shift_duration": "PT168H",
		// A stale/irrelevant handoff_day/handoff_time that must not reach the API for custom.
		"handoff_day":  "monday",
		"handoff_time": "09:00:00",
	})

	if s.HandoffDay != nil {
		t.Fatalf("expected HandoffDay to be nil for a custom strategy, got %q", string(*s.HandoffDay))
	}
	if s.HandoffTime != nil {
		t.Fatalf("expected HandoffTime to be nil for a custom strategy, got %q", *s.HandoffTime)
	}
	if s.ShiftDuration == nil || *s.ShiftDuration != "PT168H" {
		t.Fatalf("expected ShiftDuration \"PT168H\" for a custom strategy, got %v", s.ShiftDuration)
	}
}

// Guard for the FH-3378 fix: the discard logic must NOT strip handoff_day/handoff_time from a
// weekly strategy (and must drop the inapplicable shift_duration).
func TestBuildUpdateStrategy_keepsHandoffForWeeklyStrategy(t *testing.T) {
	s := buildUpdateStrategy(map[string]interface{}{
		"type":           "weekly",
		"shift_duration": "",
		"handoff_day":    "thursday",
		"handoff_time":   "10:00:00",
	})

	if s.HandoffDay == nil || string(*s.HandoffDay) != "thursday" {
		t.Fatalf("expected HandoffDay \"thursday\" for a weekly strategy, got %v", s.HandoffDay)
	}
	if s.HandoffTime == nil || *s.HandoffTime != "10:00:00" {
		t.Fatalf("expected HandoffTime \"10:00:00\" for a weekly strategy, got %v", s.HandoffTime)
	}
	if s.ShiftDuration != nil {
		t.Fatalf("expected ShiftDuration to be nil for a weekly strategy, got %q", *s.ShiftDuration)
	}
}

func TestAccOnCallScheduleResource_updateHandoffAndRestrictions(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckOnCallScheduleResourceDestroy(),
		),
		Steps: []resource.TestStep{
			{
				// Initial configuration
				Config: testAccOnCallScheduleConfig_withHandoff(rName, "monday", "09:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "name", fmt.Sprintf("test-schedule-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "time_zone", "America/New_York"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "slack_user_group_id", "test-group-1"),
					// Strategy settings
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.type", "weekly"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "monday"),
					// Initial configuration has no restrictions
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "0"),
				),
			},
			{
				// Update handoff day/time and add restrictions
				Config: testAccOnCallScheduleConfig_withHandoffAndRestrictions(rName, "wednesday", "13:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "name", fmt.Sprintf("test-schedule-%s", rName)),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "time_zone", "America/New_York"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "slack_user_group_id", "test-group-1"),
					// Changed handoff settings
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.type", "weekly"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "13:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "wednesday"),
					// Added restrictions
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "2"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.start_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.start_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.end_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.end_time", "17:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.1.start_day", "tuesday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.1.start_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.1.end_day", "tuesday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.1.end_time", "17:00:00"),
				),
			},
			{
				// Update just handoff time, keeping restrictions
				Config: testAccOnCallScheduleConfig_withHandoffAndRestrictions(rName, "wednesday", "15:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					// Changed handoff time only
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "15:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "wednesday"),
					// Restrictions still present
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "2"),
				),
			},
		},
	})
}

func testAccOnCallScheduleConfig_withHandoff(rName, handoffDay, handoffTime, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_schedule" {
		team_id = "%s"
		name = "test-schedule-%s"
		time_zone = "America/New_York"
		slack_user_group_id = "test-group-1"

		strategy {
			type         = "weekly"
			handoff_time = "%s"
			handoff_day  = "%s"
		}
	}
	`, sharedTeamID, rName, handoffTime, handoffDay)
}

func testAccOnCallScheduleConfig_withHandoffAndRestrictions(rName, handoffDay, handoffTime, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_schedule" {
		team_id = "%s"
		name = "test-schedule-%s"
		time_zone = "America/New_York"
		slack_user_group_id = "test-group-1"

		strategy {
			type         = "weekly"
			handoff_time = "%s"
			handoff_day  = "%s"
		}

		restrictions {
			start_day = "monday"
			start_time = "09:00:00"
			end_day = "monday"
			end_time = "17:00:00"
		}

		restrictions {
			start_day = "tuesday"
			start_time = "09:00:00"
			end_day = "tuesday"
			end_time = "17:00:00"
		}
	}
	`, sharedTeamID, rName, handoffTime, handoffDay)
}

func TestAccOnCallScheduleResource_scheduleModifications(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckOnCallScheduleResourceDestroy(),
		),
		Steps: []resource.TestStep{
			{
				// Initial configuration with restrictions
				Config: testAccOnCallScheduleConfig_withHandoffAndRestrictions(rName, "monday", "09:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.type", "weekly"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "2"),
				),
			},
			{
				// Change just handoff day, keeping time and restrictions
				Config: testAccOnCallScheduleConfig_withHandoffAndRestrictions(rName, "friday", "09:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "friday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "2"),
				),
			},
			// TODO: Uncomment this test step when the Go SDK omitempty limitation is fixed
			// {
			// 	// Remove all restrictions but keep handoff settings
			// 	Config: testAccOnCallScheduleConfig_withHandoff(rName, "friday", "09:00:00"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
			// 		resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "friday"),
			// 		resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "09:00:00"),
			// 		resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "0"),
			// 	),
			// },
			{
				// Add different restriction pattern
				Config: testAccOnCallScheduleConfig_withBusinessHours(rName, "friday", "09:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "friday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.#", "1"),
					// Check business hours restriction
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.start_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.start_time", "09:00:00"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.end_day", "friday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "restrictions.0.end_time", "17:00:00"),
				),
			},
		},
	})
}

func testAccOnCallScheduleConfig_withBusinessHours(rName, handoffDay, handoffTime, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_schedule" {
		team_id = "%s"
		name = "test-schedule-%s"
		time_zone = "America/New_York"
		slack_user_group_id = "test-group-1"

		strategy {
			type         = "weekly"
			handoff_time = "%s"
			handoff_day  = "%s"
		}

		restrictions {
			start_day = "monday"
			start_time = "09:00:00"
			end_day = "friday"
			end_time = "17:00:00"
		}
	}
	`, sharedTeamID, rName, handoffTime, handoffDay)
}

func TestAccOnCallScheduleResource_effectiveAt(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)
	futureTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339) // Tomorrow
	pastTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)  // Yesterday

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckOnCallScheduleResourceDestroy(),
		Steps: []resource.TestStep{
			{
				// Initial schedule setup
				Config: testAccOnCallScheduleConfig_withHandoff(rName, "monday", "09:00:00", sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "monday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "09:00:00"),
				),
			},
			{
				// Update with future effective_at
				Config: testAccOnCallScheduleConfig_withEffectiveAt(rName, "friday", "13:00:00", futureTime, sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "friday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "13:00:00"),
					// effective_at shouldn't be in state
					resource.TestCheckNoResourceAttr("firehydrant_on_call_schedule.test_schedule", "effective_at"),
				),
			},
			{
				// Update with past effective_at (should apply immediately)
				Config: testAccOnCallScheduleConfig_withEffectiveAt(rName, "wednesday", "15:00:00", pastTime, sharedTeamID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("firehydrant_on_call_schedule.test_schedule", "id"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_day", "wednesday"),
					resource.TestCheckResourceAttr("firehydrant_on_call_schedule.test_schedule", "strategy.0.handoff_time", "15:00:00"),
					// effective_at shouldn't be in state
					resource.TestCheckNoResourceAttr("firehydrant_on_call_schedule.test_schedule", "effective_at"),
				),
			},
			{
				// Test invalid timestamp format
				Config:      testAccOnCallScheduleConfig_withEffectiveAt(rName, "thursday", "10:00:00", "invalid-timestamp", sharedTeamID),
				ExpectError: regexp.MustCompile("effective_at must be a valid RFC3339 timestamp"),
			},
			{
				// Verify plan is empty when effective_at changes but nothing else does
				Config:   testAccOnCallScheduleConfig_withEffectiveAt(rName, "wednesday", "15:00:00", futureTime, sharedTeamID),
				PlanOnly: true,
			},
		},
	})
}

func testAccOnCallScheduleConfig_withEffectiveAt(rName, handoffDay, handoffTime, effectiveAt, sharedTeamID string) string {
	return fmt.Sprintf(`
	resource "firehydrant_on_call_schedule" "test_schedule" {
		team_id = "%s"
		name = "test-schedule-%s"
		time_zone = "America/New_York"
		slack_user_group_id = "test-group-1"

		strategy {
			type         = "weekly"
			handoff_time = "%s"
			handoff_day  = "%s"
		}

		effective_at = "%s"
	}
	`, sharedTeamID, rName, handoffTime, handoffDay, effectiveAt)
}

func TestAccOnCallScheduleResourceImport_basic(t *testing.T) {
	t.Parallel()
	sharedTeamID := getSharedTeamID(t)
	rName := acctest.RandStringFromCharSet(20, acctest.CharSetAlphaNum)

	resourceName := "firehydrant_on_call_schedule.test_on_call_schedule"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testFireHydrantIsSetup(t) },
		ProviderFactories: sharedProviderFactories(),
		CheckDestroy:      testAccCheckOnCallScheduleResourceDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallScheduleConfig_basic(rName, sharedTeamID),
			},
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("Not found: %s", resourceName)
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["team_id"], rs.Primary.Attributes["id"]), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
