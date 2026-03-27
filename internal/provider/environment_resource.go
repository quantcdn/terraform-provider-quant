package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"github.com/quantcdn/terraform-provider-quant/internal/client"
	"github.com/quantcdn/terraform-provider-quant/internal/resource_environment"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

var (
	_ resource.Resource                = (*environmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*environmentResource)(nil)
	_ resource.ResourceWithImportState = (*environmentResource)(nil)
)

func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

type environmentResource struct {
	client *client.Client
}

func (r *environmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *environmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_environment.EnvironmentResourceSchema(ctx)
}

func (r *environmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_environment.EnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callEnvironmentCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callEnvironmentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_environment.EnvironmentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callEnvironmentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_environment.EnvironmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callEnvironmentUpdateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callEnvironmentReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_environment.EnvironmentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callEnvironmentDeleteAPI(ctx, r, &data)...)
}

func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "application/env_name" (organization from provider config)
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'application/env_name'.",
		)
		return
	}

	var data resource_environment.EnvironmentModel
	data.Application = types.StringValue(parts[0])
	data.EnvName = types.StringValue(parts[1])

	resp.Diagnostics.Append(callEnvironmentReadAPI(ctx, &environmentResource{client: r.client}, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *environmentResource) getOrg(data *resource_environment.EnvironmentModel) string {
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		return data.Organization.ValueString()
	}
	return r.client.Organization
}

func callEnvironmentCreateAPI(ctx context.Context, r *environmentResource, data *resource_environment.EnvironmentModel) (diags diag.Diagnostics) {
	if data.EnvName.IsNull() || data.EnvName.IsUnknown() {
		diags.AddAttributeError(path.Root("env_name"), "Missing env_name", "Cannot create an environment without env_name.")
		return
	}
	if data.Application.IsNull() || data.Application.IsUnknown() {
		diags.AddAttributeError(path.Root("application"), "Missing application", "Cannot create an environment without application.")
		return
	}

	sdkReq := quantadmingo.NewCreateEnvironmentRequest(data.EnvName.ValueString())

	if !data.MinCapacity.IsNull() && !data.MinCapacity.IsUnknown() {
		v := int32(data.MinCapacity.ValueInt64())
		sdkReq.MinCapacity = &v
	}
	if !data.MaxCapacity.IsNull() && !data.MaxCapacity.IsUnknown() {
		v := int32(data.MaxCapacity.ValueInt64())
		sdkReq.MaxCapacity = &v
	}
	if !data.CloneConfigurationFrom.IsNull() && !data.CloneConfigurationFrom.IsUnknown() {
		v := data.CloneConfigurationFrom.ValueString()
		sdkReq.CloneConfigurationFrom = &v
	}
	if !data.ImageSuffix.IsNull() && !data.ImageSuffix.IsUnknown() {
		v := data.ImageSuffix.ValueString()
		sdkReq.ImageSuffix = &v
	}
	if !data.MergeEnvironment.IsNull() && !data.MergeEnvironment.IsUnknown() {
		v := data.MergeEnvironment.ValueBool()
		sdkReq.MergeEnvironment = &v
	}

	// Parse compose definition if provided
	if !data.ComposeDefinition.IsNull() && !data.ComposeDefinition.IsUnknown() {
		var compose quantadmingo.Compose
		if err := json.Unmarshal([]byte(data.ComposeDefinition.ValueString()), &compose); err != nil {
			diags.AddAttributeError(path.Root("compose_definition"), "Invalid compose_definition JSON", err.Error())
			return
		}
		sdkReq.ComposeDefinition = &compose
	}

	// Parse spot configuration if provided
	if !data.SpotConfiguration.IsNull() && !data.SpotConfiguration.IsUnknown() {
		var spot quantadmingo.SpotConfiguration
		if err := json.Unmarshal([]byte(data.SpotConfiguration.ValueString()), &spot); err != nil {
			diags.AddAttributeError(path.Root("spot_configuration"), "Invalid spot_configuration JSON", err.Error())
			return
		}
		sdkReq.SpotConfiguration = &spot
	}

	// Parse environment variables if provided
	if !data.EnvironmentVariables.IsNull() && !data.EnvironmentVariables.IsUnknown() {
		var envVars []quantadmingo.CreateEnvironmentRequestEnvironmentInner
		if err := json.Unmarshal([]byte(data.EnvironmentVariables.ValueString()), &envVars); err != nil {
			diags.AddAttributeError(path.Root("environment_variables"), "Invalid environment_variables JSON", err.Error())
			return
		}
		sdkReq.Environment = envVars
	}

	org := r.getOrg(data)
	envResp, resp, err := r.client.Instance.EnvironmentsAPI.CreateEnvironment(r.client.AuthContext, org, data.Application.ValueString()).CreateEnvironmentRequest(*sdkReq).Execute()

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
				diags.AddError("Invalid Environment Configuration", extractAPIErrorMessage(resp, err))
				return
			}
		}
		diags.AddError("Unable to Create Environment", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	data.EnvName = types.StringValue(envResp.GetEnvName())
	data.Organization = types.StringValue(org)

	// Poll until environment is ready
	createStateConf := retry.StateChangeConf{
		Pending: []string{"CREATING", "not_found"},
		Target:  []string{"ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			result, resp, err := r.client.Instance.EnvironmentsAPI.GetEnvironment(r.client.AuthContext, org, data.Application.ValueString(), data.EnvName.ValueString()).Execute()
			if err != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					return nil, "not_found", nil
				}
				return nil, "", fmt.Errorf("error checking environment status: %v", err)
			}

			status := "ACTIVE"
			if result.Status != nil {
				status = *result.Status
			}

			if status == "FAILED" {
				return result, status, fmt.Errorf("environment creation failed")
			}

			return result, status, nil
		},
		Timeout:      15 * time.Minute,
		Delay:        10 * time.Second,
		MinTimeout:   5 * time.Second,
		PollInterval: 15 * time.Second,
	}

	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		diags.AddError(
			"Environment creation timeout",
			fmt.Sprintf("Environment was created but did not become ready within the timeout period. Error: %s", err.Error()),
		)
	}

	return
}

func callEnvironmentReadAPI(ctx context.Context, r *environmentResource, data *resource_environment.EnvironmentModel) (diags diag.Diagnostics) {
	if data.EnvName.IsNull() || data.EnvName.IsUnknown() {
		diags.AddError("Unable to read environment", "The env_name is unknown.")
		return
	}
	if data.Application.IsNull() || data.Application.IsUnknown() {
		diags.AddError("Unable to read environment", "The application is unknown.")
		return
	}

	org := r.getOrg(data)
	env, resp, err := r.client.Instance.EnvironmentsAPI.GetEnvironment(r.client.AuthContext, org, data.Application.ValueString(), data.EnvName.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			diags.AddError("Environment Not Found", fmt.Sprintf("Environment '%s' not found.", data.EnvName.ValueString()))
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
		diags.AddError("Unable to read environment", fmt.Sprintf("Error: %s", errMsg))
		return
	}

	data.EnvName = types.StringValue(env.GetEnvName())
	data.Organization = types.StringValue(org)

	if env.Status != nil {
		data.Status = types.StringValue(*env.Status)
	} else {
		data.Status = types.StringNull()
	}

	if env.RunningCount != nil {
		data.RunningCount = types.Int64Value(int64(*env.RunningCount))
	} else {
		data.RunningCount = types.Int64Null()
	}

	if env.DesiredCount != nil {
		data.DesiredCount = types.Int64Value(int64(*env.DesiredCount))
	} else {
		data.DesiredCount = types.Int64Null()
	}

	if env.MinCapacity != nil {
		data.MinCapacity = types.Int64Value(int64(*env.MinCapacity))
	}

	if env.MaxCapacity != nil {
		data.MaxCapacity = types.Int64Value(int64(*env.MaxCapacity))
	}

	if env.DeploymentStatus != nil {
		data.DeploymentStatus = types.StringValue(*env.DeploymentStatus)
	} else {
		data.DeploymentStatus = types.StringNull()
	}

	if env.CreatedAt != nil {
		data.CreatedAt = types.StringValue(env.CreatedAt.Format(time.RFC3339))
	} else {
		data.CreatedAt = types.StringNull()
	}

	if env.UpdatedAt != nil {
		data.UpdatedAt = types.StringValue(env.UpdatedAt.Format(time.RFC3339))
	} else {
		data.UpdatedAt = types.StringNull()
	}

	// Resolve unknown optional/computed fields to null
	if data.CloneConfigurationFrom.IsUnknown() {
		data.CloneConfigurationFrom = types.StringNull()
	}
	if data.ImageSuffix.IsUnknown() {
		data.ImageSuffix = types.StringNull()
	}
	if data.SpotConfiguration.IsUnknown() {
		data.SpotConfiguration = types.StringNull()
	}
	if data.EnvironmentVariables.IsUnknown() {
		data.EnvironmentVariables = types.StringNull()
	}
	if data.MergeEnvironment.IsUnknown() {
		data.MergeEnvironment = types.BoolNull()
	}
	if data.ComposeDefinition.IsUnknown() {
		data.ComposeDefinition = types.StringNull()
	}

	return
}

func callEnvironmentUpdateAPI(ctx context.Context, r *environmentResource, data *resource_environment.EnvironmentModel) (diags diag.Diagnostics) {
	if data.ComposeDefinition.IsNull() || data.ComposeDefinition.IsUnknown() {
		diags.AddAttributeError(path.Root("compose_definition"), "Missing compose_definition", "Cannot update an environment without a compose_definition.")
		return
	}

	var compose quantadmingo.Compose
	if err := json.Unmarshal([]byte(data.ComposeDefinition.ValueString()), &compose); err != nil {
		diags.AddAttributeError(path.Root("compose_definition"), "Invalid compose_definition JSON", err.Error())
		return
	}

	sdkReq := quantadmingo.NewUpdateEnvironmentRequest(compose)

	if !data.MinCapacity.IsNull() && !data.MinCapacity.IsUnknown() {
		sdkReq.SetMinCapacity(int32(data.MinCapacity.ValueInt64()))
	}
	if !data.MaxCapacity.IsNull() && !data.MaxCapacity.IsUnknown() {
		sdkReq.SetMaxCapacity(int32(data.MaxCapacity.ValueInt64()))
	}

	org := r.getOrg(data)
	resp, err := r.client.Instance.EnvironmentsAPI.UpdateEnvironment(r.client.AuthContext, org, data.Application.ValueString(), data.EnvName.ValueString()).UpdateEnvironmentRequest(*sdkReq).Execute()

	if err != nil {
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				diags.AddError("Unable to update environment", string(body))
				return
			}
		}
		diags.AddError("Unable to update environment", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}

func callEnvironmentDeleteAPI(ctx context.Context, r *environmentResource, data *resource_environment.EnvironmentModel) (diags diag.Diagnostics) {
	if data.EnvName.IsNull() || data.EnvName.IsUnknown() {
		diags.AddAttributeError(path.Root("env_name"), "Missing env_name", "To delete an environment the env_name must be known.")
		return
	}

	org := r.getOrg(data)
	resp, err := r.client.Instance.EnvironmentsAPI.DeleteEnvironment(r.client.AuthContext, org, data.Application.ValueString(), data.EnvName.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return // Already deleted
		}
		diags.AddError("Unable to delete environment", fmt.Sprintf("Error: %s", err.Error()))
	}

	return
}
