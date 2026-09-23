package provider

import (
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The API rejects an on-call schedule update when effective_at is more than
// one month in the past ("effective_at can't be more than 1 month in the
// past"). The provider sends the configured effective_at on every update.
// After the configured value becomes one month old, every update to the
// schedule fails. This includes the update that Terraform makes to revert a
// change made in the UI.
//
// effective_at anchors the rotation. The first member's shift starts at
// effective_at. Each shift after that goes to the next member. A lap is one
// full rotation: the member count times the shift length. If the provider adds
// a whole number of laps to the anchor, each shift from the new anchor onward
// gets the same member as before. The on-call order does not change.

// effectiveAtSafetyMargin is the extra time that the provider keeps between a
// new effective_at and the API's one-month limit.
const effectiveAtSafetyMargin = 24 * time.Hour

// effectiveAtCutoff returns the oldest effective_at that the provider sends
// unchanged.
func effectiveAtCutoff(now time.Time) time.Time {
	return now.AddDate(0, -1, 0).Add(effectiveAtSafetyMargin)
}

// rollForwardEffectiveAt adds whole laps to effectiveAt when effectiveAt is
// older than the API accepts. It returns effectiveAt and false in these cases:
//   - effectiveAt is recent enough.
//   - The lap length is not known: a custom strategy, no time zone, or no members.
func rollForwardEffectiveAt(effectiveAt, now time.Time, loc *time.Location, strategyType string, memberCount int) (time.Time, bool) {
	cutoff := effectiveAtCutoff(now)
	if !effectiveAt.Before(cutoff) {
		return effectiveAt, false
	}

	var shiftDays int
	switch strategyType {
	case "daily":
		shiftDays = 1
	case "weekly":
		shiftDays = 7
	default:
		return effectiveAt, false
	}
	if memberCount < 1 || loc == nil {
		return effectiveAt, false
	}
	lapDays := shiftDays * memberCount

	// The provider adds laps as calendar days in the schedule's time zone. This
	// keeps the anchor's time of day the same across daylight saving changes,
	// as the handoffs do.
	anchor := effectiveAt.In(loc)
	at := func(laps int) time.Time { return anchor.AddDate(0, 0, laps*lapDays) }

	laps := int(now.Sub(effectiveAt).Hours()/24) / lapDays
	for !at(laps + 1).After(now) {
		laps++
	}
	for laps > 0 && at(laps).After(now) {
		laps--
	}

	rolled := at(laps)
	if rolled.Before(cutoff) {
		// One lap is longer than the window that the API accepts, so no lap
		// boundary is in the window. The next boundary keeps the order. The
		// update takes effect at that time.
		rolled = at(laps + 1)
	}

	return rolled.UTC(), true
}

// onCallScheduleLapInputs returns the time zone and the strategy type that
// rollForwardEffectiveAt uses. It returns false when the configuration does
// not give the lap length. Restrictions can change how the API makes shifts,
// so the provider sends the configured value for a schedule with restrictions.
func onCallScheduleLapInputs(d *schema.ResourceData) (*time.Location, string, bool) {
	if len(d.Get("restrictions").([]interface{})) > 0 {
		return nil, "", false
	}

	strategies := d.Get("strategy").([]interface{})
	if len(strategies) == 0 || strategies[0] == nil {
		return nil, "", false
	}
	strategyType, _ := strategies[0].(map[string]interface{})["type"].(string)

	loc, err := time.LoadLocation(d.Get("time_zone").(string))
	if err != nil {
		return nil, "", false
	}

	return loc, strategyType, true
}
