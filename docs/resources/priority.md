---
page_title: "FireHydrant Resource: firehydrant_priority"
---

# firehydrant_priority Resource

FireHydrant priorities define when an incident should be addressed.

## Example Usage

Basic usage:
```hcl
resource "firehydrant_priority" "example-priority" {
  slug        = "MYEXAMPLEPRIORITY"
  description = "This is an example priority"
  default     = true
}
```

## Argument Reference

The following arguments are supported:

* `slug` - (Required) The slug representing the priority. It must be unique and only contain
  alphanumeric characters and hyphens. The slug cannot be longer than 23 characters.
* `default` - (Optional) Indicates whether the priority should be the default priority for incidents. 
  At most one resource can have default set to `true`. Setting default to `true` for multiple priority 
  resources will result in inconsistent plans in Terraform. Defaults to `false`.
* `description` - (Optional) A description for the priority.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the priority. This is the same as the slug.

## Import

Priorities can be imported; use `<PRIORITY SLUG>` as the import ID. For example:

```shell
terraform import firehydrant_priority.test P1
```

