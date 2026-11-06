---
page_title: "FireHydrant Resource: firehydrant_functionality"
---

# firehydrant_functionality Resource

A functionality (function) is a programming construct that performs a specific task.
FireHydrant functionalities let you associate backend services with the features your
end users interact with.

## Example Usage

Complete usage:
```hcl
resource "firehydrant_service" "example-service1" {
  name = "my-example-service1"
}

resource "firehydrant_service" "example-service2" {
  name = "my-example-service2"
}

resource "firehydrant_environment" "example-environment" {
  name = "production"
}

resource "firehydrant_team" "example-owner-team" {
  name = "functionality-owner"
}

resource "firehydrant_team" "example-responding-team" {
  name = "functionality-responders"
}

resource "firehydrant_functionality" "example-functionality" {
  name                     = "my-example-functionality"
  alert_on_add             = true
  auto_add_responding_team = true
  description              = "This is an example functionality"

  environment_ids = [
    firehydrant_environment.example-environment.id
  ]

  external_resources {
    connection_type = "github"
    remote_id       = "freshworks-oss/terraform-provider-firehydrant"
  }

  labels = {
    lifecycle = "production"
  }

  links {
    href_url = "https://example.com/internal-dashboard"
    icon_url = "https://example.com/dashboard-icon.png"
    name     = "Internal Dashboard"
  }

  owner_id     = firehydrant_team.example-owner-team.id
  service_tier = 1

  service_ids = [
    firehydrant_service.example-service1.id,
    firehydrant_service.example-service2.id
  ]

  team_ids = [
    firehydrant_team.example-responding-team.id
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the functionality.
* `alert_on_add` - (Optional) Indicates whether FireHydrant should automatically create an
  alert, based on configured integrations, when this functionality is added to an active
  incident. Defaults to `false`.
* `auto_add_responding_team` - (Optional) Indicates whether FireHydrant should automatically
  add the responding team when this functionality is added to an active incident. Defaults
  to `false`.
* `description` - (Optional) A description of the functionality.
* `environment_ids` - (Optional) A set of IDs of the environments associated with this
  functionality.
* `external_resources` - (Optional) External resources associated with the functionality.
* `labels` - (Optional) Key-value pairs associated with the functionality.
* `links` - (Optional) Links associated with the functionality.
* `owner_id` - (Optional) The ID of the team that owns this functionality.
* `service_ids` - (Optional) A set of IDs of the services this functionality is associated with.
* `service_tier` - (Optional) The functionality's service tier, from `0` through `5`. Lower
  values represent higher criticality. Defaults to `5`.
* `team_ids` - (Optional) A set of IDs of the teams responsible for this functionality's
  incident response.

The `links` block supports:

* `href_url` - (Required) The URL for the link.
* `icon_url` - (Optional) The URL of an icon for the link.
* `name` - (Required) The display name of the link.

The `external_resources` block supports:

* `remote_id` - (Required) The ID of the resource in the remote provider.
* `connection_type` - (Optional) The FireHydrant integration slug, such as `github`,
  `opsgenie`, `pager_duty`, `statuspage`, or `victorops`. It is not required when the
  resource has already been imported into FireHydrant.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the functionality.

## Import

Functionalities can be imported; use `<FUNCTIONALITY ID>` as the import ID. For example:

```shell
terraform import firehydrant_functionality.test 3638b647-b99c-5051-b715-eda2c912c42e
```
