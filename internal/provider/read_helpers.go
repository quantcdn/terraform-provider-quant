package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_application"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_environment"
)

// ---------------------------------------------------------------------------
// Primitive helpers for SDK GetXxxOk patterns
// ---------------------------------------------------------------------------

// optionalString converts an SDK (*string, bool) pair to a Terraform string.
func optionalString(v *string, ok bool) basetypes.StringValue {
	if ok && v != nil && *v != "" {
		return types.StringValue(*v)
	}
	return types.StringNull()
}

// optionalInt32AsInt64 converts an SDK (*int32, bool) pair to a Terraform Int64.
func optionalInt32AsInt64(v *int32, ok bool) basetypes.Int64Value {
	if ok && v != nil {
		return types.Int64Value(int64(*v))
	}
	return types.Int64Null()
}

// optionalBool converts an SDK (*bool, bool) pair to a Terraform Bool.
func optionalBool(v *bool, ok bool) basetypes.BoolValue {
	if ok && v != nil {
		return types.BoolValue(*v)
	}
	return types.BoolNull()
}

// optionalTime formats a time pointer as RFC3339 string or null.
func optionalTime(v *time.Time, ok bool) basetypes.StringValue {
	if ok && v != nil {
		return types.StringValue(v.Format(time.RFC3339))
	}
	return types.StringNull()
}

// ---------------------------------------------------------------------------
// Application: build Terraform model from SDK Application response
// ---------------------------------------------------------------------------

// buildAppComposeDefinitionValue builds a resource_application.ComposeDefinitionValue
// from the SDK Compose response.
func buildAppComposeDefinitionValue(ctx context.Context, compose *quantadmingo.Compose) (resource_application.ComposeDefinitionValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build containers list
	containersList := types.ListNull(resource_application.ContainersValue{}.Type(ctx))
	if compose.HasContainers() {
		sdkContainers := compose.GetContainers()
		containerVals := make([]attr.Value, 0, len(sdkContainers))
		for _, c := range sdkContainers {
			cv, d := buildAppContainerValue(ctx, &c)
			diags.Append(d...)
			if diags.HasError() {
				return resource_application.NewComposeDefinitionValueNull(), diags
			}
			containerVals = append(containerVals, cv)
		}
		var d diag.Diagnostics
		containersList, d = types.ListValueFrom(ctx, resource_application.ContainersValue{}.Type(ctx), containerVals)
		diags.Append(d...)
		if diags.HasError() {
			return resource_application.NewComposeDefinitionValueNull(), diags
		}
	}

	// Build spot configuration
	spotAttrs := map[string]attr.Value{
		"strategy": types.StringNull(),
	}
	if compose.HasSpotConfiguration() {
		spot := compose.GetSpotConfiguration()
		spotAttrs["strategy"] = optionalString(spot.GetStrategyOk())
	}
	spotAttrTypes := resource_application.SpotConfigurationValue{}.AttributeTypes(ctx)
	spotObj, d := types.ObjectValue(spotAttrTypes, spotAttrs)
	diags.Append(d...)
	if diags.HasError() {
		return resource_application.NewComposeDefinitionValueNull(), diags
	}

	attrTypes := resource_application.ComposeDefinitionValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"architecture":               optionalString(compose.GetArchitectureOk()),
		"containers":                 containersList,
		"enable_cross_app_networking": optionalBool(compose.GetEnableCrossAppNetworkingOk()),
		"enable_cross_env_networking": optionalBool(compose.GetEnableCrossEnvNetworkingOk()),
		"max_capacity":               optionalInt32AsInt64(compose.GetMaxCapacityOk()),
		"min_capacity":               optionalInt32AsInt64(compose.GetMinCapacityOk()),
		"spot_configuration":         spotObj,
		"task_cpu":                   optionalInt32AsInt64(compose.GetTaskCpuOk()),
		"task_memory":                optionalInt32AsInt64(compose.GetTaskMemoryOk()),
	}

	result, d := resource_application.NewComposeDefinitionValue(attrTypes, attrs)
	diags.Append(d...)
	return result, diags
}

// buildAppContainerValue builds a single resource_application.ContainersValue
// from an SDK Container.
func buildAppContainerValue(ctx context.Context, c *quantadmingo.Container) (resource_application.ContainersValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// ImageReference (required)
	imgRef := c.GetImageReference()
	irAttrTypes := resource_application.ImageReferenceValue{}.AttributeTypes(ctx)
	irAttrs := map[string]attr.Value{
		"identifier": optionalString(imgRef.GetIdentifierOk()),
		"type":       optionalString(imgRef.GetTypeOk()),
	}
	irObj, d := types.ObjectValue(irAttrTypes, irAttrs)
	diags.Append(d...)
	if diags.HasError() {
		return resource_application.ContainersValue{}, diags
	}

	// HealthCheck
	hcAttrTypes := resource_application.HealthCheckValue{}.AttributeTypes(ctx)
	hcObj := types.ObjectNull(hcAttrTypes)
	if c.HasHealthCheck() {
		hc := c.GetHealthCheck()
		cmdList := types.ListNull(types.StringType)
		if cmds, ok := hc.GetCommandOk(); ok && cmds != nil {
			var d2 diag.Diagnostics
			cmdList, d2 = types.ListValueFrom(ctx, types.StringType, cmds)
			diags.Append(d2...)
		}
		hcAttrs := map[string]attr.Value{
			"command":      cmdList,
			"interval":     optionalInt32AsInt64(hc.GetIntervalOk()),
			"retries":      optionalInt32AsInt64(hc.GetRetriesOk()),
			"start_period": optionalInt32AsInt64(hc.GetStartPeriodOk()),
			"timeout":      optionalInt32AsInt64(hc.GetTimeoutOk()),
		}
		hcObj, d = types.ObjectValue(hcAttrTypes, hcAttrs)
		diags.Append(d...)
	}

	// OriginProtectionConfig
	opcAttrTypes := resource_application.OriginProtectionConfigValue{}.AttributeTypes(ctx)
	opcObj := types.ObjectNull(opcAttrTypes)
	if c.HasOriginProtectionConfig() {
		opc := c.GetOriginProtectionConfig()
		ipList := types.ListNull(types.StringType)
		if ips, ok := opc.GetIpAllowOk(); ok && ips != nil {
			var d2 diag.Diagnostics
			ipList, d2 = types.ListValueFrom(ctx, types.StringType, ips)
			diags.Append(d2...)
		}
		opcAttrs := map[string]attr.Value{
			"enabled":  optionalBool(opc.GetEnabledOk()),
			"ip_allow": ipList,
		}
		opcObj, d = types.ObjectValue(opcAttrTypes, opcAttrs)
		diags.Append(d...)
	}

	// ExposedPorts
	exposedPorts := types.ListNull(types.Int64Type)
	if c.HasExposedPorts() {
		ports := c.GetExposedPorts()
		int64Ports := make([]int64, len(ports))
		for i, p := range ports {
			int64Ports[i] = int64(p)
		}
		exposedPorts, d = types.ListValueFrom(ctx, types.Int64Type, int64Ports)
		diags.Append(d...)
	}

	// Command
	commandList := types.ListNull(types.StringType)
	if c.HasCommand() {
		commandList, d = types.ListValueFrom(ctx, types.StringType, c.GetCommand())
		diags.Append(d...)
	}

	// EntryPoint
	entryPointList := types.ListNull(types.StringType)
	if c.HasEntryPoint() {
		entryPointList, d = types.ListValueFrom(ctx, types.StringType, c.GetEntryPoint())
		diags.Append(d...)
	}

	// Environment (container env vars)
	envList := types.ListNull(resource_application.EnvironmentValue{}.Type(ctx))
	if c.HasEnvironment() {
		sdkEnvs := c.GetEnvironment()
		envVals := make([]attr.Value, 0, len(sdkEnvs))
		envAttrTypes := resource_application.EnvironmentValue{}.AttributeTypes(ctx)
		for _, ev := range sdkEnvs {
			evAttrs := map[string]attr.Value{
				"name":  optionalString(ev.GetNameOk()),
				"value": optionalString(ev.GetValueOk()),
			}
			evVal, d2 := resource_application.NewEnvironmentValue(envAttrTypes, evAttrs)
			diags.Append(d2...)
			envVals = append(envVals, evVal)
		}
		envList, d = types.ListValueFrom(ctx, resource_application.EnvironmentValue{}.Type(ctx), envVals)
		diags.Append(d...)
	}

	// Secrets
	secretsList := types.ListNull(resource_application.SecretsValue{}.Type(ctx))
	if c.HasSecrets() {
		sdkSecrets := c.GetSecrets()
		secVals := make([]attr.Value, 0, len(sdkSecrets))
		secAttrTypes := resource_application.SecretsValue{}.AttributeTypes(ctx)
		for _, s := range sdkSecrets {
			sAttrs := map[string]attr.Value{
				"name":       optionalString(s.GetNameOk()),
				"value_from": optionalString(s.GetValueFromOk()),
			}
			sVal, d2 := resource_application.NewSecretsValue(secAttrTypes, sAttrs)
			diags.Append(d2...)
			secVals = append(secVals, sVal)
		}
		secretsList, d = types.ListValueFrom(ctx, resource_application.SecretsValue{}.Type(ctx), secVals)
		diags.Append(d...)
	}

	// DependsOn
	depsList := types.ListNull(resource_application.DependsOnValue{}.Type(ctx))
	if c.HasDependsOn() {
		sdkDeps := c.GetDependsOn()
		depVals := make([]attr.Value, 0, len(sdkDeps))
		depAttrTypes := resource_application.DependsOnValue{}.AttributeTypes(ctx)
		for _, dep := range sdkDeps {
			depAttrs := map[string]attr.Value{
				"condition":      optionalString(dep.GetConditionOk()),
				"container_name": optionalString(dep.GetContainerNameOk()),
			}
			depVal, d2 := resource_application.NewDependsOnValue(depAttrTypes, depAttrs)
			diags.Append(d2...)
			depVals = append(depVals, depVal)
		}
		depsList, d = types.ListValueFrom(ctx, resource_application.DependsOnValue{}.Type(ctx), depVals)
		diags.Append(d...)
	}

	// MountPoints
	mpList := types.ListNull(resource_application.MountPointsValue{}.Type(ctx))
	if c.HasMountPoints() {
		sdkMPs := c.GetMountPoints()
		mpVals := make([]attr.Value, 0, len(sdkMPs))
		mpAttrTypes := resource_application.MountPointsValue{}.AttributeTypes(ctx)
		for _, mp := range sdkMPs {
			mpAttrs := map[string]attr.Value{
				"container_path": optionalString(mp.GetContainerPathOk()),
				"read_only":      optionalBool(mp.GetReadOnlyOk()),
				"source_volume":  optionalString(mp.GetSourceVolumeOk()),
			}
			mpVal, d2 := resource_application.NewMountPointsValue(mpAttrTypes, mpAttrs)
			diags.Append(d2...)
			mpVals = append(mpVals, mpVal)
		}
		mpList, d = types.ListValueFrom(ctx, resource_application.MountPointsValue{}.Type(ctx), mpVals)
		diags.Append(d...)
	}

	attrTypes := resource_application.ContainersValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"command":                  commandList,
		"cpu":                      optionalInt32AsInt64(c.GetCpuOk()),
		"depends_on":               depsList,
		"entry_point":              entryPointList,
		"environment":              envList,
		"essential":                optionalBool(c.GetEssentialOk()),
		"exposed_ports":            exposedPorts,
		"health_check":             hcObj,
		"image_reference":          irObj,
		"memory":                   optionalInt32AsInt64(c.GetMemoryOk()),
		"memory_reservation":       optionalInt32AsInt64(c.GetMemoryReservationOk()),
		"mount_points":             mpList,
		"name":                     types.StringValue(c.GetName()),
		"origin_protection":        optionalBool(c.GetOriginProtectionOk()),
		"origin_protection_config": opcObj,
		"readonly_root_filesystem": optionalBool(c.GetReadonlyRootFilesystemOk()),
		"secrets":                  secretsList,
		"user":                     optionalString(c.GetUserOk()),
		"working_directory":        optionalString(c.GetWorkingDirectoryOk()),
	}

	result, d := resource_application.NewContainersValue(attrTypes, attrs)
	diags.Append(d...)
	return result, diags
}

// buildAppDatabaseValue builds a resource_application.DatabaseValue from the SDK response.
func buildAppDatabaseValue(ctx context.Context, db *quantadmingo.ApplicationDatabase) (resource_application.DatabaseValue, diag.Diagnostics) {
	attrTypes := resource_application.DatabaseValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"engine":                  types.StringNull(), // engine is a create-only field, not returned by API
		"instance_class":          types.StringNull(), // instance_class is create-only
		"multi_az":                types.BoolNull(),   // multi_az is create-only
		"storage_gb":              types.Int64Null(),   // storage_gb is create-only
		"rds_instance_identifier": optionalString(db.GetRdsInstanceIdentifierOk()),
		"rds_instance_endpoint":   optionalString(db.GetRdsInstanceEndpointOk()),
		"rds_instance_engine":     optionalString(db.GetRdsInstanceEngineOk()),
		"rds_instance_status":     optionalString(db.GetRdsInstanceStatusOk()),
	}
	return resource_application.NewDatabaseValue(attrTypes, attrs)
}

// buildAppFilesystemValue builds a resource_application.FilesystemValue from the SDK response.
func buildAppFilesystemValue(ctx context.Context, fs *quantadmingo.ApplicationFilesystem) (resource_application.FilesystemValue, diag.Diagnostics) {
	attrTypes := resource_application.FilesystemValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"filesystem_id": optionalString(fs.GetFilesystemIdOk()),
		"mount_path":    optionalString(fs.GetMountPathOk()),
		"required":      types.BoolNull(), // required is create-only
	}
	return resource_application.NewFilesystemValue(attrTypes, attrs)
}

// buildAppImageReferenceValue builds a resource_application.ImageReferenceValue from the SDK response.
func buildAppImageReferenceValue(ctx context.Context, ir *quantadmingo.ApplicationImageReference) (resource_application.ImageReferenceValue, diag.Diagnostics) {
	attrTypes := resource_application.ImageReferenceValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"identifier": optionalString(ir.GetIdentifierOk()),
		"type":       optionalString(ir.GetTypeOk()),
	}
	return resource_application.NewImageReferenceValue(attrTypes, attrs)
}

// buildAppDeploymentInformationList builds a types.List of DeploymentInformationValue
// from the SDK response.
func buildAppDeploymentInformationList(ctx context.Context, deployments []quantadmingo.ApplicationDeploymentInformationInner) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(deployments) == 0 {
		return types.ListNull(resource_application.DeploymentInformationValue{}.Type(ctx)), diags
	}

	attrTypes := resource_application.DeploymentInformationValue{}.AttributeTypes(ctx)
	vals := make([]attr.Value, 0, len(deployments))
	for _, dep := range deployments {
		createdAt := types.StringNull()
		if t, ok := dep.GetCreatedAtOk(); ok && t != nil {
			createdAt = types.StringValue(t.Format(time.RFC3339))
		}
		attrs := map[string]attr.Value{
			"created_at":          createdAt,
			"deployment_id":      optionalString(dep.GetDeploymentIdOk()),
			"image_tag":          optionalString(dep.GetImageTagOk()),
			"status":             optionalString(dep.GetStatusOk()),
			"task_definition_arn": optionalString(dep.GetTaskDefinitionArnOk()),
		}
		depVal, d := resource_application.NewDeploymentInformationValue(attrTypes, attrs)
		diags.Append(d...)
		vals = append(vals, depVal)
	}

	result, d := types.ListValueFrom(ctx, resource_application.DeploymentInformationValue{}.Type(ctx), vals)
	diags.Append(d...)
	return result, diags
}

// ---------------------------------------------------------------------------
// Environment: build Terraform model values from SDK EnvironmentResponse
// ---------------------------------------------------------------------------

// buildEnvComposeDefinitionValue builds a resource_environment.ComposeDefinitionValue
// from the SDK Compose response.
func buildEnvComposeDefinitionValue(ctx context.Context, compose *quantadmingo.Compose) (resource_environment.ComposeDefinitionValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build containers list
	containersList := types.ListNull(resource_environment.ContainersValue{}.Type(ctx))
	if compose.HasContainers() {
		sdkContainers := compose.GetContainers()
		containerVals := make([]attr.Value, 0, len(sdkContainers))
		for _, c := range sdkContainers {
			cv, d := buildEnvContainerValue(ctx, &c)
			diags.Append(d...)
			if diags.HasError() {
				return resource_environment.NewComposeDefinitionValueNull(), diags
			}
			containerVals = append(containerVals, cv)
		}
		var d diag.Diagnostics
		containersList, d = types.ListValueFrom(ctx, resource_environment.ContainersValue{}.Type(ctx), containerVals)
		diags.Append(d...)
		if diags.HasError() {
			return resource_environment.NewComposeDefinitionValueNull(), diags
		}
	}

	// Build spot configuration
	spotAttrs := map[string]attr.Value{
		"strategy": types.StringNull(),
	}
	if compose.HasSpotConfiguration() {
		spot := compose.GetSpotConfiguration()
		spotAttrs["strategy"] = optionalString(spot.GetStrategyOk())
	}
	spotAttrTypes := resource_environment.SpotConfigurationValue{}.AttributeTypes(ctx)
	spotObj, d := types.ObjectValue(spotAttrTypes, spotAttrs)
	diags.Append(d...)
	if diags.HasError() {
		return resource_environment.NewComposeDefinitionValueNull(), diags
	}

	attrTypes := resource_environment.ComposeDefinitionValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"architecture":               optionalString(compose.GetArchitectureOk()),
		"containers":                 containersList,
		"enable_cross_app_networking": optionalBool(compose.GetEnableCrossAppNetworkingOk()),
		"enable_cross_env_networking": optionalBool(compose.GetEnableCrossEnvNetworkingOk()),
		"max_capacity":               optionalInt32AsInt64(compose.GetMaxCapacityOk()),
		"min_capacity":               optionalInt32AsInt64(compose.GetMinCapacityOk()),
		"spot_configuration":         spotObj,
		"task_cpu":                   optionalInt32AsInt64(compose.GetTaskCpuOk()),
		"task_memory":                optionalInt32AsInt64(compose.GetTaskMemoryOk()),
	}

	result, d := resource_environment.NewComposeDefinitionValue(attrTypes, attrs)
	diags.Append(d...)
	return result, diags
}

// buildEnvContainerValue builds a single resource_environment.ContainersValue
// from an SDK Container.
func buildEnvContainerValue(ctx context.Context, c *quantadmingo.Container) (resource_environment.ContainersValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	var d diag.Diagnostics

	// ImageReference (required)
	imgRef := c.GetImageReference()
	irAttrTypes := resource_environment.ImageReferenceValue{}.AttributeTypes(ctx)
	irAttrs := map[string]attr.Value{
		"identifier": optionalString(imgRef.GetIdentifierOk()),
		"type":       optionalString(imgRef.GetTypeOk()),
	}
	irObj, d := types.ObjectValue(irAttrTypes, irAttrs)
	diags.Append(d...)

	// HealthCheck
	hcAttrTypes := resource_environment.HealthCheckValue{}.AttributeTypes(ctx)
	hcObj := types.ObjectNull(hcAttrTypes)
	if c.HasHealthCheck() {
		hc := c.GetHealthCheck()
		cmdList := types.ListNull(types.StringType)
		if cmds, ok := hc.GetCommandOk(); ok && cmds != nil {
			cmdList, d = types.ListValueFrom(ctx, types.StringType, cmds)
			diags.Append(d...)
		}
		hcAttrs := map[string]attr.Value{
			"command":      cmdList,
			"interval":     optionalInt32AsInt64(hc.GetIntervalOk()),
			"retries":      optionalInt32AsInt64(hc.GetRetriesOk()),
			"start_period": optionalInt32AsInt64(hc.GetStartPeriodOk()),
			"timeout":      optionalInt32AsInt64(hc.GetTimeoutOk()),
		}
		hcObj, d = types.ObjectValue(hcAttrTypes, hcAttrs)
		diags.Append(d...)
	}

	// OriginProtectionConfig
	opcAttrTypes := resource_environment.OriginProtectionConfigValue{}.AttributeTypes(ctx)
	opcObj := types.ObjectNull(opcAttrTypes)
	if c.HasOriginProtectionConfig() {
		opc := c.GetOriginProtectionConfig()
		ipList := types.ListNull(types.StringType)
		if ips, ok := opc.GetIpAllowOk(); ok && ips != nil {
			ipList, d = types.ListValueFrom(ctx, types.StringType, ips)
			diags.Append(d...)
		}
		opcAttrs := map[string]attr.Value{
			"enabled":  optionalBool(opc.GetEnabledOk()),
			"ip_allow": ipList,
		}
		opcObj, d = types.ObjectValue(opcAttrTypes, opcAttrs)
		diags.Append(d...)
	}

	// ExposedPorts
	exposedPorts := types.ListNull(types.Int64Type)
	if c.HasExposedPorts() {
		ports := c.GetExposedPorts()
		int64Ports := make([]int64, len(ports))
		for i, p := range ports {
			int64Ports[i] = int64(p)
		}
		exposedPorts, d = types.ListValueFrom(ctx, types.Int64Type, int64Ports)
		diags.Append(d...)
	}

	// Command
	commandList := types.ListNull(types.StringType)
	if c.HasCommand() {
		commandList, d = types.ListValueFrom(ctx, types.StringType, c.GetCommand())
		diags.Append(d...)
	}

	// EntryPoint
	entryPointList := types.ListNull(types.StringType)
	if c.HasEntryPoint() {
		entryPointList, d = types.ListValueFrom(ctx, types.StringType, c.GetEntryPoint())
		diags.Append(d...)
	}

	// Environment (container env vars)
	envList := types.ListNull(resource_environment.EnvironmentValue{}.Type(ctx))
	if c.HasEnvironment() {
		sdkEnvs := c.GetEnvironment()
		envVals := make([]attr.Value, 0, len(sdkEnvs))
		envAttrTypes := resource_environment.EnvironmentValue{}.AttributeTypes(ctx)
		for _, ev := range sdkEnvs {
			evAttrs := map[string]attr.Value{
				"name":  optionalString(ev.GetNameOk()),
				"value": optionalString(ev.GetValueOk()),
			}
			evVal, d2 := resource_environment.NewEnvironmentValue(envAttrTypes, evAttrs)
			diags.Append(d2...)
			envVals = append(envVals, evVal)
		}
		envList, d = types.ListValueFrom(ctx, resource_environment.EnvironmentValue{}.Type(ctx), envVals)
		diags.Append(d...)
	}

	// Secrets
	secretsList := types.ListNull(resource_environment.SecretsValue{}.Type(ctx))
	if c.HasSecrets() {
		sdkSecrets := c.GetSecrets()
		secVals := make([]attr.Value, 0, len(sdkSecrets))
		secAttrTypes := resource_environment.SecretsValue{}.AttributeTypes(ctx)
		for _, s := range sdkSecrets {
			sAttrs := map[string]attr.Value{
				"name":       optionalString(s.GetNameOk()),
				"value_from": optionalString(s.GetValueFromOk()),
			}
			sVal, d2 := resource_environment.NewSecretsValue(secAttrTypes, sAttrs)
			diags.Append(d2...)
			secVals = append(secVals, sVal)
		}
		secretsList, d = types.ListValueFrom(ctx, resource_environment.SecretsValue{}.Type(ctx), secVals)
		diags.Append(d...)
	}

	// DependsOn
	depsList := types.ListNull(resource_environment.DependsOnValue{}.Type(ctx))
	if c.HasDependsOn() {
		sdkDeps := c.GetDependsOn()
		depVals := make([]attr.Value, 0, len(sdkDeps))
		depAttrTypes := resource_environment.DependsOnValue{}.AttributeTypes(ctx)
		for _, dep := range sdkDeps {
			depAttrs := map[string]attr.Value{
				"condition":      optionalString(dep.GetConditionOk()),
				"container_name": optionalString(dep.GetContainerNameOk()),
			}
			depVal, d2 := resource_environment.NewDependsOnValue(depAttrTypes, depAttrs)
			diags.Append(d2...)
			depVals = append(depVals, depVal)
		}
		depsList, d = types.ListValueFrom(ctx, resource_environment.DependsOnValue{}.Type(ctx), depVals)
		diags.Append(d...)
	}

	// MountPoints
	mpList := types.ListNull(resource_environment.MountPointsValue{}.Type(ctx))
	if c.HasMountPoints() {
		sdkMPs := c.GetMountPoints()
		mpVals := make([]attr.Value, 0, len(sdkMPs))
		mpAttrTypes := resource_environment.MountPointsValue{}.AttributeTypes(ctx)
		for _, mp := range sdkMPs {
			mpAttrs := map[string]attr.Value{
				"container_path": optionalString(mp.GetContainerPathOk()),
				"read_only":      optionalBool(mp.GetReadOnlyOk()),
				"source_volume":  optionalString(mp.GetSourceVolumeOk()),
			}
			mpVal, d2 := resource_environment.NewMountPointsValue(mpAttrTypes, mpAttrs)
			diags.Append(d2...)
			mpVals = append(mpVals, mpVal)
		}
		mpList, d = types.ListValueFrom(ctx, resource_environment.MountPointsValue{}.Type(ctx), mpVals)
		diags.Append(d...)
	}

	attrTypes := resource_environment.ContainersValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"command":                  commandList,
		"cpu":                      optionalInt32AsInt64(c.GetCpuOk()),
		"depends_on":               depsList,
		"entry_point":              entryPointList,
		"environment":              envList,
		"essential":                optionalBool(c.GetEssentialOk()),
		"exposed_ports":            exposedPorts,
		"health_check":             hcObj,
		"image_reference":          irObj,
		"memory":                   optionalInt32AsInt64(c.GetMemoryOk()),
		"memory_reservation":       optionalInt32AsInt64(c.GetMemoryReservationOk()),
		"mount_points":             mpList,
		"name":                     types.StringValue(c.GetName()),
		"origin_protection":        optionalBool(c.GetOriginProtectionOk()),
		"origin_protection_config": opcObj,
		"readonly_root_filesystem": optionalBool(c.GetReadonlyRootFilesystemOk()),
		"secrets":                  secretsList,
		"user":                     optionalString(c.GetUserOk()),
		"working_directory":        optionalString(c.GetWorkingDirectoryOk()),
	}

	result, d := resource_environment.NewContainersValue(attrTypes, attrs)
	diags.Append(d...)
	return result, diags
}

// buildEnvVolumesListFromSDK builds a types.List of VolumesValue from SDK Volume slice.
func buildEnvVolumesListFromSDK(ctx context.Context, volumes []quantadmingo.Volume) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(volumes) == 0 {
		return types.ListNull(resource_environment.VolumesValue{}.Type(ctx)), diags
	}

	attrTypes := resource_environment.VolumesValue{}.AttributeTypes(ctx)
	vals := make([]attr.Value, 0, len(volumes))
	for _, v := range volumes {
		attrs := map[string]attr.Value{
			"access_point_arn":   optionalString(v.GetAccessPointArnOk()),
			"access_point_id":    optionalString(v.GetAccessPointIdOk()),
			"created_at":         optionalString(v.GetCreatedAtOk()),
			"description":        optionalString(v.GetDescriptionOk()),
			"environment_efs_id": optionalString(v.GetEnvironmentEfsIdOk()),
			"root_directory":     optionalString(v.GetRootDirectoryOk()),
			"volume_id":          optionalString(v.GetVolumeIdOk()),
			"volume_name":        optionalString(v.GetVolumeNameOk()),
		}
		volVal, d := resource_environment.NewVolumesValue(attrTypes, attrs)
		diags.Append(d...)
		vals = append(vals, volVal)
	}

	result, d := types.ListValueFrom(ctx, resource_environment.VolumesValue{}.Type(ctx), vals)
	diags.Append(d...)
	return result, diags
}

// buildEnvCronListFromSDK builds a types.List of CronValue from SDK Cron slice.
func buildEnvCronListFromSDK(ctx context.Context, crons []quantadmingo.Cron) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(crons) == 0 {
		return types.ListNull(resource_environment.CronValue{}.Type(ctx)), diags
	}

	attrTypes := resource_environment.CronValue{}.AttributeTypes(ctx)
	vals := make([]attr.Value, 0, len(crons))
	for _, c := range crons {
		cmdList := types.ListNull(types.StringType)
		if cmds, ok := c.GetCommandOk(); ok && cmds != nil {
			var d diag.Diagnostics
			cmdList, d = types.ListValueFrom(ctx, types.StringType, cmds)
			diags.Append(d...)
		}
		attrs := map[string]attr.Value{
			"command":               cmdList,
			"description":          optionalString(c.GetDescriptionOk()),
			"is_enabled":           optionalBool(c.GetIsEnabledOk()),
			"name":                 optionalString(c.GetNameOk()),
			"schedule_expression":  optionalString(c.GetScheduleExpressionOk()),
			"target_container_name": optionalString(c.GetTargetContainerNameOk()),
		}
		cronVal, d := resource_environment.NewCronValue(attrTypes, attrs)
		diags.Append(d...)
		vals = append(vals, cronVal)
	}

	result, d := types.ListValueFrom(ctx, resource_environment.CronValue{}.Type(ctx), vals)
	diags.Append(d...)
	return result, diags
}

// buildEnvSpotConfigurationValue builds a resource_environment.SpotConfigurationValue
// from the SDK SpotConfiguration response.
func buildEnvSpotConfigurationValue(ctx context.Context, spot *quantadmingo.SpotConfiguration) (resource_environment.SpotConfigurationValue, diag.Diagnostics) {
	attrTypes := resource_environment.SpotConfigurationValue{}.AttributeTypes(ctx)
	attrs := map[string]attr.Value{
		"strategy": optionalString(spot.GetStrategyOk()),
	}
	return resource_environment.NewSpotConfigurationValue(attrTypes, attrs)
}
