package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"terraform-provider-quant/internal/client"
	"terraform-provider-quant/internal/resource_cron_job"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*cronJobResource)(nil)
	_ resource.ResourceWithConfigure   = (*cronJobResource)(nil)
	_ resource.ResourceWithImportState = (*cronJobResource)(nil)
)

func NewCronJobResource() resource.Resource {
	return &cronJobResource{}
}

type cronJobResource struct {
	client *client.Client
}

func (r *cronJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cron_job"
}

func (r *cronJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_cron_job.CronJobResourceSchema(ctx)
}

func (r *cronJobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *cronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_cron_job.CronJobModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCronJobCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_cron_job.CronJobModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCronJobReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_cron_job.CronJobModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCronJobUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCronJobReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_cron_job.CronJobModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callCronJobDeleteAPI(ctx, r, &data)...)
}

func (r *cronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "application/environment/cron_name"
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'application/environment/cron_name'.",
		)
		return
	}

	var data resource_cron_job.CronJobModel
	data.Application = types.StringValue(parts[0])
	data.Environment = types.StringValue(parts[1])
	data.Name = types.StringValue(parts[2])

	resp.Diagnostics.Append(callCronJobReadAPI(ctx, &cronJobResource{client: r.client}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *cronJobResource) getOrg(data *resource_cron_job.CronJobModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func callCronJobCreateAPI(ctx context.Context, r *cronJobResource, data *resource_cron_job.CronJobModel) (diags diag.Diagnostics) {
	if data.Name.IsNull() || data.Name.IsUnknown() {
		diags.AddAttributeError(path.Root("name"), "Missing name", "Cannot create a cron job without a name.")
		return
	}

	// Parse command JSON array
	var command []string
	if err := json.Unmarshal([]byte(data.Command.ValueString()), &command); err != nil {
		diags.AddAttributeError(path.Root("command"), "Invalid command JSON", fmt.Sprintf("command must be a JSON array of strings: %s", err.Error()))
		return
	}

	sdkReq := quantadmingo.NewCreateCronJobRequest(data.Name.ValueString(), data.ScheduleExpression.ValueString(), command)

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		sdkReq.Description = *quantadmingo.NewNullableString(strPtr(data.Description.ValueString()))
	}
	if !data.TargetContainerName.IsNull() && !data.TargetContainerName.IsUnknown() {
		sdkReq.TargetContainerName = *quantadmingo.NewNullableString(strPtr(data.TargetContainerName.ValueString()))
	}
	if !data.IsEnabled.IsNull() && !data.IsEnabled.IsUnknown() {
		v := data.IsEnabled.ValueBool()
		sdkReq.IsEnabled = *quantadmingo.NewNullableBool(&v)
	}

	org := r.getOrg(data)
	cron, resp, err := r.client.Instance.CronAPI.CreateCronJob(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString()).CreateCronJobRequest(*sdkReq).Execute()

	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				diags.AddError("Authentication Failed", "Please check your API token.")
				return
			case http.StatusForbidden:
				diags.AddError("Authorization Failed", extractAPIErrorMessage(resp, err))
				return
			case http.StatusBadRequest:
				diags.AddError("Invalid Cron Job Configuration", extractAPIErrorMessage(resp, err))
				return
			}
		}
		diags.AddError("Unable to Create Cron Job", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	mapCronResponse(cron, data)

	return
}

func callCronJobReadAPI(ctx context.Context, r *cronJobResource, data *resource_cron_job.CronJobModel) (diags diag.Diagnostics) {
	if data.Name.IsNull() || data.Name.IsUnknown() {
		diags.AddError("Unable to read cron job", "The name is unknown.")
		return
	}

	org := r.getOrg(data)
	cron, resp, err := r.client.Instance.CronAPI.GetCronJob(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString(), data.Name.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			diags.AddError("Cron Job Not Found", fmt.Sprintf("Cron job '%s' not found.", data.Name.ValueString()))
			return
		}
		var errMsg string
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				errMsg = string(body)
			}
		}
		if errMsg == "" {
			errMsg = err.Error()
		}
		diags.AddError("Unable to read cron job", fmt.Sprintf("Error: %s", errMsg))
		return
	}

	mapCronResponse(cron, data)

	return
}

func callCronJobUpdateAPI(ctx context.Context, r *cronJobResource, data *resource_cron_job.CronJobModel) (diags diag.Diagnostics) {
	sdkReq := quantadmingo.NewUpdateCronJobRequest()

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		sdkReq.Description = *quantadmingo.NewNullableString(strPtr(data.Description.ValueString()))
	}
	if !data.ScheduleExpression.IsNull() && !data.ScheduleExpression.IsUnknown() {
		sdkReq.ScheduleExpression = *quantadmingo.NewNullableString(strPtr(data.ScheduleExpression.ValueString()))
	}
	if !data.TargetContainerName.IsNull() && !data.TargetContainerName.IsUnknown() {
		sdkReq.TargetContainerName = *quantadmingo.NewNullableString(strPtr(data.TargetContainerName.ValueString()))
	}
	if !data.IsEnabled.IsNull() && !data.IsEnabled.IsUnknown() {
		v := data.IsEnabled.ValueBool()
		sdkReq.IsEnabled = *quantadmingo.NewNullableBool(&v)
	}

	// Parse command JSON array
	if !data.Command.IsNull() && !data.Command.IsUnknown() {
		var command []string
		if err := json.Unmarshal([]byte(data.Command.ValueString()), &command); err != nil {
			diags.AddAttributeError(path.Root("command"), "Invalid command JSON", err.Error())
			return
		}
		sdkReq.Command = command
	}

	org := r.getOrg(data)
	_, resp, err := r.client.Instance.CronAPI.UpdateCronJob(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString(), data.Name.ValueString()).UpdateCronJobRequest(*sdkReq).Execute()

	if err != nil {
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				diags.AddError("Unable to update cron job", string(body))
				return
			}
		}
		diags.AddError("Unable to update cron job", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func callCronJobDeleteAPI(ctx context.Context, r *cronJobResource, data *resource_cron_job.CronJobModel) (diags diag.Diagnostics) {
	if data.Name.IsNull() || data.Name.IsUnknown() {
		diags.AddAttributeError(path.Root("name"), "Missing name", "To delete a cron job the name must be known.")
		return
	}

	org := r.getOrg(data)
	resp, err := r.client.Instance.CronAPI.DeleteCronJob(r.client.AuthContext, org, data.Application.ValueString(), data.Environment.ValueString(), data.Name.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		diags.AddError("Unable to delete cron job", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func mapCronResponse(cron *quantadmingo.Cron, data *resource_cron_job.CronJobModel) {
	if cron.Name != nil {
		data.Name = types.StringValue(*cron.Name)
	}
	if cron.Schedule != nil {
		data.Schedule = types.StringValue(*cron.Schedule)
	} else {
		data.Schedule = types.StringNull()
	}
	if cron.Command != nil {
		data.Command = types.StringValue(*cron.Command)
	}

	// Resolve unknown optional/computed fields to null
	if data.Description.IsUnknown() {
		data.Description = types.StringNull()
	}
	if data.TargetContainerName.IsUnknown() {
		data.TargetContainerName = types.StringNull()
	}
	if data.ScheduleExpression.IsUnknown() {
		data.ScheduleExpression = types.StringNull()
	}
}
