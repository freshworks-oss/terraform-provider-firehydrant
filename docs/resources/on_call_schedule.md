---
page_title: "FireHydrant Resource: firehydrant_on_call_schedule"
subcategory: "Signals"
---

# firehydrant_on_call_schedule Resource

FireHydrant on-call schedules are used to define who is on-call for a given time period.

## Example Usage

Basic usage:
```hcl
data "firehydrant_user" "my-user" {
  email = "user@example.com"
}

resource "firehydrant_team" "example-team" {
  name        = "example-team"
  description = "This is an example team"

  memberships {
    user_id          = data.firehydrant_user.my-user.id
  }
}

resource "firehydrant_on_call_schedule" "primary" {
  name        = "Primary On-Call Schedule"
  description = "This is an example on-call schedule"
  team_id     = firehydrant_team.example-team.id

  member_ids = [
    data.firehydrant_user.my-user.id,
  ]

  time_zone   = "America/New_York"

  strategy {
    type         = "weekly"
    handoff_time = "10:00:00"
    handoff_day  = "thursday"
  }

  restrictions {
    start_day = "monday"
    end_day = "monday"
    start_time = "10:00:00"
    end_time = "14:00:00"
  }

  restrictions {
    start_day = "tuesday"
    end_day = "tuesday"
    start_time = "12:00:00"
    end_time = "23:00:00"
  }
}
```

Schedule with a custom strategy:
```hcl
resource "firehydrant_team" "example_team" {
  name = "example-team"
}

resource "firehydrant_on_call_schedule" "primary" {
  name        = "Primary On-Call Schedule"
  description = "This is an example on-call schedule"
  team_id     = firehydrant_team.example_team.id
  time_zone   = "America/Los_Angeles"
  start_time  = "2024-04-11T11:56:29-07:00"
  
  strategy {
    type           = "custom"
    shift_duration = "PT93600S"
  }

  restrictions {
    start_day  = "monday"
    start_time = "14:00:00"
    end_day    = "friday"
    end_time   = "17:00:00"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the on-call schedule.
* `description` - (Optional) A description for the on-call schedule.
* `team_id` - (Required) The ID of the team that the on-call schedule belongs to.
* `member_ids` - (Required) A list of user IDs that are on-call for the on-call schedule.
* `members` - (Deprecated) use `member_ids` instead.
* `time_zone` - (Required) The time zone that the on-call schedule is in.
* `slack_user_group_id` - (Optional) The ID of the Slack user group that the on-call schedule is associated with.
* `strategy` - (Required) A block to define the strategy for the on-call schedule.
* `restrictions` - (Optional) A block to define a restriction for the on-call schedule.
* `effective_at` - (Optional) The date and time that the on-call schedule becomes effective. Must be in `YYYY-MM-DDTHH:MM:SSZ` format. Defaults to the current date and time. If set to the past, the schedule will be effective immediately. This attribute is not stored in Terraform state. The API rejects an `effective_at` that is more than one month in the past. If the configured value is older than one month, the provider can send a later value on update. The provider does this when the schedule has a `daily` or `weekly` strategy, `member_ids`, and no `restrictions`. The later value is a whole number of rotations (member count times shift length) after the configured value, so the on-call order does not change. If one rotation is longer than one month, the update starts at the next rotation boundary, and the provider shows a warning. For other schedules, the provider sends the configured value.
* `start_time` - (Optional) An ISO8601 time string specifying when the initial rotation should start. This value is only used if the rotation's strategy type is "custom".
* `rotation_name` - (Optional) Name of the schedule's primary rotation (the rotation FireHydrant creates alongside the schedule itself). When omitted, the rotation inherits the schedule's name. Changes are sent to the schedule's PATCH endpoint and update the rotation in place. Useful when modeling a source system whose schedules contain named layers (e.g. PagerDuty), so the layer keeps its name in FireHydrant.
* `rotation_description` - (Optional) Description of the schedule's primary rotation. Falls back to the schedule's description when omitted. Changes are sent to the schedule's PATCH endpoint and update the rotation in place.

The `strategy` block supports:

* `type` - (Required) The type of strategy to use for the on-call schedule. Valid values are `weekly`, `daily`, or `custom`.
* `handoff_time` - (Required) The time of day that the on-call schedule handoff occurs. Must be in `HH:MM:SS` format.
* `handoff_day` - (Required) The day of the week that the on-call schedule handoff occurs. Valid values are `sunday`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, and `saturday`.
* `shift_duration` - (Optional) The duration of the on-call shift in [ISO8601 format](https://en.wikipedia.org/wiki/ISO_8601#Durations) (e.g. `PT8H`). Required for `custom` strategy.

The `restrictions` block supports:

* `start_day` - (Required) The day of the week that the restriction starts. Valid values are `sunday`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, and `saturday`.
* `end_day` - (Required) The day of the week that the restriction ends. Valid values are `sunday`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, and `saturday`.
* `start_time` - (Required) The time of day that the restriction starts. Must be in `HH:MM:SS` format.
* `end_time` - (Required) The time of day that the restriction ends. Must be in `HH:MM:SS` format.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the on-call schedule.

## Import

On-call schedules can be imported; use `<TeamID>:<ScheduleID>` as the import ID. For example:

```shell
terraform import firehydrant_on_call_schedule.example_schedule 3638b647-b99c-5051-b715-eda2c912c42e:12345678-90ab-cdef-1234-567890abcdef
```