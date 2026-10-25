package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func mustParseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad test timestamp %q: %v", s, err)
	}
	return v
}

func TestRollForwardEffectiveAt(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("loading time zone: %v", err)
	}

	cases := []struct {
		name         string
		effectiveAt  string
		now          string
		loc          *time.Location
		strategyType string
		members      int
		want         string
		wantMoved    bool
	}{
		{
			name:         "recent effective_at is sent unchanged",
			effectiveAt:  "2026-09-02T19:00:00Z",
			now:          "2026-09-23T18:00:00Z",
			loc:          newYork,
			strategyType: "weekly",
			members:      4,
			want:         "2026-09-02T19:00:00Z",
		},
		{
			// 4 members x 1 week = 28-day laps. Sep 2 is four laps after May 13,
			// so every shift from Sep 2 onward has the same member as before.
			name:         "weekly rotation moves forward by whole laps",
			effectiveAt:  "2026-05-13T19:00:00Z",
			now:          "2026-09-23T18:00:00Z",
			loc:          newYork,
			strategyType: "weekly",
			members:      4,
			want:         "2026-09-02T19:00:00Z",
			wantMoved:    true,
		},
		{
			// Daylight saving time ends on Nov 1 2026 in New York. The anchor is
			// 15:00 EDT on May 13. It must stay at 15:00 local time, which is 20:00Z
			// under EST. If the anchor changes by one hour against the 15:00 handoff,
			// the first shift can go to the wrong member.
			name:         "anchor keeps its wall-clock time across a DST change",
			effectiveAt:  "2026-05-13T19:00:00Z",
			now:          "2026-12-10T12:00:00Z",
			loc:          newYork,
			strategyType: "weekly",
			members:      4,
			want:         "2026-11-25T20:00:00Z",
			wantMoved:    true,
		},
		{
			name:         "daily rotation moves forward by whole laps",
			effectiveAt:  "2026-06-01T09:00:00Z",
			now:          "2026-09-23T12:00:00Z",
			loc:          time.UTC,
			strategyType: "daily",
			members:      3,
			want:         "2026-09-23T09:00:00Z",
			wantMoved:    true,
		},
		{
			// 6 members x 1 week = 42-day laps. The last boundary before now (Sep 16)
			// is more than one month old, so the function uses the next boundary
			// (Oct 28).
			name:         "lap longer than a month moves to the next boundary",
			effectiveAt:  "2026-01-07T15:00:00Z",
			now:          "2026-10-21T12:00:00Z",
			loc:          time.UTC,
			strategyType: "weekly",
			members:      6,
			want:         "2026-10-28T15:00:00Z",
			wantMoved:    true,
		},
		{
			name:         "custom strategy is sent unchanged",
			effectiveAt:  "2026-05-13T19:00:00Z",
			now:          "2026-09-23T18:00:00Z",
			loc:          newYork,
			strategyType: "custom",
			members:      4,
			want:         "2026-05-13T19:00:00Z",
		},
		{
			name:         "no members is sent unchanged",
			effectiveAt:  "2026-05-13T19:00:00Z",
			now:          "2026-09-23T18:00:00Z",
			loc:          newYork,
			strategyType: "weekly",
			members:      0,
			want:         "2026-05-13T19:00:00Z",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, moved := rollForwardEffectiveAt(
				mustParseRFC3339(t, tc.effectiveAt),
				mustParseRFC3339(t, tc.now),
				tc.loc, tc.strategyType, tc.members,
			)
			if moved != tc.wantMoved {
				t.Fatalf("moved = %v, want %v", moved, tc.wantMoved)
			}
			if gotStr := got.UTC().Format(time.RFC3339); gotStr != tc.want {
				t.Fatalf("effective_at = %s, want %s", gotStr, tc.want)
			}
			if moved && got.Before(effectiveAtCutoff(mustParseRFC3339(t, tc.now))) {
				t.Fatalf("rolled effective_at %s is still older than the API accepts", got.Format(time.RFC3339))
			}
		})
	}
}

// Regression test: after the configured effective_at became one month old,
// every update to the schedule failed with "effective_at can't be more than 1
// month in the past". This included the update that Terraform makes to revert
// a change made in the UI. The update now sends an effective_at that is a
// whole number of rotations later. The API accepts this value, and the on-call
// order does not change.
func TestOfflineOnCallScheduleUpdate_rollsStaleEffectiveAtForward(t *testing.T) {
	var updateBody map[string]interface{}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodPatch {
			if err := json.NewDecoder(req.Body).Decode(&updateBody); err != nil {
				t.Errorf("failed to decode update request body: %v", err)
			}
		}
		w.Write([]byte(`{
  "id": "schedule-id",
  "name": "test-schedule",
  "description": "test-description",
  "time_zone": "America/New_York",
  "members": [{"id": "user-1"}, {"id": "user-2"}, {"id": "user-3"}, {"id": "user-4"}],
  "strategy": {"type": "weekly", "handoff_time": "15:00:00", "handoff_day": "wednesday"},
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

	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("loading time zone: %v", err)
	}
	stale := time.Now().In(newYork).AddDate(0, 0, -130).Truncate(time.Hour).UTC().Format(time.RFC3339)
	members := []string{"user-1", "user-2", "user-3", "user-4"}

	res := resourceOnCallSchedule()
	rawConfig := map[string]cty.Value{}
	for name, ty := range res.CoreConfigSchema().ImpliedType().AttributeTypes() {
		rawConfig[name] = cty.NullVal(ty)
	}
	memberVals := make([]cty.Value, len(members))
	for i, id := range members {
		memberVals[i] = cty.StringVal(id)
	}
	rawConfig["effective_at"] = cty.StringVal(stale)
	rawConfig["member_ids"] = cty.ListVal(memberVals)

	attributes := map[string]string{
		"id":                      "schedule-id",
		"team_id":                 "team-1",
		"name":                    "test-schedule",
		"description":             "test-description",
		"time_zone":               "America/New_York",
		"strategy.#":              "1",
		"strategy.0.type":         "weekly",
		"strategy.0.handoff_day":  "wednesday",
		"strategy.0.handoff_time": "15:00:00",
		"restrictions.#":          "0",
		"member_ids.#":            strconv.Itoa(len(members)),
	}
	for i, id := range members {
		attributes["member_ids."+strconv.Itoa(i)] = id
	}

	d := res.Data(&terraform.InstanceState{
		ID:         "schedule-id",
		Attributes: attributes,
		RawConfig:  cty.ObjectVal(rawConfig),
	})

	diags := updateResourceFireHydrantOnCallSchedule(context.Background(), d, client)
	if diags.HasError() {
		t.Fatalf("error updating on-call schedule: %v", diags)
	}
	for _, dg := range diags {
		if dg.Severity == diag.Warning {
			t.Fatalf("unexpected warning for a 28-day lap: %s", dg.Summary)
		}
	}

	if updateBody == nil {
		t.Fatal("update request body was never captured")
	}
	sentStr, ok := updateBody["effective_at"].(string)
	if !ok {
		t.Fatalf("effective_at missing from update request body; body: %v", updateBody)
	}
	sent := mustParseRFC3339(t, sentStr)

	if sent.Before(effectiveAtCutoff(time.Now())) {
		t.Fatalf("sent effective_at %s is older than the API accepts", sentStr)
	}
	if sent.After(time.Now()) {
		t.Fatalf("sent effective_at %s is in the future; a 28-day lap always has a boundary in the last month", sentStr)
	}

	anchor := mustParseRFC3339(t, stale).In(newYork)
	lapDays := 7 * len(members)
	aligned := false
	for laps := 1; laps <= 10; laps++ {
		if anchor.AddDate(0, 0, laps*lapDays).Equal(sent) {
			aligned = true
			break
		}
	}
	if !aligned {
		t.Fatalf("sent effective_at %s is not a whole number of %d-day rotations after %s", sentStr, lapDays, stale)
	}
}
