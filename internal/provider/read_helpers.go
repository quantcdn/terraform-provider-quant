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
// Pointer helpers for SDK GetXxxOk patterns → Go struct fields
// ---------------------------------------------------------------------------

// ptrStr returns v when ok is true and v is non-nil/non-empty, else nil.
func ptrStr(v *string, ok bool) *string {
	if ok && v != nil && *v != "" {
		return v
	}
	return nil
}

// ptrInt32AsInt64 converts an SDK *int32 to *int64 when ok.
func ptrInt32AsInt64(v *int32, ok bool) *int64 {
	if ok && v != nil {
		n := int64(*v)
		return &n
	}
	return nil
}

// ptrBool returns v when ok is true and v is non-nil, else nil.
func ptrBool(v *bool, ok bool) *bool {
	if ok && v != nil {
		return v
	}
	return nil
}

// ptrTimeStr formats a time pointer as RFC3339 string pointer or nil.
func ptrTimeStr(v *time.Time, ok bool) *string {
	if ok && v != nil {
		s := v.Format(time.RFC3339)
		return &s
	}
	return nil
}

// optionalTime formats a time pointer as RFC3339 string or null.
// Kept because it is used in environment_resource.go.
func optionalTime(v *time.Time, ok bool) basetypes.StringValue {
	if ok && v != nil {
		return types.StringValue(v.Format(time.RFC3339))
	}
	return types.StringNull()
}

// ---------------------------------------------------------------------------
// autoMap: generic struct → custom TF Value type via ObjectValueFrom
// ---------------------------------------------------------------------------

// autoMap creates a custom Value type from a Go struct using ObjectValueFrom.
// T must be a generated Value type with a NewTValue(attrTypes, attrs) constructor.
func autoMap[T any](
	ctx context.Context,
	attrTypes map[string]attr.Type,
	readStruct any,
	constructor func(map[string]attr.Type, map[string]attr.Value) (T, diag.Diagnostics),
) (T, diag.Diagnostics) {
	var zero T
	objVal, diags := types.ObjectValueFrom(ctx, attrTypes, readStruct)
	if diags.HasError() {
		return zero, diags
	}
	result, d := constructor(attrTypes, objVal.Attributes())
	diags.Append(d...)
	return result, diags
}

// ---------------------------------------------------------------------------
// Read structs — intermediate Go structs with tfsdk tags
// ---------------------------------------------------------------------------

// databaseRead maps SDK ApplicationDatabase → TF DatabaseValue.
type databaseRead struct {
	Engine                *string `tfsdk:"engine"`
	InstanceClass         *string `tfsdk:"instance_class"`
	StorageGb             *int64  `tfsdk:"storage_gb"`
	MultiAz               *bool   `tfsdk:"multi_az"`
	RdsInstanceIdentifier *string `tfsdk:"rds_instance_identifier"`
	RdsInstanceEndpoint   *string `tfsdk:"rds_instance_endpoint"`
	RdsInstanceEngine     *string `tfsdk:"rds_instance_engine"`
	RdsInstanceStatus     *string `tfsdk:"rds_instance_status"`
}

// filesystemRead maps SDK ApplicationFilesystem → TF FilesystemValue.
type filesystemRead struct {
	FilesystemId *string `tfsdk:"filesystem_id"`
	MountPath    *string `tfsdk:"mount_path"`
	Required     *bool   `tfsdk:"required"`
}

// cacheRead maps SDK ApplicationCache → TF CacheValue (top-level app).
type cacheRead struct {
	CacheEndpoint    *string `tfsdk:"cache_endpoint"`
	CacheIdentifier  *string `tfsdk:"cache_identifier"`
	DataStorageMaxGb *int64  `tfsdk:"data_storage_max_gb"`
}

// imageRefRead maps SDK ApplicationImageReference → TF ImageReferenceValue (top-level app).
type imageRefRead struct {
	Identifier *string `tfsdk:"identifier"`
	Type       *string `tfsdk:"type"`
}

// deploymentInfoRead maps SDK ApplicationDeploymentInformationInner → TF DeploymentInformationValue.
type deploymentInfoRead struct {
	CreatedAt         *string `tfsdk:"created_at"`
	DeploymentId      *string `tfsdk:"deployment_id"`
	ImageTag          *string `tfsdk:"image_tag"`
	Status            *string `tfsdk:"status"`
	TaskDefinitionArn *string `tfsdk:"task_definition_arn"`
}

// containerImageRefRead maps SDK ContainerImageReference → nested object.
type containerImageRefRead struct {
	Identifier *string `tfsdk:"identifier"`
	Type       *string `tfsdk:"type"`
}

// spotConfigRead maps SDK SpotConfiguration → nested object.
type spotConfigRead struct {
	Strategy *string `tfsdk:"strategy"`
}

// healthCheckRead maps SDK ContainerHealthCheck → nested object.
type healthCheckRead struct {
	Command     types.List `tfsdk:"command"`
	Interval    *int64     `tfsdk:"interval"`
	Retries     *int64     `tfsdk:"retries"`
	StartPeriod *int64     `tfsdk:"start_period"`
	Timeout     *int64     `tfsdk:"timeout"`
}

// originProtectionConfigRead maps SDK ContainerOriginProtectionConfig → nested object.
type originProtectionConfigRead struct {
	Enabled      *bool      `tfsdk:"enabled"`
	IpAllow      types.List `tfsdk:"ip_allow"`
	RedirectHost *string    `tfsdk:"redirect_host"`
}

// containerEnvRead maps SDK ContainerEnvironmentInner → TF EnvironmentValue.
type containerEnvRead struct {
	Name  *string `tfsdk:"name"`
	Value *string `tfsdk:"value"`
}

// containerSecretRead maps SDK ContainerSecretsInner → TF SecretsValue.
type containerSecretRead struct {
	Name      *string `tfsdk:"name"`
	ValueFrom *string `tfsdk:"value_from"`
}

// containerDependsOnRead maps SDK ContainerDependsOnInner → TF DependsOnValue.
type containerDependsOnRead struct {
	Condition     *string `tfsdk:"condition"`
	ContainerName *string `tfsdk:"container_name"`
}

// containerMountPointRead maps SDK ContainerMountPointsInner → TF MountPointsValue.
type containerMountPointRead struct {
	ContainerPath *string `tfsdk:"container_path"`
	ReadOnly      *bool   `tfsdk:"read_only"`
	SourceVolume  *string `tfsdk:"source_volume"`
}

// containerRead maps SDK Container → TF ContainersValue.
type containerRead struct {
	Command                types.List   `tfsdk:"command"`
	Cpu                    *int64       `tfsdk:"cpu"`
	DependsOn              types.List   `tfsdk:"depends_on"`
	EntryPoint             types.List   `tfsdk:"entry_point"`
	Environment            types.List   `tfsdk:"environment"`
	Essential              *bool        `tfsdk:"essential"`
	ExposedPorts           types.List   `tfsdk:"exposed_ports"`
	HealthCheck            types.Object `tfsdk:"health_check"`
	ImageReference         types.Object `tfsdk:"image_reference"`
	Memory                 *int64       `tfsdk:"memory"`
	MemoryReservation      *int64       `tfsdk:"memory_reservation"`
	MountPoints            types.List   `tfsdk:"mount_points"`
	Name                   string       `tfsdk:"name"`
	OriginProtection       *bool        `tfsdk:"origin_protection"`
	OriginProtectionConfig types.Object `tfsdk:"origin_protection_config"`
	ReadonlyRootFilesystem *bool        `tfsdk:"readonly_root_filesystem"`
	Secrets                types.List   `tfsdk:"secrets"`
	User                   *string      `tfsdk:"user"`
	WorkingDirectory       *string      `tfsdk:"working_directory"`
}

// composeRead maps SDK Compose → TF ComposeDefinitionValue.
type composeRead struct {
	Architecture             *string      `tfsdk:"architecture"`
	Containers               types.List   `tfsdk:"containers"`
	EnableCrossAppNetworking *bool        `tfsdk:"enable_cross_app_networking"`
	EnableCrossEnvNetworking *bool        `tfsdk:"enable_cross_env_networking"`
	MaxCapacity              *int64       `tfsdk:"max_capacity"`
	MinCapacity              *int64       `tfsdk:"min_capacity"`
	SpotConfiguration        types.Object `tfsdk:"spot_configuration"`
	TaskCpu                  *int64       `tfsdk:"task_cpu"`
	TaskMemory               *int64       `tfsdk:"task_memory"`
}

// volumeRead maps SDK Volume → TF VolumesValue.
type volumeRead struct {
	AccessPointArn   *string `tfsdk:"access_point_arn"`
	AccessPointId    *string `tfsdk:"access_point_id"`
	CreatedAt        *string `tfsdk:"created_at"`
	Description      *string `tfsdk:"description"`
	EnvironmentEfsId *string `tfsdk:"environment_efs_id"`
	RootDirectory    *string `tfsdk:"root_directory"`
	VolumeId         *string `tfsdk:"volume_id"`
	VolumeName       *string `tfsdk:"volume_name"`
}

// cronRead maps SDK Cron → TF CronValue.
type cronRead struct {
	Command             types.List `tfsdk:"command"`
	Description         *string    `tfsdk:"description"`
	IsEnabled           *bool      `tfsdk:"is_enabled"`
	Name                *string    `tfsdk:"name"`
	ScheduleExpression  *string    `tfsdk:"schedule_expression"`
	TargetContainerName *string    `tfsdk:"target_container_name"`
}

// ---------------------------------------------------------------------------
// Shared sub-object builders (used by both App and Env container builders)
// ---------------------------------------------------------------------------

// buildImageRefObject builds a types.Object for a ContainerImageReference.
func buildImageRefObject(ctx context.Context, imgRef quantadmingo.ContainerImageReference, attrTypes map[string]attr.Type) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, attrTypes, containerImageRefRead{
		Identifier: ptrStr(imgRef.GetIdentifierOk()),
		Type:       ptrStr(imgRef.GetTypeOk()),
	})
}

// buildHealthCheckObject builds a types.Object for a ContainerHealthCheck, or null if absent.
func buildHealthCheckObject(ctx context.Context, c *quantadmingo.Container, attrTypes map[string]attr.Type) (basetypes.ObjectValue, diag.Diagnostics) {
	if !c.HasHealthCheck() {
		return types.ObjectNull(attrTypes), nil
	}
	var diags diag.Diagnostics
	hc := c.GetHealthCheck()

	cmdList := types.ListNull(types.StringType)
	if cmds, ok := hc.GetCommandOk(); ok && cmds != nil {
		var d diag.Diagnostics
		cmdList, d = types.ListValueFrom(ctx, types.StringType, cmds)
		diags.Append(d...)
	}

	obj, d := types.ObjectValueFrom(ctx, attrTypes, healthCheckRead{
		Command:     cmdList,
		Interval:    ptrInt32AsInt64(hc.GetIntervalOk()),
		Retries:     ptrInt32AsInt64(hc.GetRetriesOk()),
		StartPeriod: ptrInt32AsInt64(hc.GetStartPeriodOk()),
		Timeout:     ptrInt32AsInt64(hc.GetTimeoutOk()),
	})
	diags.Append(d...)
	return obj, diags
}

// buildOriginProtectionConfigObject builds a types.Object for ContainerOriginProtectionConfig, or null if absent.
func buildOriginProtectionConfigObject(ctx context.Context, c *quantadmingo.Container, attrTypes map[string]attr.Type) (basetypes.ObjectValue, diag.Diagnostics) {
	if !c.HasOriginProtectionConfig() {
		return types.ObjectNull(attrTypes), nil
	}
	var diags diag.Diagnostics
	opc := c.GetOriginProtectionConfig()

	ipList := types.ListNull(types.StringType)
	if ips, ok := opc.GetIpAllowOk(); ok && ips != nil {
		var d diag.Diagnostics
		ipList, d = types.ListValueFrom(ctx, types.StringType, ips)
		diags.Append(d...)
	}

	obj, d := types.ObjectValueFrom(ctx, attrTypes, originProtectionConfigRead{
		Enabled:      ptrBool(opc.GetEnabledOk()),
		IpAllow:      ipList,
		RedirectHost: ptrStr(opc.GetRedirectHostOk()),
	})
	diags.Append(d...)
	return obj, diags
}

// buildSpotConfigObject builds a types.Object for SpotConfiguration.
func buildSpotConfigObject(ctx context.Context, compose *quantadmingo.Compose, attrTypes map[string]attr.Type) (basetypes.ObjectValue, diag.Diagnostics) {
	read := spotConfigRead{}
	if compose.HasSpotConfiguration() {
		spot := compose.GetSpotConfiguration()
		read.Strategy = ptrStr(spot.GetStrategyOk())
	}
	return types.ObjectValueFrom(ctx, attrTypes, read)
}

// buildExposedPortsList converts SDK int32 ports to a types.List of Int64.
func buildExposedPortsList(ctx context.Context, c *quantadmingo.Container) (basetypes.ListValue, diag.Diagnostics) {
	if !c.HasExposedPorts() {
		return types.ListNull(types.Int64Type), nil
	}
	ports := c.GetExposedPorts()
	int64Ports := make([]int64, len(ports))
	for i, p := range ports {
		int64Ports[i] = int64(p)
	}
	return types.ListValueFrom(ctx, types.Int64Type, int64Ports)
}

// buildStringListFromSDK converts an SDK string-slice getter to a types.List.
func buildStringListFromSDK(ctx context.Context, vals []string, has bool) (basetypes.ListValue, diag.Diagnostics) {
	if !has {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, vals)
}

// ---------------------------------------------------------------------------
// Generic list-of-custom-type builder
// ---------------------------------------------------------------------------

// buildCustomList maps a slice of SDK items to a types.List of custom TF Value types.
func buildCustomList[SDK any, TFVal attr.Value](
	ctx context.Context,
	sdkItems []SDK,
	elemType attr.Type,
	attrTypes map[string]attr.Type,
	toRead func(SDK) any,
	constructor func(map[string]attr.Type, map[string]attr.Value) (TFVal, diag.Diagnostics),
) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	vals := make([]attr.Value, 0, len(sdkItems))
	for _, item := range sdkItems {
		v, d := autoMap(ctx, attrTypes, toRead(item), constructor)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(elemType), diags
		}
		vals = append(vals, v)
	}
	result, d := types.ListValueFrom(ctx, elemType, vals)
	diags.Append(d...)
	return result, diags
}

// ---------------------------------------------------------------------------
// Application: build Terraform model from SDK Application response
// ---------------------------------------------------------------------------

// buildAppDatabaseValue builds a resource_application.DatabaseValue from the SDK response.
func buildAppDatabaseValue(ctx context.Context, db *quantadmingo.ApplicationDatabase) (resource_application.DatabaseValue, diag.Diagnostics) {
	return autoMap(ctx,
		resource_application.DatabaseValue{}.AttributeTypes(ctx),
		databaseRead{
			// engine, instance_class, multi_az, storage_gb are create-only; not returned by API
			RdsInstanceIdentifier: ptrStr(db.GetRdsInstanceIdentifierOk()),
			RdsInstanceEndpoint:   ptrStr(db.GetRdsInstanceEndpointOk()),
			RdsInstanceEngine:     ptrStr(db.GetRdsInstanceEngineOk()),
			RdsInstanceStatus:     ptrStr(db.GetRdsInstanceStatusOk()),
		},
		resource_application.NewDatabaseValue,
	)
}

// buildAppFilesystemValue builds a resource_application.FilesystemValue from the SDK response.
func buildAppFilesystemValue(ctx context.Context, fs *quantadmingo.ApplicationFilesystem) (resource_application.FilesystemValue, diag.Diagnostics) {
	return autoMap(ctx,
		resource_application.FilesystemValue{}.AttributeTypes(ctx),
		filesystemRead{
			FilesystemId: ptrStr(fs.GetFilesystemIdOk()),
			MountPath:    ptrStr(fs.GetMountPathOk()),
			// required is create-only
		},
		resource_application.NewFilesystemValue,
	)
}

// buildAppCacheValue builds a resource_application.CacheValue from the SDK response.
func buildAppCacheValue(ctx context.Context, c *quantadmingo.ApplicationCache) (resource_application.CacheValue, diag.Diagnostics) {
	return autoMap(ctx,
		resource_application.CacheValue{}.AttributeTypes(ctx),
		cacheRead{
			CacheEndpoint:    ptrStr(c.GetCacheEndpointOk()),
			CacheIdentifier:  ptrStr(c.GetCacheIdentifierOk()),
			DataStorageMaxGb: ptrInt32AsInt64(c.GetDataStorageMaxGbOk()),
		},
		resource_application.NewCacheValue,
	)
}

// buildAppImageReferenceValue builds a resource_application.ImageReferenceValue from the SDK response.
func buildAppImageReferenceValue(ctx context.Context, ir *quantadmingo.ApplicationImageReference) (resource_application.ImageReferenceValue, diag.Diagnostics) {
	return autoMap(ctx,
		resource_application.ImageReferenceValue{}.AttributeTypes(ctx),
		imageRefRead{
			Identifier: ptrStr(ir.GetIdentifierOk()),
			Type:       ptrStr(ir.GetTypeOk()),
		},
		resource_application.NewImageReferenceValue,
	)
}

// buildAppDeploymentInformationList builds a types.List of DeploymentInformationValue
// from the SDK response.
func buildAppDeploymentInformationList(ctx context.Context, deployments []quantadmingo.ApplicationDeploymentInformationInner) (basetypes.ListValue, diag.Diagnostics) {
	if len(deployments) == 0 {
		return types.ListNull(resource_application.DeploymentInformationValue{}.Type(ctx)), nil
	}

	attrTypes := resource_application.DeploymentInformationValue{}.AttributeTypes(ctx)
	return buildCustomList(ctx, deployments,
		resource_application.DeploymentInformationValue{}.Type(ctx),
		attrTypes,
		func(dep quantadmingo.ApplicationDeploymentInformationInner) any {
			return deploymentInfoRead{
				CreatedAt:         ptrTimeStr(dep.GetCreatedAtOk()),
				DeploymentId:      ptrStr(dep.GetDeploymentIdOk()),
				ImageTag:          ptrStr(dep.GetImageTagOk()),
				Status:            ptrStr(dep.GetStatusOk()),
				TaskDefinitionArn: ptrStr(dep.GetTaskDefinitionArnOk()),
			}
		},
		resource_application.NewDeploymentInformationValue,
	)
}

// buildAppContainerValue builds a single resource_application.ContainersValue
// from an SDK Container.
func buildAppContainerValue(ctx context.Context, c *quantadmingo.Container) (resource_application.ContainersValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Sub-objects
	irAttrTypes := resource_application.ImageReferenceValue{}.AttributeTypes(ctx)
	irObj, d := buildImageRefObject(ctx, c.GetImageReference(), irAttrTypes)
	diags.Append(d...)

	hcAttrTypes := resource_application.HealthCheckValue{}.AttributeTypes(ctx)
	hcObj, d := buildHealthCheckObject(ctx, c, hcAttrTypes)
	diags.Append(d...)

	opcAttrTypes := resource_application.OriginProtectionConfigValue{}.AttributeTypes(ctx)
	opcObj, d := buildOriginProtectionConfigObject(ctx, c, opcAttrTypes)
	diags.Append(d...)

	// Primitive lists
	exposedPorts, d := buildExposedPortsList(ctx, c)
	diags.Append(d...)

	commandList, d := buildStringListFromSDK(ctx, c.GetCommand(), c.HasCommand())
	diags.Append(d...)

	entryPointList, d := buildStringListFromSDK(ctx, c.GetEntryPoint(), c.HasEntryPoint())
	diags.Append(d...)

	// Custom-type lists
	envList := types.ListNull(resource_application.EnvironmentValue{}.Type(ctx))
	if c.HasEnvironment() {
		envList, d = buildCustomList(ctx, c.GetEnvironment(),
			resource_application.EnvironmentValue{}.Type(ctx),
			resource_application.EnvironmentValue{}.AttributeTypes(ctx),
			func(ev quantadmingo.ContainerEnvironmentInner) any {
				return containerEnvRead{Name: ptrStr(ev.GetNameOk()), Value: ptrStr(ev.GetValueOk())}
			},
			resource_application.NewEnvironmentValue,
		)
		diags.Append(d...)
	}

	secretsList := types.ListNull(resource_application.SecretsValue{}.Type(ctx))
	if c.HasSecrets() {
		secretsList, d = buildCustomList(ctx, c.GetSecrets(),
			resource_application.SecretsValue{}.Type(ctx),
			resource_application.SecretsValue{}.AttributeTypes(ctx),
			func(s quantadmingo.ContainerSecretsInner) any {
				return containerSecretRead{Name: ptrStr(s.GetNameOk()), ValueFrom: ptrStr(s.GetValueFromOk())}
			},
			resource_application.NewSecretsValue,
		)
		diags.Append(d...)
	}

	depsList := types.ListNull(resource_application.DependsOnValue{}.Type(ctx))
	if c.HasDependsOn() {
		depsList, d = buildCustomList(ctx, c.GetDependsOn(),
			resource_application.DependsOnValue{}.Type(ctx),
			resource_application.DependsOnValue{}.AttributeTypes(ctx),
			func(dep quantadmingo.ContainerDependsOnInner) any {
				return containerDependsOnRead{Condition: ptrStr(dep.GetConditionOk()), ContainerName: ptrStr(dep.GetContainerNameOk())}
			},
			resource_application.NewDependsOnValue,
		)
		diags.Append(d...)
	}

	mpList := types.ListNull(resource_application.MountPointsValue{}.Type(ctx))
	if c.HasMountPoints() {
		mpList, d = buildCustomList(ctx, c.GetMountPoints(),
			resource_application.MountPointsValue{}.Type(ctx),
			resource_application.MountPointsValue{}.AttributeTypes(ctx),
			func(mp quantadmingo.ContainerMountPointsInner) any {
				return containerMountPointRead{
					ContainerPath: ptrStr(mp.GetContainerPathOk()),
					ReadOnly:      ptrBool(mp.GetReadOnlyOk()),
					SourceVolume:  ptrStr(mp.GetSourceVolumeOk()),
				}
			},
			resource_application.NewMountPointsValue,
		)
		diags.Append(d...)
	}

	if diags.HasError() {
		return resource_application.ContainersValue{}, diags
	}

	result, d := autoMap(ctx,
		resource_application.ContainersValue{}.AttributeTypes(ctx),
		containerRead{
			Command:                commandList,
			Cpu:                    ptrInt32AsInt64(c.GetCpuOk()),
			DependsOn:              depsList,
			EntryPoint:             entryPointList,
			Environment:            envList,
			Essential:              ptrBool(c.GetEssentialOk()),
			ExposedPorts:           exposedPorts,
			HealthCheck:            hcObj,
			ImageReference:         irObj,
			Memory:                 ptrInt32AsInt64(c.GetMemoryOk()),
			MemoryReservation:      ptrInt32AsInt64(c.GetMemoryReservationOk()),
			MountPoints:            mpList,
			Name:                   c.GetName(),
			OriginProtection:       ptrBool(c.GetOriginProtectionOk()),
			OriginProtectionConfig: opcObj,
			ReadonlyRootFilesystem: ptrBool(c.GetReadonlyRootFilesystemOk()),
			Secrets:                secretsList,
			User:                   ptrStr(c.GetUserOk()),
			WorkingDirectory:       ptrStr(c.GetWorkingDirectoryOk()),
		},
		resource_application.NewContainersValue,
	)
	diags.Append(d...)
	return result, diags
}

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
	spotAttrTypes := resource_application.SpotConfigurationValue{}.AttributeTypes(ctx)
	spotObj, d := buildSpotConfigObject(ctx, compose, spotAttrTypes)
	diags.Append(d...)
	if diags.HasError() {
		return resource_application.NewComposeDefinitionValueNull(), diags
	}

	result, d := autoMap(ctx,
		resource_application.ComposeDefinitionValue{}.AttributeTypes(ctx),
		composeRead{
			Architecture:             ptrStr(compose.GetArchitectureOk()),
			Containers:               containersList,
			EnableCrossAppNetworking: ptrBool(compose.GetEnableCrossAppNetworkingOk()),
			EnableCrossEnvNetworking: ptrBool(compose.GetEnableCrossEnvNetworkingOk()),
			MaxCapacity:              ptrInt32AsInt64(compose.GetMaxCapacityOk()),
			MinCapacity:              ptrInt32AsInt64(compose.GetMinCapacityOk()),
			SpotConfiguration:        spotObj,
			TaskCpu:                  ptrInt32AsInt64(compose.GetTaskCpuOk()),
			TaskMemory:               ptrInt32AsInt64(compose.GetTaskMemoryOk()),
		},
		resource_application.NewComposeDefinitionValue,
	)
	diags.Append(d...)
	return result, diags
}

// ---------------------------------------------------------------------------
// Environment: build Terraform model values from SDK EnvironmentResponse
// ---------------------------------------------------------------------------

// buildEnvContainerValue is kept for potential future use.
//
//nolint:unused
func buildEnvContainerValue(ctx context.Context, c *quantadmingo.Container) (resource_environment.ContainersValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Sub-objects
	irAttrTypes := resource_environment.ImageReferenceValue{}.AttributeTypes(ctx)
	irObj, d := buildImageRefObject(ctx, c.GetImageReference(), irAttrTypes)
	diags.Append(d...)

	hcAttrTypes := resource_environment.HealthCheckValue{}.AttributeTypes(ctx)
	hcObj, d := buildHealthCheckObject(ctx, c, hcAttrTypes)
	diags.Append(d...)

	opcAttrTypes := resource_environment.OriginProtectionConfigValue{}.AttributeTypes(ctx)
	opcObj, d := buildOriginProtectionConfigObject(ctx, c, opcAttrTypes)
	diags.Append(d...)

	// Primitive lists
	exposedPorts, d := buildExposedPortsList(ctx, c)
	diags.Append(d...)

	commandList, d := buildStringListFromSDK(ctx, c.GetCommand(), c.HasCommand())
	diags.Append(d...)

	entryPointList, d := buildStringListFromSDK(ctx, c.GetEntryPoint(), c.HasEntryPoint())
	diags.Append(d...)

	// Custom-type lists
	envList := types.ListNull(resource_environment.EnvironmentValue{}.Type(ctx))
	if c.HasEnvironment() {
		envList, d = buildCustomList(ctx, c.GetEnvironment(),
			resource_environment.EnvironmentValue{}.Type(ctx),
			resource_environment.EnvironmentValue{}.AttributeTypes(ctx),
			func(ev quantadmingo.ContainerEnvironmentInner) any {
				return containerEnvRead{Name: ptrStr(ev.GetNameOk()), Value: ptrStr(ev.GetValueOk())}
			},
			resource_environment.NewEnvironmentValue,
		)
		diags.Append(d...)
	}

	secretsList := types.ListNull(resource_environment.SecretsValue{}.Type(ctx))
	if c.HasSecrets() {
		secretsList, d = buildCustomList(ctx, c.GetSecrets(),
			resource_environment.SecretsValue{}.Type(ctx),
			resource_environment.SecretsValue{}.AttributeTypes(ctx),
			func(s quantadmingo.ContainerSecretsInner) any {
				return containerSecretRead{Name: ptrStr(s.GetNameOk()), ValueFrom: ptrStr(s.GetValueFromOk())}
			},
			resource_environment.NewSecretsValue,
		)
		diags.Append(d...)
	}

	depsList := types.ListNull(resource_environment.DependsOnValue{}.Type(ctx))
	if c.HasDependsOn() {
		depsList, d = buildCustomList(ctx, c.GetDependsOn(),
			resource_environment.DependsOnValue{}.Type(ctx),
			resource_environment.DependsOnValue{}.AttributeTypes(ctx),
			func(dep quantadmingo.ContainerDependsOnInner) any {
				return containerDependsOnRead{Condition: ptrStr(dep.GetConditionOk()), ContainerName: ptrStr(dep.GetContainerNameOk())}
			},
			resource_environment.NewDependsOnValue,
		)
		diags.Append(d...)
	}

	mpList := types.ListNull(resource_environment.MountPointsValue{}.Type(ctx))
	if c.HasMountPoints() {
		mpList, d = buildCustomList(ctx, c.GetMountPoints(),
			resource_environment.MountPointsValue{}.Type(ctx),
			resource_environment.MountPointsValue{}.AttributeTypes(ctx),
			func(mp quantadmingo.ContainerMountPointsInner) any {
				return containerMountPointRead{
					ContainerPath: ptrStr(mp.GetContainerPathOk()),
					ReadOnly:      ptrBool(mp.GetReadOnlyOk()),
					SourceVolume:  ptrStr(mp.GetSourceVolumeOk()),
				}
			},
			resource_environment.NewMountPointsValue,
		)
		diags.Append(d...)
	}

	if diags.HasError() {
		return resource_environment.ContainersValue{}, diags
	}

	result, d := autoMap(ctx,
		resource_environment.ContainersValue{}.AttributeTypes(ctx),
		containerRead{
			Command:                commandList,
			Cpu:                    ptrInt32AsInt64(c.GetCpuOk()),
			DependsOn:              depsList,
			EntryPoint:             entryPointList,
			Environment:            envList,
			Essential:              ptrBool(c.GetEssentialOk()),
			ExposedPorts:           exposedPorts,
			HealthCheck:            hcObj,
			ImageReference:         irObj,
			Memory:                 ptrInt32AsInt64(c.GetMemoryOk()),
			MemoryReservation:      ptrInt32AsInt64(c.GetMemoryReservationOk()),
			MountPoints:            mpList,
			Name:                   c.GetName(),
			OriginProtection:       ptrBool(c.GetOriginProtectionOk()),
			OriginProtectionConfig: opcObj,
			ReadonlyRootFilesystem: ptrBool(c.GetReadonlyRootFilesystemOk()),
			Secrets:                secretsList,
			User:                   ptrStr(c.GetUserOk()),
			WorkingDirectory:       ptrStr(c.GetWorkingDirectoryOk()),
		},
		resource_environment.NewContainersValue,
	)
	diags.Append(d...)
	return result, diags
}

// buildEnvComposeDefinitionValue is kept for potential future use.
//
//nolint:unused
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
	spotAttrTypes := resource_environment.SpotConfigurationValue{}.AttributeTypes(ctx)
	spotObj, d := buildSpotConfigObject(ctx, compose, spotAttrTypes)
	diags.Append(d...)
	if diags.HasError() {
		return resource_environment.NewComposeDefinitionValueNull(), diags
	}

	result, d := autoMap(ctx,
		resource_environment.ComposeDefinitionValue{}.AttributeTypes(ctx),
		composeRead{
			Architecture:             ptrStr(compose.GetArchitectureOk()),
			Containers:               containersList,
			EnableCrossAppNetworking: ptrBool(compose.GetEnableCrossAppNetworkingOk()),
			EnableCrossEnvNetworking: ptrBool(compose.GetEnableCrossEnvNetworkingOk()),
			MaxCapacity:              ptrInt32AsInt64(compose.GetMaxCapacityOk()),
			MinCapacity:              ptrInt32AsInt64(compose.GetMinCapacityOk()),
			SpotConfiguration:        spotObj,
			TaskCpu:                  ptrInt32AsInt64(compose.GetTaskCpuOk()),
			TaskMemory:               ptrInt32AsInt64(compose.GetTaskMemoryOk()),
		},
		resource_environment.NewComposeDefinitionValue,
	)
	diags.Append(d...)
	return result, diags
}

// buildEnvVolumesListFromSDK builds a types.List of VolumesValue from SDK Volume slice.
func buildEnvVolumesListFromSDK(ctx context.Context, volumes []quantadmingo.Volume) (basetypes.ListValue, diag.Diagnostics) {
	if len(volumes) == 0 {
		return types.ListNull(resource_environment.VolumesValue{}.Type(ctx)), nil
	}

	return buildCustomList(ctx, volumes,
		resource_environment.VolumesValue{}.Type(ctx),
		resource_environment.VolumesValue{}.AttributeTypes(ctx),
		func(v quantadmingo.Volume) any {
			return volumeRead{
				AccessPointArn:   ptrStr(v.GetAccessPointArnOk()),
				AccessPointId:    ptrStr(v.GetAccessPointIdOk()),
				CreatedAt:        ptrStr(v.GetCreatedAtOk()),
				Description:      ptrStr(v.GetDescriptionOk()),
				EnvironmentEfsId: ptrStr(v.GetEnvironmentEfsIdOk()),
				RootDirectory:    ptrStr(v.GetRootDirectoryOk()),
				VolumeId:         ptrStr(v.GetVolumeIdOk()),
				VolumeName:       ptrStr(v.GetVolumeNameOk()),
			}
		},
		resource_environment.NewVolumesValue,
	)
}

// buildEnvCronListFromSDK builds a types.List of CronValue from SDK Cron slice.
func buildEnvCronListFromSDK(ctx context.Context, crons []quantadmingo.Cron) (basetypes.ListValue, diag.Diagnostics) {
	if len(crons) == 0 {
		return types.ListNull(resource_environment.CronValue{}.Type(ctx)), nil
	}

	var diags diag.Diagnostics
	attrTypes := resource_environment.CronValue{}.AttributeTypes(ctx)
	vals := make([]attr.Value, 0, len(crons))

	for _, c := range crons {
		cmdList := types.ListNull(types.StringType)
		if cmds, ok := c.GetCommandOk(); ok && cmds != nil {
			var d diag.Diagnostics
			cmdList, d = types.ListValueFrom(ctx, types.StringType, cmds)
			diags.Append(d...)
		}

		v, d := autoMap(ctx, attrTypes,
			cronRead{
				Command:             cmdList,
				Description:         ptrStr(c.GetDescriptionOk()),
				IsEnabled:           ptrBool(c.GetIsEnabledOk()),
				Name:                ptrStr(c.GetNameOk()),
				ScheduleExpression:  ptrStr(c.GetScheduleExpressionOk()),
				TargetContainerName: ptrStr(c.GetTargetContainerNameOk()),
			},
			resource_environment.NewCronValue,
		)
		diags.Append(d...)
		vals = append(vals, v)
	}

	result, d := types.ListValueFrom(ctx, resource_environment.CronValue{}.Type(ctx), vals)
	diags.Append(d...)
	return result, diags
}

// buildEnvSpotConfigurationValue is kept for potential future use.
//
//nolint:unused
func buildEnvSpotConfigurationValue(ctx context.Context, spot *quantadmingo.SpotConfiguration) (resource_environment.SpotConfigurationValue, diag.Diagnostics) {
	return autoMap(ctx,
		resource_environment.SpotConfigurationValue{}.AttributeTypes(ctx),
		spotConfigRead{
			Strategy: ptrStr(spot.GetStrategyOk()),
		},
		resource_environment.NewSpotConfigurationValue,
	)
}
