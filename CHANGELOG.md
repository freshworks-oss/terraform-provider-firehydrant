## Unreleased

FEATURES:

* **New Resource:** `firehydrant_signal_webhook_target` ([#244](https://github.com/firehydrant/terraform-provider-firehydrant/pull/244))

## 0.15.2

BUG FIXES:

* `firehydrant_on_call_schedule` no longer silently drops `start_time` when the strategy is `weekly` or `daily`. The provider discarded the value for every non-`custom` strategy before sending the create request, so the schedule's initial rotation started its shifts at the strategy boundary (e.g. the beginning of the week) instead of the configured start time. Because the API does not echo `start_time` back, the drop never surfaced as a state diff.

## 0.15.1

ENHANCEMENTS:

* `firehydrant_on_call_schedule` now accepts `rotation_name` and `rotation_description` attributes that override the name/description of the schedule's primary rotation. Without these, the rotation inherits the schedule's name, which makes layer-style rotations indistinguishable from the schedule itself in the UI. Both attributes are read back from the API on refresh and update the rotation in place via the schedule's PATCH endpoint when changed.

## 0.15.0

BREAKING CHANGES:

* Setting a member with an empty string as user_id will now create a blank rotation shift. Previously these would be filtered out, moving forward users will be able to intentionally set to blank rotation shifts as a method to create a placeholder/gap in a rotation.

BUG FIXES:

* Fixes an error accessing and managing rotation members.

## 0.14.12

BUG FIXES:

* Fixed an error accessing AutoAddRespondingTeam on Functionalities

## 0.14.11

BUG FIXES:

* Upgrades Go SDK Client to fix errors on unmarshalling empty or optional fields
* Allows clearing of empty fields for resources using Go SDK client
* Add default matching strategy to inbound email resource, preventing terraform state drift

## 0.14.10

ENHANCEMENTS:

* **New Data Sources**: 
`firehydrant_escalation_policy` ([#209](https://github.com/firehydrant/terraform-provider-firehydrant/pull/209))
`firehydrant_signal_rule` ([#205](https://github.com/firehydrant/terraform-provider-firehydrant/pull/205))

Documentation updated for `firehydrant_team`, `firehydrant_escalation_policy`, `firehydrant_signal_rule`

Adds support for dynamic escalation policies to `firehydrant_escalation_policy`
Adds support for deduplication expiry, create incident condition when, target specifications to `firehydrant_signal_rule`

## 0.14.9

BUG FIXES:

* Fixes error on retrieving schedules when schedule names have a common substring, if an exact match is possible

## 0.14.8

ENHANCEMENTS:

* **New Resource**: `firehydrant_incident_type` ([#199](https://github.com/firehydrant/terraform-provider-firehydrant/pull/199))

## 0.14.7

ENHANCEMENTS:

* Added `memberships` as an attribute for `firehydrant_teams` data source

## 0.14.6

ENHANCEMENTS:

* Added `ingest_url` as a computed attribute for `firehydrant_custom_event_source`

## 0.14.5

BUG FIXES: 

* Fixes 500 on unmarshalling example_payload for transposers 

## 0.14.4

ENHANCEMENTS:

* **New Resource**: `firehydrant_custom_event_source` ([#192](https://github.com/firehydrant/terraform-provider-firehydrant/pull/192))
* Documentation added for using `incident_type` in runbook action logic

## 0.14.3

ENHANCEMENTS:

* `firehydrant_on_call_schedule` and `firehydrant_rotation` now support imports.

## 0.14.2

ENHANCEMENTS:

* **New Resource**: `firehydrant_role` ([#188](https://github.com/firehydrant/terraform-provider-firehydrant/pull/188))
* **New Data Source**: `firehydrant_role` ([#188](https://github.com/firehydrant/terraform-provider-firehydrant/pull/188))
* **New Data Source**: `firehydrant_permissions` ([#188](https://github.com/firehydrant/terraform-provider-firehydrant/pull/188))

## 0.14.1

Documentation for on-call schedules updated to show example for `start_time` parameter.

## 0.14.0

ENHANCEMENTS:
* **New Data Source**: `firehydrant_rotation` ([#184](https://github.com/firehydrant/terraform-provider-firehydrant/pull/184))
* **New Resource**: `firehydrant_rotation` ([#184](https://github.com/firehydrant/terraform-provider-firehydrant/pull/184))

## 0.13.5

ENHANCEMENTS:
* **New Data Source**: `firehydrant_on_call_schedule` ([#174](https://github.com/firehydrant/terraform-provider-firehydrant/pull/174))
* **New Data Source**: `firehydrant_on_call_schedules` ([#174](https://github.com/firehydrant/terraform-provider-firehydrant/pull/174))

## 0.13.4

BUG FIXES:
* Fixes inbound email targets panic when attempting to access index out of range

## 0.13.3

BUG FIXES:
* Fixes a discrepancy between the FireHydrant API and `firehydrant_inbound_email` resource where the `target` attribute was erroneously being required.
* Fixes a bug where `notification_priority_override` could not be unset on a `firehydrant_signal_rule` resource.

## 0.13.2

BUG FIXES:
* Fixes a bug where on-call schedule handoff days were not being updated in FireHydrant

EHNANCEMENTS:
* Adds a new `effective_at` attribute to the `firehydrant_on_call_schedule` resource to allow for scheduling changes to take effect at a future date

## 0.13.1

BUG FIXES:
* Fixes a bug where escalation policy resources were not being removed from state if deleted outside of Terraform

## 0.13.0

ENHANCEMENTS:
* **New Resource**: `firehydrant_inbound_email` ([#165](https://github.com/firehydrant/terraform-provider-firehydrant/pull/165/))
* Adds the attribute for notification priority override to the `firehydrant_signal_rule` resource ([#166](https://github.com/firehydrant/terraform-provider-firehydrant/pull/166))

## 0.12.1

REVERTED:
* Reverted the client-side rate limiting and burst control ([#164](https://github.com/firehydrant/terraform-provider-firehydrant/pull/164))

## 0.12.0

ENHANCEMENTS:
* Resilient client-side rate limiting and burst control ([#162](https://github.com/firehydrant/terraform-provider-firehydrant/pull/162))

## 0.11.0

BUG FIXES:
* `firehydrant_teams` and `firehydrant_services` data sources are now aware of all pages of teams ([#160](https://github.com/firehydrant/terraform-provider-firehydrant/pull/160))

## 0.10.0

BUG FIXES:
* `firehydrant_services` data source pulls all services correctly

## 0.9.1

BUG FIXES:
* `firehydrant_escalation_policy` example and documentation updated to correctly reflect the usage requirements.

## 0.9.0

* **New Data Source**: `firehydrant_ingest_url` ([#152](https://github.com/firehydrant/terraform-provider-firehydrant/pull/152)).
* **Added to Resource**: `firehydrant_signal_rule` `incident_type_id` can now be specified for rules ([#153](https://github.com/firehydrant/terraform-provider-firehydrant/pull/153)).

BUG FIXES:
* `EffectiveAt` is now sent for all on call schedule updates. ([#154](https://github.com/firehydrant/terraform-provider-firehydrant/pull/154)).

## 0.8.0

* `firehydrant_on_call_schedule` scheduling strategy now supports `custom` shift durations ([#151](https://github.com/firehydrant/terraform-provider-firehydrant/pull/151)).


## 0.7.1

* `firehydrant_runbook` now supports `restricted` field for private incidents runbook ([#149](https://github.com/firehydrant/terraform-provider-firehydrant/pull/149))

## 0.7.0

ENHANCEMENTS:
* **Added to Data Source**: `firehydrant_slack_channel`. Now you can pass either the channel id or the channel name to the data source.

## 0.6.0

ENHANCEMENTS:

* **New Data Source**: `firehydrant_slack_channel` ([#147](https://github.com/firehydrant/terraform-provider-firehydrant/pull/147)).
* Improved "resource not found error" with more details of URL.

## 0.5.0

* `firehydrant_team` now supports `slug` attribute ([#145](https://github.com/firehydrant/terraform-provider-firehydrant/pull/145)).

## 0.4.2

ENHANCEMENTS:

* Bump Go version to 1.22 ([#144](https://github.com/firehydrant/terraform-provider-firehydrant/pull/144))

BUG FIXES:

* Resource `firehydrant_on_call_schedule` now handles `member_ids` correctly. Field `members` has been deprecated. ([#143](https://github.com/firehydrant/terraform-provider-firehydrant/pull/143))

## 0.4.1

BUG FIXES:

* Fixes a bug where handoff steps in escalation policies were assigning to the incorrect schema and causing a panic.

## 0.4.0

* Add labels to functionalities
* Add initial Signals resources for beta period

## 0.3.6

* Fix client versioning

## 0.3.5

* Use timestamps for data source IDs
* Better document how to have runbooks automatically attach

## 0.3.4

* New Data Source: `firehydrant_team` ([#121](https://github.com/firehydrant/terraform-provider-firehydrant/pull/121))
* New Data Source: `firehydrant_teams` ([#121](https://github.com/firehydrant/terraform-provider-firehydrant/pull/121))

## 0.3.3

* Bump golang from 1.16 to 1.18
* resource/service: Added `external_resources` attribute to service ([#123](https://github.com/firehydrant/terraform-provider-firehydrant/pull/123))

## 0.3.2

ENHANCEMENTS:

* resource/service: Added `auto_add_responding_team` attribute to service ([#117](https://github.com/firehydrant/terraform-provider-firehydrant/pull/117))
* data_source/service: Added `auto_add_responding_team` attribute to service ([#117](https://github.com/firehydrant/terraform-provider-firehydrant/pull/117))
* data_source/services: Added `auto_add_responding_team` attribute to service ([#117](https://github.com/firehydrant/terraform-provider-firehydrant/pull/117))
* resource/team: Add the ability to attach memberships to teams ([#116](https://github.com/firehydrant/terraform-provider-firehydrant/pull/116))

## 0.3.1

BUG FIXES:

* documentation: Fixed broken links in runbook resource documentation ([#101](https://github.com/firehydrant/terraform-provider-firehydrant/pull/101))

## 0.3.0

BREAKING CHANGES:

* resource/functionality: Removed deprecated `services` attribute, preferring `service_ids` instead ([#94](https://github.com/firehydrant/terraform-provider-firehydrant/pull/94))
* resource/runbook: Changed the type of the `steps` `config` attribute to a JSON string ([#79](https://github.com/firehydrant/terraform-provider-firehydrant/pull/79))
* resource/runbook: Removed deprecated `type` attribute ([#80](https://github.com/firehydrant/terraform-provider-firehydrant/pull/80))
* resource/runbook: Removed deprecated `severities` attribute ([#80](https://github.com/firehydrant/terraform-provider-firehydrant/pull/80))
* resource/runbook: Changed `steps` attribute to be required ([#80](https://github.com/firehydrant/terraform-provider-firehydrant/pull/80))

BUG FIXES:

* resource/environment: Fixed bug that prevented the `description` attribute from being unset ([#93](https://github.com/firehydrant/terraform-provider-firehydrant/pull/93))
* resource/runbook: Fixed bug that prevented the `description` attribute from being unset ([#80](https://github.com/firehydrant/terraform-provider-firehydrant/pull/80))
* resource/severity: Fixed bug that prevented the `description` attribute from being unset ([#88](https://github.com/firehydrant/terraform-provider-firehydrant/pull/88))
* data_source/runbook_action: Fixed bug that prevented the Slack `add_bookmark_to_incident_channel` action from working by making `type` optional ([#92](https://github.com/firehydrant/terraform-provider-firehydrant/pull/92))

FEATURES:

* **New Resource:** `firehydrant_incident_role` ([#87](https://github.com/firehydrant/terraform-provider-firehydrant/pull/87))
* **New Resource:** `firehydrant_priority` ([#65](https://github.com/firehydrant/terraform-provider-firehydrant/pull/65))
* **New Resource:** `firehydrant_service_dependency` ([#89](https://github.com/firehydrant/terraform-provider-firehydrant/pull/89))
* **New Resource:** `firehydrant_task_list` ([#85](https://github.com/firehydrant/terraform-provider-firehydrant/pull/85))
* **New Data Source:** `firehydrant_incident_role` ([#87](https://github.com/firehydrant/terraform-provider-firehydrant/pull/87))
* **New Data Source:** `firehydrant_priority` ([#65](https://github.com/firehydrant/terraform-provider-firehydrant/pull/65))
* **New Data Source:** `firehydrant_severity` ([#88](https://github.com/firehydrant/terraform-provider-firehydrant/pull/88))
* **New Data Source:** `firehydrant_task_list` ([#85](https://github.com/firehydrant/terraform-provider-firehydrant/pull/85))

ENHANCEMENTS:

* provider: Improved error messages by adding details from the API response ([#75](https://github.com/firehydrant/terraform-provider-firehydrant/pull/75))
* provider: Improved documentation for configuring the provider ([#95](https://github.com/firehydrant/terraform-provider-firehydrant/pull/95))
* resource/environment: Added logging ([#93](https://github.com/firehydrant/terraform-provider-firehydrant/pull/93))
* resource/functionality: Added logging ([#94](https://github.com/firehydrant/terraform-provider-firehydrant/pull/94))
* resource/runbook: Added logging ([#74](https://github.com/firehydrant/terraform-provider-firehydrant/pull/74))
* resource/runbook: Added the `owner_id` attribute to runbook ([#76](https://github.com/firehydrant/terraform-provider-firehydrant/pull/76))
* resource/runbook: Added the `repeats` and `repeat_duration` attribute to runbook step ([#78](https://github.com/firehydrant/terraform-provider-firehydrant/pull/78))
* resource/runbook: Added the `attachment_rule` attribute to runbook ([#82](https://github.com/firehydrant/terraform-provider-firehydrant/pull/82))
* resource/runbook: Added default value of `false` to the steps `automatic` attribute ([#83](https://github.com/firehydrant/terraform-provider-firehydrant/pull/83))
* resource/runbook: Added the `rule` attribute to runbook steps ([#84](https://github.com/firehydrant/terraform-provider-firehydrant/pull/84))
* resource/runbook: Added documentation for step configuration for every runbook action ([#81](https://github.com/firehydrant/terraform-provider-firehydrant/pull/81))
* resource/service: Added logging ([#96](https://github.com/firehydrant/terraform-provider-firehydrant/pull/96))
* resource/severity: Added logging to the resource and validation to the `slug` attribute ([#88](https://github.com/firehydrant/terraform-provider-firehydrant/pull/88))
* resource/severity: Added support for the `type` attribute ([#88](https://github.com/firehydrant/terraform-provider-firehydrant/pull/88))
* resource/team: Added logging ([#96](https://github.com/firehydrant/terraform-provider-firehydrant/pull/96))
* data_source/environment: Added logging ([#93](https://github.com/firehydrant/terraform-provider-firehydrant/pull/93))
* data_source/functionality: Added logging ([#94](https://github.com/firehydrant/terraform-provider-firehydrant/pull/94))
* data_source/runbook: Added logging ([#74](https://github.com/firehydrant/terraform-provider-firehydrant/pull/74))
* data_source/runbook: Added the `owner_id` attribute to runbook ([#76](https://github.com/firehydrant/terraform-provider-firehydrant/pull/76))
* data_source/runbook: Added the `attachment_rule` attribute to runbook ([#82](https://github.com/firehydrant/terraform-provider-firehydrant/pull/82))
* data_source/runbook_action: Added logging ([#74](https://github.com/firehydrant/terraform-provider-firehydrant/pull/74))
* data_source/service: Added logging ([#96](https://github.com/firehydrant/terraform-provider-firehydrant/pull/96))
* data_source/services: Added logging ([#96](https://github.com/firehydrant/terraform-provider-firehydrant/pull/96))

NOTES:

* resource/functionality: The deprecated `services` attribute has been removed. See the ["Notes" section in 0.2.0](#020)
  or the [original deprecation PR](https://github.com/firehydrant/terraform-provider-firehydrant/pull/49) for more information.
* resource/runbook: There are a number of breaking changes for the runbook resource. The `steps` attribute is now required,
  the `steps` `config` attribute is now a JSON string, and the `type` and `severities` attribute have been removed. In order
  to upgrade to 0.3.0, you will need to destroy your existing runbooks and recreate them after changing your configuration to
  account for the breaking changes.
  As an example, the configuration below was valid in 0.2.1
   ```hcl
   # An example of a valid 0.2.1 configuration
   resource "firehydrant_runbook" "example-runbook" {
     name = "example-runbook"
     type = "incident"

     steps {
       name    = "Send me an email"
       action_id = data.firehydrant_runbook_action.firehydrant_email_notification.id

       config = {
         email_address   = "test@example.com"
         email_subject   = "Incident opened on FireHydrant"
         default_message = "Message"
       }
     }
   }
   ```
  To upgrade to 0.3.0, that configuration would have to change to the following:
   ```hcl
   # The same configuration as above, updated to be valid for 0.3.0
   resource "firehydrant_runbook" "example-runbook" {
     name = "example-runbook"

     steps {
       name    = "Send me an email"
       action_id = data.firehydrant_runbook_action.firehydrant_email_notification.id

       config = jsonencode({
         email_address   = "test@example.com"
         email_subject   = "Incident opened on FireHydrant"
         default_message = "Message"
       })
     }
   }
   ```

## 0.2.1

ENHANCEMENTS:

* documentation: Refactored documentation to follow best practices, added descriptions for all arguments, and added configuration examples for all resources and data sources ([#69](https://github.com/firehydrant/terraform-provider-firehydrant/pull/69))

## 0.2.0

BREAKING CHANGES:

* resource/team: Removed `services` attribute. Use resource/service to associate teams with a service. See "Notes" for more information ([#54](https://github.com/firehydrant/terraform-provider-firehydrant/pull/54))

BUG FIXES:

* provider: Fixed bug where errors weren't being checked or handled for various API requests ([#58](https://github.com/firehydrant/terraform-provider-firehydrant/pull/58))
* resource/functionality: Fixed bug that prevented the `description` attribute from being unset ([#49](https://github.com/firehydrant/terraform-provider-firehydrant/pull/49))
* resource/functionality: Fixed bug that prevented the `services` attribute from being unset ([#49](https://github.com/firehydrant/terraform-provider-firehydrant/pull/49))
* resource/functionality: Fixed bug that prevented functionalities from being removed from state when they had been deleted outside of Terraform ([#58](https://github.com/firehydrant/terraform-provider-firehydrant/pull/58))
* resource/service: Fixed bug that prevented the `description` attribute from being unset ([#51](https://github.com/firehydrant/terraform-provider-firehydrant/pull/51))
* resource/service: Fixed bug that prevented `labels` from being removed from services ([#52](https://github.com/firehydrant/terraform-provider-firehydrant/pull/52))
* resource/service: Fixed bug that prevented services from being removed from state when they had been deleted outside of Terraform ([#58](https://github.com/firehydrant/terraform-provider-firehydrant/pull/58))
* resource/team: Fixed bug that prevented teams from being removed from state when they had been deleted outside of Terraform ([#58](https://github.com/firehydrant/terraform-provider-firehydrant/pull/58))
* data_source/runbook_action: Fixed bug that caused the wrong action to be returned when multiple actions existed for the same slug ([#56](https://github.com/firehydrant/terraform-provider-firehydrant/pull/56))

ENHANCEMENTS:

* provider: Added Terraform version to the user agent header ([#24](https://github.com/firehydrant/terraform-provider-firehydrant/pull/24))
* resource/functionality: Added deprecation warning to the `services` attribute, preferring `service_ids` instead ([#49](https://github.com/firehydrant/terraform-provider-firehydrant/pull/49))
* resource/service: Added the `alert_on_add` attribute to services ([#24](https://github.com/firehydrant/terraform-provider-firehydrant/pull/24))
* resource/service: Added the `owner_id` attribute to services ([#23](https://github.com/firehydrant/terraform-provider-firehydrant/pull/23))
* resource/service: Added the `team_ids` attribute to services ([#54](https://github.com/firehydrant/terraform-provider-firehydrant/pull/54))
* resource/service: Added the `links` attribute to services ([#30](https://github.com/firehydrant/terraform-provider-firehydrant/pull/30))
* data_source/functionality: Added the `service_ids` attribute to functionality ([#49](https://github.com/firehydrant/terraform-provider-firehydrant/pull/49))
* data_source/service: Added the `alert_on_add` attribute to services ([#24](https://github.com/firehydrant/terraform-provider-firehydrant/pull/24))
* data_source/service: Added the `owner_id` attribute to services ([#23](https://github.com/firehydrant/terraform-provider-firehydrant/pull/23))
* data_source/service: Added the `team_ids` attribute to services ([#54](https://github.com/firehydrant/terraform-provider-firehydrant/pull/54))
* data_source/service: Added the `links` and `labels` attributes to services ([#30](https://github.com/firehydrant/terraform-provider-firehydrant/pull/30))
* data_source/services: Added the `alert_on_add` attribute to services ([#24](https://github.com/firehydrant/terraform-provider-firehydrant/pull/24))
* data_source/services: Added the `owner_id` attribute to services ([#23](https://github.com/firehydrant/terraform-provider-firehydrant/pull/23))
* data_source/services: Added the `team_ids` attribute to services ([#54](https://github.com/firehydrant/terraform-provider-firehydrant/pull/54))
* data_source/services: Added the `links` and `labels` attributes to services ([#30](https://github.com/firehydrant/terraform-provider-firehydrant/pull/30))

NOTES:

* The services attribute has been removed from resource/team. If you need to add a team as an owner or responder to a service, use resource/service and specify `owner_id` or `team_ids`.
   When upgrading to 0.2.0, you should remove the `services` attribute from any team resources and instead add `team_ids` to each service resource.
   As an example, the configuration below was valid in 0.1.4.
   ```hcl
   # An example of a valid 0.1.4 configuration
   resource "firehydrant_service" "service1" {
     name        = "service1"
     description = "description1"

     labels = {
       language  = "ruby",
       lifecycle = "production"
     }

     service_tier = 1
   }

   resource "firehydrant_team" "team1" {
     name = "team1"

     services {
       id = firehydrant_service.service1.id
     }
   }

   resource "firehydrant_team" "team2" {
     name        = "team2"
     description = "description2"

     services {
       id = firehydrant_service.service1.id
     }
   }
   ```
   To upgrade to 0.2.0, that configuration would have to change to the following:
   ```hcl
   # The same configuration as above, updated to be valid for 0.2.0
   resource "firehydrant_service" "service1" {
     name        = "service1"
     description = "description1"

     labels = {
       language  = "ruby",
       lifecycle = "production"
     }

     service_tier = 1

     team_ids = [
       firehydrant_team.team1.id,
       firehydrant_team.team2.id
     ]
   }


   resource "firehydrant_team" "team1" {
     name = "team1"
   }

   resource "firehydrant_team" "team2" {
     name        = "team2"
     description = "description2"
   }
   ```
* The deprecated attribute `services` will be removed from resource/functionality 3 months after the release of v0.2.0. You will have until June 30, 2022 to migrate to the preferred attribute.
  More information about this deprecation can be found in the description of ([#49](https://github.com/firehydrant/terraform-provider-firehydrant/pull/49))

## 0.1.4

BUG FIXES:
* Only one page of results was returned when using`data "firehydrant_runbook_action"` to lookup runbook steps. This meant some steps were completely inaccessible through the provider. This patch will return all steps.

FEATURES:
* CHANGELOG.md added to track contents of each release

IMPROVEMENTS
* The `examples` directory now is separated by resource and includes more and extended examples of `service` resources

## 0.1.3
FEATURES:
* Automated release process using GitHub Actions
* Added `service_tier` to service resource
## 0.1.2 (Dec 3, 2020)

NO CHANGES
## 0.1.1 (Dec 3, 2020)

IMPROVEMENTS
* Added autogenerated documentation

## 0.1.0 (Dec 3, 2020)

INITIAL RELEASE
