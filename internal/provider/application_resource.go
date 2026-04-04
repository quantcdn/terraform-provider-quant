package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_application"
)

var (
	_ resource.Resource                = (*applicationResource)(nil)
	_ resource.ResourceWithConfigure   = (*applicationResource)(nil)
	_ resource.ResourceWithImportState = (*applicationResource)(nil)
)

func NewApplicationResource() resource.Resource {
	return &applicationResource{}
}

type applicationResource struct {
	client *client.Client
}

func (r *applicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_application.ApplicationResourceSchema(ctx)
}

func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *applicationResource) getOrg(data *resource_application.ApplicationModel) string {
	if !data.Organisation.IsNull() && !data.Organisation.IsUnknown() {
		return data.Organisation.ValueString()
	}
	return r.client.Organization
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_application.ApplicationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callApplicationCreateAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the created application to populate computed fields
	resp.Diagnostics.Append(callApplicationReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_application.ApplicationModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callApplicationReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The V3 Applications API does not have an Update endpoint.
	// If only computed fields changed (drift in server-computed values),
	// refresh state from the API instead of erroring.
	var data resource_application.ApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callApplicationReadAPI(ctx, r, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_application.ApplicationModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(callApplicationDeleteAPI(ctx, r, &data)...)
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "app_name" (organization comes from provider config)
	resource.ImportStatePassthroughID(ctx, path.Root("app_name"), req, resp)
}

// buildCreateApplicationRequest constructs the SDK CreateApplicationRequest from the Terraform model.
func buildCreateApplicationRequest(ctx context.Context, data *resource_application.ApplicationModel) (*quantadmingo.CreateApplicationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	// ComposeDefinition — build the SDK Compose struct field-by-field from
	// the typed Terraform value to avoid json.Marshal on Terraform internal types.
	var compose quantadmingo.Compose
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
		var d diag.Diagnostics
		compose, d = buildSDKCompose(ctx, cf)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
	}

	sdkReq := quantadmingo.NewCreateApplicationRequest(data.AppName.ValueString(), compose)

	// Set optional fields
	if !data.MinCapacity.IsNull() && !data.MinCapacity.IsUnknown() {
		sdkReq.SetMinCapacity(int32(data.MinCapacity.ValueInt64()))
	}
	if !data.MaxCapacity.IsNull() && !data.MaxCapacity.IsUnknown() {
		sdkReq.SetMaxCapacity(int32(data.MaxCapacity.ValueInt64()))
	}

	// Database configuration — now a nested DatabaseValue object.
	if !data.Database.IsNull() && !data.Database.IsUnknown() {
		db := quantadmingo.NewCreateApplicationRequestDatabase()
		hasDatabase := false

		if !data.Database.Engine.IsNull() && !data.Database.Engine.IsUnknown() {
			db.SetEngine(data.Database.Engine.ValueString())
			hasDatabase = true
		}
		if !data.Database.InstanceClass.IsNull() && !data.Database.InstanceClass.IsUnknown() {
			db.SetInstanceClass(data.Database.InstanceClass.ValueString())
			hasDatabase = true
		}
		if !data.Database.StorageGb.IsNull() && !data.Database.StorageGb.IsUnknown() {
			db.SetStorageGb(int32(data.Database.StorageGb.ValueInt64()))
			hasDatabase = true
		}
		if !data.Database.MultiAz.IsNull() && !data.Database.MultiAz.IsUnknown() {
			db.SetMultiAz(data.Database.MultiAz.ValueBool())
			hasDatabase = true
		}

		if hasDatabase {
			sdkReq.SetDatabase(*db)
		}
	}

	// Filesystem configuration — now a nested FilesystemValue object.
	if !data.Filesystem.IsNull() && !data.Filesystem.IsUnknown() {
		fs := quantadmingo.NewCreateApplicationRequestFilesystem()
		hasFilesystem := false

		if !data.Filesystem.Required.IsNull() && !data.Filesystem.Required.IsUnknown() {
			fs.SetRequired(data.Filesystem.Required.ValueBool())
			hasFilesystem = true
		}
		if !data.Filesystem.MountPath.IsNull() && !data.Filesystem.MountPath.IsUnknown() {
			fs.SetMountPath(data.Filesystem.MountPath.ValueString())
			hasFilesystem = true
		}

		if hasFilesystem {
			sdkReq.SetFilesystem(*fs)
		}
	}

	// Environment variables — now a types.List of nested EnvironmentValue objects.
	if !data.Environment.IsNull() && !data.Environment.IsUnknown() {
		var envValues []resource_application.EnvironmentValue
		diags.Append(data.Environment.ElementsAs(ctx, &envValues, false)...)
		if diags.HasError() {
			return nil, diags
		}
		var envVars []quantadmingo.CreateApplicationRequestEnvironmentInner
		for _, ev := range envValues {
			item := quantadmingo.NewCreateApplicationRequestEnvironmentInner()
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
		if len(envVars) > 0 {
			sdkReq.SetEnvironment(envVars)
		}
	}

	return sdkReq, diags
}

// callApplicationCreateAPI creates a new application via the V3 API.
func callApplicationCreateAPI(ctx context.Context, r *applicationResource, data *resource_application.ApplicationModel) (diags diag.Diagnostics) {
	if data.AppName.IsNull() || data.AppName.IsUnknown() {
		diags.AddAttributeError(
			path.Root("app_name"),
			"Missing app_name attribute",
			"Cannot create an application without an app_name.",
		)
		return
	}

	sdkReq, buildDiags := buildCreateApplicationRequest(ctx, data)
	diags.Append(buildDiags...)
	if diags.HasError() {
		return
	}

	org := r.getOrg(data)

	app, resp, err := r.client.Instance.ApplicationsAPI.CreateApplication(r.client.AuthContext, org).CreateApplicationRequest(*sdkReq).Execute()

	if err != nil {
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				diags.AddError(
					"Authentication Failed",
					"Failed to authenticate with the Quant API. Please check your API token is valid.",
				)
				return
			case http.StatusForbidden:
				apiMsg := extractAPIErrorMessage(resp, err)
				diags.AddError("Authorization Failed", apiMsg)
				return
			case http.StatusBadRequest:
				apiMsg := extractAPIErrorMessage(resp, err)
				diags.AddError("Invalid Application Configuration", apiMsg)
				return
			case http.StatusConflict:
				apiMsg := extractAPIErrorMessage(resp, err)
				diags.AddError("Application Already Exists", apiMsg)
				return
			}
		}
		diags.AddError(
			"Unable to Create Application",
			fmt.Sprintf("An unexpected error occurred: %s", err.Error()),
		)
		return
	}

	// Store the app name from the response (should match what was sent)
	data.AppName = types.StringValue(app.GetAppName())

	// Application creation is asynchronous — poll until it becomes available.
	createStateConf := retry.StateChangeConf{
		Pending: []string{"CREATING", "not_found"},
		Target:  []string{"ACTIVE"},
		Refresh: func() (interface{}, string, error) {
			appResult, resp, err := r.client.Instance.ApplicationsAPI.GetApplication(r.client.AuthContext, org, data.AppName.ValueString()).Execute()
			if err != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					return nil, "not_found", nil
				}
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				return nil, "", fmt.Errorf("error checking application status (HTTP %d): %v", statusCode, err)
			}

			status := "ACTIVE"
			if appResult.HasStatus() {
				status = appResult.GetStatus()
			}

			if status == "FAILED" {
				return appResult, status, fmt.Errorf("application creation failed")
			}

			return appResult, status, nil
		},
		Timeout:      15 * time.Minute,
		Delay:        10 * time.Second,
		MinTimeout:   5 * time.Second,
		PollInterval: 15 * time.Second,
	}

	_, err = createStateConf.WaitForStateContext(ctx)
	if err != nil {
		diags.AddError(
			"Application creation timeout",
			fmt.Sprintf("Application was created but did not become ready within the timeout period. This may indicate the application is still being provisioned. Error: %s", err.Error()),
		)
	}

	return
}

// callApplicationReadAPI reads an application from the V3 API and updates the model.
func callApplicationReadAPI(ctx context.Context, r *applicationResource, data *resource_application.ApplicationModel) (diags diag.Diagnostics) {
	if data.AppName.IsNull() || data.AppName.IsUnknown() {
		diags.AddError(
			"Unable to read application",
			"The application app_name is unknown.",
		)
		return
	}

	org := r.getOrg(data)

	app, resp, err := r.client.Instance.ApplicationsAPI.GetApplication(r.client.AuthContext, org, data.AppName.ValueString()).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			diags.AddError(
				"Application Not Found",
				fmt.Sprintf("Application '%s' was not found in organization '%s'.", data.AppName.ValueString(), org),
			)
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
		diags.AddError(
			"Unable to read application",
			fmt.Sprintf("Error: %s", errMsg),
		)
		return
	}

	// --- Scalar fields ---
	data.AppName = types.StringValue(app.GetAppName())
	data.Organisation = types.StringValue(app.GetOrganisation())

	if app.HasStatus() {
		data.Status = types.StringValue(app.GetStatus())
	} else {
		data.Status = types.StringNull()
	}

	if app.HasRunningCount() {
		data.RunningCount = types.Int64Value(int64(app.GetRunningCount()))
	} else {
		data.RunningCount = types.Int64Null()
	}

	if app.HasDesiredCount() {
		data.DesiredCount = types.Int64Value(int64(app.GetDesiredCount()))
	} else {
		data.DesiredCount = types.Int64Null()
	}

	if app.HasMinCapacity() {
		data.MinCapacity = types.Int64Value(int64(app.GetMinCapacity()))
	} else {
		data.MinCapacity = types.Int64Null()
	}

	if app.HasMaxCapacity() {
		data.MaxCapacity = types.Int64Value(int64(app.GetMaxCapacity()))
	} else {
		data.MaxCapacity = types.Int64Null()
	}

	// --- ContainerNames ---
	if app.HasContainerNames() {
		names := app.GetContainerNames()
		namesList, d := types.ListValueFrom(ctx, types.StringType, names)
		diags.Append(d...)
		data.ContainerNames = namesList
	} else {
		data.ContainerNames = types.ListNull(types.StringType)
	}

	// --- ComposeDefinition ---
	if app.HasComposeDefinition() {
		compose := app.GetComposeDefinition()
		cd, d := buildAppComposeDefinitionValue(ctx, &compose)
		diags.Append(d...)
		if !diags.HasError() {
			data.ComposeDefinition = cd
		}
	} else {
		data.ComposeDefinition = resource_application.NewComposeDefinitionValueNull()
	}

	// --- Database ---
	if dbPtr, ok := app.GetDatabaseOk(); ok && dbPtr != nil {
		db, d := buildAppDatabaseValue(ctx, dbPtr)
		diags.Append(d...)
		if !diags.HasError() {
			data.Database = db
		}
	} else {
		data.Database = resource_application.NewDatabaseValueNull()
	}

	// --- Filesystem ---
	if fsPtr, ok := app.GetFilesystemOk(); ok && fsPtr != nil {
		fs, d := buildAppFilesystemValue(ctx, fsPtr)
		diags.Append(d...)
		if !diags.HasError() {
			data.Filesystem = fs
		}
	} else {
		data.Filesystem = resource_application.NewFilesystemValueNull()
	}

	// --- ImageReference ---
	if irPtr, ok := app.GetImageReferenceOk(); ok && irPtr != nil {
		ir, d := buildAppImageReferenceValue(ctx, irPtr)
		diags.Append(d...)
		if !diags.HasError() {
			data.ImageReference = ir
		}
	} else {
		data.ImageReference = resource_application.NewImageReferenceValueNull()
	}

	// --- DeploymentInformation ---
	if app.HasDeploymentInformation() {
		depList, d := buildAppDeploymentInformationList(ctx, app.GetDeploymentInformation())
		diags.Append(d...)
		if !diags.HasError() {
			data.DeploymentInformation = depList
		}
	} else {
		data.DeploymentInformation = types.ListNull(resource_application.DeploymentInformationValue{}.Type(ctx))
	}

	// --- EnvironmentNames ---
	// The API returns environments as a list of objects with envName; extract names.
	if app.HasEnvironments() {
		envs := app.GetEnvironments()
		names := make([]string, 0, len(envs))
		for _, e := range envs {
			if n, ok := e.GetEnvNameOk(); ok && n != nil {
				names = append(names, *n)
			}
		}
		namesList, d := types.ListValueFrom(ctx, types.StringType, names)
		diags.Append(d...)
		data.EnvironmentNames = namesList
	} else {
		data.EnvironmentNames = types.ListNull(types.StringType)
	}

	// --- Environment (env vars) ---
	// The Application GET response does not return environment variables.
	// Preserve current state if known; otherwise null.
	if data.Environment.IsUnknown() {
		data.Environment = types.ListNull(resource_application.EnvironmentValue{}.Type(ctx))
	}

	// --- Application (self-link / computed ID) ---
	if data.Application.IsUnknown() {
		data.Application = types.StringNull()
	}

	return
}

// nullifyUnknownAttrValue recursively replaces unknown attr.Value instances
// with their null equivalents.  For objects (basetypes.ObjectValue) it descends
// into child attributes so that deeply nested unknowns are also resolved.  For
// lists (basetypes.ListValue) it processes each element.
func nullifyUnknownAttrValue(ctx context.Context, v attr.Value) attr.Value {
	if v.IsUnknown() {
		// Replace with null of the same type.
		switch tv := v.(type) {
		case basetypes.StringValue:
			return types.StringNull()
		case basetypes.Int64Value:
			return types.Int64Null()
		case basetypes.BoolValue:
			return types.BoolNull()
		case basetypes.Float64Value:
			return types.Float64Null()
		case basetypes.ObjectValue:
			return types.ObjectNull(tv.AttributeTypes(ctx))
		case basetypes.ListValue:
			return types.ListNull(tv.ElementType(ctx))
		default:
			// For custom types implementing ObjectValuable, try to get attr types.
			if ov, ok := v.(basetypes.ObjectValuable); ok {
				objVal, diags := ov.ToObjectValue(ctx)
				if !diags.HasError() {
					return types.ObjectNull(objVal.AttributeTypes(ctx))
				}
			}
			return v
		}
	}

	// Not unknown — recurse into objects and lists to resolve nested unknowns.
	switch tv := v.(type) {
	case basetypes.ObjectValue:
		attrs := tv.Attributes()
		resolved := make(map[string]attr.Value, len(attrs))
		for k, child := range attrs {
			resolved[k] = nullifyUnknownAttrValue(ctx, child)
		}
		obj, _ := types.ObjectValue(tv.AttributeTypes(ctx), resolved)
		return obj
	case basetypes.ListValue:
		elems := tv.Elements()
		if len(elems) == 0 {
			return v
		}
		resolved := make([]attr.Value, len(elems))
		for i, elem := range elems {
			// Custom types (ContainersValue, etc.) implement ObjectValuable
			// but aren't basetypes.ObjectValue. Convert first, then recurse.
			if _, isObj := elem.(basetypes.ObjectValue); !isObj {
				if ov, ok := elem.(basetypes.ObjectValuable); ok {
					objVal, d := ov.ToObjectValue(ctx)
					if !d.HasError() {
						resolved[i] = nullifyUnknownAttrValue(ctx, objVal)
						continue
					}
				}
			}
			resolved[i] = nullifyUnknownAttrValue(ctx, elem)
		}
		list, _ := types.ListValue(tv.ElementType(ctx), resolved)
		return list
	}

	return v
}

// resolveApplicationUnknowns walks the Database, Filesystem, and
// ComposeDefinition nested objects and replaces any unknown sub-fields with
// null values.  This prevents the Pulumi bridge from panicking when it
// encounters unknown PropertyValue values in the state.
func resolveApplicationUnknowns(ctx context.Context, data *resource_application.ApplicationModel) {
	// --- Database ---
	if !data.Database.IsNull() && !data.Database.IsUnknown() {
		db := data.Database
		attrTypes := db.AttributeTypes(ctx)
		attrs := map[string]attr.Value{
			"engine":                  nullifyUnknownAttrValue(ctx, db.Engine),
			"instance_class":          nullifyUnknownAttrValue(ctx, db.InstanceClass),
			"multi_az":                nullifyUnknownAttrValue(ctx, db.MultiAz),
			"rds_instance_endpoint":   nullifyUnknownAttrValue(ctx, db.RdsInstanceEndpoint),
			"rds_instance_engine":     nullifyUnknownAttrValue(ctx, db.RdsInstanceEngine),
			"rds_instance_identifier": nullifyUnknownAttrValue(ctx, db.RdsInstanceIdentifier),
			"rds_instance_status":     nullifyUnknownAttrValue(ctx, db.RdsInstanceStatus),
			"storage_gb":              nullifyUnknownAttrValue(ctx, db.StorageGb),
		}
		newDB, diags := resource_application.NewDatabaseValue(attrTypes, attrs)
		if !diags.HasError() {
			data.Database = newDB
		}
	}

	// --- Filesystem ---
	if !data.Filesystem.IsNull() && !data.Filesystem.IsUnknown() {
		fs := data.Filesystem
		attrTypes := fs.AttributeTypes(ctx)
		attrs := map[string]attr.Value{
			"filesystem_id": nullifyUnknownAttrValue(ctx, fs.FilesystemId),
			"mount_path":    nullifyUnknownAttrValue(ctx, fs.MountPath),
			"required":      nullifyUnknownAttrValue(ctx, fs.Required),
		}
		newFS, diags := resource_application.NewFilesystemValue(attrTypes, attrs)
		if !diags.HasError() {
			data.Filesystem = newFS
		}
	}

	// --- ComposeDefinition ---
	if !data.ComposeDefinition.IsNull() && !data.ComposeDefinition.IsUnknown() {
		cd := data.ComposeDefinition

		// Resolve containers list: each container element may have unknown
		// sub-fields (e.g. health_check, origin_protection_config).
		containers := cd.Containers
		if !containers.IsNull() && !containers.IsUnknown() {
			resolved := nullifyUnknownAttrValue(ctx, containers)
			if lv, ok := resolved.(basetypes.ListValue); ok {
				containers = lv
			}
		}

		// Resolve spot_configuration sub-fields.
		spotConfig := cd.SpotConfiguration
		if !spotConfig.IsNull() && !spotConfig.IsUnknown() {
			resolved := nullifyUnknownAttrValue(ctx, spotConfig)
			if ov, ok := resolved.(basetypes.ObjectValue); ok {
				spotConfig = ov
			}
		}

		attrTypes := cd.AttributeTypes(ctx)
		attrs := map[string]attr.Value{
			"architecture":               nullifyUnknownAttrValue(ctx, cd.Architecture),
			"containers":                 containers,
			"enable_cross_app_networking": nullifyUnknownAttrValue(ctx, cd.EnableCrossAppNetworking),
			"enable_cross_env_networking": nullifyUnknownAttrValue(ctx, cd.EnableCrossEnvNetworking),
			"max_capacity":               nullifyUnknownAttrValue(ctx, cd.MaxCapacity),
			"min_capacity":               nullifyUnknownAttrValue(ctx, cd.MinCapacity),
			"spot_configuration":         spotConfig,
			"task_cpu":                    nullifyUnknownAttrValue(ctx, cd.TaskCpu),
			"task_memory":                 nullifyUnknownAttrValue(ctx, cd.TaskMemory),
		}
		newCD, diags := resource_application.NewComposeDefinitionValue(attrTypes, attrs)
		if !diags.HasError() {
			data.ComposeDefinition = newCD
		}
	}

	// --- ImageReference ---
	if !data.ImageReference.IsNull() && !data.ImageReference.IsUnknown() {
		ir := data.ImageReference
		attrTypes := ir.AttributeTypes(ctx)
		attrs := map[string]attr.Value{
			"identifier": nullifyUnknownAttrValue(ctx, ir.Identifier),
			"type":       nullifyUnknownAttrValue(ctx, ir.ImageReferenceType),
		}
		newIR, diags := resource_application.NewImageReferenceValue(attrTypes, attrs)
		if !diags.HasError() {
			data.ImageReference = newIR
		}
	}
}

// callApplicationDeleteAPI deletes an application via the V3 API.
func callApplicationDeleteAPI(ctx context.Context, r *applicationResource, data *resource_application.ApplicationModel) (diags diag.Diagnostics) {
	if data.AppName.IsNull() || data.AppName.IsUnknown() {
		diags.AddAttributeError(
			path.Root("app_name"),
			"Missing app_name attribute",
			"To delete an application the app_name must be known.",
		)
		return
	}

	org := r.getOrg(data)

	resp, err := r.client.Instance.ApplicationsAPI.DeleteApplication(r.client.AuthContext, org, data.AppName.ValueString()).Execute()

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Already deleted — not an error
			return
		}
		if resp != nil && resp.StatusCode == http.StatusBadRequest {
			diags.AddError("Unable to delete application", extractAPIErrorMessage(resp, err))
			return
		}
		diags.AddError(
			"Unable to delete application",
			fmt.Sprintf("Error: %s", err.Error()),
		)
		return
	}

	return
}
