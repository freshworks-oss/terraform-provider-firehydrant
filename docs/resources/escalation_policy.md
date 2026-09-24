---
page_title: "FireHydrant Resource: firehydrant_escalation_policy"
subcategory: "Signals"
---

# firehydrant_escalation_policy Resource

FireHydrant escalation policies are used to define the order in which teams are notified of an alert.

## Example Usage

Basic static escalation policy:
```hcl
data "firehydrant_user" "my-user" {
  email = "user@example.com"
}

data "firehydrant_user" "backup-user" {
  email = "backup-user@example.com"
}

resource "firehydrant_team" "example-team" {
  name        = "example-team"
  description = "This is an example team"

  memberships {
    user_id          = data.firehydrant_user.my-user.id
  }
}

resource "firehydrant_team" "backup-team" {
  name        = "backup-team"
  description = "The backup for the example team"

  memberships {
    user_id          = data.firehydrant_user.backup-user.id
  }
}

resource "firehydrant_escalation_policy" "default_policy" {
  name = "Default Policy"
  description = "This is an example escalation policy"
  team_id = firehydrant_team.example-team.id
  step_strategy = "static"

  step {
    timeout     = "PT1M"

    targets {
      type = "OnCallSchedule"
      id   = firehydrant_on_call_schedule.primary.id
    }

    targets {
      type = "User"
      id   = data.firehydrant_user.my-user.id
    }
  }

  repetitions = 2

  handoff_step {
    target_type = "Team"
    target_id   = firehydrant_team.backup-team.id
  }
}
```

Dynamic escalation policy with priority-specific settings:
```hcl
resource "firehydrant_escalation_policy" "dynamic_policy" {
  name = "Dynamic Policy"
  description = "A dynamic escalation policy with priority-specific settings"
  team_id = firehydrant_team.example-team.id
  step_strategy = "dynamic_by_priority"

  step {
    timeout     = "PT1M"
    priorities  = ["HIGH", "MEDIUM"]

    targets {
      type = "OnCallSchedule"
      id   = firehydrant_on_call_schedule.primary.id
    }
  }

  step {
    timeout     = "PT5M"
    priorities  = ["LOW"]

    targets {
      type = "User"
      id   = data.firehydrant_user.my-user.id
    }
  }

  notification_priority_policies {
    priority = "HIGH"
    repetitions = 3
    
    handoff_step {
      target_type = "Team"
      target_id   = firehydrant_team.backup-team.id
    }
  }

  notification_priority_policies {
    priority = "MEDIUM"
    repetitions = 2
  }

  notification_priority_policies {
    priority = "LOW"
    repetitions = 1
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the escalation policy.
* `description` - (Optional) A description for the escalation policy.
* `team_id` - (Required) The ID of the team to associate the escalation policy with.
* `step` - (Required) A block to define a step in the escalation policy. (a minimum of one is required)
* `handoff_step` - (Optional) A block to define a handoff step in the escalation policy.
* `default` - (Optional) Set this to `true` to mark this as the default escalation policy for this team.
* `repetitions` - (Required) The number of times to repeat the escalation policy. Defaults to 0.
* `step_strategy` - (Optional) The strategy for handling steps in the escalation policy. Can be 'static' or 'dynamic_by_priority'. Defaults to 'static'.
* `notification_priority_policies` - (Optional) Priority-specific policies for dynamic escalation policies.

The `step` block supports:

* `timeout` - (Required) The amount of time to wait before escalating to the next step. Must be in ISO 8601 duration format.
* `targets` - (Required) A block to define a target for the step. (a minimum of one is required)
* `priorities` - (Optional) A list of priorities this step applies to. Only used when `step_strategy` is 'dynamic_by_priority'. Must be one or more of 'HIGH', 'MEDIUM', 'LOW'.

The `targets` block supports:

* `id` - (Required) The ID of the target for this step.
* `type` - (Required) The type of target for this step. Must be one of `User`, `SlackChannel`, or `OnCallSchedule`.

The `handoff_step` block supports:

* `target_id` - (Required) The ID of the target to handoff to.
* `target_type` - (Required) The type of target to handoff to. Must be one of `Team` or `EscalationPolicy`.

The `notification_priority_policies` block supports:

* `priority` - (Required) The priority level. Must be one of 'HIGH', 'MEDIUM', or 'LOW'.
* `repetitions` - (Optional) The number of repetitions for this priority level.
* `handoff_step` - (Optional) A handoff step for this priority level.

The `notification_priority_policies.handoff_step` block supports:

* `target_id` - (Required) The ID of the target to handoff to for this priority level.
* `target_type` - (Required) The type of target to handoff to for this priority level. Must be one of `Team` or `EscalationPolicy`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the escalation policy.

## Import

Escalation policies can be imported; use `<TeamID>:<EscalationPolicyID>` as the import ID. For example:

```shell
terraform import firehydrant_escalation_policy.default_policy 3638b647-b99c-5051-b715-eda2c912c42e:12345678-90ab-cdef-1234-567890abcdef
```

## Escalation Policy Strategies

### Static Escalation Policy

Static escalation policies execute steps in order for all signals regardless of priority. This is the default behavior when `step_strategy` is omitted or set to 'static'.

- Steps execute sequentially for all signals
- Uses the top-level `repetitions` and `handoff_step` settings
- All steps apply to all signal priorities

### Dynamic Escalation Policy

Dynamic escalation policies allow different escalation behavior based on signal priority. Set `step_strategy` to 'dynamic_by_priority' to enable this mode.

- Steps can specify which priorities they apply to via the `priorities` field
- `notification_priority_policies` define priority-specific repetitions and handoff steps
- Allows different escalation paths for HIGH, MEDIUM, and LOW priority signals
- Steps without explicit priorities will apply to all priorities defined in `notification_priority_policies`
