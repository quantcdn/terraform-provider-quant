package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// errReader is a reader that always returns an error, used to simulate
// unreadable HTTP response bodies.
type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

// ---------- parseAPIError ----------

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name     string
		httpResp *http.Response
		want     string
	}{
		{
			name:     "nil response",
			httpResp: nil,
			want:     "",
		},
		{
			name:     "nil body",
			httpResp: &http.Response{Body: nil},
			want:     "",
		},
		{
			name: "unreadable body",
			httpResp: &http.Response{
				Body: io.NopCloser(errReader{}),
			},
			want: "",
		},
		{
			name: "invalid JSON",
			httpResp: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString("not json")),
			},
			want: "",
		},
		{
			name: "valid JSON with message",
			httpResp: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"error":true,"message":"something went wrong"}`)),
			},
			want: "something went wrong",
		},
		{
			name: "valid JSON without message",
			httpResp: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"error":true}`)),
			},
			want: "",
		},
		{
			name: "valid JSON with empty message",
			httpResp: &http.Response{
				Body: io.NopCloser(bytes.NewBufferString(`{"error":true,"message":""}`)),
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAPIError(tc.httpResp)
			if got != tc.want {
				t.Errorf("parseAPIError() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------- extractStringList ----------

func TestExtractStringList(t *testing.T) {
	ctx := context.Background()

	validList, diags := types.ListValueFrom(ctx, types.StringType, []string{"a", "b", "c"})
	if diags.HasError() {
		t.Fatalf("failed to create valid list: %s", diags.Errors())
	}

	emptyList, diags := types.ListValueFrom(ctx, types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("failed to create empty list: %s", diags.Errors())
	}

	tests := []struct {
		name      string
		list      types.List
		wantLen   int
		wantNil   bool
		wantItems []string
	}{
		{
			name:    "null list",
			list:    types.ListNull(types.StringType),
			wantNil: true,
		},
		{
			name:    "unknown list",
			list:    types.ListUnknown(types.StringType),
			wantNil: true,
		},
		{
			name:      "valid list with elements",
			list:      validList,
			wantLen:   3,
			wantNil:   false,
			wantItems: []string{"a", "b", "c"},
		},
		{
			name:    "empty list",
			list:    emptyList,
			wantLen: 0,
			wantNil: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, diags := extractStringList(ctx, tc.list)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags.Errors())
			}
			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if len(result) != tc.wantLen {
				t.Errorf("len = %d, want %d", len(result), tc.wantLen)
			}
			for i, want := range tc.wantItems {
				if result[i] != want {
					t.Errorf("result[%d] = %q, want %q", i, result[i], want)
				}
			}
		})
	}
}

// ---------- setConditionalList ----------

func TestSetConditionalList(t *testing.T) {
	ctx := context.Background()

	strPtr := func(s string) *string { return &s }

	t.Run("nil apiValue sets null/empty", func(t *testing.T) {
		var modelValue types.String
		var modelIs, modelIsNot types.List

		diags := setConditionalList(ctx, nil, nil, nil, &modelValue, &modelIs, &modelIsNot)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if !modelValue.IsNull() {
			t.Error("expected modelValue to be null")
		}
		// modelIs and modelIsNot should be empty lists (not null)
		isElems, _ := extractStringList(ctx, modelIs)
		if len(isElems) != 0 {
			t.Errorf("expected empty modelIs, got %v", isElems)
		}
		isNotElems, _ := extractStringList(ctx, modelIsNot)
		if len(isNotElems) != 0 {
			t.Errorf("expected empty modelIsNot, got %v", isNotElems)
		}
	})

	t.Run("empty string apiValue sets null/empty", func(t *testing.T) {
		var modelValue types.String
		var modelIs, modelIsNot types.List

		empty := ""
		diags := setConditionalList(ctx, &empty, nil, nil, &modelValue, &modelIs, &modelIsNot)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if !modelValue.IsNull() {
			t.Error("expected modelValue to be null")
		}
	})

	t.Run("non-nil apiValue with is-list", func(t *testing.T) {
		var modelValue types.String
		var modelIs, modelIsNot types.List

		diags := setConditionalList(ctx,
			strPtr("country"), []string{"US", "AU"}, nil,
			&modelValue, &modelIs, &modelIsNot,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if modelValue.ValueString() != "country" {
			t.Errorf("modelValue = %q, want %q", modelValue.ValueString(), "country")
		}
		isElems, _ := extractStringList(ctx, modelIs)
		if len(isElems) != 2 || isElems[0] != "US" || isElems[1] != "AU" {
			t.Errorf("modelIs = %v, want [US AU]", isElems)
		}
		isNotElems, _ := extractStringList(ctx, modelIsNot)
		if len(isNotElems) != 0 {
			t.Errorf("expected empty modelIsNot, got %v", isNotElems)
		}
	})

	t.Run("non-nil apiValue with is-not-list", func(t *testing.T) {
		var modelValue types.String
		var modelIs, modelIsNot types.List

		diags := setConditionalList(ctx,
			strPtr("ip"), nil, []string{"10.0.0.1"},
			&modelValue, &modelIs, &modelIsNot,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if modelValue.ValueString() != "ip" {
			t.Errorf("modelValue = %q, want %q", modelValue.ValueString(), "ip")
		}
		isElems, _ := extractStringList(ctx, modelIs)
		if len(isElems) != 0 {
			t.Errorf("expected empty modelIs, got %v", isElems)
		}
		isNotElems, _ := extractStringList(ctx, modelIsNot)
		if len(isNotElems) != 1 || isNotElems[0] != "10.0.0.1" {
			t.Errorf("modelIsNot = %v, want [10.0.0.1]", isNotElems)
		}
	})

	t.Run("non-nil apiValue with both is and is-not lists", func(t *testing.T) {
		var modelValue types.String
		var modelIs, modelIsNot types.List

		diags := setConditionalList(ctx,
			strPtr("method"), []string{"GET"}, []string{"POST", "PUT"},
			&modelValue, &modelIs, &modelIsNot,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if modelValue.ValueString() != "method" {
			t.Errorf("modelValue = %q, want %q", modelValue.ValueString(), "method")
		}
		isElems, _ := extractStringList(ctx, modelIs)
		if len(isElems) != 1 || isElems[0] != "GET" {
			t.Errorf("modelIs = %v, want [GET]", isElems)
		}
		isNotElems, _ := extractStringList(ctx, modelIsNot)
		if len(isNotElems) != 2 || isNotElems[0] != "POST" || isNotElems[1] != "PUT" {
			t.Errorf("modelIsNot = %v, want [POST PUT]", isNotElems)
		}
	})
}

// ---------- buildConditionalList ----------

func TestBuildConditionalList(t *testing.T) {
	ctx := context.Background()

	t.Run("null modelValue does nothing", func(t *testing.T) {
		called := false
		diags := buildConditionalList(ctx,
			types.StringNull(),
			types.ListNull(types.StringType),
			types.ListNull(types.StringType),
			func(string) { called = true },
			func([]string) { called = true },
			func([]string) { called = true },
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if called {
			t.Error("no setters should be called for null modelValue")
		}
	})

	t.Run("unknown modelValue does nothing", func(t *testing.T) {
		called := false
		diags := buildConditionalList(ctx,
			types.StringUnknown(),
			types.ListNull(types.StringType),
			types.ListNull(types.StringType),
			func(string) { called = true },
			func([]string) { called = true },
			func([]string) { called = true },
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if called {
			t.Error("no setters should be called for unknown modelValue")
		}
	})

	t.Run("valid modelValue with is-list", func(t *testing.T) {
		isList, d := types.ListValueFrom(ctx, types.StringType, []string{"US", "AU"})
		if d.HasError() {
			t.Fatalf("failed to create list: %s", d.Errors())
		}
		emptyList, d := types.ListValueFrom(ctx, types.StringType, []string{})
		if d.HasError() {
			t.Fatalf("failed to create empty list: %s", d.Errors())
		}

		var gotValue string
		var gotIs []string
		isNotCalled := false

		diags := buildConditionalList(ctx,
			types.StringValue("country"),
			isList,
			emptyList,
			func(v string) { gotValue = v },
			func(v []string) { gotIs = v },
			func([]string) { isNotCalled = true },
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if gotValue != "country" {
			t.Errorf("setValue got %q, want %q", gotValue, "country")
		}
		if len(gotIs) != 2 || gotIs[0] != "US" || gotIs[1] != "AU" {
			t.Errorf("setIs got %v, want [US AU]", gotIs)
		}
		if isNotCalled {
			t.Error("setIsNot should not be called for empty is-not list")
		}
	})

	t.Run("valid modelValue with is-not-list", func(t *testing.T) {
		isNotList, d := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.1"})
		if d.HasError() {
			t.Fatalf("failed to create list: %s", d.Errors())
		}
		emptyList, d := types.ListValueFrom(ctx, types.StringType, []string{})
		if d.HasError() {
			t.Fatalf("failed to create empty list: %s", d.Errors())
		}

		var gotValue string
		isCalled := false
		var gotIsNot []string

		diags := buildConditionalList(ctx,
			types.StringValue("ip"),
			emptyList,
			isNotList,
			func(v string) { gotValue = v },
			func([]string) { isCalled = true },
			func(v []string) { gotIsNot = v },
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if gotValue != "ip" {
			t.Errorf("setValue got %q, want %q", gotValue, "ip")
		}
		if isCalled {
			t.Error("setIs should not be called for empty is list")
		}
		if len(gotIsNot) != 1 || gotIsNot[0] != "10.0.0.1" {
			t.Errorf("setIsNot got %v, want [10.0.0.1]", gotIsNot)
		}
	})

	t.Run("valid modelValue with null lists", func(t *testing.T) {
		var gotValue string
		isCalled := false
		isNotCalled := false

		diags := buildConditionalList(ctx,
			types.StringValue("method"),
			types.ListNull(types.StringType),
			types.ListNull(types.StringType),
			func(v string) { gotValue = v },
			func([]string) { isCalled = true },
			func([]string) { isNotCalled = true },
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if gotValue != "method" {
			t.Errorf("setValue got %q, want %q", gotValue, "method")
		}
		if isCalled {
			t.Error("setIs should not be called for null is list")
		}
		if isNotCalled {
			t.Error("setIsNot should not be called for null is-not list")
		}
	})
}

// ---------- setConditionalListsFromAPI ----------

func TestSetConditionalListsFromAPI(t *testing.T) {
	ctx := context.Background()
	strPtr := func(s string) *string { return &s }

	t.Run("mixed nil and non-nil API values", func(t *testing.T) {
		var (
			modelCountry, modelIp, modelMethod             types.String
			modelCountryIs, modelCountryIsNot              types.List
			modelIpIs, modelIpIsNot                        types.List
			modelMethodIs, modelMethodIsNot                types.List
		)

		diags := setConditionalListsFromAPI(ctx,
			// Country: non-nil with is-list
			strPtr("country"), []string{"US"}, nil,
			// IP: nil
			nil, nil, nil,
			// Method: non-nil with is-not-list
			strPtr("method"), nil, []string{"DELETE"},
			// Country model
			&modelCountry, &modelCountryIs, &modelCountryIsNot,
			// IP model
			&modelIp, &modelIpIs, &modelIpIsNot,
			// Method model
			&modelMethod, &modelMethodIs, &modelMethodIsNot,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}

		// Country should be set
		if modelCountry.ValueString() != "country" {
			t.Errorf("modelCountry = %q, want %q", modelCountry.ValueString(), "country")
		}
		countryIs, _ := extractStringList(ctx, modelCountryIs)
		if len(countryIs) != 1 || countryIs[0] != "US" {
			t.Errorf("countryIs = %v, want [US]", countryIs)
		}

		// IP should be null
		if !modelIp.IsNull() {
			t.Error("expected modelIp to be null")
		}

		// Method should be set
		if modelMethod.ValueString() != "method" {
			t.Errorf("modelMethod = %q, want %q", modelMethod.ValueString(), "method")
		}
		methodIsNot, _ := extractStringList(ctx, modelMethodIsNot)
		if len(methodIsNot) != 1 || methodIsNot[0] != "DELETE" {
			t.Errorf("methodIsNot = %v, want [DELETE]", methodIsNot)
		}
	})

	t.Run("all nil API values", func(t *testing.T) {
		var (
			modelCountry, modelIp, modelMethod             types.String
			modelCountryIs, modelCountryIsNot              types.List
			modelIpIs, modelIpIsNot                        types.List
			modelMethodIs, modelMethodIsNot                types.List
		)

		diags := setConditionalListsFromAPI(ctx,
			nil, nil, nil,
			nil, nil, nil,
			nil, nil, nil,
			&modelCountry, &modelCountryIs, &modelCountryIsNot,
			&modelIp, &modelIpIs, &modelIpIsNot,
			&modelMethod, &modelMethodIs, &modelMethodIsNot,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}

		if !modelCountry.IsNull() {
			t.Error("expected modelCountry to be null")
		}
		if !modelIp.IsNull() {
			t.Error("expected modelIp to be null")
		}
		if !modelMethod.IsNull() {
			t.Error("expected modelMethod to be null")
		}
	})

	t.Run("all non-nil API values", func(t *testing.T) {
		var (
			modelCountry, modelIp, modelMethod             types.String
			modelCountryIs, modelCountryIsNot              types.List
			modelIpIs, modelIpIsNot                        types.List
			modelMethodIs, modelMethodIsNot                types.List
		)

		diags := setConditionalListsFromAPI(ctx,
			strPtr("country"), []string{"US", "GB"}, nil,
			strPtr("ip"), nil, []string{"192.168.1.1"},
			strPtr("method"), []string{"GET", "POST"}, nil,
			&modelCountry, &modelCountryIs, &modelCountryIsNot,
			&modelIp, &modelIpIs, &modelIpIsNot,
			&modelMethod, &modelMethodIs, &modelMethodIsNot,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}

		if modelCountry.ValueString() != "country" {
			t.Errorf("modelCountry = %q, want %q", modelCountry.ValueString(), "country")
		}
		if modelIp.ValueString() != "ip" {
			t.Errorf("modelIp = %q, want %q", modelIp.ValueString(), "ip")
		}
		if modelMethod.ValueString() != "method" {
			t.Errorf("modelMethod = %q, want %q", modelMethod.ValueString(), "method")
		}

		countryIs, _ := extractStringList(ctx, modelCountryIs)
		if len(countryIs) != 2 {
			t.Errorf("countryIs len = %d, want 2", len(countryIs))
		}
		ipIsNot, _ := extractStringList(ctx, modelIpIsNot)
		if len(ipIsNot) != 1 || ipIsNot[0] != "192.168.1.1" {
			t.Errorf("ipIsNot = %v, want [192.168.1.1]", ipIsNot)
		}
		methodIs, _ := extractStringList(ctx, modelMethodIs)
		if len(methodIs) != 2 {
			t.Errorf("methodIs len = %d, want 2", len(methodIs))
		}
	})
}

// ---------- buildConditionalListsForRequest ----------

func TestBuildConditionalListsForRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("mixed null and valid model values", func(t *testing.T) {
		countryIsList, d := types.ListValueFrom(ctx, types.StringType, []string{"US", "AU"})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		emptyList, d := types.ListValueFrom(ctx, types.StringType, []string{})
		if d.HasError() {
			t.Fatal(d.Errors())
		}
		methodIsNotList, d := types.ListValueFrom(ctx, types.StringType, []string{"DELETE"})
		if d.HasError() {
			t.Fatal(d.Errors())
		}

		var gotCountry, gotMethod string
		var gotCountryIs, gotCountryIsNot []string
		var gotIpIs, gotIpIsNot []string
		var gotMethodIs, gotMethodIsNot []string
		ipValueCalled := false

		diags := buildConditionalListsForRequest(ctx,
			// Country: valid with is-list
			types.StringValue("country"), countryIsList, emptyList,
			// IP: null
			types.StringNull(), types.ListNull(types.StringType), types.ListNull(types.StringType),
			// Method: valid with is-not-list
			types.StringValue("method"), emptyList, methodIsNotList,
			// Country setters
			func(v string) { gotCountry = v },
			func(v []string) { gotCountryIs = v },
			func(v []string) { gotCountryIsNot = v },
			// IP setters
			func(string) { ipValueCalled = true },
			func(v []string) { gotIpIs = v },
			func(v []string) { gotIpIsNot = v },
			// Method setters
			func(v string) { gotMethod = v },
			func(v []string) { gotMethodIs = v },
			func(v []string) { gotMethodIsNot = v },
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}

		// Country
		if gotCountry != "country" {
			t.Errorf("country value = %q, want %q", gotCountry, "country")
		}
		if len(gotCountryIs) != 2 || gotCountryIs[0] != "US" || gotCountryIs[1] != "AU" {
			t.Errorf("countryIs = %v, want [US AU]", gotCountryIs)
		}
		if gotCountryIsNot != nil {
			t.Errorf("countryIsNot = %v, want nil", gotCountryIsNot)
		}

		// IP: nothing should be set
		if ipValueCalled {
			t.Error("IP setters should not be called for null modelValue")
		}
		if gotIpIs != nil {
			t.Errorf("ipIs = %v, want nil", gotIpIs)
		}
		if gotIpIsNot != nil {
			t.Errorf("ipIsNot = %v, want nil", gotIpIsNot)
		}

		// Method
		if gotMethod != "method" {
			t.Errorf("method value = %q, want %q", gotMethod, "method")
		}
		if gotMethodIs != nil {
			t.Errorf("methodIs = %v, want nil", gotMethodIs)
		}
		if len(gotMethodIsNot) != 1 || gotMethodIsNot[0] != "DELETE" {
			t.Errorf("methodIsNot = %v, want [DELETE]", gotMethodIsNot)
		}
	})

	t.Run("all null model values", func(t *testing.T) {
		anyCalled := false
		noop := func(string) { anyCalled = true }
		noopSlice := func([]string) { anyCalled = true }

		diags := buildConditionalListsForRequest(ctx,
			types.StringNull(), types.ListNull(types.StringType), types.ListNull(types.StringType),
			types.StringNull(), types.ListNull(types.StringType), types.ListNull(types.StringType),
			types.StringNull(), types.ListNull(types.StringType), types.ListNull(types.StringType),
			noop, noopSlice, noopSlice,
			noop, noopSlice, noopSlice,
			noop, noopSlice, noopSlice,
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}
		if anyCalled {
			t.Error("no setters should be called when all model values are null")
		}
	})

	t.Run("all valid model values", func(t *testing.T) {
		countryIs, _ := types.ListValueFrom(ctx, types.StringType, []string{"US"})
		countryIsNot, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		ipIs, _ := types.ListValueFrom(ctx, types.StringType, []string{})
		ipIsNot, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.1"})
		methodIs, _ := types.ListValueFrom(ctx, types.StringType, []string{"GET"})
		methodIsNot, _ := types.ListValueFrom(ctx, types.StringType, []string{})

		var gotCountry, gotIp, gotMethod string
		var gotCountryIs, gotIpIsNot, gotMethodIs []string

		diags := buildConditionalListsForRequest(ctx,
			types.StringValue("country"), countryIs, countryIsNot,
			types.StringValue("ip"), ipIs, ipIsNot,
			types.StringValue("method"), methodIs, methodIsNot,
			func(v string) { gotCountry = v },
			func(v []string) { gotCountryIs = v },
			func([]string) {},
			func(v string) { gotIp = v },
			func([]string) {},
			func(v []string) { gotIpIsNot = v },
			func(v string) { gotMethod = v },
			func(v []string) { gotMethodIs = v },
			func([]string) {},
		)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %s", diags.Errors())
		}

		if gotCountry != "country" {
			t.Errorf("country = %q, want %q", gotCountry, "country")
		}
		if len(gotCountryIs) != 1 || gotCountryIs[0] != "US" {
			t.Errorf("countryIs = %v, want [US]", gotCountryIs)
		}
		if gotIp != "ip" {
			t.Errorf("ip = %q, want %q", gotIp, "ip")
		}
		if len(gotIpIsNot) != 1 || gotIpIsNot[0] != "10.0.0.1" {
			t.Errorf("ipIsNot = %v, want [10.0.0.1]", gotIpIsNot)
		}
		if gotMethod != "method" {
			t.Errorf("method = %q, want %q", gotMethod, "method")
		}
		if len(gotMethodIs) != 1 || gotMethodIs[0] != "GET" {
			t.Errorf("methodIs = %v, want [GET]", gotMethodIs)
		}
	})
}

// ---------- addUseStateForUnknown ----------

func TestAddUseStateForUnknown(t *testing.T) {
	t.Run("StringAttribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.StringAttribute{Computed: true},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.StringAttribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("BoolAttribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.BoolAttribute{Computed: true},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.BoolAttribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("Int64Attribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.Int64Attribute{Computed: true},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.Int64Attribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("Float64Attribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.Float64Attribute{Computed: true},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.Float64Attribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("ListAttribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.ListAttribute{Computed: true, ElementType: types.StringType},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.ListAttribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("MapAttribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.MapAttribute{Computed: true, ElementType: types.StringType},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.MapAttribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("ListNestedAttribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"inner": schema.StringAttribute{Optional: true},
					},
				},
			},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.ListNestedAttribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("ObjectAttribute computed gets modifier", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.ObjectAttribute{
				Computed:       true,
				AttributeTypes: map[string]attr.Type{"name": types.StringType},
			},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.ObjectAttribute)
		if len(a.PlanModifiers) != 1 {
			t.Fatalf("expected 1 plan modifier, got %d", len(a.PlanModifiers))
		}
	})

	t.Run("SingleNestedAttribute recurses into children", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"parent": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"child": schema.StringAttribute{Computed: true},
				},
			},
		}
		addUseStateForUnknown(attrs)
		parent := attrs["parent"].(schema.SingleNestedAttribute)
		child := parent.Attributes["child"].(schema.StringAttribute)
		if len(child.PlanModifiers) != 1 {
			t.Fatalf("expected child to have 1 plan modifier, got %d", len(child.PlanModifiers))
		}
	})

	t.Run("non-computed attributes unchanged", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"required_str": schema.StringAttribute{Required: true},
			"optional_bool": schema.BoolAttribute{Optional: true},
		}
		addUseStateForUnknown(attrs)
		s := attrs["required_str"].(schema.StringAttribute)
		if len(s.PlanModifiers) != 0 {
			t.Errorf("required string should have 0 modifiers, got %d", len(s.PlanModifiers))
		}
		b := attrs["optional_bool"].(schema.BoolAttribute)
		if len(b.PlanModifiers) != 0 {
			t.Errorf("optional bool should have 0 modifiers, got %d", len(b.PlanModifiers))
		}
	})

	t.Run("preserves existing modifiers", func(t *testing.T) {
		attrs := map[string]schema.Attribute{
			"test": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		}
		addUseStateForUnknown(attrs)
		a := attrs["test"].(schema.StringAttribute)
		if len(a.PlanModifiers) != 2 {
			t.Fatalf("expected 2 plan modifiers (existing + UseStateForUnknown), got %d", len(a.PlanModifiers))
		}
	})
}

func TestReadFailure(t *testing.T) {
	t.Run("404 becomes a not-found diagnostic", func(t *testing.T) {
		d := readFailure(&http.Response{StatusCode: http.StatusNotFound}, "summary", "detail")
		if _, ok := d.(notFoundDiagnostic); !ok {
			t.Fatalf("expected notFoundDiagnostic, got %T", d)
		}
		if d.Severity() != diag.SeverityError || d.Summary() != "summary" || d.Detail() != "detail" {
			t.Fatalf("unexpected diagnostic: %v %q %q", d.Severity(), d.Summary(), d.Detail())
		}
	})

	t.Run("other status stays an ordinary error", func(t *testing.T) {
		d := readFailure(&http.Response{StatusCode: http.StatusInternalServerError}, "summary", "detail")
		if _, ok := d.(notFoundDiagnostic); ok {
			t.Fatal("a 500 must not be treated as not found")
		}
	})

	t.Run("nil response stays an ordinary error", func(t *testing.T) {
		if _, ok := readFailure(nil, "summary", "detail").(notFoundDiagnostic); ok {
			t.Fatal("a transport error must not be treated as not found")
		}
	})
}

func TestStripNotFound(t *testing.T) {
	other := diag.NewErrorDiagnostic("other", "boom")
	var diags diag.Diagnostics
	diags.Append(other, readFailure(&http.Response{StatusCode: http.StatusNotFound}, "gone", "x"))

	rest, found := stripNotFound(diags)
	if !found {
		t.Fatal("expected the not-found diagnostic to be detected")
	}
	if len(rest) != 1 || !rest[0].Equal(other) {
		t.Fatalf("expected only the unrelated error to remain, got %v", rest)
	}

	if _, found := stripNotFound(diag.Diagnostics{other}); found {
		t.Fatal("a plain error must not be reported as not found")
	}
}

func TestNotFoundDiagnosticEqual(t *testing.T) {
	a := readFailure(&http.Response{StatusCode: http.StatusNotFound}, "gone", "x")
	b := readFailure(&http.Response{StatusCode: http.StatusNotFound}, "gone", "x")
	plain := diag.NewErrorDiagnostic("gone", "x")

	if !a.Equal(b) {
		t.Fatal("two identical not-found diagnostics must be equal, or Diagnostics.Append cannot de-duplicate them")
	}
	if a.Equal(plain) || plain.Equal(a) {
		t.Fatal("a not-found diagnostic must not equal a plain error with the same text")
	}
}
