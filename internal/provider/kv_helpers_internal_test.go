package provider

import (
	"strings"
	"testing"
	"time"

	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
)

func TestParseKVImportID(t *testing.T) {
	ref, err := parseKVImportID("p/s/n")
	if err != nil || ref != (kvRef{Project: "p", StoreID: "s", Name: "n"}) {
		t.Fatalf("3 parts: %v %v", ref, err)
	}
	ref, err = parseKVImportID("o/p/s/n")
	if err != nil || ref.Organization != "o" || ref.Name != "n" {
		t.Fatalf("4 parts: %v %v", ref, err)
	}
	for _, bad := range []string{"p/s", "o/p/s/n/x", "p//n"} {
		if _, err := parseKVImportID(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}

func TestWaitTimeoutFromEnvironment(t *testing.T) {
	t.Setenv("QUANTCDN_TEST_TIMEOUT", "")
	if got := waitTimeout("QUANTCDN_TEST_TIMEOUT", time.Minute); got != time.Minute {
		t.Fatalf("unset: %s", got)
	}
	t.Setenv("QUANTCDN_TEST_TIMEOUT", "30m")
	if got := waitTimeout("QUANTCDN_TEST_TIMEOUT", time.Minute); got != 30*time.Minute {
		t.Fatalf("30m: %s", got)
	}
	t.Setenv("QUANTCDN_TEST_TIMEOUT", "-1s")
	if got := waitTimeout("QUANTCDN_TEST_TIMEOUT", time.Minute); got != time.Minute {
		t.Fatalf("negative: %s", got)
	}
}

func kvErr(body string) error {
	e := quantadmingo.GenericOpenAPIError{}
	return withBody{e, []byte(body)}
}

type withBody struct {
	quantadmingo.GenericOpenAPIError
	b []byte
}

func (w withBody) Body() []byte  { return w.b }
func (w withBody) Error() string { return "409 Conflict" }

func TestKVLedgerDropRefusedMessages(t *testing.T) {
	retained := kvLedgerDropRefused("aiUse", kvErr(`{"code":"LEDGER_RETAINED","reason":"retention","retainUntil":"2033-10-14T00:00:00.000Z","message":"m"}`))
	for _, want := range []string{"2033-10-14T00:00:00.000Z", "removed {", "destroy = false", "terraform state rm", "retainOnDelete", "pulumi state delete"} {
		if !strings.Contains(retained, want) {
			t.Errorf("retained message lacks %q: %s", want, retained)
		}
	}
	unarchived := kvLedgerDropRefused("aiUse", kvErr(`{"code":"LEDGER_RETAINED","reason":"unarchived","collection":"ai-use:a","shard":"03","message":"m"}`))
	if !strings.Contains(unarchived, "Try again in a few minutes") || !strings.Contains(unarchived, "ai-use:a 03") {
		t.Errorf("unarchived message: %s", unarchived)
	}
	other := kvLedgerDropRefused("aiUse", kvErr(`{"code":"FORBIDDEN","message":"no"}`))
	if strings.Contains(other, "removed {") {
		t.Errorf("a non-retention refusal must not suggest removed: %s", other)
	}
}
