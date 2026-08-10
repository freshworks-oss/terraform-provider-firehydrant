resource "firehydrant_signal_webhook_target" "example" {
  name        = "Example Webhook"
  url         = "https://example.com/webhook"
  description = "Forwards signals to the example service"
  signing_key = "example-signing-key"
}
