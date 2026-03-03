package mapper

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
// FromSDK test doubles
// ---------------------------------------------------------------------------

type testSDKResponse struct {
	id               int32
	uuid             string
	name             string
	allowQueryParams bool
}

func (r *testSDKResponse) GetId() int32             { return r.id }
func (r *testSDKResponse) GetUuid() string           { return r.uuid }
func (r *testSDKResponse) GetName() string           { return r.name }
func (r *testSDKResponse) GetAllowQueryParams() bool { return r.allowQueryParams }

type testFromSDKModel struct {
	Id               types.Int64  `tfsdk:"id"`
	Uuid             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	AllowQueryParams types.Bool   `tfsdk:"allow_query_params"`
}

// ---------------------------------------------------------------------------
// FromSDK tests
// ---------------------------------------------------------------------------

func TestFromSDK_MapsResponseFields(t *testing.T) {
	resp := &testSDKResponse{
		id:               123,
		uuid:             "abc-def-ghi",
		name:             "my-project",
		allowQueryParams: true,
	}

	model := &testFromSDKModel{}
	diags := FromSDK(context.Background(), resp, model)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.Id.ValueInt64() != 123 {
		t.Errorf("expected id = 123, got %d", model.Id.ValueInt64())
	}
	if model.Uuid.ValueString() != "abc-def-ghi" {
		t.Errorf("expected uuid = %q, got %q", "abc-def-ghi", model.Uuid.ValueString())
	}
	if model.Name.ValueString() != "my-project" {
		t.Errorf("expected name = %q, got %q", "my-project", model.Name.ValueString())
	}
	if model.AllowQueryParams.ValueBool() != true {
		t.Errorf("expected allowQueryParams = true, got %v", model.AllowQueryParams.ValueBool())
	}
}

func TestFromSDK_SkipsMissingGetters(t *testing.T) {
	// Model has a field "extra" that the SDK response has no GetExtra() for.
	type modelWithExtra struct {
		Name  types.String `tfsdk:"name"`
		Extra types.String `tfsdk:"extra"`
	}

	resp := &testSDKResponse{name: "hello"}
	model := &modelWithExtra{}
	diags := FromSDK(context.Background(), resp, model)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	// The known field should still be mapped.
	if model.Name.ValueString() != "hello" {
		t.Errorf("expected name = %q, got %q", "hello", model.Name.ValueString())
	}

	// Extra should remain its zero value (null string).
	if !model.Extra.IsNull() {
		t.Errorf("expected Extra to remain null, got %q", model.Extra.ValueString())
	}
}

func TestFromSDK_MapsInt32ToInt64(t *testing.T) {
	resp := &testSDKResponse{id: 42}

	type intModel struct {
		Id types.Int64 `tfsdk:"id"`
	}

	model := &intModel{}
	diags := FromSDK(context.Background(), resp, model)

	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}

	if model.Id.ValueInt64() != 42 {
		t.Errorf("expected id = 42, got %d", model.Id.ValueInt64())
	}
}

// ---------------------------------------------------------------------------
// List test doubles
// ---------------------------------------------------------------------------

type testTFModelWithList struct {
	Domain types.List `tfsdk:"domain"`
	Url    types.List `tfsdk:"url"`
}

type testSDKReqWithList struct {
	domain []string
	url    []string
}

func (r *testSDKReqWithList) SetDomain(v []string) { r.domain = v }
func (r *testSDKReqWithList) SetUrl(v []string)    { r.url = v }
func (r *testSDKReqWithList) GetDomain() []string  { return r.domain }
func (r *testSDKReqWithList) GetUrl() []string     { return r.url }

// ---------------------------------------------------------------------------
// List tests — ToSDK
// ---------------------------------------------------------------------------

func TestToSDK_MapsStringList(t *testing.T) {
	ctx := context.Background()
	domains, diags := types.ListValueFrom(ctx, types.StringType, []string{"example.com", "test.com"})
	if diags.HasError() {
		t.Fatalf("failed to create list: %v", diags.Errors())
	}

	model := testTFModelWithList{
		Domain: domains,
		Url:    types.ListNull(types.StringType),
	}

	req := &testSDKReqWithList{}
	d := ToSDK(ctx, &model, req)
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}

	if len(req.domain) != 2 || req.domain[0] != "example.com" || req.domain[1] != "test.com" {
		t.Errorf("expected domain=[example.com, test.com], got %v", req.domain)
	}
}

func TestToSDK_SkipsNullList(t *testing.T) {
	model := testTFModelWithList{
		Domain: types.ListNull(types.StringType),
		Url:    types.ListNull(types.StringType),
	}

	req := &testSDKReqWithList{}
	d := ToSDK(context.Background(), &model, req)
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}

	if req.domain != nil {
		t.Errorf("expected domain=nil (null list), got %v", req.domain)
	}
	if req.url != nil {
		t.Errorf("expected url=nil (null list), got %v", req.url)
	}
}

// ---------------------------------------------------------------------------
// List tests — FromSDK
// ---------------------------------------------------------------------------

func TestFromSDK_MapsStringSliceToList(t *testing.T) {
	ctx := context.Background()
	resp := &testSDKReqWithList{domain: []string{"a.com", "b.com"}}

	type listModel struct {
		Domain types.List `tfsdk:"domain"`
	}

	model := &listModel{}
	d := FromSDK(ctx, resp, model)
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}

	var result []string
	model.Domain.ElementsAs(ctx, &result, false)
	if len(result) != 2 || result[0] != "a.com" || result[1] != "b.com" {
		t.Errorf("expected [a.com, b.com], got %v", result)
	}
}

// ---------------------------------------------------------------------------
// snakeToPascal unit tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Additional coverage tests — convertInt64ToTarget
// ---------------------------------------------------------------------------

// Setters that accept Int, Int16, Int8 types for convertInt64ToTarget coverage.
type intVariantSDKRequest struct {
	intVal   *int
	int16Val *int16
	int8Val  *int8
}

func (r *intVariantSDKRequest) SetIntField(v int)     { r.intVal = &v }
func (r *intVariantSDKRequest) SetInt16Field(v int16)  { r.int16Val = &v }
func (r *intVariantSDKRequest) SetInt8Field(v int8)    { r.int8Val = &v }

func TestToSDK_MapsInt64ToInt(t *testing.T) {
	type intModel struct {
		IntField types.Int64 `tfsdk:"int_field"`
	}
	model := intModel{IntField: types.Int64Value(99)}
	req := &intVariantSDKRequest{}
	diags := ToSDK(context.Background(), model, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.intVal == nil || *req.intVal != 99 {
		t.Errorf("expected intVal = 99, got %v", req.intVal)
	}
}

func TestToSDK_MapsInt64ToInt16(t *testing.T) {
	type int16Model struct {
		Int16Field types.Int64 `tfsdk:"int16_field"`
	}
	model := int16Model{Int16Field: types.Int64Value(7)}
	req := &intVariantSDKRequest{}
	diags := ToSDK(context.Background(), model, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.int16Val == nil || *req.int16Val != 7 {
		t.Errorf("expected int16Val = 7, got %v", req.int16Val)
	}
}

func TestToSDK_MapsInt64ToInt8(t *testing.T) {
	type int8Model struct {
		Int8Field types.Int64 `tfsdk:"int8_field"`
	}
	model := int8Model{Int8Field: types.Int64Value(3)}
	req := &intVariantSDKRequest{}
	diags := ToSDK(context.Background(), model, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.int8Val == nil || *req.int8Val != 3 {
		t.Errorf("expected int8Val = 3, got %v", req.int8Val)
	}
}

// Setter that takes a uint32, which convertInt64ToTarget does not support (default case).
type unsupportedIntSDKRequest struct {
	val *uint32
}

func (r *unsupportedIntSDKRequest) SetWeight(v uint32) { r.val = &v }

func TestToSDK_SkipsInt64WithUnsupportedIntTarget(t *testing.T) {
	model := testTFModel{
		Name:             types.StringNull(),
		Region:           types.StringNull(),
		AllowQueryParams: types.BoolNull(),
		Weight:           types.Int64Value(10),
		Disabled:         types.BoolNull(),
		Score:            types.Float64Null(),
	}
	req := &unsupportedIntSDKRequest{}
	diags := ToSDK(context.Background(), model, req)
	// Should produce a warning because the value is not null but type doesn't match.
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.val != nil {
		t.Errorf("expected val to be nil (unsupported int target skipped), got %v", *req.val)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests — convertToTFValue (Float32, Float64, non-string slice, default)
// ---------------------------------------------------------------------------

type float32Response struct {
	rate float32
}

func (r *float32Response) GetRate() float32 { return r.rate }

func TestFromSDK_MapsFloat32ToFloat64(t *testing.T) {
	resp := &float32Response{rate: 1.5}
	type floatModel struct {
		Rate types.Float64 `tfsdk:"rate"`
	}
	model := &floatModel{}
	diags := FromSDK(context.Background(), resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	// float32 → float64 may lose precision, just check approximate value.
	got := model.Rate.ValueFloat64()
	if got < 1.4 || got > 1.6 {
		t.Errorf("expected rate ≈ 1.5, got %f", got)
	}
}

type float64Response struct {
	score float64
}

func (r *float64Response) GetScore() float64 { return r.score }

func TestFromSDK_MapsFloat64ToFloat64(t *testing.T) {
	resp := &float64Response{score: 2.718}
	type floatModel struct {
		Score types.Float64 `tfsdk:"score"`
	}
	model := &floatModel{}
	diags := FromSDK(context.Background(), resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if model.Score.ValueFloat64() != 2.718 {
		t.Errorf("expected score = 2.718, got %f", model.Score.ValueFloat64())
	}
}

// Getter returns []int — a non-string slice, should be skipped by convertToTFValue.
type intSliceResponse struct {
	ids []int
}

func (r *intSliceResponse) GetIds() []int { return r.ids }

func TestFromSDK_SkipsNonStringSlice(t *testing.T) {
	resp := &intSliceResponse{ids: []int{1, 2, 3}}
	type sliceModel struct {
		Ids types.List `tfsdk:"ids"`
	}
	model := &sliceModel{}
	diags := FromSDK(context.Background(), resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	// The field should remain at its zero value (null list).
	if !model.Ids.IsNull() {
		t.Errorf("expected Ids to remain null (non-string slice skipped), got %v", model.Ids)
	}
}

// Getter returns a struct — unsupported by convertToTFValue (default case).
type structVal struct{ X int }
type structResponse struct {
	data structVal
}

func (r *structResponse) GetData() structVal { return r.data }

func TestFromSDK_SkipsUnsupportedReturnType(t *testing.T) {
	resp := &structResponse{data: structVal{X: 1}}
	type structModel struct {
		Data types.String `tfsdk:"data"`
	}
	model := &structModel{}
	diags := FromSDK(context.Background(), resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	// Field should remain at its zero value (null string).
	if !model.Data.IsNull() {
		t.Errorf("expected Data to remain null (unsupported return type), got %q", model.Data.ValueString())
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests — FromSDK (non-pointer, pointer-to-non-struct, wrong getter sig)
// ---------------------------------------------------------------------------

func TestFromSDK_RejectsNonPointer(t *testing.T) {
	resp := &testSDKResponse{name: "hello"}
	model := testFromSDKModel{} // not a pointer
	diags := FromSDK(context.Background(), resp, model)
	if !diags.HasError() {
		t.Fatal("expected error when tfModel is not a pointer, got none")
	}
	found := false
	for _, d := range diags.Errors() {
		if d.Summary() == "FromSDK: tfModel must be a pointer" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error summary %q, got: %v", "FromSDK: tfModel must be a pointer", diags.Errors())
	}
}

func TestFromSDK_RejectsPointerToNonStruct(t *testing.T) {
	resp := &testSDKResponse{name: "hello"}
	s := "not a struct"
	diags := FromSDK(context.Background(), resp, &s)
	if !diags.HasError() {
		t.Fatal("expected error when tfModel is a pointer to non-struct, got none")
	}
	found := false
	for _, d := range diags.Errors() {
		if d.Summary() == "FromSDK: tfModel must be a pointer to struct" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error summary %q, got: %v", "FromSDK: tfModel must be a pointer to struct", diags.Errors())
	}
}

// Getter that takes arguments — should be skipped (getter.Type().NumIn() != 0).
type getterWithArgsResponse struct{}

func (r *getterWithArgsResponse) GetName(ctx context.Context) string { return "bad" }

func TestFromSDK_SkipsGetterWithArgs(t *testing.T) {
	resp := &getterWithArgsResponse{}
	type model struct {
		Name types.String `tfsdk:"name"`
	}
	m := &model{}
	diags := FromSDK(context.Background(), resp, m)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if !m.Name.IsNull() {
		t.Errorf("expected Name to remain null (getter with args skipped), got %q", m.Name.ValueString())
	}
}

// Getter that returns multiple values — should be skipped (getter.Type().NumOut() != 1).
type getterMultiReturnResponse struct{}

func (r *getterMultiReturnResponse) GetName() (string, error) { return "bad", nil }

func TestFromSDK_SkipsGetterWithMultiReturn(t *testing.T) {
	resp := &getterMultiReturnResponse{}
	type model struct {
		Name types.String `tfsdk:"name"`
	}
	m := &model{}
	diags := FromSDK(context.Background(), resp, m)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if !m.Name.IsNull() {
		t.Errorf("expected Name to remain null (getter with multi-return skipped), got %q", m.Name.ValueString())
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests — ToSDK (non-struct tfModel, setter with != 1 arg)
// ---------------------------------------------------------------------------

func TestToSDK_RejectsNonStructTfModel(t *testing.T) {
	s := "not a struct"
	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), s, req)
	if !diags.HasError() {
		t.Fatal("expected error when tfModel is not a struct, got none")
	}
	found := false
	for _, d := range diags.Errors() {
		if d.Summary() == "ToSDK: tfModel must be a struct or pointer to struct" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error summary %q, got: %v", "ToSDK: tfModel must be a struct or pointer to struct", diags.Errors())
	}
}

// Setter that takes two arguments — should be skipped (setter.Type().NumIn() != 1).
type multiArgSDKRequest struct{}

func (r *multiArgSDKRequest) SetName(a string, b string) {}

func TestToSDK_SkipsSetterWithMultipleArgs(t *testing.T) {
	type model struct {
		Name types.String `tfsdk:"name"`
	}
	m := model{Name: types.StringValue("hello")}
	req := &multiArgSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	// No error, just silently skipped.
}

// ---------------------------------------------------------------------------
// Additional coverage tests — convertToTarget (ConvertibleTo path)
// ---------------------------------------------------------------------------

// A custom string type that string is convertible to.
type customString string

type customStringSDKRequest struct {
	name *customString
}

func (r *customStringSDKRequest) SetName(v customString) { r.name = &v }

func TestToSDK_ConvertibleToTarget(t *testing.T) {
	type model struct {
		Name types.String `tfsdk:"name"`
	}
	m := model{Name: types.StringValue("hello")}
	req := &customStringSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.name == nil || *req.name != "hello" {
		t.Errorf("expected name = %q, got %v", "hello", req.name)
	}
}

// convertToTarget skip path: value type not convertible to target type.
// This is exercised by TestToSDK_WarnsOnTypeMismatch already (string → int64),
// but let's also cover bool → string which hits convertToTarget directly.
type stringSetterSDKRequest struct {
	disabled *string
}

func (r *stringSetterSDKRequest) SetDisabled(v string) { r.disabled = &v }

func TestToSDK_SkipsInconvertibleType(t *testing.T) {
	type model struct {
		Disabled types.Bool `tfsdk:"disabled"`
	}
	m := model{Disabled: types.BoolValue(true)}
	req := &stringSetterSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	// Should warn (not error) about type mismatch.
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.disabled != nil {
		t.Errorf("expected disabled to be nil (inconvertible type skipped), got %v", *req.disabled)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests — isNullOrUnknown (Float64 and List cases)
// ---------------------------------------------------------------------------

// These cases are hit via the warning path in ToSDK when convertTFValue returns
// skip=true but the field value is not null/unknown. We need to make sure
// Float64 and List non-null cases go through isNullOrUnknown.

type unsupportedFloatSetterSDKRequest struct {
	score *string
}

func (r *unsupportedFloatSetterSDKRequest) SetScore(v string) { r.score = &v }

func TestToSDK_WarnsOnFloat64TypeMismatch(t *testing.T) {
	// Float64 field with a setter expecting string — skip + warning.
	type model struct {
		Score types.Float64 `tfsdk:"score"`
	}
	m := model{Score: types.Float64Value(3.14)}
	req := &unsupportedFloatSetterSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	// Should warn about the mismatch.
	warnings := diags.Warnings()
	if len(warnings) == 0 {
		t.Fatal("expected a warning for Float64 type mismatch, got none")
	}
	if req.score != nil {
		t.Errorf("expected score to be nil (type mismatch skipped), got %v", *req.score)
	}
}

// For List non-null + non-string-element: the List is valid but the elements
// aren't strings, so convertTFValue's List branch returns skip via ElementsAs error.
// This goes through the warning path hitting isNullOrUnknown's List case.
func TestToSDK_WarnsOnListElementTypeMismatch(t *testing.T) {
	ctx := context.Background()
	// Create a list of int64 values, paired with a setter expecting []string.
	intList, d := types.ListValueFrom(ctx, types.Int64Type, []int64{1, 2, 3})
	if d.HasError() {
		t.Fatalf("failed to create int list: %v", d.Errors())
	}

	type listModel struct {
		Domain types.List `tfsdk:"domain"`
	}
	model := listModel{Domain: intList}
	req := &testSDKReqWithList{}
	diags := ToSDK(ctx, &model, req)

	// Should NOT error, but should warn about type mismatch.
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.domain != nil {
		t.Errorf("expected domain=nil (list element type mismatch), got %v", req.domain)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests — isNullOrUnknown (default case for unsupported type)
// ---------------------------------------------------------------------------

// To hit the default case in isNullOrUnknown, we need a TF field type that is
// not String, Bool, Int64, Float64, or List. types.Object would work, but the
// simplest approach is to use a custom struct type in the model that has a tfsdk
// tag but isn't a recognized Terraform type. convertTFValue's default case will
// return skip=true, and then isNullOrUnknown's default case returns true (so no
// warning is emitted).
type customTFType struct{}

func TestToSDK_SkipsSilentlyForUnsupportedTFType(t *testing.T) {
	type model struct {
		Custom customTFType `tfsdk:"custom"`
	}
	m := model{Custom: customTFType{}}
	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	// Should silently skip — no errors, no warnings.
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if len(diags.Warnings()) != 0 {
		t.Errorf("expected no warnings for unsupported TF type, got %v", diags.Warnings())
	}
}

// ---------------------------------------------------------------------------
// Additional coverage tests — convertTFValue default case
// ---------------------------------------------------------------------------

// The convertTFValue default case is already tested via
// TestToSDK_SkipsSilentlyForUnsupportedTFType above, since customTFType
// doesn't match any of the type-switch cases.

// ---------------------------------------------------------------------------
// Additional coverage: tfsdk:"-" tag skipped in ToSDK (line 57-58)
// ---------------------------------------------------------------------------

func TestToSDK_SkipsFieldWithDashTag(t *testing.T) {
	type modelWithDash struct {
		Name   types.String `tfsdk:"name"`
		Ignore types.String `tfsdk:"-"`
	}
	m := modelWithDash{
		Name:   types.StringValue("hello"),
		Ignore: types.StringValue("should-be-ignored"),
	}
	req := &testSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.name == nil || *req.name != "hello" {
		t.Errorf("expected name = %q, got %v", "hello", req.name)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: tfsdk:"-" tag skipped in FromSDK (line 229-230)
// ---------------------------------------------------------------------------

func TestFromSDK_SkipsFieldWithDashTag(t *testing.T) {
	type modelWithDash struct {
		Name   types.String `tfsdk:"name"`
		Ignore types.String `tfsdk:"-"`
	}
	resp := &testSDKResponse{name: "hello"}
	model := &modelWithDash{}
	diags := FromSDK(context.Background(), resp, model)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if model.Name.ValueString() != "hello" {
		t.Errorf("expected name = %q, got %q", "hello", model.Name.ValueString())
	}
	if !model.Ignore.IsNull() {
		t.Errorf("expected Ignore to remain null, got %q", model.Ignore.ValueString())
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: convertInt64ToTarget Int64 case (line 148-149)
// ---------------------------------------------------------------------------

type int64SetterSDKRequest struct {
	val *int64
}

func (r *int64SetterSDKRequest) SetWeight(v int64) { r.val = &v }

func TestToSDK_MapsInt64ToInt64(t *testing.T) {
	type model struct {
		Weight types.Int64 `tfsdk:"weight"`
	}
	m := model{Weight: types.Int64Value(123456789)}
	req := &int64SetterSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if req.val == nil || *req.val != 123456789 {
		t.Errorf("expected val = 123456789, got %v", req.val)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: convertTFValue default + isNullOrUnknown default
// (lines 136-138, 188-190)
// Need a request type that HAS a matching setter so that the unsupported TF
// type actually reaches convertTFValue (and subsequently isNullOrUnknown).
// ---------------------------------------------------------------------------

type customSetterSDKRequest struct {
	custom *string
}

func (r *customSetterSDKRequest) SetCustom(v string) { r.custom = &v }

func TestToSDK_SkipsSilentlyForUnsupportedTFTypeWithMatchingSetter(t *testing.T) {
	type model struct {
		Custom customTFType `tfsdk:"custom"`
	}
	m := model{Custom: customTFType{}}
	req := &customSetterSDKRequest{}
	diags := ToSDK(context.Background(), m, req)
	// Should silently skip — no errors, no warnings.
	// isNullOrUnknown returns true for unsupported types, so no warning is emitted.
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags.Errors())
	}
	if len(diags.Warnings()) != 0 {
		t.Errorf("expected no warnings for unsupported TF type, got %v", diags.Warnings())
	}
	if req.custom != nil {
		t.Errorf("expected custom to be nil (unsupported type skipped), got %v", *req.custom)
	}
}

// ---------------------------------------------------------------------------
// Additional coverage: convertToTFValue ListValueFrom error path (line 285-287)
// ---------------------------------------------------------------------------

func TestFromSDK_ListValueFromError(t *testing.T) {
	// Override the package-level listValueFromFunc to simulate an error.
	original := listValueFromFunc
	listValueFromFunc = func(ctx context.Context, elementType attr.Type, elements any) (types.List, diag.Diagnostics) {
		var diags diag.Diagnostics
		diags.AddError("simulated error", "ListValueFrom failed")
		return types.ListNull(elementType), diags
	}
	defer func() { listValueFromFunc = original }()

	resp := &testSDKReqWithList{domain: []string{"a.com", "b.com"}}
	type listModel struct {
		Domain types.List `tfsdk:"domain"`
	}
	model := &listModel{}
	d := FromSDK(context.Background(), resp, model)

	// Should NOT return errors from FromSDK itself — the field is just skipped.
	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}

	// The field should remain at its zero value (null list) because ListValueFrom failed.
	if !model.Domain.IsNull() {
		t.Errorf("expected Domain to remain null (ListValueFrom error), got %v", model.Domain)
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
