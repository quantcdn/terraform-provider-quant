terraform {
  required_providers {
    quant = {
      source = "registry.terraform.io/quantcdn/quant"
    }
  }
  required_version = ">= 1.1.0"
}

resource "quant_rule_headers" "header_rule" {
  project = "api-test"
  name = "Terraform header rule"
  disabled = false
  domain = "any"
  url = "/another-change-2/*"

  # Selection criteria
  # country_include = false
  # countries = []
  # method_include = false
  # methods = []
  # ip_include = false
  # ips = []
  # only_with_cookie = "test-cookie"

  headers = {
    "x-customer-header": "value",
  }
}

output "header_rule_result" {
  description = "The status of the header rule"
  value = quant_rule_headers.header_rule.uuid
}

resource "quant_rule_custom_response" "custom_response" {
  project = "api-test"
  name = "Terraform custom response rule"
  disabled = false
  domain = "any"
  url = "/test-response"

  custom_response_status_code = 200
  custom_response_body = "<h1>This is a custom response</h1>"
}

# resource "quant_rule_proxy" "proxy_rule" {
#   project = "api-test"
#   name = "Proxy rule"
#   disabled = false
#   url = "any"

#   # Selection criteria
#   # country_include = false
#   # countries = []
#   # method_include = false
#   # methods = []
#   # ip_include = false
#   # ips = []
#   # only_with_cookie = "test-cookie"

#   url = "*"
#   to = "https://origin.server"
#   host = "forward-for.com"

#   # Original authentication details
#   auth_user = ""
#   auth_pass = ""

#   disable_ssl_verify = true
#   only_404 = true

#   strip_headers = []

#   waf_config = {
#     mode = "block"
#     paranoia_level = 4
#     allow_rules = []
#     allow_ip = []
#     block_ip = []
#     block_ua = []
#     block_referer = []
#     notify_slack = ""
#     notify_slack_rpm = 5
#     httpbl = {
#       enabled = false
#       block_suspicious = false
#       block_harvester = false
#       block_spam = false
#       block_search_engine = false
#     }
#   }

# }

