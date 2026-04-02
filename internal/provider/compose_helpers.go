package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

// composeFields is a portable intermediate representation of the Terraform
// ComposeDefinitionValue fields.  Both resource_application and
// resource_environment generate identical structs; this lets us share the
// conversion logic without duplicating it.
type composeFields struct {
	Architecture             basetypes.StringValue
	Containers               basetypes.ListValue
	EnableCrossAppNetworking basetypes.BoolValue
	EnableCrossEnvNetworking basetypes.BoolValue
	MaxCapacity              basetypes.Int64Value
	MinCapacity              basetypes.Int64Value
	SpotConfiguration        basetypes.ObjectValue
	TaskCpu                  basetypes.Int64Value
	TaskMemory               basetypes.Int64Value
}

// containerFields is a portable intermediate representation of the Terraform
// ContainersValue fields.
type containerFields struct {
	Command                basetypes.ListValue
	Cpu                    basetypes.Int64Value
	DependsOn              basetypes.ListValue
	EntryPoint             basetypes.ListValue
	Environment            basetypes.ListValue
	Essential              basetypes.BoolValue
	ExposedPorts           basetypes.ListValue
	HealthCheck            basetypes.ObjectValue
	ImageReference         basetypes.ObjectValue
	Memory                 basetypes.Int64Value
	MemoryReservation      basetypes.Int64Value
	MountPoints            basetypes.ListValue
	Name                   basetypes.StringValue
	OriginProtection       basetypes.BoolValue
	OriginProtectionConfig basetypes.ObjectValue
	ReadonlyRootFilesystem basetypes.BoolValue
	Secrets                basetypes.ListValue
	User                   basetypes.StringValue
	WorkingDirectory       basetypes.StringValue
}

// buildSDKCompose converts the portable compose fields into a quantadmingo.Compose struct.
func buildSDKCompose(ctx context.Context, cf composeFields) (quantadmingo.Compose, diag.Diagnostics) {
	var diags diag.Diagnostics
	compose := *quantadmingo.NewCompose()

	if !cf.TaskCpu.IsNull() && !cf.TaskCpu.IsUnknown() {
		compose.SetTaskCpu(int32(cf.TaskCpu.ValueInt64()))
	}
	if !cf.TaskMemory.IsNull() && !cf.TaskMemory.IsUnknown() {
		compose.SetTaskMemory(int32(cf.TaskMemory.ValueInt64()))
	}
	if !cf.Architecture.IsNull() && !cf.Architecture.IsUnknown() {
		compose.SetArchitecture(cf.Architecture.ValueString())
	}
	if !cf.MinCapacity.IsNull() && !cf.MinCapacity.IsUnknown() {
		compose.SetMinCapacity(int32(cf.MinCapacity.ValueInt64()))
	}
	if !cf.MaxCapacity.IsNull() && !cf.MaxCapacity.IsUnknown() {
		compose.SetMaxCapacity(int32(cf.MaxCapacity.ValueInt64()))
	}
	if !cf.EnableCrossEnvNetworking.IsNull() && !cf.EnableCrossEnvNetworking.IsUnknown() {
		compose.SetEnableCrossEnvNetworking(cf.EnableCrossEnvNetworking.ValueBool())
	}
	if !cf.EnableCrossAppNetworking.IsNull() && !cf.EnableCrossAppNetworking.IsUnknown() {
		compose.SetEnableCrossAppNetworking(cf.EnableCrossAppNetworking.ValueBool())
	}

	// Spot configuration
	if !cf.SpotConfiguration.IsNull() && !cf.SpotConfiguration.IsUnknown() {
		attrs := cf.SpotConfiguration.Attributes()
		if s, ok := attrs["strategy"]; ok {
			if sv, ok := s.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
				spot := quantadmingo.NewSpotConfiguration(sv.ValueString())
				compose.SetSpotConfiguration(*spot)
			}
		}
	}

	// Containers
	if !cf.Containers.IsNull() && !cf.Containers.IsUnknown() {
		sdkContainers, d := buildSDKContainers(ctx, cf.Containers)
		diags.Append(d...)
		if diags.HasError() {
			return compose, diags
		}
		compose.SetContainers(sdkContainers)
	}

	return compose, diags
}

// buildSDKContainers extracts container fields from a Terraform ListValue and
// converts each element into a quantadmingo.Container.
func buildSDKContainers(ctx context.Context, containersList basetypes.ListValue) ([]quantadmingo.Container, diag.Diagnostics) {
	var diags diag.Diagnostics
	var sdkContainers []quantadmingo.Container

	// Extract each element — may be a generated ContainersValue (custom type)
	// or a plain ObjectValue. Both implement ObjectValuable with ToObjectValue().
	elements := containersList.Elements()
	for i, elem := range elements {
		var objVal basetypes.ObjectValue

		// Try direct ObjectValue first, then fall back to ToObjectValue()
		if ov, ok := elem.(basetypes.ObjectValue); ok {
			objVal = ov
		} else if ova, ok := elem.(basetypes.ObjectValuable); ok {
			var d diag.Diagnostics
			objVal, d = ova.ToObjectValue(ctx)
			diags.Append(d...)
			if diags.HasError() {
				return nil, diags
			}
		} else {
			diags.AddAttributeError(
				path.Root("compose_definition").AtName("containers"),
				"Invalid container element",
				"Expected an object value for container element.",
			)
			continue
		}

		cf := extractContainerFields(objVal)
		container, d := buildSDKContainer(ctx, cf, i)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		sdkContainers = append(sdkContainers, container)
	}

	return sdkContainers, diags
}

// extractContainerFields pulls the typed attributes from an ObjectValue into
// the portable containerFields struct.
func extractContainerFields(obj basetypes.ObjectValue) containerFields {
	attrs := obj.Attributes()
	cf := containerFields{}

	if v, ok := attrs["name"]; ok {
		if sv, ok := v.(basetypes.StringValue); ok {
			cf.Name = sv
		}
	}
	if v, ok := attrs["cpu"]; ok {
		if iv, ok := v.(basetypes.Int64Value); ok {
			cf.Cpu = iv
		}
	}
	if v, ok := attrs["memory"]; ok {
		if iv, ok := v.(basetypes.Int64Value); ok {
			cf.Memory = iv
		}
	}
	if v, ok := attrs["memory_reservation"]; ok {
		if iv, ok := v.(basetypes.Int64Value); ok {
			cf.MemoryReservation = iv
		}
	}
	if v, ok := attrs["essential"]; ok {
		if bv, ok := v.(basetypes.BoolValue); ok {
			cf.Essential = bv
		}
	}
	if v, ok := attrs["user"]; ok {
		if sv, ok := v.(basetypes.StringValue); ok {
			cf.User = sv
		}
	}
	if v, ok := attrs["working_directory"]; ok {
		if sv, ok := v.(basetypes.StringValue); ok {
			cf.WorkingDirectory = sv
		}
	}
	if v, ok := attrs["readonly_root_filesystem"]; ok {
		if bv, ok := v.(basetypes.BoolValue); ok {
			cf.ReadonlyRootFilesystem = bv
		}
	}
	if v, ok := attrs["origin_protection"]; ok {
		if bv, ok := v.(basetypes.BoolValue); ok {
			cf.OriginProtection = bv
		}
	}
	if v, ok := attrs["image_reference"]; ok {
		if ov, ok := v.(basetypes.ObjectValue); ok {
			cf.ImageReference = ov
		}
	}
	if v, ok := attrs["health_check"]; ok {
		if ov, ok := v.(basetypes.ObjectValue); ok {
			cf.HealthCheck = ov
		}
	}
	if v, ok := attrs["origin_protection_config"]; ok {
		if ov, ok := v.(basetypes.ObjectValue); ok {
			cf.OriginProtectionConfig = ov
		}
	}
	if v, ok := attrs["exposed_ports"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.ExposedPorts = lv
		}
	}
	if v, ok := attrs["command"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.Command = lv
		}
	}
	if v, ok := attrs["entry_point"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.EntryPoint = lv
		}
	}
	if v, ok := attrs["environment"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.Environment = lv
		}
	}
	if v, ok := attrs["secrets"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.Secrets = lv
		}
	}
	if v, ok := attrs["depends_on"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.DependsOn = lv
		}
	}
	if v, ok := attrs["mount_points"]; ok {
		if lv, ok := v.(basetypes.ListValue); ok {
			cf.MountPoints = lv
		}
	}

	return cf
}

// buildSDKContainer converts a single containerFields into a quantadmingo.Container.
func buildSDKContainer(ctx context.Context, cf containerFields, index int) (quantadmingo.Container, diag.Diagnostics) {
	var diags diag.Diagnostics

	// ImageReference is required — extract type and identifier.
	imgRef := quantadmingo.ContainerImageReference{}
	if !cf.ImageReference.IsNull() && !cf.ImageReference.IsUnknown() {
		imgAttrs := cf.ImageReference.Attributes()
		if v, ok := imgAttrs["type"]; ok {
			if sv, ok := v.(basetypes.StringValue); ok {
				imgRef.SetType(sv.ValueString())
			}
		}
		if v, ok := imgAttrs["identifier"]; ok {
			if sv, ok := v.(basetypes.StringValue); ok {
				imgRef.SetIdentifier(sv.ValueString())
			}
		}
	}

	container := *quantadmingo.NewContainer(cf.Name.ValueString(), imgRef)

	// Optional scalar fields
	if !cf.Cpu.IsNull() && !cf.Cpu.IsUnknown() {
		container.SetCpu(int32(cf.Cpu.ValueInt64()))
	}
	if !cf.Memory.IsNull() && !cf.Memory.IsUnknown() {
		container.SetMemory(int32(cf.Memory.ValueInt64()))
	}
	if !cf.MemoryReservation.IsNull() && !cf.MemoryReservation.IsUnknown() {
		container.SetMemoryReservation(int32(cf.MemoryReservation.ValueInt64()))
	}
	if !cf.Essential.IsNull() && !cf.Essential.IsUnknown() {
		container.SetEssential(cf.Essential.ValueBool())
	}
	if !cf.User.IsNull() && !cf.User.IsUnknown() {
		container.SetUser(cf.User.ValueString())
	}
	if !cf.WorkingDirectory.IsNull() && !cf.WorkingDirectory.IsUnknown() {
		container.SetWorkingDirectory(cf.WorkingDirectory.ValueString())
	}
	if !cf.ReadonlyRootFilesystem.IsNull() && !cf.ReadonlyRootFilesystem.IsUnknown() {
		container.SetReadonlyRootFilesystem(cf.ReadonlyRootFilesystem.ValueBool())
	}
	if !cf.OriginProtection.IsNull() && !cf.OriginProtection.IsUnknown() {
		container.SetOriginProtection(cf.OriginProtection.ValueBool())
	}

	// ExposedPorts — list of int64 → []int32
	if !cf.ExposedPorts.IsNull() && !cf.ExposedPorts.IsUnknown() {
		var ports []int64
		diags.Append(cf.ExposedPorts.ElementsAs(ctx, &ports, false)...)
		if diags.HasError() {
			return container, diags
		}
		int32Ports := make([]int32, len(ports))
		for j, p := range ports {
			int32Ports[j] = int32(p)
		}
		container.SetExposedPorts(int32Ports)
	}

	// Command — list of strings
	if !cf.Command.IsNull() && !cf.Command.IsUnknown() {
		var cmds []string
		diags.Append(cf.Command.ElementsAs(ctx, &cmds, false)...)
		if diags.HasError() {
			return container, diags
		}
		container.SetCommand(cmds)
	}

	// EntryPoint — list of strings
	if !cf.EntryPoint.IsNull() && !cf.EntryPoint.IsUnknown() {
		var ep []string
		diags.Append(cf.EntryPoint.ElementsAs(ctx, &ep, false)...)
		if diags.HasError() {
			return container, diags
		}
		container.SetEntryPoint(ep)
	}

	// Environment — list of objects with name/value
	if !cf.Environment.IsNull() && !cf.Environment.IsUnknown() {
		var envVars []quantadmingo.ContainerEnvironmentInner
		for _, elem := range cf.Environment.Elements() {
			envObj, ok := elem.(basetypes.ObjectValue)
			if !ok {
				continue
			}
			envAttrs := envObj.Attributes()
			ev := quantadmingo.NewContainerEnvironmentInner("", "")
			if v, ok := envAttrs["name"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					ev.SetName(sv.ValueString())
				}
			}
			if v, ok := envAttrs["value"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					ev.SetValue(sv.ValueString())
				}
			}
			envVars = append(envVars, *ev)
		}
		container.SetEnvironment(envVars)
	}

	// Secrets — list of objects with name/value_from
	if !cf.Secrets.IsNull() && !cf.Secrets.IsUnknown() {
		var secrets []quantadmingo.ContainerSecretsInner
		for _, elem := range cf.Secrets.Elements() {
			secObj, ok := elem.(basetypes.ObjectValue)
			if !ok {
				continue
			}
			secAttrs := secObj.Attributes()
			s := quantadmingo.NewContainerSecretsInner("", "")
			if v, ok := secAttrs["name"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					s.SetName(sv.ValueString())
				}
			}
			if v, ok := secAttrs["value_from"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					s.SetValueFrom(sv.ValueString())
				}
			}
			secrets = append(secrets, *s)
		}
		container.SetSecrets(secrets)
	}

	// DependsOn — list of objects with container_name/condition
	if !cf.DependsOn.IsNull() && !cf.DependsOn.IsUnknown() {
		var deps []quantadmingo.ContainerDependsOnInner
		for _, elem := range cf.DependsOn.Elements() {
			depObj, ok := elem.(basetypes.ObjectValue)
			if !ok {
				continue
			}
			depAttrs := depObj.Attributes()
			d := quantadmingo.NewContainerDependsOnInner("")
			if v, ok := depAttrs["container_name"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					d.SetContainerName(sv.ValueString())
				}
			}
			if v, ok := depAttrs["condition"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					d.SetCondition(sv.ValueString())
				}
			}
			deps = append(deps, *d)
		}
		container.SetDependsOn(deps)
	}

	// MountPoints — list of objects with source_volume/container_path/read_only
	if !cf.MountPoints.IsNull() && !cf.MountPoints.IsUnknown() {
		var mps []quantadmingo.ContainerMountPointsInner
		for _, elem := range cf.MountPoints.Elements() {
			mpObj, ok := elem.(basetypes.ObjectValue)
			if !ok {
				continue
			}
			mpAttrs := mpObj.Attributes()
			mp := quantadmingo.NewContainerMountPointsInner("", "")
			if v, ok := mpAttrs["source_volume"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					mp.SetSourceVolume(sv.ValueString())
				}
			}
			if v, ok := mpAttrs["container_path"]; ok {
				if sv, ok := v.(basetypes.StringValue); ok && !sv.IsNull() && !sv.IsUnknown() {
					mp.SetContainerPath(sv.ValueString())
				}
			}
			if v, ok := mpAttrs["read_only"]; ok {
				if bv, ok := v.(basetypes.BoolValue); ok && !bv.IsNull() && !bv.IsUnknown() {
					mp.SetReadOnly(bv.ValueBool())
				}
			}
			mps = append(mps, *mp)
		}
		container.SetMountPoints(mps)
	}

	// HealthCheck — object with command/interval/timeout/retries/start_period
	if !cf.HealthCheck.IsNull() && !cf.HealthCheck.IsUnknown() {
		hcAttrs := cf.HealthCheck.Attributes()
		hc := quantadmingo.NewContainerHealthCheck()

		if v, ok := hcAttrs["command"]; ok {
			if lv, ok := v.(basetypes.ListValue); ok && !lv.IsNull() && !lv.IsUnknown() {
				var cmds []string
				diags.Append(lv.ElementsAs(ctx, &cmds, false)...)
				if !diags.HasError() {
					hc.SetCommand(cmds)
				}
			}
		}
		if v, ok := hcAttrs["interval"]; ok {
			if iv, ok := v.(basetypes.Int64Value); ok && !iv.IsNull() && !iv.IsUnknown() {
				hc.SetInterval(int32(iv.ValueInt64()))
			}
		}
		if v, ok := hcAttrs["timeout"]; ok {
			if iv, ok := v.(basetypes.Int64Value); ok && !iv.IsNull() && !iv.IsUnknown() {
				hc.SetTimeout(int32(iv.ValueInt64()))
			}
		}
		if v, ok := hcAttrs["retries"]; ok {
			if iv, ok := v.(basetypes.Int64Value); ok && !iv.IsNull() && !iv.IsUnknown() {
				hc.SetRetries(int32(iv.ValueInt64()))
			}
		}
		if v, ok := hcAttrs["start_period"]; ok {
			if iv, ok := v.(basetypes.Int64Value); ok && !iv.IsNull() && !iv.IsUnknown() {
				hc.SetStartPeriod(int32(iv.ValueInt64()))
			}
		}

		container.SetHealthCheck(*hc)
	}

	// OriginProtectionConfig — object with enabled/ip_allow
	if !cf.OriginProtectionConfig.IsNull() && !cf.OriginProtectionConfig.IsUnknown() {
		opcAttrs := cf.OriginProtectionConfig.Attributes()
		opc := quantadmingo.NewContainerOriginProtectionConfig()

		if v, ok := opcAttrs["enabled"]; ok {
			if bv, ok := v.(basetypes.BoolValue); ok && !bv.IsNull() && !bv.IsUnknown() {
				opc.SetEnabled(bv.ValueBool())
			}
		}
		if v, ok := opcAttrs["ip_allow"]; ok {
			if lv, ok := v.(basetypes.ListValue); ok && !lv.IsNull() && !lv.IsUnknown() {
				var ips []string
				diags.Append(lv.ElementsAs(ctx, &ips, false)...)
				if !diags.HasError() {
					opc.SetIpAllow(ips)
				}
			}
		}

		container.SetOriginProtectionConfig(*opc)
	}

	return container, diags
}

