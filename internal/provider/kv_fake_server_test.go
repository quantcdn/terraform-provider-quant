package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// kvFake is an httptest server for the KV API of one project. Routes are
// "METHOD /path" below /api/v2/organizations/test-org/projects/test-project/kv.
// An unknown route answers 404 with a KV error body.
type kvFake struct {
	t      *testing.T
	srv    *httptest.Server
	mu     sync.Mutex
	routes map[string]func(body map[string]any) (int, any)
	calls  []kvCall
}

type kvCall struct {
	Route string
	Body  map[string]any
}

const kvFakeBase = "/api/v2/organizations/test-org/projects/test-project/kv"

func newKVFake(t *testing.T) *kvFake {
	t.Helper()
	f := &kvFake{t: t, routes: map[string]func(map[string]any) (int, any){}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *kvFake) on(route string, h func(body map[string]any) (int, any)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[route] = h
}

func (f *kvFake) json(route string, status int, body any) {
	f.on(route, func(map[string]any) (int, any) { return status, body })
}

func (f *kvFake) serve(w http.ResponseWriter, req *http.Request) {
	var body map[string]any
	if b, _ := io.ReadAll(req.Body); len(b) > 0 {
		_ = json.Unmarshal(b, &body)
	}
	route := req.Method + " " + strings.TrimPrefix(req.URL.Path, kvFakeBase)
	f.mu.Lock()
	h, ok := f.routes[route]
	f.calls = append(f.calls, kvCall{Route: route, Body: body})
	f.mu.Unlock()
	status, out := 404, any(map[string]any{"error": true, "code": "NOT_FOUND", "message": "no route " + route, "retryable": false})
	if ok {
		status, out = h(body)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(out)
}

// bodies are the request bodies sent to route, in order.
func (f *kvFake) bodies(route string) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []map[string]any
	for _, c := range f.calls {
		if c.Route == route {
			out = append(out, c.Body)
		}
	}
	return out
}

func (f *kvFake) count(route string) int { return len(f.bodies(route)) }

// providerBlock configures the provider against the fake, with no retries.
func (f *kvFake) providerBlock() string {
	return fmt.Sprintf(`
provider "quant" {
  organization = "test-org"
  bearer       = "test-token"
  base_url     = %q
  max_retries  = 0
}
`, f.srv.URL)
}
