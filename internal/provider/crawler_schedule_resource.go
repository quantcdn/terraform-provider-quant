package provider

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-quant/internal/client"

	"terraform-provider-quant/internal/resource_crawler_schedule"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	openapi "github.com/quantcdn/quant-admin-go"
)

var (
	_ resource.Resource              = (*crawlerScheduleResource)(nil)
	_ resource.ResourceWithConfigure = (*crawlerScheduleResource)(nil)
)

func NewCrawlerScheduleResource() resource.Resource {
	return &crawlerScheduleResource{}
}

type crawlerScheduleResource struct {
	client *client.Client
}

func (r *crawlerScheduleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crawler_schedule"
}

func (r *crawlerScheduleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_crawler_schedule.CrawlerScheduleResourceSchema(ctx)
}

func (r *crawlerScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *internal.Client, got: %T. Please report this issue to the provider developers", req.ProviderData),
		)
	}
	r.client = client
}

func (r *crawlerScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerScheduleCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerScheduleReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update the crawler schedule object.
	resp.Diagnostics.Append(callCrawlerScheduleUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete API call logic
	resp.Diagnostics.Append(callCrawlerScheduleDeleteAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func callCrawlerScheduleCreateAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	req := *openapi.NewCrawlerScheduleRequest(schedule.ScheduleCronString.ValueString())
	req.SetName(schedule.Name.ValueString())

	api, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesCreate(r.client.AuthContext, r.client.Organization, schedule.Project.ValueString(), schedule.Crawler.ValueString()).CrawlerScheduleRequest(req).Execute()

	if err != nil {
		diags.AddError(
			"Unable to create crawler schedule",
			fmt.Sprintf("Error: %s", err.Error()),
		)
		return diags
	}

	// Set the ID from the create response before calling read
	schedule.Id = types.Int64Value(int64(api.GetId()))

	return callCrawlerScheduleReadAPI(ctx, r, schedule)
}

func callCrawlerScheduleReadAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	if schedule.Id.IsUnknown() || schedule.Id.IsNull() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing schedule.id attribute",
			"To read schedule information, id must be provided.",
		)
		return diags
	}

	if schedule.Project.IsNull() || schedule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing schedule.project attribute",
			"To read schedule information, project must be provided.",
		)
		return
	}

	scheduleId := strconv.FormatInt(schedule.Id.ValueInt64(), 10)
	api, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesRead(ctx, r.client.Organization, schedule.Project.ValueString(), schedule.Crawler.ValueString(), scheduleId).Execute()
	if err != nil {
		diags.AddError("Unable to load crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
		return diags
	}

	// Set all fields from the API response
	schedule.Id = types.Int64Value(int64(api.GetId()))
	schedule.CrawlerConfigId = types.Int64Value(int64(api.GetCrawlerConfigId()))
	schedule.ProjectId = types.Int64Value(int64(api.GetProjectId()))
	schedule.CrawlerLastRunId = types.Int64Value(int64(api.GetCrawlerLastRunId()))
	schedule.ScheduleCronString = types.StringValue(api.GetScheduleCronString())
	schedule.Name = types.StringValue(api.GetName())
	schedule.CreatedAt = types.StringValue(api.GetCreatedAt())
	schedule.UpdatedAt = types.StringValue(api.GetUpdatedAt())
	schedule.Organization = types.StringValue(r.client.Organization)
	schedule.CrawlerSchedule = types.StringValue(api.GetScheduleCronString())  // This might need a different value

	// Set deleted_at (null if not deleted)
	if api.DeletedAt != nil {
		schedule.DeletedAt = types.StringValue(*api.DeletedAt)
	} else {
		schedule.DeletedAt = types.StringNull()
	}

	return diags
}

func callCrawlerScheduleDeleteAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	if schedule.Id.IsUnknown() || schedule.Id.IsNull() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing schedule.id attribute",
			"To delete schedule information the schedule id must be provided",
		)
		return
	}

	if schedule.Project.IsNull() || schedule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing schedule.project attribute",
			"To delete schedule information the schedule project must be provided",
		)
		return
	}

	scheduleId := strconv.FormatInt(schedule.Id.ValueInt64(), 10)
	_, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesDelete(ctx, r.client.Organization, schedule.Project.ValueString(), schedule.Crawler.ValueString(), scheduleId).Execute()
	if err != nil {
		diags.AddError("Unable to delete crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
	}
	return diags
}

func callCrawlerScheduleUpdateAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	if schedule.Id.IsUnknown() || schedule.Id.IsNull() {
		diags.AddAttributeError(
			path.Root("id"),
			"Missing schedule.id attribute",
			"To update schedule information the schedule id must be provided",
		)
		return
	}

	if schedule.Project.IsNull() || schedule.Project.IsUnknown() {
		diags.AddAttributeError(
			path.Root("project"),
			"Missing schedule.project attribute",
			"To update schedule information the schedule project must be provided",
		)
		return
	}

	req := *openapi.NewCrawlerScheduleRequestUpdateWithDefaults()
	req.SetScheduleCronString(schedule.ScheduleCronString.ValueString())

	scheduleId := strconv.FormatInt(schedule.Id.ValueInt64(), 10)
	api, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesUpdate(ctx, r.client.Organization, schedule.Project.ValueString(), schedule.Crawler.ValueString(), scheduleId).CrawlerScheduleRequestUpdate(req).Execute()
	if err != nil {
		diags.AddError("Unable to update crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	schedule.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	return diags
}
