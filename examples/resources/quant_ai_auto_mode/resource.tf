resource "quant_ai_governance" "example" {
  ai_enabled   = true
  model_policy = "unrestricted"
}

resource "quant_ai_auto_mode" "example" {
  enabled                  = true
  orchestrator_model       = "example-orchestrator-model"
  watcher_interval_minutes = 5
  max_parallel_tasks       = 4
  enforcement              = "default" # QuantGov accepts "required" only.
  autonomous_default       = false

  group = {
    roles = {
      fast     = ["example-fast-model"]
      standard = ["example-standard-model", "example-fallback-model"]
      deep     = ["example-deep-model"]
    }
  }

  # The API refuses auto mode until the organisation has governance settings.
  depends_on = [quant_ai_governance.example]
}
