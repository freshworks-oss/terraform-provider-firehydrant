---
page_title: "FireHydrant Resource: firehydrant_signal_webhook_target"
subcategory: "Signals"
---

# firehydrant_signal_webhook_target Resource

FireHydrant signal webhook targets are URLs that FireHydrant notifies when signals are received.

## Example Usage

Basic usage:
```hcl
resource "firehydrant_signal_webhook_target" "example" {
  name = "Example Webhook"
  url  = "https://example.com/webhook"
}
```

Using all available attributes:
```hcl
resource "firehydrant_signal_webhook_target" "example" {
  name        = "Example Webhook"
  url         = "https://example.com/webhook"
  description = "Forwards signals to the example service"
  signing_key = var.webhook_signing_key
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the webhook target.
* `url` - (Required) The URL that the webhook target will notify. Must be an `http` or `https` URL.
* `description` - (Optional) A description of the webhook target.
* `signing_key` - (Optional, Sensitive) A secret FireHydrant provides in the `FH-Signature` header when sending
  payloads to the webhook target. The API never returns this value once it has been set, so the provider only
  ever sends it and never reads it back. Two consequences of that: removing `signing_key` from your
  configuration does not remove the key from the webhook target in FireHydrant, and an imported webhook target
  never has a `signing_key` in state. To rotate the key, set `signing_key` to its new value.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the webhook target.

## Import

Signal webhook target resources can be imported using the resource ID, e.g.,

```
$ terraform import firehydrant_signal_webhook_target.example 12345678-90ab-cdef-1234-567890abcdef
```

If your configuration sets `signing_key`, the first plan after importing shows a diff for it, because the API
does not return the key for the provider to store in state. Applying that diff sends the configured key to
FireHydrant.
