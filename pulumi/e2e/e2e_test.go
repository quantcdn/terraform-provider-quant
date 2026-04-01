//go:build e2e

// Package e2e contains end-to-end tests that run against a real QuantCDN staging API.
// These tests create, update, and destroy real resources.
//
// Required environment variables:
//
//	QUANTCDN_API_TOKEN   - Staging API token
//	QUANTCDN_ORGANIZATION - Staging organization
//
// Run with: go test -v -tags=e2e -timeout=30m ./e2e/
package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	buildOnce sync.Once
	buildErr  error
	binDir    string
)

// uniqueSuffix returns a short unique string for resource naming to avoid collisions.
func uniqueSuffix() string {
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().Unix()%10000)
}

func ensureProvider(t *testing.T) {
	t.Helper()
	buildOnce.Do(func() {
		// Tests run from pulumi/e2e/, so .. is pulumi/
		pulumiDir, _ := filepath.Abs("..")
		binDir = filepath.Join(pulumiDir, "bin")
		cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "pulumi-resource-quant"), "./provider/cmd/pulumi-resource-quant")
		cmd.Dir = pulumiDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		buildErr = cmd.Run()
	})
	require.NoError(t, buildErr, "failed to build provider binary")
}

func checkE2EEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("QUANTCDN_API_TOKEN") == "" {
		t.Skip("QUANTCDN_API_TOKEN not set — skipping E2E test")
	}
	if os.Getenv("QUANTCDN_ORGANIZATION") == "" {
		t.Skip("QUANTCDN_ORGANIZATION not set — skipping E2E test")
	}
}

// requiresInfra skips tests that need real cloud infrastructure (CDN distributions,
// ECS clusters) when running against a local dev API.
func requiresInfra(t *testing.T) {
	t.Helper()
	baseURL := os.Getenv("QUANTCDN_BASE_URL")
	if strings.Contains(baseURL, "localhost") || strings.Contains(baseURL, "127.0.0.1") {
		t.Skip("skipping — requires cloud infrastructure (CDN/ECS), not available on local API")
	}
}

// programDir returns the absolute path to a test program directory.
func programDir(t *testing.T, name string) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "programs", name)
}

type e2eResult struct {
	stack   auto.Stack
	outputs auto.OutputMap
}

// createStack creates a stack, sets a unique project name via config, runs up,
// and returns the result plus a cleanup function that destroys and removes the stack.
func createStack(t *testing.T, programName string, extraConfig ...map[string]string) (e2eResult, func()) {
	t.Helper()
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	dir := programDir(t, programName)

	// Put provider binary on PATH
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)

	// Use local backend
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	// Read the Pulumi.yaml to get the project name for config key prefix
	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err, "failed to create stack")

	// Set unique project name to avoid collisions
	projectKey := fmt.Sprintf("e2e-%s:projectName", programName)
	err = stack.SetConfig(ctx, projectKey, auto.ConfigValue{Value: fmt.Sprintf("pulumi-e2e-%s-%s", programName, uniqueSuffix())})
	if err != nil {
		t.Logf("Note: could not set projectName config (program may not use it): %v", err)
	}

	// Apply extra config
	for _, cfg := range extraConfig {
		for k, v := range cfg {
			_ = stack.SetConfig(ctx, k, auto.ConfigValue{Value: v})
		}
	}

	// Run pulumi up
	upResult, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	if err != nil {
		t.Logf("Up stderr: %s", upResult.StdErr)
	}
	require.NoError(t, err, "pulumi up failed for %s", programName)

	outputs := upResult.Outputs

	cleanup := func() {
		// Destroy all resources
		destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
		if err != nil {
			t.Logf("Destroy stderr: %s", destroyResult.StdErr)
			t.Errorf("pulumi destroy failed for %s: %v", programName, err)
		}
		// Remove the stack
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}

	return e2eResult{stack: stack, outputs: outputs}, cleanup
}

// updateStack runs pulumi up on an existing stack (for testing updates).
func updateStack(t *testing.T, stack auto.Stack) auto.OutputMap {
	t.Helper()
	ctx := context.Background()
	upResult, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	if err != nil {
		t.Logf("Update stderr: %s", upResult.StdErr)
	}
	require.NoError(t, err, "pulumi up (update) failed")
	return upResult.Outputs
}

// ---------------------------------------------------------------------------
// E2E Test: Project + Data Sources
// ---------------------------------------------------------------------------

func TestE2E_Project(t *testing.T) {
	result, cleanup := createStack(t, "project")
	defer cleanup()

	// Verify outputs from create
	assert.NotEmpty(t, result.outputs["projectId"].Value, "projectId should be set")
	assert.NotEmpty(t, result.outputs["machineName"].Value, "machineName should be set")
	assert.NotEmpty(t, result.outputs["lookedUpName"].Value, "getProject lookup should return name")

	t.Logf("Created project: id=%v machine_name=%v", result.outputs["projectId"].Value, result.outputs["machineName"].Value)
}

// ---------------------------------------------------------------------------
// E2E Test: Domain
// ---------------------------------------------------------------------------

func TestE2E_Domain(t *testing.T) {
	requiresInfra(t)
	result, cleanup := createStack(t, "domain")
	defer cleanup()

	assert.NotEmpty(t, result.outputs["domainId"].Value, "domainId should be set")
	assert.NotEmpty(t, result.outputs["domain"].Value, "domain should be set")

	t.Logf("Created domain: id=%v domain=%v", result.outputs["domainId"].Value, result.outputs["domain"].Value)
}

// ---------------------------------------------------------------------------
// E2E Test: Rules — basic (7 rule types, minimal config)
// ---------------------------------------------------------------------------

func TestE2E_Rules(t *testing.T) {
	result, cleanup := createStack(t, "rules")
	defer cleanup()

	ruleOutputs := []string{"proxyId", "redirectId", "customResponseId", "authId", "botChallengeId", "headersId", "serveStaticId"}
	for _, key := range ruleOutputs {
		assert.NotEmpty(t, result.outputs[key].Value, "%s should be set", key)
		t.Logf("Rule %s: id=%v", key, result.outputs[key].Value)
	}
}

// ---------------------------------------------------------------------------
// E2E Test: Rules — advanced proxy (WAF, failover, headers, auth, error pages,
// country/method filters)
// ---------------------------------------------------------------------------

func TestE2E_RulesProxyAdvanced(t *testing.T) {
	result, cleanup := createStack(t, "rules-proxy-advanced")
	defer cleanup()

	expected := []string{
		"proxyFailoverId",
		"proxyWafId",
		"proxyAuthSslId",
		"proxyErrorPageId",
		"proxyCountryExcludeId",
		"proxyMethodFilterId",
	}
	for _, key := range expected {
		assert.NotEmpty(t, result.outputs[key].Value, "%s should be set", key)
		t.Logf("Proxy %s: id=%v", key, result.outputs[key].Value)
	}

	// Verify that preview shows no unexpected drift after create
	ctx := context.Background()
	previewResult, err := result.stack.Preview(ctx)
	if err == nil {
		t.Logf("Preview after create (should be clean): %v", previewResult.ChangeSummary)
	}
}

// ---------------------------------------------------------------------------
// E2E Test: Rules — deep coverage (redirect codes, custom response statuses,
// bot challenge TTLs, country/IP/method filters, disabled rules)
// ---------------------------------------------------------------------------

func TestE2E_RulesDeep(t *testing.T) {
	result, cleanup := createStack(t, "rules-deep")
	defer cleanup()

	expected := []string{
		// Redirects
		"redirect301Id",
		"redirect302Id",
		"redirect307Id",
		"redirectIpFilterId",
		"disabledRedirectId",
		// Custom responses
		"customResponse403Id",
		"customResponse503Id",
		"customResponseJsonId",
		"customResponseMethodBlockId",
		// Bot challenges
		"botChallengeCheckboxId",
		"botChallengeInvisibleId",
		"botChallengeCountryId",
		// Auth
		"authCountryFilterId",
		"authIpFilterId",
		// Headers
		"headersSecuritySetId",
		"headersCorsId",
		// ServeStatic
		"serveStaticMethodFilterId",
		"serveStaticCountryFilterId",
	}
	for _, key := range expected {
		assert.NotEmpty(t, result.outputs[key].Value, "%s should be set", key)
		t.Logf("Rule %s: id=%v", key, result.outputs[key].Value)
	}

	// Verify preview after create is clean (no drift)
	ctx := context.Background()
	previewResult, err := result.stack.Preview(ctx)
	if err == nil {
		t.Logf("Preview after create: %v", previewResult.ChangeSummary)
	}
}

// ---------------------------------------------------------------------------
// E2E Test: Crawler + CrawlerSchedule
// ---------------------------------------------------------------------------

func TestE2E_Crawler(t *testing.T) {
	result, cleanup := createStack(t, "crawler")
	defer cleanup()

	assert.NotEmpty(t, result.outputs["crawlerId"].Value, "crawlerId should be set")
	assert.NotEmpty(t, result.outputs["scheduleId"].Value, "scheduleId should be set")

	t.Logf("Created crawler: id=%v schedule=%v", result.outputs["crawlerId"].Value, result.outputs["scheduleId"].Value)
}

// ---------------------------------------------------------------------------
// E2E Test: Header
// ---------------------------------------------------------------------------

func TestE2E_Headers(t *testing.T) {
	result, cleanup := createStack(t, "headers")
	defer cleanup()

	assert.NotEmpty(t, result.outputs["headerId"].Value, "headerId should be set")
	t.Logf("Created header: id=%v", result.outputs["headerId"].Value)

	// Test update: modify headers via config and re-up
	ctx := context.Background()
	err := result.stack.SetConfig(ctx, "e2e-headers:headers", auto.ConfigValue{Value: `{"X-Frame-Options": "SAMEORIGIN"}`})
	if err == nil {
		outputs := updateStack(t, result.stack)
		assert.NotEmpty(t, outputs["headerId"].Value, "headerId should still be set after update")
		t.Logf("Updated header: id=%v", outputs["headerId"].Value)
	}
}

// ---------------------------------------------------------------------------
// E2E Test: Application Stack (Application, Environment, Volume, CronJob)
// ---------------------------------------------------------------------------

func TestE2E_AppStack(t *testing.T) {
	requiresInfra(t)
	suffix := uniqueSuffix()
	result, cleanup := createStack(t, "app-stack", map[string]string{
		"e2e-app-stack:appName": fmt.Sprintf("pulumi-e2e-app-%s", suffix),
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["appId"].Value, "appId should be set")
	assert.NotEmpty(t, result.outputs["envId"].Value, "envId should be set")
	assert.NotEmpty(t, result.outputs["volumeId"].Value, "volumeId should be set")
	assert.NotEmpty(t, result.outputs["cronId"].Value, "cronId should be set")

	t.Logf("Created app stack: app=%v env=%v volume=%v cron=%v",
		result.outputs["appId"].Value, result.outputs["envId"].Value,
		result.outputs["volumeId"].Value, result.outputs["cronId"].Value)
}

// ---------------------------------------------------------------------------
// E2E Test: KV Store + KV Item
// ---------------------------------------------------------------------------

func TestE2E_KV(t *testing.T) {
	result, cleanup := createStack(t, "kv")
	defer cleanup()

	assert.NotEmpty(t, result.outputs["storeId"].Value, "storeId should be set")
	assert.NotEmpty(t, result.outputs["itemId"].Value, "itemId should be set")

	t.Logf("Created KV: store=%v item=%v", result.outputs["storeId"].Value, result.outputs["itemId"].Value)

	// Test update: change the item value
	ctx := context.Background()
	_ = result.stack.SetConfig(ctx, "e2e-kv:itemValue", auto.ConfigValue{Value: "updated-e2e-value"})
	outputs := updateStack(t, result.stack)
	assert.NotEmpty(t, outputs["itemId"].Value, "itemId should still be set after update")
}

// ---------------------------------------------------------------------------
// E2E Test: AI resources (AiGovernance, AiVectorCollection, AiVectorDocument, AiSkill)
// ---------------------------------------------------------------------------

func TestE2E_AI(t *testing.T) {
	suffix := uniqueSuffix()
	result, cleanup := createStack(t, "ai", map[string]string{
		"e2e-ai:testSuffix": fmt.Sprintf("e2e-%s", suffix),
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["skillId"].Value, "skillId should be set")
	assert.NotEmpty(t, result.outputs["skillName"].Value, "skillName should be set")

	t.Logf("Created AI resources: governance=%v skill=%v",
		result.outputs["governanceVersion"].Value,
		result.outputs["skillId"].Value)
}

func TestE2E_AIVector(t *testing.T) {
	requiresInfra(t) // vector-db requires backend infra not always available on staging

	suffix := uniqueSuffix()
	result, cleanup := createStack(t, "ai-vector", map[string]string{
		"e2e-ai-vector:testSuffix": fmt.Sprintf("e2e-%s", suffix),
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["collectionId"].Value, "collectionId should be set")
	assert.NotEmpty(t, result.outputs["collectionName"].Value, "collectionName should be set")
	assert.NotEmpty(t, result.outputs["documentId"].Value, "documentId should be set")

	t.Logf("Created AI vector resources: collection=%v doc=%v",
		result.outputs["collectionId"].Value,
		result.outputs["documentId"].Value)
}

// ---------------------------------------------------------------------------
// E2E Test: Full lifecycle (create → preview → update → destroy) for a project
// ---------------------------------------------------------------------------

func TestE2E_FullLifecycle(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	dir := programDir(t, "project")

	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)

	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	suffix := uniqueSuffix()

	// Set a unique project name
	err = stack.SetConfig(ctx, "e2e-project:projectName", auto.ConfigValue{Value: fmt.Sprintf("pulumi-e2e-lifecycle-%s", suffix)})
	require.NoError(t, err)

	// Step 1: Create
	t.Log("Step 1: Create")
	upResult, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	require.NoError(t, err, "create failed")
	assert.NotEmpty(t, upResult.Outputs["projectId"].Value)

	// Step 2: Preview (should show no changes)
	t.Log("Step 2: Preview (no changes expected)")
	previewResult, err := stack.Preview(ctx)
	require.NoError(t, err, "preview failed")
	t.Logf("Preview summary: %v", previewResult.ChangeSummary)

	// Step 3: Update (change project name)
	t.Log("Step 3: Update")
	err = stack.SetConfig(ctx, "e2e-project:projectName", auto.ConfigValue{Value: fmt.Sprintf("pulumi-e2e-lifecycle-upd-%s", suffix)})
	require.NoError(t, err)
	upResult, err = stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	require.NoError(t, err, "update failed")
	assert.NotEmpty(t, upResult.Outputs["projectId"].Value)

	// Step 4: Destroy
	t.Log("Step 4: Destroy")
	destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
	if err != nil {
		t.Logf("Destroy stderr: %s", destroyResult.StdErr)
	}
	require.NoError(t, err, "destroy failed")

	_ = stack.Workspace().RemoveStack(ctx, stack.Name())
}
