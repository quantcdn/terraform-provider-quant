# Crawler Schedule Resource

Manages automated crawler schedules using cron expressions.

## Example Usage

### Basic Daily Schedule

```hcl
resource "quant_crawler" "example" {
  project = "my-project"
  name    = "Main Crawler"
  domain  = "https://example.com"
}

resource "quant_crawler_schedule" "daily" {
  project              = "my-project"
  crawler_uuid         = quant_crawler.example.uuid
  name                 = "Daily Crawl"
  schedule_cron_string = "0 2 * * *"  # 2 AM every day
}
```

### Multiple Schedules

```hcl
# Weekday schedule
resource "quant_crawler_schedule" "weekday" {
  project              = "my-project"
  crawler_uuid         = quant_crawler.example.uuid
  name                 = "Weekday Crawl"
  schedule_cron_string = "0 8 * * 1-5"  # 8 AM Monday-Friday
}

# Weekend schedule  
resource "quant_crawler_schedule" "weekend" {
  project              = "my-project"
  crawler_uuid         = quant_crawler.example.uuid
  name                 = "Weekend Crawl"
  schedule_cron_string = "0 10 * * 0,6"  # 10 AM Saturday and Sunday
}
```

### Hourly Schedule

```hcl
resource "quant_crawler_schedule" "hourly" {
  project              = "my-project"
  crawler_uuid         = quant_crawler.example.uuid
  name                 = "Hourly Updates"
  schedule_cron_string = "0 * * * *"  # Every hour on the hour
}
```

## Argument Reference

### Required Arguments

- `project` - (Required) The machine name of the Quant project.
- `crawler_uuid` - (Required) The UUID of the crawler to schedule.
- `schedule_cron_string` - (Required) Cron expression defining when the crawler should run.

### Optional Arguments

- `name` - (Optional) A descriptive name for the schedule.
- `crawler` - (Optional) Alternative to `crawler_uuid` for specifying the crawler.
- `organization` - (Optional) Organization identifier (typically inherited from provider).

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The numeric ID of the crawler schedule.
- `crawler_config_id` - The ID of the associated crawler configuration.
- `project_id` - The numeric ID of the project.
- `crawler_last_run_id` - The ID of the last crawler run triggered by this schedule.
- `created_at` - Timestamp when the schedule was created.
- `updated_at` - Timestamp when the schedule was last updated.

## Cron Expression Format

The cron expression follows standard cron syntax:

```
* * * * *
│ │ │ │ │
│ │ │ │ └─── Day of week (0-6, Sunday=0)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

### Common Examples

- `0 0 * * *` - Daily at midnight
- `0 */6 * * *` - Every 6 hours
- `30 2 * * *` - Daily at 2:30 AM
- `0 9 * * 1` - Every Monday at 9 AM
- `0 0 1 * *` - First day of every month at midnight
- `0 12 * * 1-5` - Weekdays at noon
- `*/15 * * * *` - Every 15 minutes

## Import

Crawler schedules can be imported using the format `organization/project/id`:

```shell
terraform import quant_crawler_schedule.example my-org/my-project/123
```

## Notes

- Multiple schedules can be created for the same crawler.
- Cron expressions are evaluated in UTC timezone.
- Modifying the `schedule_cron_string` will update the existing schedule rather than creating a new one.
- The crawler must exist before creating a schedule for it.
