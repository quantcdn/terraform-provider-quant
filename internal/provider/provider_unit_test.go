package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_cron_job"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/resource_volume"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

// ===================================================================
// 1. Provider Schema / Metadata / Resources / DataSources
// ===================================================================

func TestUnitProviderMetadata(t *testing.T) {
	p := &quantProvider{}
	resp := &fwprovider.MetadataResponse{}
	p.Metadata(context.Background(), fwprovider.MetadataRequest{}, resp)

	if resp.TypeName != "quant" {
		t.Errorf("expected TypeName 'quant', got %q", resp.TypeName)
	}
}

func TestUnitProviderSchema(t *testing.T) {
	p := &quantProvider{}
	resp := &fwprovider.SchemaResponse{}
	p.Schema(context.Background(), fwprovider.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %s", resp.Diagnostics)
	}

	expectedAttrs := []string{
		"bearer", "organization", "base_url",
		"requests_per_second", "max_retries", "base_delay_ms",
		"max_delay_ms", "enable_jitter", "timeout_seconds",
	}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing expected schema attribute: %s", attr)
		}
	}
}

func TestUnitProviderResources(t *testing.T) {
	p := &quantProvider{}
	resources := p.Resources(context.Background())

	// There should be a reasonable number of resources registered
	if len(resources) == 0 {
		t.Fatal("expected at least one resource, got 0")
	}

	// Verify at least the known resource types are present by instantiating them
	expectedTypeNames := map[string]bool{
		"_project":          false,
		"_domain":           false,
		"_crawler":          false,
		"_crawler_schedule": false,
		"_rule_proxy":       false,
		"_rule_redirect":    false,
		"_header":           false,
		"_application":      false,
		"_environment":      false,
		"_volume":           false,
		"_cron_job":         false,
		"_kv_store":         false,
		"_kv_item":          false,
	}

	for _, rf := range resources {
		r := rf()
		metaResp := &fwresource.MetadataResponse{}
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "quant"}, metaResp)
		suffix := metaResp.TypeName[len("quant"):]
		if _, ok := expectedTypeNames[suffix]; ok {
			expectedTypeNames[suffix] = true
		}
	}

	for name, found := range expectedTypeNames {
		if !found {
			t.Errorf("expected resource 'quant%s' not found in provider resources", name)
		}
	}
}

func TestUnitProviderDataSources(t *testing.T) {
	p := &quantProvider{}
	dataSources := p.DataSources(context.Background())

	if len(dataSources) != 2 {
		t.Fatalf("expected 2 data sources, got %d", len(dataSources))
	}

	expectedTypeNames := map[string]bool{
		"quant_projects": false,
		"quant_project":  false,
	}

	for _, dsf := range dataSources {
		ds := dsf()
		metaResp := &fwdatasource.MetadataResponse{}
		ds.Metadata(context.Background(), fwdatasource.MetadataRequest{ProviderTypeName: "quant"}, metaResp)
		expectedTypeNames[metaResp.TypeName] = true
	}

	for name, found := range expectedTypeNames {
		if !found {
			t.Errorf("expected data source %q not found", name)
		}
	}
}

// ===================================================================
// 2. Provider Configure
// ===================================================================

// newProviderConfigVal builds a tftypes.Value for the provider config
// using the given overrides. Nil string means null.
func newProviderConfigVal(bearer, org, baseURL *string) tftypes.Value {
	toBearerVal := tftypes.NewValue(tftypes.String, nil)
	if bearer != nil {
		toBearerVal = tftypes.NewValue(tftypes.String, *bearer)
	}
	toOrgVal := tftypes.NewValue(tftypes.String, nil)
	if org != nil {
		toOrgVal = tftypes.NewValue(tftypes.String, *org)
	}
	toBaseURLVal := tftypes.NewValue(tftypes.String, nil)
	if baseURL != nil {
		toBaseURLVal = tftypes.NewValue(tftypes.String, *baseURL)
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"bearer":              tftypes.String,
			"organization":        tftypes.String,
			"base_url":            tftypes.String,
			"requests_per_second": tftypes.Number,
			"max_retries":         tftypes.Number,
			"base_delay_ms":       tftypes.Number,
			"max_delay_ms":        tftypes.Number,
			"enable_jitter":       tftypes.Bool,
			"timeout_seconds":     tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"bearer":              toBearerVal,
		"organization":        toOrgVal,
		"base_url":            toBaseURLVal,
		"requests_per_second": tftypes.NewValue(tftypes.Number, nil),
		"max_retries":         tftypes.NewValue(tftypes.Number, nil),
		"base_delay_ms":       tftypes.NewValue(tftypes.Number, nil),
		"max_delay_ms":        tftypes.NewValue(tftypes.Number, nil),
		"enable_jitter":       tftypes.NewValue(tftypes.Bool, nil),
		"timeout_seconds":     tftypes.NewValue(tftypes.Number, nil),
	})
}

func getProviderSchema() fwprovider.SchemaResponse {
	p := &quantProvider{}
	schemaResp := &fwprovider.SchemaResponse{}
	p.Schema(context.Background(), fwprovider.SchemaRequest{}, schemaResp)
	return *schemaResp
}

func TestUnitProviderConfigure_MissingBearerAndOrg(t *testing.T) {
	// Clear env vars to ensure they don't interfere
	t.Setenv("QUANTCDN_API_TOKEN", "")
	t.Setenv("QUANTCDN_ORGANIZATION", "")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	configVal := newProviderConfigVal(nil, nil, nil)

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Fatal("expected errors when bearer and org are both missing, but got none")
	}

	// Check we got errors about both bearer and organization
	errCount := 0
	for _, d := range configResp.Diagnostics.Errors() {
		if d.Summary() == "Missing QuantCDN API bearer token" || d.Summary() == "Missing QuantCDN organization" {
			errCount++
		}
	}
	if errCount != 2 {
		t.Errorf("expected 2 specific errors (bearer + org), found %d", errCount)
	}
}

func TestUnitProviderConfigure_ValidBearerAndOrg(t *testing.T) {
	// Clear env vars
	t.Setenv("QUANTCDN_API_TOKEN", "")
	t.Setenv("QUANTCDN_ORGANIZATION", "")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	bearer := "test-token-123"
	org := "test-org"
	configVal := newProviderConfigVal(&bearer, &org, nil)

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if configResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", configResp.Diagnostics)
	}

	if configResp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set after successful configure")
	}
	if configResp.ResourceData == nil {
		t.Error("expected ResourceData to be set after successful configure")
	}
}

func TestUnitProviderConfigure_EnvVarFallback(t *testing.T) {
	t.Setenv("QUANTCDN_API_TOKEN", "env-token")
	t.Setenv("QUANTCDN_ORGANIZATION", "env-org")
	t.Setenv("QUANTCDN_BASE_URL", "https://custom.api.example.com")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	// All null config values -- should fall back to env vars
	configVal := newProviderConfigVal(nil, nil, nil)

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if configResp.Diagnostics.HasError() {
		t.Fatalf("expected no errors with env var fallback, got: %s", configResp.Diagnostics)
	}

	if configResp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set from env var fallback")
	}
}

func TestUnitProviderConfigure_UnknownBearer(t *testing.T) {
	t.Setenv("QUANTCDN_API_TOKEN", "")
	t.Setenv("QUANTCDN_ORGANIZATION", "")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	// Create config with unknown bearer value
	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"bearer":              tftypes.String,
			"organization":        tftypes.String,
			"base_url":            tftypes.String,
			"requests_per_second": tftypes.Number,
			"max_retries":         tftypes.Number,
			"base_delay_ms":       tftypes.Number,
			"max_delay_ms":        tftypes.Number,
			"enable_jitter":       tftypes.Bool,
			"timeout_seconds":     tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"bearer":              tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"organization":        tftypes.NewValue(tftypes.String, "test-org"),
		"base_url":            tftypes.NewValue(tftypes.String, nil),
		"requests_per_second": tftypes.NewValue(tftypes.Number, nil),
		"max_retries":         tftypes.NewValue(tftypes.Number, nil),
		"base_delay_ms":       tftypes.NewValue(tftypes.Number, nil),
		"max_delay_ms":        tftypes.NewValue(tftypes.Number, nil),
		"enable_jitter":       tftypes.NewValue(tftypes.Bool, nil),
		"timeout_seconds":     tftypes.NewValue(tftypes.Number, nil),
	})

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Fatal("expected error for unknown bearer token, got none")
	}

	found := false
	for _, d := range configResp.Diagnostics.Errors() {
		if d.Summary() == "Unknown QuantCDN API bearer token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Unknown QuantCDN API bearer token' error not found")
	}
}

func TestUnitProviderConfigure_UnknownOrg(t *testing.T) {
	t.Setenv("QUANTCDN_API_TOKEN", "")
	t.Setenv("QUANTCDN_ORGANIZATION", "")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	configVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"bearer":              tftypes.String,
			"organization":        tftypes.String,
			"base_url":            tftypes.String,
			"requests_per_second": tftypes.Number,
			"max_retries":         tftypes.Number,
			"base_delay_ms":       tftypes.Number,
			"max_delay_ms":        tftypes.Number,
			"enable_jitter":       tftypes.Bool,
			"timeout_seconds":     tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"bearer":              tftypes.NewValue(tftypes.String, "test-token"),
		"organization":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"base_url":            tftypes.NewValue(tftypes.String, nil),
		"requests_per_second": tftypes.NewValue(tftypes.Number, nil),
		"max_retries":         tftypes.NewValue(tftypes.Number, nil),
		"base_delay_ms":       tftypes.NewValue(tftypes.Number, nil),
		"max_delay_ms":        tftypes.NewValue(tftypes.Number, nil),
		"enable_jitter":       tftypes.NewValue(tftypes.Bool, nil),
		"timeout_seconds":     tftypes.NewValue(tftypes.Number, nil),
	})

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Fatal("expected error for unknown organization, got none")
	}

	found := false
	for _, d := range configResp.Diagnostics.Errors() {
		if d.Summary() == "Unknown QuantCDN organization" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Unknown QuantCDN organization' error not found")
	}
}

func TestUnitProviderConfigure_ConfigOverridesEnvVars(t *testing.T) {
	t.Setenv("QUANTCDN_API_TOKEN", "env-token")
	t.Setenv("QUANTCDN_ORGANIZATION", "env-org")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	// Config values should override env vars
	bearer := "config-token"
	org := "config-org"
	configVal := newProviderConfigVal(&bearer, &org, nil)

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if configResp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", configResp.Diagnostics)
	}

	if configResp.DataSourceData == nil {
		t.Error("expected DataSourceData to be set")
	}
}

func TestUnitProviderConfigure_MissingBearerOnly(t *testing.T) {
	t.Setenv("QUANTCDN_API_TOKEN", "")
	t.Setenv("QUANTCDN_ORGANIZATION", "")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	org := "test-org"
	configVal := newProviderConfigVal(nil, &org, nil)

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Fatal("expected error for missing bearer, got none")
	}
}

func TestUnitProviderConfigure_MissingOrgOnly(t *testing.T) {
	t.Setenv("QUANTCDN_API_TOKEN", "")
	t.Setenv("QUANTCDN_ORGANIZATION", "")

	p := &quantProvider{}
	schemaResp := getProviderSchema()

	bearer := "test-token"
	configVal := newProviderConfigVal(&bearer, nil, nil)

	configReq := fwprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configVal,
		},
	}
	configResp := &fwprovider.ConfigureResponse{}
	p.Configure(context.Background(), configReq, configResp)

	if !configResp.Diagnostics.HasError() {
		t.Fatal("expected error for missing org, got none")
	}
}

// ===================================================================
// 3. extractAPIErrorMessage
// ===================================================================

// unitErrReader is a reader that always returns an error.
type unitErrReader struct{}

func (unitErrReader) Read([]byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestUnitExtractAPIErrorMessage_NilResponse(t *testing.T) {
	msg := extractAPIErrorMessage(nil, fmt.Errorf("fallback error"))
	if msg != "fallback error" {
		t.Errorf("expected 'fallback error', got %q", msg)
	}
}

func TestUnitExtractAPIErrorMessage_NilBody(t *testing.T) {
	resp := &http.Response{Body: nil}
	msg := extractAPIErrorMessage(resp, fmt.Errorf("fallback error"))
	if msg != "fallback error" {
		t.Errorf("expected 'fallback error', got %q", msg)
	}
}

func TestUnitExtractAPIErrorMessage_UnreadableBody(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(unitErrReader{}),
	}
	msg := extractAPIErrorMessage(resp, fmt.Errorf("fallback error"))
	if msg != "fallback error" {
		t.Errorf("expected 'fallback error', got %q", msg)
	}
}

func TestUnitExtractAPIErrorMessage_InvalidJSON(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(bytes.NewBufferString("not json at all")),
	}
	msg := extractAPIErrorMessage(resp, fmt.Errorf("fallback error"))
	if msg != "fallback error" {
		t.Errorf("expected 'fallback error', got %q", msg)
	}
}

func TestUnitExtractAPIErrorMessage_ValidJSONWithMessage(t *testing.T) {
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(`{"error":true,"message":"bad request details"}`)),
		StatusCode: 400,
	}
	msg := extractAPIErrorMessage(resp, fmt.Errorf("generic error"))
	if msg != "bad request details" {
		t.Errorf("expected 'bad request details', got %q", msg)
	}
}

func TestUnitExtractAPIErrorMessage_ValidJSONEmptyMessage(t *testing.T) {
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(`{"error":true,"message":""}`)),
		StatusCode: 400,
	}
	msg := extractAPIErrorMessage(resp, fmt.Errorf("fallback error"))
	if msg != "fallback error" {
		t.Errorf("expected 'fallback error', got %q", msg)
	}
}

func TestUnitExtractAPIErrorMessage_ValidJSONNoMessage(t *testing.T) {
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(`{"error":true}`)),
		StatusCode: 400,
	}
	msg := extractAPIErrorMessage(resp, fmt.Errorf("fallback error"))
	if msg != "fallback error" {
		t.Errorf("expected 'fallback error', got %q", msg)
	}
}

// ===================================================================
// 4. RuleBaseConfigValidator and RuleBaseAttributes
// ===================================================================

func TestUnitRuleBaseConfigValidator(t *testing.T) {
	validators := RuleBaseConfigValidator()
	if validators == nil {
		t.Fatal("RuleBaseConfigValidator returned nil")
	}
	// Currently returns an empty slice
	if len(validators) != 0 {
		t.Errorf("expected 0 validators, got %d", len(validators))
	}
}

func TestUnitRuleBaseAttributes(t *testing.T) {
	ctx := context.Background()
	attrs := RuleBaseAttributes(ctx)

	expectedKeys := []string{
		"organization", "project", "name", "uuid", "rule_id",
		"weight", "url", "domain", "disabled", "only_with_cookie",
		"method", "method_is", "method_is_not",
		"ip", "ip_is", "ip_is_not",
		"country", "country_is", "country_is_not",
	}

	for _, key := range expectedKeys {
		if _, ok := attrs[key]; !ok {
			t.Errorf("missing expected attribute: %s", key)
		}
	}

	// Verify the total count matches
	if len(attrs) != len(expectedKeys) {
		t.Errorf("expected %d attributes, got %d", len(expectedKeys), len(attrs))
	}
}

// ===================================================================
// 5. Project data source
// ===================================================================

func TestUnitProjectDataSource_Metadata(t *testing.T) {
	ds := NewProjectDataSource()
	resp := &fwdatasource.MetadataResponse{}
	ds.Metadata(context.Background(), fwdatasource.MetadataRequest{ProviderTypeName: "quant"}, resp)

	if resp.TypeName != "quant_project" {
		t.Errorf("expected TypeName 'quant_project', got %q", resp.TypeName)
	}
}

func TestUnitProjectDataSource_Schema(t *testing.T) {
	ds := NewProjectDataSource()
	resp := &fwdatasource.SchemaResponse{}
	ds.Metadata(context.Background(), fwdatasource.MetadataRequest{ProviderTypeName: "quant"}, &fwdatasource.MetadataResponse{})
	ds.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %s", resp.Diagnostics)
	}

	expectedAttrs := []string{
		"machine_name", "with_token", "id", "name", "uuid",
		"created_at", "updated_at", "region", "organization_id",
		"security_score", "git_url", "write_token",
	}
	for _, attr := range expectedAttrs {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing expected project data source attribute: %s", attr)
		}
	}
}

// ===================================================================
// 6. Projects data source
// ===================================================================

func TestUnitProjectsDataSource_Metadata(t *testing.T) {
	ds := NewProjectsDataSource()
	resp := &fwdatasource.MetadataResponse{}
	ds.Metadata(context.Background(), fwdatasource.MetadataRequest{ProviderTypeName: "quant"}, resp)

	if resp.TypeName != "quant_projects" {
		t.Errorf("expected TypeName 'quant_projects', got %q", resp.TypeName)
	}
}

func TestUnitProjectsDataSource_Schema(t *testing.T) {
	ds := NewProjectsDataSource()
	resp := &fwdatasource.SchemaResponse{}
	ds.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %s", resp.Diagnostics)
	}

	if _, ok := resp.Schema.Attributes["projects"]; !ok {
		t.Error("missing 'projects' attribute in projects data source schema")
	}
}

// ===================================================================
// 7. Crawler resource helper functions
// ===================================================================

func TestUnitNullableInt64_Present(t *testing.T) {
	result := nullableInt64(42, true)
	if result.IsNull() {
		t.Fatal("expected non-null value")
	}
	if result.ValueInt64() != 42 {
		t.Errorf("expected 42, got %d", result.ValueInt64())
	}
}

func TestUnitNullableInt64_Absent(t *testing.T) {
	result := nullableInt64(42, false)
	if !result.IsNull() {
		t.Error("expected null value when present=false")
	}
}

func TestUnitNullableInt64_Zero(t *testing.T) {
	result := nullableInt64(0, true)
	if result.IsNull() {
		t.Fatal("expected non-null value")
	}
	if result.ValueInt64() != 0 {
		t.Errorf("expected 0, got %d", result.ValueInt64())
	}
}

func TestUnitNullableFloat64_Present(t *testing.T) {
	result := nullableFloat64(3.14, true)
	if result.IsNull() {
		t.Fatal("expected non-null value")
	}
	if result.ValueFloat64() != 3.14 {
		t.Errorf("expected 3.14, got %f", result.ValueFloat64())
	}
}

func TestUnitNullableFloat64_Absent(t *testing.T) {
	result := nullableFloat64(3.14, false)
	if !result.IsNull() {
		t.Error("expected null value when present=false")
	}
}

func TestUnitNullableFloat64_Zero(t *testing.T) {
	result := nullableFloat64(0.0, true)
	if result.IsNull() {
		t.Fatal("expected non-null value")
	}
	if result.ValueFloat64() != 0.0 {
		t.Errorf("expected 0.0, got %f", result.ValueFloat64())
	}
}

func TestUnitStringListOrPreserve_NonEmptySlice(t *testing.T) {
	existing := types.ListNull(types.StringType)
	result := stringListOrPreserve([]string{"a", "b", "c"}, existing)

	if result.IsNull() {
		t.Fatal("expected non-null list")
	}

	elems := result.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	if elems[0].(types.String).ValueString() != "a" {
		t.Errorf("expected 'a', got %q", elems[0].(types.String).ValueString())
	}
	if elems[1].(types.String).ValueString() != "b" {
		t.Errorf("expected 'b', got %q", elems[1].(types.String).ValueString())
	}
	if elems[2].(types.String).ValueString() != "c" {
		t.Errorf("expected 'c', got %q", elems[2].(types.String).ValueString())
	}
}

func TestUnitStringListOrPreserve_EmptySlice_ExistingNull(t *testing.T) {
	existing := types.ListNull(types.StringType)
	result := stringListOrPreserve([]string{}, existing)

	if result.IsNull() {
		t.Fatal("expected non-null empty list, got null")
	}
	if len(result.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result.Elements()))
	}
}

func TestUnitStringListOrPreserve_EmptySlice_ExistingUnknown(t *testing.T) {
	existing := types.ListUnknown(types.StringType)
	result := stringListOrPreserve([]string{}, existing)

	if result.IsNull() || result.IsUnknown() {
		t.Fatal("expected non-null non-unknown empty list")
	}
	if len(result.Elements()) != 0 {
		t.Errorf("expected 0 elements, got %d", len(result.Elements()))
	}
}

func TestUnitStringListOrPreserve_EmptySlice_PreservesExisting(t *testing.T) {
	ctx := context.Background()
	existing, diags := types.ListValueFrom(ctx, types.StringType, []string{"preserved"})
	if diags.HasError() {
		t.Fatalf("setup error: %s", diags)
	}

	result := stringListOrPreserve([]string{}, existing)

	if result.IsNull() || result.IsUnknown() {
		t.Fatal("expected non-null non-unknown list")
	}
	elems := result.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 preserved element, got %d", len(elems))
	}
	if elems[0].(types.String).ValueString() != "preserved" {
		t.Errorf("expected 'preserved', got %q", elems[0].(types.String).ValueString())
	}
}

func TestUnitParseCacheLifetime_Valid(t *testing.T) {
	val, err := parseCacheLifetime(types.StringValue("3600"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 3600 {
		t.Errorf("expected 3600, got %d", val)
	}
}

func TestUnitParseCacheLifetime_Zero(t *testing.T) {
	val, err := parseCacheLifetime(types.StringValue("0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}
}

func TestUnitParseCacheLifetime_Null(t *testing.T) {
	_, err := parseCacheLifetime(types.StringNull())
	if err == nil {
		t.Fatal("expected error for null value, got nil")
	}
}

func TestUnitParseCacheLifetime_Unknown(t *testing.T) {
	_, err := parseCacheLifetime(types.StringUnknown())
	if err == nil {
		t.Fatal("expected error for unknown value, got nil")
	}
}

func TestUnitParseCacheLifetime_Empty(t *testing.T) {
	_, err := parseCacheLifetime(types.StringValue(""))
	if err == nil {
		t.Fatal("expected error for empty string, got nil")
	}
}

func TestUnitParseCacheLifetime_NonNumeric(t *testing.T) {
	_, err := parseCacheLifetime(types.StringValue("not-a-number"))
	if err == nil {
		t.Fatal("expected error for non-numeric string, got nil")
	}
}

func TestUnitFormatCacheLifetime(t *testing.T) {
	result := formatCacheLifetime("3600")
	if result.ValueString() != "3600" {
		t.Errorf("expected '3600', got %q", result.ValueString())
	}

	result = formatCacheLifetime("0")
	if result.ValueString() != "0" {
		t.Errorf("expected '0', got %q", result.ValueString())
	}

	result = formatCacheLifetime("")
	if result.ValueString() != "" {
		t.Errorf("expected empty string, got %q", result.ValueString())
	}
}

// ===================================================================
// 8. strPtr utility function
// ===================================================================

func TestUnitStrPtr(t *testing.T) {
	s := "hello"
	ptr := strPtr(s)
	if ptr == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *ptr != "hello" {
		t.Errorf("expected 'hello', got %q", *ptr)
	}

	// Empty string
	ptr = strPtr("")
	if ptr == nil {
		t.Fatal("expected non-nil pointer for empty string")
	}
	if *ptr != "" {
		t.Errorf("expected empty string, got %q", *ptr)
	}
}

// ===================================================================
// 9. mapCronResponse and mapVolumeResponse
// ===================================================================

func TestUnitMapCronResponse_AllFieldsSet(t *testing.T) {
	name := "my-cron"
	schedule := "0 * * * *"
	command := []string{"echo", "hello"}

	cron := &quantadmingo.Cron{
		Name:               &name,
		ScheduleExpression: &schedule,
		Command:            command,
	}

	data := &resource_cron_job.CronJobModel{
		Description:         types.StringUnknown(),
		TargetContainerName: types.StringUnknown(),
		ScheduleExpression:  types.StringUnknown(),
		Cron:                types.StringUnknown(),
		IsEnabled:           types.BoolUnknown(),
	}
	mapCronResponse(cron, data)

	if data.Name.ValueString() != "my-cron" {
		t.Errorf("expected name 'my-cron', got %q", data.Name.ValueString())
	}
	if data.ScheduleExpression.ValueString() != "0 * * * *" {
		t.Errorf("expected schedule_expression '0 * * * *', got %q", data.ScheduleExpression.ValueString())
	}
	if data.Cron.ValueString() != "0 * * * *" {
		t.Errorf("expected cron '0 * * * *', got %q", data.Cron.ValueString())
	}
	// Command is now a types.List
	var cmdElems []string
	data.Command.ElementsAs(context.Background(), &cmdElems, false)
	if len(cmdElems) != 2 || cmdElems[0] != "echo" || cmdElems[1] != "hello" {
		t.Errorf("expected command [echo, hello], got %v", cmdElems)
	}
	// Unknown fields should be resolved to null
	if !data.Description.IsNull() {
		t.Error("expected Description to be null after mapping (was unknown)")
	}
	if !data.TargetContainerName.IsNull() {
		t.Error("expected TargetContainerName to be null after mapping (was unknown)")
	}
	if !data.IsEnabled.IsNull() {
		t.Error("expected IsEnabled to be null after mapping (was unknown, API returned nil)")
	}
}

func TestUnitMapCronResponse_NilSchedule(t *testing.T) {
	name := "cron-no-schedule"
	cron := &quantadmingo.Cron{
		Name: &name,
	}

	data := &resource_cron_job.CronJobModel{
		Description:         types.StringValue("existing"),
		TargetContainerName: types.StringValue("existing"),
		ScheduleExpression:  types.StringValue("existing"),
	}
	mapCronResponse(cron, data)

	if data.Name.ValueString() != "cron-no-schedule" {
		t.Errorf("expected name 'cron-no-schedule', got %q", data.Name.ValueString())
	}
	if !data.Cron.IsNull() {
		t.Error("expected Cron to be null when API returns nil")
	}
	// Non-unknown fields should be preserved as-is
	if data.Description.ValueString() != "existing" {
		t.Errorf("expected existing Description to be preserved, got %q", data.Description.ValueString())
	}
}

func TestUnitMapCronResponse_AllNilFields(t *testing.T) {
	cron := &quantadmingo.Cron{}

	data := &resource_cron_job.CronJobModel{
		Description:         types.StringUnknown(),
		TargetContainerName: types.StringUnknown(),
		ScheduleExpression:  types.StringUnknown(),
		Cron:                types.StringUnknown(),
		IsEnabled:           types.BoolUnknown(),
	}
	mapCronResponse(cron, data)

	if !data.Cron.IsNull() {
		t.Error("expected Cron to be null")
	}
	if !data.IsEnabled.IsNull() {
		t.Error("expected IsEnabled to be null")
	}
	if !data.Description.IsNull() {
		t.Error("expected Description to be null")
	}
	if !data.TargetContainerName.IsNull() {
		t.Error("expected TargetContainerName to be null")
	}
	if !data.ScheduleExpression.IsNull() {
		t.Error("expected ScheduleExpression to be null")
	}
}

func TestUnitMapVolumeResponse_AllFieldsSet(t *testing.T) {
	volId := "vol-123"
	volName := "my-volume"
	desc := "test volume"
	efsId := "fs-abc"
	rootDir := "/data"
	apId := "fsap-123"
	apArn := "arn:aws:elasticfilesystem:us-east-1:123456789012:access-point/fsap-123"
	createdAt := "2025-01-01T00:00:00Z"

	vol := &quantadmingo.Volume{
		VolumeId:         &volId,
		VolumeName:       &volName,
		Description:      &desc,
		EnvironmentEfsId: &efsId,
		RootDirectory:    &rootDir,
		AccessPointId:    &apId,
		AccessPointArn:   &apArn,
		CreatedAt:        &createdAt,
	}

	data := &resource_volume.VolumeModel{}
	mapVolumeResponse(vol, data)

	if data.VolumeId.ValueString() != "vol-123" {
		t.Errorf("expected VolumeId 'vol-123', got %q", data.VolumeId.ValueString())
	}
	if data.VolumeName.ValueString() != "my-volume" {
		t.Errorf("expected VolumeName 'my-volume', got %q", data.VolumeName.ValueString())
	}
	if data.Description.ValueString() != "test volume" {
		t.Errorf("expected Description 'test volume', got %q", data.Description.ValueString())
	}
	if data.EnvironmentEfsId.ValueString() != "fs-abc" {
		t.Errorf("expected EnvironmentEfsId 'fs-abc', got %q", data.EnvironmentEfsId.ValueString())
	}
	if data.RootDirectory.ValueString() != "/data" {
		t.Errorf("expected RootDirectory '/data', got %q", data.RootDirectory.ValueString())
	}
	if data.AccessPointId.ValueString() != "fsap-123" {
		t.Errorf("expected AccessPointId 'fsap-123', got %q", data.AccessPointId.ValueString())
	}
	if data.AccessPointArn.ValueString() != apArn {
		t.Errorf("expected AccessPointArn %q, got %q", apArn, data.AccessPointArn.ValueString())
	}
	if data.CreatedAt.ValueString() != "2025-01-01T00:00:00Z" {
		t.Errorf("expected CreatedAt '2025-01-01T00:00:00Z', got %q", data.CreatedAt.ValueString())
	}
}

func TestUnitMapVolumeResponse_AllNilFields(t *testing.T) {
	vol := &quantadmingo.Volume{}

	data := &resource_volume.VolumeModel{
		Description:   types.StringUnknown(),
		RootDirectory: types.StringUnknown(),
	}
	mapVolumeResponse(vol, data)

	if !data.VolumeId.IsNull() {
		t.Error("expected VolumeId to be null")
	}
	if !data.EnvironmentEfsId.IsNull() {
		t.Error("expected EnvironmentEfsId to be null")
	}
	if !data.AccessPointId.IsNull() {
		t.Error("expected AccessPointId to be null")
	}
	if !data.AccessPointArn.IsNull() {
		t.Error("expected AccessPointArn to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Error("expected CreatedAt to be null")
	}
	// Description and RootDirectory should be null since they were unknown
	if !data.Description.IsNull() {
		t.Error("expected Description to be null when API returns nil and model was unknown")
	}
	if !data.RootDirectory.IsNull() {
		t.Error("expected RootDirectory to be null when API returns nil and model was unknown")
	}
}

func TestUnitMapVolumeResponse_PartialFields(t *testing.T) {
	volId := "vol-456"
	volName := "partial-volume"

	vol := &quantadmingo.Volume{
		VolumeId:   &volId,
		VolumeName: &volName,
	}

	data := &resource_volume.VolumeModel{
		Description:   types.StringValue("keep-me"),
		RootDirectory: types.StringValue("keep-me-too"),
	}
	mapVolumeResponse(vol, data)

	if data.VolumeId.ValueString() != "vol-456" {
		t.Errorf("expected VolumeId 'vol-456', got %q", data.VolumeId.ValueString())
	}
	if data.VolumeName.ValueString() != "partial-volume" {
		t.Errorf("expected VolumeName 'partial-volume', got %q", data.VolumeName.ValueString())
	}
	// Description and RootDirectory should be preserved since they are not unknown
	if data.Description.ValueString() != "keep-me" {
		t.Errorf("expected Description 'keep-me', got %q", data.Description.ValueString())
	}
	if data.RootDirectory.ValueString() != "keep-me-too" {
		t.Errorf("expected RootDirectory 'keep-me-too', got %q", data.RootDirectory.ValueString())
	}
	if !data.EnvironmentEfsId.IsNull() {
		t.Error("expected EnvironmentEfsId to be null")
	}
	if !data.AccessPointId.IsNull() {
		t.Error("expected AccessPointId to be null")
	}
	if !data.AccessPointArn.IsNull() {
		t.Error("expected AccessPointArn to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Error("expected CreatedAt to be null")
	}
}

func TestUnitMapVolumeResponse_NilVolumeIdSetsNull(t *testing.T) {
	volName := "name-only"
	vol := &quantadmingo.Volume{
		VolumeName: &volName,
	}

	data := &resource_volume.VolumeModel{
		VolumeId: types.StringValue("old-id"),
	}
	mapVolumeResponse(vol, data)

	if !data.VolumeId.IsNull() {
		t.Error("expected VolumeId to be set to null when API returns nil")
	}
}

// ===================================================================
// Additional coverage: New() factory function
// ===================================================================

func TestUnitNewFactory(t *testing.T) {
	factory := New()
	if factory == nil {
		t.Fatal("New() returned nil factory")
	}
	p := factory()
	if p == nil {
		t.Fatal("factory function returned nil provider")
	}
}

// ===================================================================
// Additional coverage: Resource Metadata tests
// ===================================================================

func TestUnitProjectResourceMetadata(t *testing.T) {
	r := NewProjectResource()
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "quant"}, resp)
	if resp.TypeName != "quant_project" {
		t.Errorf("expected TypeName 'quant_project', got %q", resp.TypeName)
	}
}

func TestUnitCrawlerResourceMetadata(t *testing.T) {
	r := NewCrawlerResource()
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "quant"}, resp)
	if resp.TypeName != "quant_crawler" {
		t.Errorf("expected TypeName 'quant_crawler', got %q", resp.TypeName)
	}
}

func TestUnitCrawlerScheduleResourceMetadata(t *testing.T) {
	r := NewCrawlerScheduleResource()
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "quant"}, resp)
	if resp.TypeName != "quant_crawler_schedule" {
		t.Errorf("expected TypeName 'quant_crawler_schedule', got %q", resp.TypeName)
	}
}

func TestUnitVolumeResourceMetadata(t *testing.T) {
	r := NewVolumeResource()
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "quant"}, resp)
	if resp.TypeName != "quant_volume" {
		t.Errorf("expected TypeName 'quant_volume', got %q", resp.TypeName)
	}
}

func TestUnitCronJobResourceMetadata(t *testing.T) {
	r := NewCronJobResource()
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "quant"}, resp)
	if resp.TypeName != "quant_cron_job" {
		t.Errorf("expected TypeName 'quant_cron_job', got %q", resp.TypeName)
	}
}

// ===================================================================
// Resource Configure with nil ProviderData (early return path)
// ===================================================================

func TestUnitProjectResourceConfigure_NilProviderData(t *testing.T) {
	r := &projectResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil ProviderData: %s", resp.Diagnostics)
	}
	if r.client != nil {
		t.Error("expected client to remain nil")
	}
}

func TestUnitCrawlerResourceConfigure_NilProviderData(t *testing.T) {
	r := &crawlerResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil ProviderData: %s", resp.Diagnostics)
	}
	if r.client != nil {
		t.Error("expected client to remain nil")
	}
}

func TestUnitVolumeResourceConfigure_NilProviderData(t *testing.T) {
	r := &volumeResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil ProviderData: %s", resp.Diagnostics)
	}
	if r.client != nil {
		t.Error("expected client to remain nil")
	}
}

func TestUnitCronJobResourceConfigure_NilProviderData(t *testing.T) {
	r := &cronJobResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil ProviderData: %s", resp.Diagnostics)
	}
	if r.client != nil {
		t.Error("expected client to remain nil")
	}
}

// ===================================================================
// Resource Configure with wrong type (error path)
// ===================================================================

func TestUnitProjectResourceConfigure_WrongType(t *testing.T) {
	r := &projectResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "wrong-type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestUnitCrawlerResourceConfigure_WrongType(t *testing.T) {
	r := &crawlerResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "wrong-type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestUnitVolumeResourceConfigure_WrongType(t *testing.T) {
	r := &volumeResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "wrong-type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestUnitCronJobResourceConfigure_WrongType(t *testing.T) {
	r := &cronJobResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: "wrong-type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// ===================================================================
// DataSource Configure with nil and wrong type
// ===================================================================

func TestUnitProjectsDataSourceConfigure_NilProviderData(t *testing.T) {
	ds := &projectsDataSource{}
	resp := &fwdatasource.ConfigureResponse{}
	ds.Configure(context.Background(), fwdatasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil ProviderData: %s", resp.Diagnostics)
	}
}

func TestUnitProjectsDataSourceConfigure_WrongType(t *testing.T) {
	ds := &projectsDataSource{}
	resp := &fwdatasource.ConfigureResponse{}
	ds.Configure(context.Background(), fwdatasource.ConfigureRequest{ProviderData: "wrong-type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestUnitProjectDataSourceConfigure_NilProviderData(t *testing.T) {
	ds := &projectDataSource{}
	resp := &fwdatasource.ConfigureResponse{}
	ds.Configure(context.Background(), fwdatasource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil ProviderData: %s", resp.Diagnostics)
	}
}

func TestUnitProjectDataSourceConfigure_WrongType(t *testing.T) {
	ds := &projectDataSource{}
	resp := &fwdatasource.ConfigureResponse{}
	ds.Configure(context.Background(), fwdatasource.ConfigureRequest{ProviderData: "wrong-type"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// ===================================================================
// CrawlerSchedule resource Configure paths
// ===================================================================

func TestUnitCrawlerScheduleResourceConfigure_NilProviderData(t *testing.T) {
	r := &crawlerScheduleResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: nil}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics)
	}
}

func TestUnitCrawlerScheduleResourceConfigure_WrongType(t *testing.T) {
	r := &crawlerScheduleResource{}
	resp := &fwresource.ConfigureResponse{}
	r.Configure(context.Background(), fwresource.ConfigureRequest{ProviderData: 12345}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// ===================================================================
// Volume resource Update (not supported)
// ===================================================================

func TestUnitVolumeResourceUpdate_NotSupported(t *testing.T) {
	r := &volumeResource{}
	resp := &fwresource.UpdateResponse{}
	r.Update(context.Background(), fwresource.UpdateRequest{}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for volume update (not supported)")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Update Not Supported" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Update Not Supported' error not found")
	}
}

// ===================================================================
// generateID
// ===================================================================

func TestUnitGenerateID(t *testing.T) {
	t.Run("same input same hash", func(t *testing.T) {
		h1 := generateID(map[string]string{"X-Foo": "bar"})
		h2 := generateID(map[string]string{"X-Foo": "bar"})
		if h1 != h2 {
			t.Errorf("expected same hash, got %q vs %q", h1, h2)
		}
	})

	t.Run("different input different hash", func(t *testing.T) {
		h1 := generateID(map[string]string{"X-Foo": "bar"})
		h2 := generateID(map[string]string{"X-Foo": "baz"})
		if h1 == h2 {
			t.Error("expected different hashes for different values")
		}
	})

	t.Run("empty map valid hash", func(t *testing.T) {
		h := generateID(map[string]string{})
		if len(h) != 64 { // sha256 hex = 64 chars
			t.Errorf("expected 64-char hex hash, got len %d: %q", len(h), h)
		}
	})

	t.Run("order independent", func(t *testing.T) {
		h1 := generateID(map[string]string{"A": "1", "B": "2"})
		h2 := generateID(map[string]string{"B": "2", "A": "1"})
		if h1 != h2 {
			t.Errorf("expected same hash regardless of insertion order, got %q vs %q", h1, h2)
		}
	})
}

// ===================================================================
// extractHeadersMap
// ===================================================================

func TestUnitExtractHeadersMap(t *testing.T) {
	ctx := context.Background()

	t.Run("valid map", func(t *testing.T) {
		mapVal, d := types.MapValueFrom(ctx, types.StringType, map[string]string{
			"X-Custom": "value1",
			"X-Other":  "value2",
		})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		model := &headerResourceModel{Headers: mapVal}
		result, diags := extractHeadersMap(model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if result["X-Custom"] != "value1" || result["X-Other"] != "value2" {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		mapVal, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		model := &headerResourceModel{Headers: mapVal}
		result, diags := extractHeadersMap(model)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("non-string element type causes error", func(t *testing.T) {
		// Build a map with a non-string attr.Value to trigger the type assertion failure
		badMap, d := types.MapValue(types.Int64Type, map[string]attr.Value{
			"key": types.Int64Value(42),
		})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		model := &headerResourceModel{Headers: badMap}
		_, diags := extractHeadersMap(model)
		if !diags.HasError() {
			t.Error("expected error for non-string element type")
		}
	})
}

// ===================================================================
// headersMapToSDK
// ===================================================================

func TestUnitHeadersMapToSDK(t *testing.T) {
	ctx := context.Background()

	t.Run("valid map", func(t *testing.T) {
		mapVal, d := types.MapValueFrom(ctx, types.StringType, map[string]string{
			"X-Foo": "bar",
		})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		result, diags := headersMapToSDK(ctx, mapVal)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if result["X-Foo"] != "bar" {
			t.Errorf("expected X-Foo=bar, got %v", result)
		}
	})

	t.Run("null map returns empty", func(t *testing.T) {
		result, diags := headersMapToSDK(ctx, types.MapNull(types.StringType))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if len(result) != 0 {
			t.Errorf("expected empty map for null, got %v", result)
		}
	})

	t.Run("unknown map returns empty", func(t *testing.T) {
		result, diags := headersMapToSDK(ctx, types.MapUnknown(types.StringType))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if len(result) != 0 {
			t.Errorf("expected empty map for unknown, got %v", result)
		}
	})

	t.Run("empty map returns empty", func(t *testing.T) {
		mapVal, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		result, diags := headersMapToSDK(ctx, mapVal)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})
}

// ===================================================================
// setWafStringList
// ===================================================================

func TestUnitSetWafStringList(t *testing.T) {
	ctx := context.Background()

	t.Run("null list skips setter", func(t *testing.T) {
		called := false
		var diags diag.Diagnostics
		setWafStringList(ctx, types.ListNull(types.StringType), func([]string) { called = true }, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if called {
			t.Error("setter should not be called for null list")
		}
	})

	t.Run("unknown list skips setter", func(t *testing.T) {
		called := false
		var diags diag.Diagnostics
		setWafStringList(ctx, types.ListUnknown(types.StringType), func([]string) { called = true }, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if called {
			t.Error("setter should not be called for unknown list")
		}
	})

	t.Run("valid list calls setter", func(t *testing.T) {
		list, d := types.ListValueFrom(ctx, types.StringType, []string{"a", "b"})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		var got []string
		var diags diag.Diagnostics
		setWafStringList(ctx, list, func(v []string) { got = v }, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("expected [a b], got %v", got)
		}
	})

	t.Run("empty list calls setter with empty slice", func(t *testing.T) {
		list, d := types.ListValueFrom(ctx, types.StringType, []string{})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		called := false
		var diags diag.Diagnostics
		setWafStringList(ctx, list, func(v []string) { called = true }, &diags)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		// extractStringList returns non-nil []string{} for empty list, so setter IS called
		if !called {
			t.Error("setter should be called for empty (non-null) list")
		}
	})
}

// ===================================================================
// setStringListFromAPI
// ===================================================================

func TestUnitSetStringListFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("nil apiVal sets null list", func(t *testing.T) {
		var target types.List
		diags := setStringListFromAPI(ctx, nil, &target)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if !target.IsNull() {
			t.Error("expected null list for nil apiVal")
		}
	})

	t.Run("non-nil apiVal sets list value", func(t *testing.T) {
		var target types.List
		diags := setStringListFromAPI(ctx, []string{"x", "y"}, &target)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if target.IsNull() || target.IsUnknown() {
			t.Fatal("expected non-null list")
		}
		elems, d := extractStringList(ctx, target)
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		if len(elems) != 2 || elems[0] != "x" || elems[1] != "y" {
			t.Errorf("expected [x y], got %v", elems)
		}
	})

	t.Run("empty slice sets empty list", func(t *testing.T) {
		var target types.List
		diags := setStringListFromAPI(ctx, []string{}, &target)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if target.IsNull() {
			t.Error("expected non-null list for empty slice")
		}
		elems, d := extractStringList(ctx, target)
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		if len(elems) != 0 {
			t.Errorf("expected empty list, got %v", elems)
		}
	})
}
