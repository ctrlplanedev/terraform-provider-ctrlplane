resource "ctrlplane_system" "example" {
  name        = "payments-platform"
  description = "Payment processing platform services"

  metadata = {
    team  = "payments"
    owner = "platform-engineering"
  }
}
