package mapper

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type testTFModel struct {
	Name             types.String  `tfsdk:"name"`
	Region           types.String  `tfsdk:"region"`
	AllowQueryParams types.Bool    `tfsdk:"allow_query_params"`
	Weight           types.Int64   `tfsdk:"weight"`
	Disabled         types.Bool    `tfsdk:"disabled"`
	Score            types.Float64 `tfsdk:"score"`
}

type testSDKRequest struct {
	name             *string
	region           *string
	allowQueryParams *bool
	weight           *int32
	disabled         *bool
	score            *float64
}

func (r *testSDKRequest) SetName(v string)             { r.name = &v }
func (r *testSDKRequest) SetRegion(v string)            { r.region = &v }
func (r *testSDKRequest) SetAllowQueryParams(v bool)    { r.allowQueryParams = &v }
func (r *testSDKRequest) SetWeight(v int32)             { r.weight = &v }
func (r *testSDKRequest) SetDisabled(v bool)            { r.disabled = &v }
func (r *testSDKRequest) SetScore(v float64)            { r.score = &v }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestToSDK_MapsStringFields(t *testing.T) {
	model := testTFModel{
		Name:             types.StringValue("my-project"),
		Region:           types.StringValue("us-east-1"),
		AllowQueryParams: types.BoolNull(),
		Weight:           types.Int64Null(),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Null(),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if req.name == nil || *req.name != "my-project" {
		t.Errorf("expected name = %q, got %v", "my-project", req.name)
	}
	if req.region == nil || *req.region != "us-east-1" {
		t.Errorf("expected region = %q, got %v", "us-east-1", req.region)
	}
}

func TestToSDK_MapsBoolFields(t *testing.T) {
	model := testTFModel{
		Name:             types.StringNull(),
		Region:           types.StringNull(),
		AllowQueryParams: types.BoolValue(true),
		Weight:           types.Int64Null(),
		Disabled:         types.BoolValue(false),
		Score:            types.Float64Null(),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if req.allowQueryParams == nil || *req.allowQueryParams != true {
		t.Errorf("expected allowQueryParams = true, got %v", req.allowQueryParams)
	}
	if req.disabled == nil || *req.disabled != false {
		t.Errorf("expected disabled = false, got %v", req.disabled)
	}
}

func TestToSDK_MapsInt64ToInt32(t *testing.T) {
	model := testTFModel{
		Name:             types.StringNull(),
		Region:           types.StringNull(),
		AllowQueryParams: types.BoolNull(),
		Weight:           types.Int64Value(42),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Null(),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if req.weight == nil || *req.weight != 42 {
		t.Errorf("expected weight = 42, got %v", req.weight)
	}
}

func TestToSDK_MapsFloat64Fields(t *testing.T) {
	model := testTFModel{
		Name:             types.StringNull(),
		Region:           types.StringNull(),
		AllowQueryParams: types.BoolNull(),
		Weight:           types.Int64Null(),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Value(3.14),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if req.score == nil || *req.score != 3.14 {
		t.Errorf("expected score = 3.14, got %v", req.score)
	}
}

func TestToSDK_SkipsNullFields(t *testing.T) {
	model := testTFModel{
		Name:             types.StringValue("set-this"),
		Region:           types.StringNull(),
		AllowQueryParams: types.BoolNull(),
		Weight:           types.Int64Null(),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Null(),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if req.name == nil || *req.name != "set-this" {
		t.Errorf("expected name = %q, got %v", "set-this", req.name)
	}

	// Null fields must NOT have their setters called.
	if req.region != nil {
		t.Errorf("expected region to be nil (null field skipped), got %v", *req.region)
	}
	if req.allowQueryParams != nil {
		t.Errorf("expected allowQueryParams to be nil (null field skipped), got %v", *req.allowQueryParams)
	}
	if req.weight != nil {
		t.Errorf("expected weight to be nil (null field skipped), got %v", *req.weight)
	}
	if req.disabled != nil {
		t.Errorf("expected disabled to be nil (null field skipped), got %v", *req.disabled)
	}
}

func TestToSDK_SkipsUnknownFields(t *testing.T) {
	model := testTFModel{
		Name:             types.StringUnknown(),
		Region:           types.StringValue("eu-west-1"),
		AllowQueryParams: types.BoolUnknown(),
		Weight:           types.Int64Unknown(),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Unknown(),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	// Unknown fields must NOT have their setters called.
	if req.name != nil {
		t.Errorf("expected name to be nil (unknown field skipped), got %v", *req.name)
	}
	if req.allowQueryParams != nil {
		t.Errorf("expected allowQueryParams to be nil (unknown field skipped), got %v", *req.allowQueryParams)
	}
	if req.weight != nil {
		t.Errorf("expected weight to be nil (unknown field skipped), got %v", *req.weight)
	}
	if req.score != nil {
		t.Errorf("expected score to be nil (unknown field skipped), got %v", *req.score)
	}

	// The one set field should still be mapped.
	if req.region == nil || *req.region != "eu-west-1" {
		t.Errorf("expected region = %q, got %v", "eu-west-1", req.region)
	}
}

func TestToSDK_SkipsFieldsWithNoSetter(t *testing.T) {
	// Model has a field "extra" that the SDK request has no SetExtra method for.
	type modelWithExtra struct {
		Name  types.String `tfsdk:"name"`
		Extra types.String `tfsdk:"extra"`
	}

	model := modelWithExtra{
		Name:  types.StringValue("hello"),
		Extra: types.StringValue("should-be-ignored"),
	}

	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	// The known field should still be mapped.
	if req.name == nil || *req.name != "hello" {
		t.Errorf("expected name = %q, got %v", "hello", req.name)
	}
}

func TestToSDK_RejectsNonPointerSDKReq(t *testing.T) {
	model := testTFModel{
		Name:             types.StringValue("hello"),
		Region:           types.StringNull(),
		AllowQueryParams: types.BoolNull(),
		Weight:           types.Int64Null(),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Null(),
	}

	// Pass a non-pointer SDK request — should return an error diagnostic.
	req := testSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	if !diags.HasError() {
		t.Fatal("expected error diagnostic when sdkReq is not a pointer, got none")
	}

	found := false
	for _, d := range diags.Errors() {
		if d.Summary() == "ToSDK: sdkReq must be a pointer" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error with summary %q, got: %v", "ToSDK: sdkReq must be a pointer", diags.Errors())
	}
}

// mismatchSDKRequest is a test double whose setter expects int64, creating a
// type mismatch when paired with a types.String TF field.
type mismatchSDKRequest struct {
	name *int64
}

func (r *mismatchSDKRequest) SetName(v int64) { r.name = &v }

func TestToSDK_WarnsOnTypeMismatch(t *testing.T) {
	// TF model has Name as types.String, but the SDK setter expects int64.
	type mismatchModel struct {
		Name types.String `tfsdk:"name"`
	}

	model := mismatchModel{
		Name: types.StringValue("hello"),
	}

	req := &mismatchSDKRequest{}
	diags := ToSDK(context.Background(), model, req)

	// Should NOT have errors (type mismatch is a warning, not an error).
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags.Errors())
	}

	// Should have a warning diagnostic about the type mismatch.
	warnings := diags.Warnings()
	if len(warnings) == 0 {
		t.Fatal("expected a warning diagnostic for type mismatch, got none")
	}

	found := false
	for _, w := range warnings {
		if w.Summary() == `ToSDK: type mismatch for field "name"` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning with summary %q, got: %v", `ToSDK: type mismatch for field "name"`, warnings)
	}

	// The setter should NOT have been called.
	if req.name != nil {
		t.Errorf("expected name to be nil (setter should not be called on type mismatch), got %v", *req.name)
	}
}

// ---------------------------------------------------------------------------
// snakeToPascal unit tests
// ---------------------------------------------------------------------------

func TestSnakeToPascal(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"name", "Name"},
		{"region", "Region"},
		{"allow_query_params", "AllowQueryParams"},
		{"weight", "Weight"},
		{"a_b_c", "ABC"},
		{"already", "Already"},
		{"", ""},
	}

	for _, tc := range cases {
		got := snakeToPascal(tc.input)
		if got != tc.expected {
			t.Errorf("snakeToPascal(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
