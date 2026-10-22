---
page_title: "Quant: quant_application"
subcategory: "Quant Cloud"
description: |-
  Manages a Quant Cloud application and its production environment.
---

# Application Resource

Manages a Quant Cloud application and its production environment.

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
resource "quant_application" "example" {
  app_name = "example-app"

  compose_definition = {
    containers = [
      {
        name            = "web"
        image_reference = {
          type       = "external"
          identifier = "nginx:stable"
        }
        exposed_ports = [80]
        health_check = {
          command  = ["CMD-SHELL", "curl -f http://localhost/ || exit 1"]
          interval = 30
          retries  = 3
          timeout  = 5
        }
      }
    ]
    task_cpu    = 512
    task_memory = 1024
    min_capacity = 1
    max_capacity = 2
  }

  database = {
    engine     = "mysql"
    storage_gb = 20
  }
}
```

## Argument Reference

### Required

* `app_name` (String) - (Required) Application name.
* `compose_definition` (Object) - (Required) Container definition for the application: containers, capacity, and networking.
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
  * `max_capacity` (Number) - (Optional) Maximum number of instances. Defaults to `1`.
  * `min_capacity` (Number) - (Optional) Minimum number of instances. Defaults to `1`.
  * `single_task_only` (Boolean) - (Optional) Optional. Forces single-task mode for data-safe applications (max one running task). When true: capacity is locked to 1. When false: explicitly allows scaling. When omitted: the platform auto-detects stateful containers and enables single-task mode if found.
  * `spot_configuration` (Object) - (Optional) Spot instance strategy configuration for controlling cost vs reliability. Spot instances provide significant cost savings (~70%) but may be interrupted by the platform. Available for non-production environments.
    * `strategy` (String) - (Optional) Spot instance strategy. 'off' = On-Demand only (highest reliability, no savings). 'spot-only' = 100% Spot instances (~70% savings, default for non-prod). 'mixed-safe' = 50% Spot instances (~35% savings, requires multiple instances). 'mixed-aggressive' = 80% Spot instances (~56% savings, requires multiple instances). Valid values: `off`, `spot-only`, `mixed-safe`, `mixed-aggressive`.
  * `startup_grace_period_seconds` (Number) - (Optional) Optional. Seconds the routing layer waits after a task starts before an unhealthy health check can replace it (applied as the service's healthCheckGracePeriodSeconds when a routing layer is attached). Raise for apps that are slow to boot, e.g. run migrations on startup. Tasks that become healthy sooner still enter service immediately. Defaults to 120 when omitted. Range: 0 to 3600.
  * `task_cpu` (Number) - (Optional) Task-level CPU units (e.g., 256, 512, 1024).
  * `task_memory` (Number) - (Optional) Task-level memory in MB.

### Optional

* `application` (String) - (Optional) The application ID.
* `database` (Object) - (Optional) Optional database configuration.
  * `engine` (String) - (Optional) Database engine type (MySQL 8.4, Postgres). Valid values: `mysql`, `postgres`. Defaults to `mysql`.
  * `instance_class` (String) - (Optional) Database instance class (database size class). Defaults to `db.t4g.micro`.
  * `multi_az` (Boolean) - (Optional) Enable a high-availability standby (higher availability and cost). Defaults to `false`.
  * `rds_instance_endpoint` (String) - (Read-Only) Database instance endpoint address.
  * `rds_instance_engine` (String) - (Read-Only) Database engine.
  * `rds_instance_identifier` (String) - (Read-Only) Database instance identifier.
  * `rds_instance_status` (String) - (Read-Only) Database instance status.
  * `storage_gb` (Number) - (Optional) Allocated storage in GiB. Defaults to `20`.
* `environment` (List of Object) - (Optional) Optional. Initial environment variables for the production environment.
  * `name` (String) - (Optional) Environment variable name.
  * `value` (String) - (Optional) Environment variable value.
* `filesystem` (Object) - (Optional) Optional filesystem configuration.
  * `filesystem_id` (String) - (Read-Only) Filesystem ID.
  * `mount_path` (String) - (Optional) Mount path inside containers. Defaults to `/mnt/data`.
  * `required` (Boolean) - (Optional) Whether to create a shared filesystem. Defaults to `true`.
* `max_capacity` (Number) - (Optional) Maximum task count for auto-scaling. Defaults to `1`.
* `min_capacity` (Number) - (Optional) Minimum task count for auto-scaling. Defaults to `1`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `cache` (Object) - (Read-Only) Managed cache configuration.
* `container_names` (List of String) - (Read-Only) List of container names.
* `deployment_information` (List of Object) - (Read-Only) Deployment history.
* `desired_count` (Number) - (Read-Only) Desired task count.
* `environment_names` (List of String) - (Read-Only) List of environment names (read-only).
* `image_reference` (Object) - (Read-Only) Image reference information.
* `organisation` (String) - (Read-Only) Organisation machine name.
* `running_count` (Number) - (Read-Only) Currently running task count.
* `status` (String) - (Read-Only) Application status.

## Import

Import an application with its name. The organisation comes from the provider configuration.

```bash
terraform import quant_application.example example-app
```
