# Crawler Schedule Resource

Manages a Quant crawler schedule.

## Example Usage

```hcl
resource "quant_crawler_schedule" "crawler_schedule" {
    project = quant_project.test.machine_name
    crawler = quant_crawler.crawler.uuid
    name = "test-crawler-schedule"
    schedule_cron_string = "0 0 * * *"
}
```

## Argument Reference

- `project` - (Required) The machine name of the project.
- `crawler` - (Required) The UUID of the crawler.
- `schedule_cron_string` - (Required) The cron string to schedule the crawler.

## Attributes Reference

- `id` - The ID of the crawler schedule.
- `crawler_config_id` - The ID of the crawler configuration.
- `project_id` - The ID of the project.
- `crawler_last_run_id` - The ID of the last crawler run.
- `schedule_cron_string` - The cron string to schedule the crawler.
- `created_at` - The date and time the crawler schedule was created.
