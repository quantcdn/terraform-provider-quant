package resource_rule_proxy

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// toRecomputeWhenAppProxyPlanModifier ensures `to` is treated as unknown whenever application_proxy is true.
type toRecomputeWhenAppProxyPlanModifier struct{}

func (m toRecomputeWhenAppProxyPlanModifier) Description(_ context.Context) string {
	return "Marks 'to' unknown (to be recomputed) when application_proxy is true so backend-computed value is refreshed after save"
}

func (m toRecomputeWhenAppProxyPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Marks `to` unknown when `application_proxy` is true so the backend-computed value is refreshed after save"
}

func (m toRecomputeWhenAppProxyPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating and there is no prior state, defer to other modifiers/validators
	// We only force unknown when application_proxy is explicitly true in the planned config.
	var appProxy types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("application_proxy"), &appProxy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !appProxy.IsUnknown() && appProxy.ValueBool() {
		resp.PlanValue = basetypes.NewStringUnknown()
	}
}

// hostRecomputeWhenAppProxyPlanModifier ensures `host` is treated as unknown whenever application_proxy is true.
type hostRecomputeWhenAppProxyPlanModifier struct{}

func (m hostRecomputeWhenAppProxyPlanModifier) Description(_ context.Context) string {
	return "Marks 'host' unknown (to be recomputed) when application_proxy is true so backend-computed value is refreshed after save"
}

func (m hostRecomputeWhenAppProxyPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Marks `host` unknown when `application_proxy` is true so the backend-computed value is refreshed after save"
}

func (m hostRecomputeWhenAppProxyPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var appProxy types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("application_proxy"), &appProxy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !appProxy.IsUnknown() && appProxy.ValueBool() {
		resp.PlanValue = basetypes.NewStringUnknown()
	}
}

// ToRecomputeWhenAppProxy returns a plan modifier for the `to` attribute.
func ToRecomputeWhenAppProxy() planmodifier.String {
	return toRecomputeWhenAppProxyPlanModifier{}
}

// HostRecomputeWhenAppProxy returns a plan modifier for the `host` attribute.
func HostRecomputeWhenAppProxy() planmodifier.String {
	return hostRecomputeWhenAppProxyPlanModifier{}
}


