package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// addReplaceKeepingState makes each attribute force replacement. An
// Optional+Computed attribute first keeps its prior state when the config
// omits it; without that, the unknown planned value differs from state and
// every plan would replace the resource.
func addReplaceKeepingState(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		attrs[name] = withPlanModifiers(attrs[name], true)
	}
}

// addKeepState keeps the prior state of each attribute when the plan has no
// value for it (Optional+Computed inputs the config omits).
func addKeepState(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		attrs[name] = withPlanModifiers(attrs[name], false)
	}
}

// requireSubjectField makes each string attribute need subject_field in the
// same configuration (a plan-time error instead of an API 400).
func requireSubjectField(attrs map[string]schema.Attribute, names ...string) {
	for _, name := range names {
		if t, ok := attrs[name].(schema.StringAttribute); ok {
			t.Validators = append(t.Validators, stringvalidator.AlsoRequires(path.MatchRoot("subject_field")))
			attrs[name] = t
		}
	}
}

func withPlanModifiers(a schema.Attribute, replace bool) schema.Attribute {
	switch t := a.(type) {
	case schema.StringAttribute:
		t.PlanModifiers = append(t.PlanModifiers, stringplanmodifier.UseStateForUnknown())
		if replace {
			t.PlanModifiers = append(t.PlanModifiers, stringplanmodifier.RequiresReplace())
		}
		return t
	case schema.BoolAttribute:
		t.PlanModifiers = append(t.PlanModifiers, boolplanmodifier.UseStateForUnknown())
		if replace {
			t.PlanModifiers = append(t.PlanModifiers, boolplanmodifier.RequiresReplace())
		}
		return t
	case schema.ListAttribute:
		t.PlanModifiers = append(t.PlanModifiers, listplanmodifier.UseStateForUnknown())
		if replace {
			t.PlanModifiers = append(t.PlanModifiers, listplanmodifier.RequiresReplace())
		}
		return t
	case schema.Int64Attribute:
		t.PlanModifiers = append(t.PlanModifiers, int64planmodifier.UseStateForUnknown())
		if replace {
			t.PlanModifiers = append(t.PlanModifiers, int64planmodifier.RequiresReplace())
		}
		return t
	}
	return a
}

// kvAPIStatus is the HTTP status of an SDK call, or 0 when there was no response.
func kvAPIStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// kvAPIOK is true for a 2xx answer. A drop or declare that the API accepted
// succeeded even when the SDK cannot decode the answer body.
func kvAPIOK(resp *http.Response) bool {
	status := kvAPIStatus(resp)
	return status >= 200 && status < 300
}

// kvAPIError is the KV error body (V2KvError) fields the provider reports.
type kvAPIError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Reason      string `json:"reason"`
	RetainUntil string `json:"retainUntil"`
	Collection  string `json:"collection"`
	Shard       string `json:"shard"`
	Hint        string `json:"hint"`
}

// parseKVAPIError reads the KV error body that the SDK keeps on its error.
func parseKVAPIError(err error) kvAPIError {
	var out kvAPIError
	var withBody interface{ Body() []byte }
	if errors.As(err, &withBody) {
		_ = json.Unmarshal(withBody.Body(), &out)
	}
	return out
}

// kvRef is the path that names an index or a ledger in a store.
type kvRef struct {
	Organization, Project, StoreID, Name string
}

// parseKVImportID accepts project/store_id/name or organization/project/store_id/name.
// An empty organization means the provider organization.
func parseKVImportID(id string) (kvRef, error) {
	parts := strings.Split(id, "/")
	for _, p := range parts {
		if p == "" {
			return kvRef{}, fmt.Errorf("import ID %q has an empty part", id)
		}
	}
	switch len(parts) {
	case 3:
		return kvRef{Project: parts[0], StoreID: parts[1], Name: parts[2]}, nil
	case 4:
		return kvRef{Organization: parts[0], Project: parts[1], StoreID: parts[2], Name: parts[3]}, nil
	}
	return kvRef{}, fmt.Errorf("import ID %q must be project/store_id/name or organization/project/store_id/name", id)
}

// sleepCtx waits for d or until ctx ends.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// waitTimeout is the duration in the environment variable, or def when it is
// unset or not a valid positive Go duration ("20m", "1h").
func waitTimeout(envVar string, def time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(envVar))
	if err != nil || v <= 0 {
		return def
	}
	return v
}

// optionalString is null for a nil or empty string.
func optionalString(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func timeString(t time.Time) types.String {
	if t.IsZero() {
		return types.StringNull()
	}
	return types.StringValue(t.UTC().Format(time.RFC3339Nano))
}

func kvOptionalTime(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return timeString(*t)
}

// orgOr is the organization in v when it is known, else the provider organization.
func orgOr(v types.String, providerOrg string) string {
	if isKnown(v) && v.ValueString() != "" {
		return v.ValueString()
	}
	return providerOrg
}
