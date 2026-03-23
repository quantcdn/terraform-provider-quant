package provider

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/mapper"
	"terraform-provider-quant/internal/resource_crawler_schedule"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerScheduleCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerScheduleReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var state resource_crawler_schedule.CrawlerScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	// Preserve computed ID from state.
	data.Id = state.Id

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerScheduleUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *crawlerScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_crawler_schedule.CrawlerScheduleModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCrawlerScheduleDeleteAPI(ctx, r, &data)...)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------
func callCrawlerScheduleCreateAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	req := quantadmingo.NewV2CrawlerScheduleRequest(
		schedule.ScheduleCronString.ValueString(),
	)
	name := schedule.Name.ValueString()
	req.Name = &name

	api, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesAdd(
		r.client.AuthContext, r.client.Organization,
		schedule.Project.ValueString(), schedule.Crawler.ValueString(),
	).V2CrawlerScheduleRequest(*req).Execute()

	if err != nil {
		diags.AddError("Unable to create crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	schedule.Id = types.Int64Value(int64(api.GetId()))

	return callCrawlerScheduleReadAPI(ctx, r, schedule)
}

// ---------------------------------------------------------------------------
// Read — uses mapper.FromSDK for simple fields, manual for timestamps.
// ---------------------------------------------------------------------------
func callCrawlerScheduleReadAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	if schedule.Id.IsUnknown() || schedule.Id.IsNull() {
		diags.AddAttributeError(path.Root("id"), "Missing schedule.id attribute",
			"To read schedule information, id must be provided.")
		return
	}
	if schedule.Project.IsNull() || schedule.Project.IsUnknown() {
		diags.AddAttributeError(path.Root("project"), "Missing schedule.project attribute",
			"To read schedule information, project must be provided.")
		return
	}

	scheduleId := strconv.FormatInt(schedule.Id.ValueInt64(), 10)
	api, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesShow(
		ctx, r.client.Organization,
		schedule.Project.ValueString(), schedule.Crawler.ValueString(), scheduleId,
	).Execute()

	if err != nil {
		diags.AddError("Unable to load crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	// mapper.FromSDK handles: Id, Name, CrawlerConfigId, CrawlerUuid,
	// ProjectId, CrawlerLastRunId, ScheduleCronString.
	diags.Append(mapper.FromSDK(ctx, api, schedule)...)

	// Timestamps — time.Time not supported by mapper.
	schedule.CreatedAt = types.StringValue(api.GetCreatedAt().Format("2006-01-02T15:04:05Z07:00"))
	schedule.UpdatedAt = types.StringValue(api.GetUpdatedAt().Format("2006-01-02T15:04:05Z07:00"))

	// Fields not on the API response object.
	schedule.Organization = types.StringValue(r.client.Organization)
	schedule.CrawlerSchedule = types.StringValue(api.GetScheduleCronString())

	if api.CrawlerUuid != nil {
		schedule.CrawlerUuid = types.StringValue(api.GetCrawlerUuid())
	}

	return
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------
func callCrawlerScheduleDeleteAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	if schedule.Id.IsUnknown() || schedule.Id.IsNull() {
		diags.AddAttributeError(path.Root("id"), "Missing schedule.id attribute",
			"To delete schedule information the schedule id must be provided")
		return
	}
	if schedule.Project.IsNull() || schedule.Project.IsUnknown() {
		diags.AddAttributeError(path.Root("project"), "Missing schedule.project attribute",
			"To delete schedule information the schedule project must be provided")
		return
	}

	scheduleId := strconv.FormatInt(schedule.Id.ValueInt64(), 10)
	_, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesDelete(
		ctx, r.client.Organization,
		schedule.Project.ValueString(), schedule.Crawler.ValueString(), scheduleId,
	).Execute()

	if err != nil {
		diags.AddError("Unable to delete crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
	}
	return
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------
func callCrawlerScheduleUpdateAPI(ctx context.Context, r *crawlerScheduleResource, schedule *resource_crawler_schedule.CrawlerScheduleModel) (diags diag.Diagnostics) {
	if schedule.Id.IsUnknown() || schedule.Id.IsNull() {
		diags.AddAttributeError(path.Root("id"), "Missing schedule.id attribute",
			"To update schedule information the schedule id must be provided")
		return
	}
	if schedule.Project.IsNull() || schedule.Project.IsUnknown() {
		diags.AddAttributeError(path.Root("project"), "Missing schedule.project attribute",
			"To update schedule information the schedule project must be provided")
		return
	}

	req := quantadmingo.NewV2CrawlerScheduleRequest(
		schedule.ScheduleCronString.ValueString(),
	)
	updateName := schedule.Name.ValueString()
	req.Name = &updateName

	scheduleId := strconv.FormatInt(schedule.Id.ValueInt64(), 10)
	_, _, err := r.client.Instance.CrawlerSchedulesAPI.CrawlerSchedulesEdit(
		ctx, r.client.Organization,
		schedule.Project.ValueString(), schedule.Crawler.ValueString(), scheduleId,
	).V2CrawlerScheduleRequest(*req).Execute()

	if err != nil {
		diags.AddError("Unable to update crawler schedule", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	return callCrawlerScheduleReadAPI(ctx, r, schedule)
}
