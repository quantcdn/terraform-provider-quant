package resource_rule_proxy

import (
    "context"

    "github.com/hashicorp/terraform-plugin-framework/path"
    schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

// toRequiredUnlessAppProxyValidator ensures that `to` is provided unless `application_proxy` is true.
type toRequiredUnlessAppProxyValidator struct{}

func (v toRequiredUnlessAppProxyValidator) Description(_ context.Context) string {
    return "to must be set unless application_proxy is true"
}

func (v toRequiredUnlessAppProxyValidator) MarkdownDescription(_ context.Context) string {
    return "to must be set unless `application_proxy` is true"
}

func (v toRequiredUnlessAppProxyValidator) ValidateString(ctx context.Context, req schemavalidator.StringRequest, resp *schemavalidator.StringResponse) {
    // Read application_proxy from the config
    var appProxy types.Bool
    diags := req.Config.GetAttribute(ctx, path.Root("application_proxy"), &appProxy)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // If application_proxy is true, `to` must NOT be set in config
    if !appProxy.IsUnknown() && appProxy.ValueBool() {
        if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                req.Path,
                "Invalid argument: to",
                "When `application_proxy` is true, do not set `to`. It is computed by the API.",
            )
        }
        return
    }

    // Otherwise, ensure `to` is present and non-empty
    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() == "" {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Missing required argument: to",
            "The attribute `to` is required when `application_proxy` is not true.",
        )
    }
}

// ToRequiredUnlessAppProxy returns the validator instance for schema wiring.
func ToRequiredUnlessAppProxy() schemavalidator.String {
    return toRequiredUnlessAppProxyValidator{}
}

// hostRequiredUnlessAppProxyValidator ensures that `host` is provided unless `application_proxy` is true.
type hostRequiredUnlessAppProxyValidator struct{}

func (v hostRequiredUnlessAppProxyValidator) Description(_ context.Context) string {
    return "host must be set unless application_proxy is true"
}

func (v hostRequiredUnlessAppProxyValidator) MarkdownDescription(_ context.Context) string {
    return "host must be set unless `application_proxy` is true"
}

func (v hostRequiredUnlessAppProxyValidator) ValidateString(ctx context.Context, req schemavalidator.StringRequest, resp *schemavalidator.StringResponse) {
    var appProxy types.Bool
    diags := req.Config.GetAttribute(ctx, path.Root("application_proxy"), &appProxy)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    if !appProxy.IsUnknown() && appProxy.ValueBool() {
        if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                req.Path,
                "Invalid argument: host",
                "When `application_proxy` is true, do not set `host`. It is computed by the API.",
            )
        }
        return
    }

    if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() == "" {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Missing required argument: host",
            "The attribute `host` is required when `application_proxy` is not true.",
        )
    }
}

// HostRequiredUnlessAppProxy returns the validator instance for schema wiring.
func HostRequiredUnlessAppProxy() schemavalidator.String {
    return hostRequiredUnlessAppProxyValidator{}
}


