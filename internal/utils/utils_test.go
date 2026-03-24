package utils

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// ---------------------------------------------------------------------------
// rule_import.go — GetRuleImportId
// ---------------------------------------------------------------------------

func TestGetRuleImportId(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantProject string
		wantUUID    string
		wantErr     string
	}{
		{
			name:        "valid import id",
			input:       "my-project/550e8400-e29b-41d4-a716-446655440000",
			wantProject: "my-project",
			wantUUID:    "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:        "valid import id uppercase",
			input:       "proj/550E8400-E29B-41D4-A716-446655440000",
			wantProject: "proj",
			wantUUID:    "550E8400-E29B-41D4-A716-446655440000",
		},
		{
			name:    "no slash — single segment",
			input:   "noslashhere",
			wantErr: "the ID must follow the pattern project/uuid to import",
		},
		{
			name:    "too many slashes",
			input:   "a/b/c",
			wantErr: "the ID must follow the pattern project/uuid to import",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "the ID must follow the pattern project/uuid to import",
		},
		{
			name:    "invalid UUID format",
			input:   "project/not-a-uuid",
			wantErr: "invalid UUID format",
		},
		{
			name:    "UUID missing hyphens",
			input:   "project/550e8400e29b41d4a716446655440000",
			wantErr: "invalid UUID format",
		},
		{
			name:    "UUID with bad version digit (0)",
			input:   "project/550e8400-e29b-01d4-a716-446655440000",
			wantErr: "invalid UUID format",
		},
		{
			name:    "UUID with bad variant nibble (0)",
			input:   "project/550e8400-e29b-41d4-0716-446655440000",
			wantErr: "invalid UUID format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proj, uuid, err := GetRuleImportId(tc.input)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
				}
				if !proj.IsNull() {
					t.Fatal("expected project to be null on error")
				}
				if !uuid.IsNull() {
					t.Fatal("expected uuid to be null on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if proj.ValueString() != tc.wantProject {
				t.Errorf("project: want %q, got %q", tc.wantProject, proj.ValueString())
			}
			if uuid.ValueString() != tc.wantUUID {
				t.Errorf("uuid: want %q, got %q", tc.wantUUID, uuid.ValueString())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rule_import.go — GetDomainImportId
// ---------------------------------------------------------------------------

func TestGetDomainImportId(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantProject string
		wantID      int64
		wantErr     string
	}{
		{
			name:        "valid domain import",
			input:       "my-project/12345",
			wantProject: "my-project",
			wantID:      12345,
		},
		{
			name:        "zero id",
			input:       "proj/0",
			wantProject: "proj",
			wantID:      0,
		},
		{
			name:        "negative id",
			input:       "proj/-1",
			wantProject: "proj",
			wantID:      -1,
		},
		{
			name:    "no slash",
			input:   "noslash",
			wantErr: "the ID must follow the pattern project/uuid to import",
		},
		{
			name:    "too many slashes",
			input:   "a/b/c",
			wantErr: "the ID must follow the pattern project/uuid to import",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "the ID must follow the pattern project/uuid to import",
		},
		{
			name:    "non-numeric id",
			input:   "project/abc",
			wantErr: "invalid domain ID format",
		},
		{
			name:    "float id",
			input:   "project/1.5",
			wantErr: "invalid domain ID format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proj, id, err := GetDomainImportId(tc.input)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("expected error %q, got %q", tc.wantErr, err.Error())
				}
				if !proj.IsNull() {
					t.Fatal("expected project to be null on error")
				}
				if !id.IsNull() {
					t.Fatal("expected id to be null on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if proj.ValueString() != tc.wantProject {
				t.Errorf("project: want %q, got %q", tc.wantProject, proj.ValueString())
			}
			if id.ValueInt64() != tc.wantID {
				t.Errorf("id: want %d, got %d", tc.wantID, id.ValueInt64())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rules.go — GetRuleAny, GetFilterIs, GetFilterIsNot
// ---------------------------------------------------------------------------

func TestGetRuleAny(t *testing.T) {
	result := GetRuleAny()
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != "any" {
		t.Errorf("want %q, got %q", "any", *result)
	}
}

func TestGetFilterIs(t *testing.T) {
	tests := []struct {
		filter string
		want   string
	}{
		{"country", "country_is"},
		{"method", "method_is"},
		{"ip", "ip_is"},
		{"", "_is"},
	}
	for _, tc := range tests {
		t.Run(tc.filter, func(t *testing.T) {
			result := GetFilterIs(tc.filter)
			if result == nil {
				t.Fatal("expected non-nil pointer")
			}
			if *result != tc.want {
				t.Errorf("want %q, got %q", tc.want, *result)
			}
		})
	}
}

func TestGetFilterIsNot(t *testing.T) {
	tests := []struct {
		filter string
		want   string
	}{
		{"country", "country_is_not"},
		{"method", "method_is_not"},
		{"ip", "ip_is_not"},
		{"", "_is_not"},
	}
	for _, tc := range tests {
		t.Run(tc.filter, func(t *testing.T) {
			result := GetFilterIsNot(tc.filter)
			if result == nil {
				t.Fatal("expected non-nil pointer")
			}
			if *result != tc.want {
				t.Errorf("want %q, got %q", tc.want, *result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// rules.go — IsValidRuleFilter
// ---------------------------------------------------------------------------

func TestIsValidRuleFilter(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Valid "_is_not" variants
		{"country_is_not", true},
		{"method_is_not", true},
		{"ip_is_not", true},
		// Valid "_is" variants
		{"country_is", true},
		{"method_is", true},
		{"ip_is", true},
		// Valid "any"
		{"any", true},
		// Invalid values
		{"", false},
		{"country", false},
		{"invalid", false},
		{"ANY", false},
		{"country_is_", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := IsValidRuleFilter(tc.input)
			if got != tc.want {
				t.Errorf("IsValidRuleFilter(%q): want %v, got %v", tc.input, tc.want, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// retry.go — RetryRuleRead
// ---------------------------------------------------------------------------

func TestRetryRuleRead(t *testing.T) {
	type result struct {
		value string
	}

	t.Run("success on first attempt", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			return result{value: "ok"}, &http.Response{StatusCode: 200}, nil
		}

		val, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val.value != "ok" {
			t.Errorf("want value %q, got %q", "ok", val.value)
		}
		if resp.StatusCode != 200 {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	})

	t.Run("success on retry after 404", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			if calls == 1 {
				return result{}, &http.Response{StatusCode: 404}, errors.New("not found")
			}
			return result{value: "found"}, &http.Response{StatusCode: 200}, nil
		}

		val, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val.value != "found" {
			t.Errorf("want value %q, got %q", "found", val.value)
		}
		if resp.StatusCode != 200 {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("success on retry after 400", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			if calls == 1 {
				return result{}, &http.Response{StatusCode: 400}, errors.New("bad request")
			}
			return result{value: "recovered"}, &http.Response{StatusCode: 200}, nil
		}

		val, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val.value != "recovered" {
			t.Errorf("want value %q, got %q", "recovered", val.value)
		}
		if resp.StatusCode != 200 {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("non-retryable error stops immediately", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			return result{}, &http.Response{StatusCode: 500}, errors.New("server error")
		}

		_, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "server error" {
			t.Errorf("want error %q, got %q", "server error", err.Error())
		}
		if resp.StatusCode != 500 {
			t.Errorf("want status 500, got %d", resp.StatusCode)
		}
		if calls != 1 {
			t.Errorf("expected 1 call (no retry for 500), got %d", calls)
		}
	})

	t.Run("error with nil response retries then fails", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			return result{}, nil, errors.New("connection refused")
		}

		_, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "connection refused" {
			t.Errorf("want error %q, got %q", "connection refused", err.Error())
		}
		if resp != nil {
			t.Error("expected nil response")
		}
		if calls != 2 {
			t.Errorf("expected 2 calls (nil response is retryable), got %d", calls)
		}
	})

	t.Run("error with nil response then success", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			if calls == 1 {
				return result{}, nil, errors.New("connection refused")
			}
			return result{value: "recovered"}, &http.Response{StatusCode: 200}, nil
		}

		val, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val.value != "recovered" {
			t.Errorf("want value %q, got %q", "recovered", val.value)
		}
		if resp.StatusCode != 200 {
			t.Errorf("want status 200, got %d", resp.StatusCode)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls, got %d", calls)
		}
	})

	t.Run("404 on all attempts exhausts retries", func(t *testing.T) {
		calls := 0
		readFunc := func() (result, *http.Response, error) {
			calls++
			return result{}, &http.Response{StatusCode: 404}, errors.New("not found")
		}

		_, resp, err := RetryRuleRead[result](context.Background(), readFunc, "test_resource")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "not found" {
			t.Errorf("want error %q, got %q", "not found", err.Error())
		}
		if resp.StatusCode != 404 {
			t.Errorf("want status 404, got %d", resp.StatusCode)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls (exhausted retries), got %d", calls)
		}
	})
}
