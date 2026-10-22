---
page_title: "Quant: quant_environment"
subcategory: "Quant Cloud"
description: |-
  Manages an environment of a Quant Cloud application. An environment can clone the configuration of another environment.
---

# Environment Resource

Manages an environment of a Quant Cloud application. An environment can clone the configuration of another environment.

## Example Usage

```hcl
provider "quant" {
  bearer       = var.quant_api_token
  organization = "example-org"
  # AU Government customers: uncomment to use the QuantGov dashboard.
  # base_url   = "https://dash.quantgov.cloud"
}
```

```hcl
resource "quant_environment" "staging" {
  application = "example-app"
  env_name    = "staging"

  clone_configuration_from = "production"
  merge_environment        = true

  environment = [
    {
      name  = "APP_ENV"
      value = "staging"
    }
  ]

  min_capacity = 1
  max_capacity = 1
}
```

## Argument Reference

### Required

* `env_name` (String) - (Required) Environment name (e.g., 'staging', 'development').

### Optional

* `application` (String) - (Optional) The application ID.
* `clone_configuration_from` (String) - (Optional) Clone configuration from an existing environment.
* `compose_definition` (Object) - (Optional) Container definition for the application: containers, capacity, and networking.
  * `architecture` (String) - (Optional) CPU architecture (X86_64 or ARM64).
  * `containers` (List of Object) - (Optional) Containers in the task.
    * `command` (List of String) - (Optional) Command to run, as a list of arguments.
    * `cpu` (Number) - (Optional) Container-level CPU units.
    * `depends_on` (List of Object) - (Optional) Container startup dependencies.
      * `condition` (String) - (Optional) The condition to wait for on the dependency. Valid values: `START`, `HEALTHY`, `COMPLETE`, `SUCCESS`.
      * `container_name` (String) - (Required) The name of the container this container depends on.
    * `entry_point` (List of String) - (Optional) Container entry point, as a list of arguments.
    * `environment` (List of Object) - (Optional) Environment variables specific to this container.
      * `name` (String) - (Required) Environment variable name.
      * `value` (String) - (Required) Environment variable value.
    * `essential` (Boolean) - (Optional) Whether the task stops if this container stops. Defaults to `true`.
    * `exposed_ports` (List of Number) - (Optional) List of container ports to expose.
    * `health_check` (Object) - (Optional) Container health check configuration.
      * `command` (List of String) - (Optional) The command to run to determine if the container is healthy.
      * `interval` (Number) - (Optional) Time period (seconds) between health checks. Range: 5 to 300. Defaults to `30`.
      * `retries` (Number) - (Optional) Number of times to retry a failed health check. Range: 1 to 10. Defaults to `3`.
      * `start_period` (Number) - (Optional) Grace period (seconds) to ignore unhealthy checks after container starts. Range: 0 to 300.
      * `timeout` (Number) - (Optional) Time period (seconds) to wait for a health check to return. Range: 2 to 60. Defaults to `5`.
    * `image_reference` (Object) - (Required) Reference to the container image.
      * `identifier` (String) - (Required) The image identifier. For 'internal' type, this is the image tag. For 'external' type, this is the full image name.
      * `type` (String) - (Required) Specifies whether the image is internal (internal registry) or external (e.g., Docker Hub). Valid values: `internal`, `external`.
    * `memory` (Number) - (Optional) Container-level memory hard limit (MiB).
    * `memory_reservation` (Number) - (Optional) Container-level memory soft limit (MiB).
    * `mount_points` (List of Object) - (Optional) Volume mount points for the container.
      * `container_path` (String) - (Required) The path inside the container where the volume is mounted.
      * `read_only` (Boolean) - (Optional) Mount the volume read-only. Defaults to `false`.
      * `source_volume` (String) - (Required) The name of the logical volume.
    * `name` (String) - (Required) Name of the container.
    * `origin_protection` (Boolean) - (Optional) Enable origin protection for all exposed ports on this container. Use originProtectionConfig for advanced options like IP allow lists. Defaults to `false`.
    * `origin_protection_config` (Object) - (Optional) Extended origin protection configuration with IP allow list support.
      * `enabled` (Boolean) - (Optional) Whether origin protection is enabled. Defaults to true if this config object is provided.
      * `ip_allow` (List of String) - (Optional) List of IP addresses or CIDR ranges that can bypass origin protection for direct access (e.g., VPN IPs).
      * `redirect_host` (String) - (Optional) Optional bare host (e.g. www.example.com). When set, requests denied by origin protection are 302-redirected to https://<redirectHost> with path and query preserved, instead of receiving a 403. Allowed IPs and valid-header (CDN) traffic are unaffected.
    * `readonly_root_filesystem` (Boolean) - (Optional) Mount the container root filesystem read-only. Defaults to `false`.
    * `secrets` (List of Object) - (Optional) Secrets mapped to environment variables.
      * `name` (String) - (Required) The environment variable name to be set in the container.
      * `value_from` (String) - (Required) The key of the secret in the environment's 'app-secrets' store.
    * `user` (String) - (Optional) User to run the container process as.
    * `working_directory` (String) - (Optional) Working directory inside the container.
  * `enable_cross_app_networking` (Boolean) - (Optional) Optional. Enable cross-application networking within the same organization. When false (default): Uses shared/app-specific network access group based on enableCrossEnvNetworking. When true: Uses org-specific network access group to enable container-to-container communication with ALL applications in the same organization via service discovery (microservices architecture). This setting takes priority over enableCrossEnvNetworking.
  * `enable_cross_env_networking` (Boolean) - (Optional) Optional. Enable cross-environment networking within the same application. When false (default): Uses shared network access group for complete isolation (most secure). When true: Uses app-specific network access group to enable communication between environments of the same application (e.g., staging can connect to production database). Note: If enableCrossAppNetworking is true, this setting is overridden.
  * `max_capacity` (Number) - (Optional) Maximum number of instances.
  * `min_capacity` (Number) - (Optional) Minimum number of instances.
  * `single_task_only` (Boolean) - (Optional) Optional. Forces single-task mode for data-safe applications (max one running task). When true: capacity is locked to 1. When false: explicitly allows scaling. When omitted: the platform auto-detects stateful containers and enables single-task mode if found.
  * `spot_configuration` (Object) - (Optional) Spot instance strategy configuration for controlling cost vs reliability. Spot instances provide significant cost savings (~70%) but may be interrupted by the platform. Available for non-production environments.
    * `strategy` (String) - (Optional) Spot instance strategy. 'off' = On-Demand only (highest reliability, no savings). 'spot-only' = 100% Spot instances (~70% savings, default for non-prod). 'mixed-safe' = 50% Spot instances (~35% savings, requires multiple instances). 'mixed-aggressive' = 80% Spot instances (~56% savings, requires multiple instances). Valid values: `off`, `spot-only`, `mixed-safe`, `mixed-aggressive`.
  * `startup_grace_period_seconds` (Number) - (Optional) Optional. Seconds the routing layer waits after a task starts before an unhealthy health check can replace it (applied as the service's healthCheckGracePeriodSeconds when a routing layer is attached). Raise for apps that are slow to boot, e.g. run migrations on startup. Tasks that become healthy sooner still enter service immediately. Defaults to 120 when omitted. Range: 0 to 3600.
  * `task_cpu` (Number) - (Optional) Task-level CPU units (e.g., 256, 512, 1024).
  * `task_memory` (Number) - (Optional) Task-level memory in MB.
* `environment` (List of Object) - (Optional) Environment variables to inject.
  * `name` (String) - (Optional) Variable name.
  * `value` (String) - (Optional) Variable value.
* `image_suffix` (String) - (Optional) Optional image tag suffix for cloning.
* `max_capacity` (Number) - (Optional) Maximum number of instances.
* `merge_environment` (Boolean) - (Optional) Whether to merge environment variables with cloned ones (true) or replace them (false). Default: false.
* `min_capacity` (Number) - (Optional) Minimum number of instances.
* `organisation` (String) - (Optional) The organisation ID.
* `single_task_only` (Boolean) - (Optional) Optional. Forces single-task mode (max one running task). When omitted, the platform auto-detects stateful containers.
* `spot_configuration` (Object) - (Optional) Spot instance strategy configuration for controlling cost vs reliability. Spot instances provide significant cost savings (~70%) but may be interrupted by the platform. Available for non-production environments.
  * `strategy` (String) - (Optional) Spot instance strategy. 'off' = On-Demand only (highest reliability, no savings). 'spot-only' = 100% Spot instances (~70% savings, default for non-prod). 'mixed-safe' = 50% Spot instances (~35% savings, requires multiple instances). 'mixed-aggressive' = 80% Spot instances (~56% savings, requires multiple instances). Valid values: `off`, `spot-only`, `mixed-safe`, `mixed-aggressive`.
* `startup_grace_period_seconds` (Number) - (Optional) Optional. Seconds the routing layer waits after a task starts before an unhealthy health check can replace it. If not set, the value from composeDefinition (or default 120) is used. Range: 0 to 3600.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `alb_routing` (Object) - (Read-Only) Routing layer routing configuration.
* `container_names` (List of String) - (Read-Only) Container name list.
* `created_at` (String) - (Read-Only) Creation timestamp.
* `cron` (List of Object) - (Read-Only) Scheduled cron jobs.
* `deployment_failure_reason` (String) - (Read-Only) Reason for deployment failure.
* `deployment_failure_type` (String) - (Read-Only) Why the most recent task stopped. SPOT_INTERRUPTION is informational: the platform reclaimed the task and container platform replaces it.
* `deployment_status` (String) - (Read-Only) Current deployment status.
* `desired_count` (Number) - (Read-Only) Desired number of tasks.
* `load_balancer` (Object) - (Read-Only) Routing configuration.
* `public_ip_address` (String) - (Read-Only) Public IP address for SSH access.
* `running_count` (Number) - (Read-Only) Number of running tasks.
* `security_group` (Object) - (Read-Only) Network access configuration.
* `service` (Object) - (Read-Only) Service details.
* `status` (String) - (Read-Only) Environment status.
* `subnet` (Object) - (Read-Only) Network segment configuration.
* `task_definition` (Object) - (Read-Only) Task definition details.
* `updated_at` (String) - (Read-Only) Last update timestamp.
* `volumes` (List of Object) - (Read-Only) Persistent storage volumes.
* `vpc` (Object) - (Read-Only) Network configuration.

## Import

Import an environment with the ID `application/env_name`. The organisation comes from the provider configuration.

```bash
terraform import quant_environment.staging example-app/staging
```
