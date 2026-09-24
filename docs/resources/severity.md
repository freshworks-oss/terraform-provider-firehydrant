---
page_title: "FireHydrant Resource: firehydrant_severity"
---

# firehydrant_severity Resource

FireHydrant severities represent different levels of severity for incidents.

## Example Usage

Basic usage:
```hcl
resource "firehydrant_severity" "example-severity" {
  slug        = "EXAMPLESEVERITY"
  description = "This is an example severity"
  type        = "maintenance"
}
```

## Argument Reference

The following arguments are supported:

* `slug` - (Required) The slug representing the severity. It must be unique and only contain 
  alphanumeric characters and hyphens. The slug cannot be longer than 23 characters.
* `description` - (Optional) A description for the severity.
* `type` - (Optional) The type of the severity.
  Valid values are `gameday`, `maintenance`, and `unexpected_downtime`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the severity. This is the same as the slug.

## Import

Severities can be imported; use `<SEVERITY SLUG>` as the import ID. For example:

```shell
terraform import firehydrant_severity.test SEV3
```
