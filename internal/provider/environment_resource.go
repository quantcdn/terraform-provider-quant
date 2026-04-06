package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_environment"
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
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
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

	// ComposeDefinition — build the SDK Compose struct field-by-field from
	// the typed Terraform value to avoid json.Marshal on Terraform internal types.
	if !data.ComposeDefinition.IsNull() && !data.ComposeDefinition.IsUnknown() {
		cd := data.ComposeDefinition
		cf := composeFields{
			Architecture:             cd.Architecture,
			Containers:               cd.Containers,
			EnableCrossAppNetworking: cd.EnableCrossAppNetworking,
			EnableCrossEnvNetworking: cd.EnableCrossEnvNetworking,
			MaxCapacity:              cd.MaxCapacity,
			MinCapacity:              cd.MinCapacity,
			SpotConfiguration:        cd.SpotConfiguration,
			TaskCpu:                  cd.TaskCpu,
			TaskMemory:               cd.TaskMemory,
		}
		compose, d := buildSDKCompose(ctx, cf)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		sdkReq.ComposeDefinition = &compose
	}

	// SpotConfiguration — build from typed Terraform value.
	if !data.SpotConfiguration.IsNull() && !data.SpotConfiguration.IsUnknown() {
		if !data.SpotConfiguration.Strategy.IsNull() && !data.SpotConfiguration.Strategy.IsUnknown() {
			spot := quantadmingo.NewSpotConfiguration(data.SpotConfiguration.Strategy.ValueString())
			sdkReq.SpotConfiguration = spot
		}
	}

	// Environment variables — now a types.List of nested EnvironmentValue objects.
	if !data.Environment.IsNull() && !data.Environment.IsUnknown() {
		var envValues []resource_environment.EnvironmentValue
		diags.Append(data.Environment.ElementsAs(ctx, &envValues, false)...)
		if diags.HasError() {
			return
		}
		var envVars []quantadmingo.CreateEnvironmentRequestEnvironmentInner
		for _, ev := range envValues {
			item := quantadmingo.NewCreateEnvironmentRequestEnvironmentInner()
			if !ev.Name.IsNull() && !ev.Name.IsUnknown() {
				v := ev.Name.ValueString()
				item.Name = &v
			}
			if !ev.Value.IsNull() && !ev.Value.IsUnknown() {
				v := ev.Value.ValueString()
				item.Value = &v
			}
			envVars = append(envVars, *item)
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
	data.Organisation = types.StringValue(org)

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

	// --- Scalar fields ---
	data.EnvName = types.StringValue(env.GetEnvName())
	data.Organisation = types.StringValue(org)

	if env.HasStatus() {
		data.Status = types.StringValue(env.GetStatus())
	} else {
		data.Status = types.StringNull()
	}

	if env.HasRunningCount() {
		data.RunningCount = types.Int64Value(int64(env.GetRunningCount()))
	} else {
		data.RunningCount = types.Int64Null()
	}

	if env.HasDesiredCount() {
		data.DesiredCount = types.Int64Value(int64(env.GetDesiredCount()))
	} else {
		data.DesiredCount = types.Int64Null()
	}

	if env.HasMinCapacity() {
		data.MinCapacity = types.Int64Value(int64(env.GetMinCapacity()))
	} else {
		data.MinCapacity = types.Int64Null()
	}

	if env.HasMaxCapacity() {
		data.MaxCapacity = types.Int64Value(int64(env.GetMaxCapacity()))
	} else {
		data.MaxCapacity = types.Int64Null()
	}

	if env.HasDeploymentStatus() {
		data.DeploymentStatus = types.StringValue(env.GetDeploymentStatus())
	} else {
		data.DeploymentStatus = types.StringNull()
	}

	if env.HasDeploymentFailureType() {
		data.DeploymentFailureType = types.StringValue(env.GetDeploymentFailureType())
	} else {
		data.DeploymentFailureType = types.StringNull()
	}

	if env.HasDeploymentFailureReason() {
		data.DeploymentFailureReason = types.StringValue(env.GetDeploymentFailureReason())
	} else {
		data.DeploymentFailureReason = types.StringNull()
	}

	if env.HasPublicIpAddress() {
		data.PublicIpAddress = types.StringValue(env.GetPublicIpAddress())
	} else {
		data.PublicIpAddress = types.StringNull()
	}

	data.CreatedAt = optionalTime(env.GetCreatedAtOk())
	data.UpdatedAt = optionalTime(env.GetUpdatedAtOk())

	// --- Volumes ---
	if env.HasVolumes() {
		volList, d := buildEnvVolumesListFromSDK(ctx, env.GetVolumes())
		diags.Append(d...)
		if !diags.HasError() {
			data.Volumes = volList
		}
	} else {
		data.Volumes = types.ListNull(resource_environment.VolumesValue{}.Type(ctx))
	}

	// --- Cron ---
	if env.HasCron() {
		cronList, d := buildEnvCronListFromSDK(ctx, env.GetCron())
		diags.Append(d...)
		if !diags.HasError() {
			data.Cron = cronList
		}
	} else {
		data.Cron = types.ListNull(resource_environment.CronValue{}.Type(ctx))
	}

	// --- ContainerNames ---
	// EnvironmentResponse returns containers as []map[string]interface{};
	// extract "name" from each for the containerNames list.
	if env.HasContainers() {
		sdkContainers := env.GetContainers()
		names := make([]string, 0, len(sdkContainers))
		for _, c := range sdkContainers {
			if name, ok := c["name"]; ok {
				if nameStr, ok := name.(string); ok && nameStr != "" {
					names = append(names, nameStr)
				}
			}
		}
		if len(names) > 0 {
			namesList, d := types.ListValueFrom(ctx, types.StringType, names)
			diags.Append(d...)
			data.ContainerNames = namesList
		} else {
			data.ContainerNames = types.ListNull(types.StringType)
		}
	} else {
		data.ContainerNames = types.ListNull(types.StringType)
	}

	// --- ComposeDefinition ---
	// The EnvironmentResponse SDK model does not include composeDefinition.
	// Preserve user-configured state if known; null out unknowns.
	if data.ComposeDefinition.IsUnknown() {
		data.ComposeDefinition = resource_environment.NewComposeDefinitionValueNull()
	}

	// --- SpotConfiguration ---
	// The EnvironmentResponse SDK model does not include spotConfiguration.
	// Preserve user-configured state if known; null out unknowns.
	if data.SpotConfiguration.IsUnknown() {
		data.SpotConfiguration = resource_environment.NewSpotConfigurationValueNull()
	}

	// --- Environment (env vars) ---
	// The EnvironmentResponse SDK model does not include environment variables.
	// Preserve user-configured state if known; null out unknowns.
	// Build a canonical EnvironmentValue to get the correct DocumentsType
	// (zero-value types would have empty AttrTypes → dynamic element type).
	if data.Environment.IsUnknown() || data.Environment.IsNull() || data.Environment.ElementType(ctx) == nil {
		envAttrTypes := resource_environment.EnvironmentValue{}.AttributeTypes(ctx)
		canonicalEnv, _ := resource_environment.NewEnvironmentValue(envAttrTypes, map[string]attr.Value{
			"name":  types.StringNull(),
			"value": types.StringNull(),
		})
		data.Environment = types.ListNull(canonicalEnv.Type(ctx))
	}

	// --- Read-only server infrastructure fields ---
	// These are readOnly=true in the OA spec (users never set them).
	// Null them to avoid drift issues with opaque server-side data.
	data.AlbRouting = resource_environment.NewAlbRoutingValueNull()
	data.LoadBalancer = resource_environment.NewLoadBalancerValueNull()
	data.SecurityGroup = resource_environment.NewSecurityGroupValueNull()
	data.Service = resource_environment.NewServiceValueNull()
	data.Subnet = resource_environment.NewSubnetValueNull()
	data.TaskDefinition = resource_environment.NewTaskDefinitionValueNull()
	data.Vpc = resource_environment.NewVpcValueNull()

	// --- Create-only scalars: preserve if known, null out unknowns ---
	if data.CloneConfigurationFrom.IsUnknown() {
		data.CloneConfigurationFrom = types.StringNull()
	}
	if data.ImageSuffix.IsUnknown() {
		data.ImageSuffix = types.StringNull()
	}
	if data.MergeEnvironment.IsUnknown() {
		data.MergeEnvironment = types.BoolNull()
	}
	if data.Application.IsUnknown() {
		data.Application = types.StringNull()
	}

	return
}

func callEnvironmentUpdateAPI(ctx context.Context, r *environmentResource, data *resource_environment.EnvironmentModel) (diags diag.Diagnostics) {
	if data.ComposeDefinition.IsNull() || data.ComposeDefinition.IsUnknown() {
		diags.AddAttributeError(path.Root("compose_definition"), "Missing compose_definition", "Cannot update an environment without a compose_definition.")
		return
	}

	cd := data.ComposeDefinition
	cf := composeFields{
		Architecture:             cd.Architecture,
		Containers:               cd.Containers,
		EnableCrossAppNetworking: cd.EnableCrossAppNetworking,
		EnableCrossEnvNetworking: cd.EnableCrossEnvNetworking,
		MaxCapacity:              cd.MaxCapacity,
		MinCapacity:              cd.MinCapacity,
		SpotConfiguration:        cd.SpotConfiguration,
		TaskCpu:                  cd.TaskCpu,
		TaskMemory:               cd.TaskMemory,
	}
	compose, d := buildSDKCompose(ctx, cf)
	diags.Append(d...)
	if diags.HasError() {
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
