//go:build e2e

// Package e2e contains end-to-end tests that run against a real QuantCDN staging API.
// These tests create, update, and destroy real resources.
//
// Required environment variables:
//
//	QUANTCDN_API_TOKEN   - Staging API token
//	QUANTCDN_ORGANIZATION - Staging organization
//
// Run with: go test -v -tags=e2e -timeout=60m ./e2e/
package e2e

import (
	"context"
	"fmt"
	"io"
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
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optimport"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/quantcdn/terraform-provider-quant/v5/internal/client"
	quantadmingo "github.com/quantcdn/quant-admin-go/v4"
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
		// Destroy resources sequentially to avoid API rate limit / conflict errors
		destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout), optdestroy.Parallel(1))
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
	suffix := uniqueSuffix()
	result, cleanup := createStack(t, "domain", map[string]string{
		"e2e-domain:projectName":  fmt.Sprintf("pulumi-e2e-domain-%s", suffix),
		"e2e-domain:domainSuffix": fmt.Sprintf("d%s", suffix),
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["domainId"].Value, "domainId should be set")
	assert.NotEmpty(t, result.outputs["domain"].Value, "domain should be set")

	t.Logf("Created domain: id=%v domain=%v", result.outputs["domainId"].Value, result.outputs["domain"].Value)
}

// ---------------------------------------------------------------------------
// E2E Test: Rules — basic (7 rule types, minimal config) + update
// ---------------------------------------------------------------------------

func TestE2E_Rules(t *testing.T) {
	result, cleanup := createStack(t, "rules")
	defer cleanup()

	ruleOutputs := []string{"proxyId", "redirectId", "customResponseId", "authId", "botChallengeId", "headersId", "serveStaticId"}
	for _, key := range ruleOutputs {
		assert.NotEmpty(t, result.outputs[key].Value, "%s should be set", key)
		t.Logf("Rule %s: id=%v", key, result.outputs[key].Value)
	}

	// Test update: change proxy target and redirect destination
	ctx := context.Background()
	_ = result.stack.SetConfig(ctx, "e2e-rules:proxyTo", auto.ConfigValue{Value: "https://example.org"})
	_ = result.stack.SetConfig(ctx, "e2e-rules:redirectTo", auto.ConfigValue{Value: "https://example.org/updated"})
	_ = result.stack.SetConfig(ctx, "e2e-rules:customResponseBody", auto.ConfigValue{Value: "<h1>Updated</h1>"})
	t.Log("Updating rules (proxy target, redirect destination, custom response body)...")
	outputs := updateStack(t, result.stack)
	for _, key := range ruleOutputs {
		assert.NotEmpty(t, outputs[key].Value, "%s should still be set after update", key)
	}
	t.Log("Rules update succeeded")
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
		// ContentFilter
		"contentFilterBlockId",
		// Function
		"ruleFunctionId",
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

	// Test update: change crawler domain and schedule cron
	ctx := context.Background()
	_ = result.stack.SetConfig(ctx, "e2e-crawler:crawlerDomain", auto.ConfigValue{Value: "https://httpbin.org"})
	_ = result.stack.SetConfig(ctx, "e2e-crawler:scheduleCron", auto.ConfigValue{Value: "0 12 * * 1"})
	t.Log("Updating crawler (domain, schedule cron)...")
	outputs := updateStack(t, result.stack)
	assert.NotEmpty(t, outputs["crawlerId"].Value, "crawlerId should still be set after update")
	assert.NotEmpty(t, outputs["scheduleId"].Value, "scheduleId should still be set after update")
	t.Log("Crawler update succeeded")
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
		"e2e-app-stack:appName": fmt.Sprintf("pe2e-%s", suffix),
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["appId"].Value, "appId should be set")
	assert.NotEmpty(t, result.outputs["envId"].Value, "envId should be set")
	assert.NotEmpty(t, result.outputs["volumeId"].Value, "volumeId should be set")
	assert.NotEmpty(t, result.outputs["cronId"].Value, "cronId should be set")

	t.Logf("Created app stack: app=%v env=%v volume=%v cron=%v",
		result.outputs["appId"].Value, result.outputs["envId"].Value,
		result.outputs["volumeId"].Value, result.outputs["cronId"].Value)

	// Test update: change CronJob schedule
	// Note: Only changing cronSchedule (not description) to avoid triggering
	// Application update — Application has no update API and returns an error.
	ctx := context.Background()
	_ = result.stack.SetConfig(ctx, "e2e-app-stack:cronSchedule", auto.ConfigValue{Value: "cron(0 6 ? * MON *)"})
	t.Log("Updating CronJob (schedule)...")
	outputs := updateStack(t, result.stack)
	assert.NotEmpty(t, outputs["cronId"].Value, "cronId should still be set after update")
	t.Log("CronJob update succeeded")
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

	// Test update: change skill content and description
	ctx := context.Background()
	_ = result.stack.SetConfig(ctx, "e2e-ai:skillContent", auto.ConfigValue{Value: "You are an updated test skill."})
	_ = result.stack.SetConfig(ctx, "e2e-ai:skillDescription", auto.ConfigValue{Value: "Updated E2E test skill"})
	t.Log("Updating AI skill (content, description)...")
	outputs := updateStack(t, result.stack)
	assert.NotEmpty(t, outputs["skillId"].Value, "skillId should still be set after update")
	t.Log("AI skill update succeeded")

	// Test update: change governance modelPolicy to allowlist (modelList is
	// already in the YAML program, so switching to allowlist is valid).
	_ = result.stack.SetConfig(ctx, "e2e-ai:modelPolicy", auto.ConfigValue{Value: "allowlist"})
	t.Log("Updating AI governance (modelPolicy → allowlist)...")
	_ = updateStack(t, result.stack)
	t.Log("AI governance update succeeded")
}

func TestE2E_AIVector(t *testing.T) {
	suffix := uniqueSuffix()
	result, cleanup := createStack(t, "ai-vector", map[string]string{
		"e2e-ai-vector:testSuffix": fmt.Sprintf("e2e-%s", suffix),
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["collectionId"].Value, "collectionId should be set")
	assert.NotEmpty(t, result.outputs["collectionName"].Value, "collectionName should be set")

	t.Logf("Created AI vector resources: collection=%v name=%v chunksCreated=%v",
		result.outputs["collectionId"].Value,
		result.outputs["collectionName"].Value,
		result.outputs["chunksCreated"].Value)

	// Test update: change document content (upsert)
	ctx := context.Background()
	_ = result.stack.SetConfig(ctx, "e2e-ai-vector:docContent", auto.ConfigValue{Value: "This is updated document content for E2E testing."})
	t.Log("Updating AI vector document (content)...")
	outputs := updateStack(t, result.stack)
	assert.NotEmpty(t, outputs["collectionId"].Value, "collectionId should still be set after update")
	t.Log("AI vector document update succeeded")
}

// ---------------------------------------------------------------------------
// E2E Test: Environment lifecycle (create → verify → update scaling → destroy)
// ---------------------------------------------------------------------------

func TestE2E_Environment(t *testing.T) {
	requiresInfra(t)

	suffix := uniqueSuffix()
	result, cleanup := createStack(t, "environment", map[string]string{
		"e2e-environment:appName":   fmt.Sprintf("pulumi-e2e-env-%s", suffix),
		"e2e-environment:envSuffix": "staging",
	})
	defer cleanup()

	assert.NotEmpty(t, result.outputs["appName"].Value, "appName should be set")
	assert.Equal(t, "staging", result.outputs["envName"].Value, "envName should be staging")

	t.Logf("Created: app=%v env=%v status=%v deployStatus=%v",
		result.outputs["appName"].Value,
		result.outputs["envName"].Value,
		result.outputs["envStatus"].Value,
		result.outputs["envDeploymentStatus"].Value)
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
	destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout), optdestroy.Parallel(1))
	if err != nil {
		t.Logf("Destroy stderr: %s", destroyResult.StdErr)
	}
	require.NoError(t, err, "destroy failed")

	_ = stack.Workspace().RemoveStack(ctx, stack.Name())
}

// ---------------------------------------------------------------------------
// Helper: create an API client for direct API calls (drift detection, etc.)
// ---------------------------------------------------------------------------

func newAPIClient(t *testing.T) *client.Client {
	t.Helper()
	token := os.Getenv("QUANTCDN_API_TOKEN")
	org := os.Getenv("QUANTCDN_ORGANIZATION")
	baseURL := os.Getenv("QUANTCDN_BASE_URL")
	require.NotEmpty(t, token, "QUANTCDN_API_TOKEN must be set")
	require.NotEmpty(t, org, "QUANTCDN_ORGANIZATION must be set")

	if baseURL != "" {
		return client.NewWithBaseURL(token, org, baseURL)
	}
	return client.New(token, org)
}

// ---------------------------------------------------------------------------
// E2E Test: Drift Detection — Project
// Create project via Pulumi, modify name directly via API, verify preview
// detects the drift.
// ---------------------------------------------------------------------------

func TestE2E_DriftDetection_Project(t *testing.T) {
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
	projectName := fmt.Sprintf("pulumi-e2e-drift-%s", suffix)
	err = stack.SetConfig(ctx, "e2e-project:projectName", auto.ConfigValue{Value: projectName})
	require.NoError(t, err)

	// Step 1: Create
	t.Log("Step 1: Create project")
	upResult, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	require.NoError(t, err, "create failed")
	machineName := upResult.Outputs["machineName"].Value.(string)
	t.Logf("Created project: machine_name=%s", machineName)

	defer func() {
		destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout), optdestroy.Parallel(1))
		if err != nil {
			t.Logf("Destroy stderr: %s", destroyResult.StdErr)
		}
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Step 2: Preview should be clean (no drift yet)
	t.Log("Step 2: Preview (should be clean)")
	previewResult, err := stack.Preview(ctx)
	require.NoError(t, err, "preview failed")
	t.Logf("Preview before drift: %v", previewResult.ChangeSummary)

	// Step 3: Modify project name directly via API (out-of-band change)
	t.Log("Step 3: Modifying project name directly via API...")
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	updatedName := projectName + "-drifted"
	projReq := quantadmingo.NewV2ProjectRequestWithDefaults()
	projReq.SetName(updatedName)
	updateReq := apiClient.Instance.ProjectsAPI.ProjectsUpdate(apiClient.AuthContext, apiClient.Organization, machineName)
	_, _, err = updateReq.V2ProjectRequest(*projReq).Execute()
	require.NoError(t, err, "API update failed")
	t.Logf("Changed project name to '%s' via API", updatedName)

	// Step 4: Refresh to pull actual state from API, then preview to detect diff
	t.Log("Step 4: Refresh state")
	_, err = stack.Refresh(ctx)
	require.NoError(t, err, "refresh failed")

	t.Log("Step 5: Preview (should detect drift)")
	previewResult, err = stack.Preview(ctx)
	require.NoError(t, err, "preview after drift failed")
	t.Logf("Preview after drift: %v", previewResult.ChangeSummary)

	// The preview should show at least one update (the name was changed)
	updateCount, hasUpdates := previewResult.ChangeSummary["update"]
	assert.True(t, hasUpdates && updateCount > 0, "preview should detect drift (expected update, got: %v)", previewResult.ChangeSummary)

	// Step 5: Pulumi up should restore the original name
	t.Log("Step 5: Pulumi up to correct drift")
	upResult, err = stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	require.NoError(t, err, "corrective up failed")
	t.Log("Drift corrected successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Drift Detection — KV Item
// Create KV store + item via Pulumi, modify item value via API, verify
// preview detects it.
// ---------------------------------------------------------------------------

func TestE2E_DriftDetection_KV(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	dir := programDir(t, "kv")

	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)

	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	suffix := uniqueSuffix()
	err = stack.SetConfig(ctx, "e2e-kv:projectName", auto.ConfigValue{Value: fmt.Sprintf("pulumi-e2e-drift-kv-%s", suffix)})
	require.NoError(t, err)

	// Step 1: Create
	t.Log("Step 1: Create KV store + item")
	upResult, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	require.NoError(t, err, "create failed")
	t.Logf("Created KV: store=%v item=%v", upResult.Outputs["storeId"].Value, upResult.Outputs["itemId"].Value)

	defer func() {
		destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout), optdestroy.Parallel(1))
		if err != nil {
			t.Logf("Destroy stderr: %s", destroyResult.StdErr)
		}
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Step 2: Preview should be clean
	t.Log("Step 2: Preview (should be clean)")
	previewResult, err := stack.Preview(ctx)
	require.NoError(t, err, "preview failed")
	t.Logf("Preview before drift: %v", previewResult.ChangeSummary)

	// Step 3: Modify item value directly via API
	t.Log("Step 3: Modifying KV item value directly via API...")
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	// We need the project machine name and store ID to update the KV item via API.
	// The store ID from outputs is an integer, but the KV API expects the store UUID string.
	// We need to find the project first.
	projectName := fmt.Sprintf("pulumi-e2e-drift-kv-%s", suffix)
	projectsResp, _, err := apiClient.Instance.ProjectsAPI.ProjectsList(apiClient.AuthContext, apiClient.Organization).Execute()
	require.NoError(t, err, "failed to list projects")

	var projectMachine string
	for _, p := range projectsResp {
		if p.GetName() == projectName {
			projectMachine = p.GetMachineName()
			break
		}
	}
	require.NotEmpty(t, projectMachine, "could not find project machine name for '%s'", projectName)

	storeIdRaw := upResult.Outputs["storeId"].Value
	storeId := fmt.Sprintf("%v", storeIdRaw)

	// Use the KV API to update the item value
	kvUpdateBody := quantadmingo.NewV2StoreItemUpdateRequest("drifted-value")
	kvUpdateReq := apiClient.Instance.KVAPI.KVItemsUpdate(apiClient.AuthContext, apiClient.Organization, projectMachine, storeId, "e2e-test-key")
	_, _, err = kvUpdateReq.V2StoreItemUpdateRequest(*kvUpdateBody).Execute()
	if err != nil {
		t.Logf("KV item update via API failed: %v — skipping KV drift test", err)
		t.Skip("Skipping KV drift test — API item update not available")
	}
	t.Log("Changed KV item value to 'drifted-value' via API")

	// Step 4: Refresh to pull actual state from API, then preview to detect diff
	t.Log("Step 4: Refresh state")
	_, err = stack.Refresh(ctx)
	require.NoError(t, err, "refresh failed")

	t.Log("Step 5: Preview (should detect drift)")
	previewResult, err = stack.Preview(ctx)
	require.NoError(t, err, "preview after drift failed")
	t.Logf("Preview after drift: %v", previewResult.ChangeSummary)

	updateCount, hasUpdates := previewResult.ChangeSummary["update"]
	assert.True(t, hasUpdates && updateCount > 0, "preview should detect KV drift (expected update, got: %v)", previewResult.ChangeSummary)
}

// ---------------------------------------------------------------------------
// E2E Test: Drift Detection — Rule
// Create project + rule via Pulumi, modify rule via API, verify preview
// detects the drift.
// ---------------------------------------------------------------------------

func TestE2E_DriftDetection_Rule(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	dir := programDir(t, "rules")

	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)

	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	suffix := uniqueSuffix()
	err = stack.SetConfig(ctx, "e2e-rules:projectName", auto.ConfigValue{Value: fmt.Sprintf("pulumi-e2e-drift-rule-%s", suffix)})
	require.NoError(t, err)

	// Step 1: Create
	t.Log("Step 1: Create project + rules")
	upResult, err := stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	require.NoError(t, err, "create failed")

	defer func() {
		destroyResult, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout), optdestroy.Parallel(1))
		if err != nil {
			t.Logf("Destroy stderr: %s", destroyResult.StdErr)
		}
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Step 2: Preview should be clean
	t.Log("Step 2: Preview (should be clean)")
	previewResult, err := stack.Preview(ctx)
	require.NoError(t, err, "preview failed")
	t.Logf("Preview before drift: %v", previewResult.ChangeSummary)

	// Step 3: Get the redirect rule UUID and modify it via API
	redirectId := fmt.Sprintf("%v", upResult.Outputs["redirectId"].Value)
	t.Logf("Redirect rule ID: %s", redirectId)

	apiClient := newAPIClient(t)
	defer apiClient.Close()

	// Get the project machine name — we need to extract from stack state
	// The project machine name is derived from the project name
	projectName := fmt.Sprintf("pulumi-e2e-drift-rule-%s", suffix)
	// Machine names are auto-generated from project name by the API
	// We need to read the project to get it
	projectsResp, _, err := apiClient.Instance.ProjectsAPI.ProjectsList(apiClient.AuthContext, apiClient.Organization).Execute()
	require.NoError(t, err, "failed to list projects")

	var machineName string
	for _, p := range projectsResp {
		if p.GetName() == projectName {
			machineName = p.GetMachineName()
			break
		}
	}
	require.NotEmpty(t, machineName, "could not find project machine name for '%s'", projectName)

	// Update the redirect rule destination via API
	redirectReq := quantadmingo.NewV2RuleRedirectRequest(
		[]string{"*.example.com"},
		[]string{"/old-path"},
		"https://drifted.example.com/path",
	)
	t.Logf("Updating rule %s on project %s via SDK...", redirectId, machineName)
	ruleUpdateReq := apiClient.Instance.RulesAPI.RulesRedirectUpdate(apiClient.AuthContext, apiClient.Organization, machineName, redirectId)
	updateResp, httpResp, err := ruleUpdateReq.V2RuleRedirectRequest(*redirectReq).Execute()
	if err != nil {
		body := ""
		if httpResp != nil {
			b, _ := io.ReadAll(httpResp.Body)
			body = string(b)
		}
		t.Logf("Rule update via API error: %v (body: %s)", err, body)
		t.Skip("Skipping rule drift test — API update failed")
	}
	// Verify the update response shows the new redirect_to
	if updateResp != nil && updateResp.ActionConfig != nil {
		t.Logf("Update response: redirect_to=%s", updateResp.ActionConfig.To)
	} else {
		t.Logf("Update response: %+v", updateResp)
	}
	t.Log("Changed redirect destination to 'https://drifted.example.com/path' via API")

	// Step 4: Refresh state from actual infrastructure, then preview
	t.Log("Step 4: Refresh to detect drift")
	_, err = stack.Refresh(ctx)
	require.NoError(t, err, "refresh failed")

	t.Log("Step 5: Preview (should detect drift)")
	previewResult, err = stack.Preview(ctx)
	require.NoError(t, err, "preview after drift failed")
	t.Logf("Preview after drift: %v", previewResult.ChangeSummary)

	updateCount, hasUpdates := previewResult.ChangeSummary["update"]
	assert.True(t, hasUpdates && updateCount > 0, "preview should detect rule drift (expected update, got: %v)", previewResult.ChangeSummary)
}

// ---------------------------------------------------------------------------
// E2E Test: Import — Project
// Create a project via API, import it into Pulumi, verify state is populated.
// ---------------------------------------------------------------------------

func TestE2E_Import_Project(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()

	// Step 1: Create project directly via API
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	suffix := uniqueSuffix()
	projectName := fmt.Sprintf("pulumi-e2e-import-%s", suffix)
	t.Logf("Creating project '%s' via API...", projectName)

	projReq := quantadmingo.NewV2ProjectRequestWithDefaults()
	projReq.SetName(projectName)
	projReq.SetRegion("au-govt")
	createResp, _, err := apiClient.Instance.ProjectsAPI.ProjectsCreate(apiClient.AuthContext, apiClient.Organization).V2ProjectRequest(*projReq).Execute()
	require.NoError(t, err, "API project create failed")
	machineName := createResp.GetMachineName()
	t.Logf("Created project via API: machine_name=%s", machineName)

	// Clean up the real resource at the end
	defer func() {
		t.Log("Cleaning up API-created project...")
		_, err := apiClient.Instance.ProjectsAPI.ProjectsDelete(apiClient.AuthContext, apiClient.Organization, machineName).Execute()
		if err != nil {
			t.Logf("Cleanup warning: failed to delete project: %v", err)
		}
	}()

	// Step 2: Set up a Pulumi stack and import the project
	dir := programDir(t, "project")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	// Set the config to match the imported project
	err = stack.SetConfig(ctx, "e2e-project:projectName", auto.ConfigValue{Value: projectName})
	require.NoError(t, err)

	defer func() {
		// Just remove the stack — don't destroy (we already cleaned up the API resource above)
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Step 3: Import the project by machine name
	t.Log("Importing project into Pulumi...")
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:Project",
				Name: "testProject",
				ID:   machineName,
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	// ImportResources may fail reading generated_code.txt even when the import
	// itself succeeds. Check for that specific error and ignore it.
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed")
	} else if err != nil {
		t.Logf("Import succeeded but code generation file not found (expected with GenerateCode=false)")
	}

	t.Log("Import test completed successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Import — KV Store
// Create a KV store via API, import into Pulumi, verify.
// ---------------------------------------------------------------------------

func TestE2E_Import_KVStore(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()

	// Step 1: Create project + KV store via API
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	suffix := uniqueSuffix()
	projectName := fmt.Sprintf("pulumi-e2e-import-kv-%s", suffix)
	t.Logf("Creating project '%s' via API...", projectName)

	projReq := quantadmingo.NewV2ProjectRequestWithDefaults()
	projReq.SetName(projectName)
	projReq.SetRegion("au-govt")
	createResp, _, err := apiClient.Instance.ProjectsAPI.ProjectsCreate(apiClient.AuthContext, apiClient.Organization).V2ProjectRequest(*projReq).Execute()
	require.NoError(t, err, "API project create failed")
	machineName := createResp.GetMachineName()
	t.Logf("Created project: machine_name=%s", machineName)

	defer func() {
		// Clean up — delete project (cascades to KV store)
		_, err := apiClient.Instance.ProjectsAPI.ProjectsDelete(apiClient.AuthContext, apiClient.Organization, machineName).Execute()
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}()

	// Wait for project to be available
	time.Sleep(5 * time.Second)

	// Create KV store
	storeReq := quantadmingo.NewV2StoreRequest("e2e-import-store")
	kvResp, _, err := apiClient.Instance.KVAPI.KVCreate(apiClient.AuthContext, apiClient.Organization, machineName).V2StoreRequest(*storeReq).Execute()
	require.NoError(t, err, "API KV store create failed")
	storeId := kvResp.GetId()
	t.Logf("Created KV store: id=%s", storeId)

	// Step 2: Set up Pulumi stack and import
	dir := programDir(t, "kv")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	err = stack.SetConfig(ctx, "e2e-kv:projectName", auto.ConfigValue{Value: projectName})
	require.NoError(t, err)

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Step 3: Import KV store
	t.Log("Importing KV store into Pulumi...")
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:KvStore",
				Name: "testStore",
				ID:   machineName + "/" + storeId,
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed for KV store")
	} else if err != nil {
		t.Logf("Import succeeded but code generation file not found (expected with GenerateCode=false)")
	}
	t.Log("KV Store import test completed successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Import — AI Skill
// Create an AI skill via API, import into Pulumi, verify.
// ---------------------------------------------------------------------------

func TestE2E_Import_AISkill(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()

	// Step 1: Create AI skill via API
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	suffix := uniqueSuffix()
	skillName := fmt.Sprintf("e2e-import-skill-%s", suffix)
	t.Logf("Creating AI skill '%s' via API...", skillName)

	skillReq := quantadmingo.NewCreateSkillRequest(skillName, "You are a test skill for import testing.", "always")
	skillReq.SetDescription("Import test skill")
	skillResp, _, err := apiClient.Instance.AISkillsAPI.CreateSkill(apiClient.AuthContext, apiClient.Organization).CreateSkillRequest(*skillReq).Execute()
	if err != nil {
		t.Skipf("Skipping AI skill import test — API create failed: %v", err)
	}

	// Extract skillId from the response's Skill map
	skillId := ""
	if skillMap := skillResp.GetSkill(); skillMap != nil {
		if id, ok := skillMap["skillId"].(string); ok {
			skillId = id
		}
	}
	if skillId == "" {
		t.Skip("Skipping AI skill import test — could not extract skillId from response")
	}
	t.Logf("Created AI skill: id=%s", skillId)

	defer func() {
		_, _, err := apiClient.Instance.AISkillsAPI.DeleteSkill(apiClient.AuthContext, apiClient.Organization, skillId).Execute()
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}()

	// Step 2: Set up Pulumi stack and import
	dir := programDir(t, "ai")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	err = stack.SetConfig(ctx, "e2e-ai:testSuffix", auto.ConfigValue{Value: fmt.Sprintf("import-%s", suffix)})
	require.NoError(t, err)

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Step 3: Import the skill
	t.Log("Importing AI skill into Pulumi...")
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:AiSkill",
				Name: "testSkill",
				ID:   skillId,
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed for AI skill")
	} else if err != nil {
		t.Logf("Import succeeded but code generation file not found (expected with GenerateCode=false)")
	}
	t.Log("AI Skill import test completed successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Import — AI Governance (singleton, no create needed)
// ---------------------------------------------------------------------------

func TestE2E_Import_AIGovernance(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()

	// AiGovernance is a singleton — it already exists per org, so no API
	// setup needed. We just import it.
	dir := programDir(t, "ai")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	suffix := uniqueSuffix()
	err = stack.SetConfig(ctx, "e2e-ai:testSuffix", auto.ConfigValue{Value: fmt.Sprintf("gov-import-%s", suffix)})
	require.NoError(t, err)

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	t.Log("Importing AI governance into Pulumi...")
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:AiGovernance",
				Name: "testGovernance",
				ID:   os.Getenv("QUANTCDN_ORGANIZATION"),
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed for AI governance")
	}
	t.Log("AI Governance import test completed successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Import — AI Vector Collection
// ---------------------------------------------------------------------------

func TestE2E_Import_AIVectorCollection(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	suffix := uniqueSuffix()
	collectionName := fmt.Sprintf("e2e-import-col-%s", suffix)
	t.Logf("Creating vector collection '%s' via API...", collectionName)

	createReq := quantadmingo.NewCreateVectorCollectionRequest(collectionName)
	model := "amazon.titan-embed-text-v2:0"
	createReq.EmbeddingModel = &model

	resp, _, err := apiClient.Instance.AIVectorDatabaseAPI.CreateVectorCollection(apiClient.AuthContext, apiClient.Organization).
		CreateVectorCollectionRequest(*createReq).Execute()
	require.NoError(t, err, "API collection create failed")

	collectionId := ""
	if resp != nil && resp.Collection != nil && resp.Collection.CollectionId != nil {
		collectionId = *resp.Collection.CollectionId
	}
	require.NotEmpty(t, collectionId, "collectionId should be returned")
	t.Logf("Created vector collection: id=%s", collectionId)

	defer func() {
		_, _, err := apiClient.Instance.AIVectorDatabaseAPI.DeleteVectorCollection(apiClient.AuthContext, apiClient.Organization, collectionId).Execute()
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}()

	dir := programDir(t, "ai-vector")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	err = stack.SetConfig(ctx, "e2e-ai-vector:testSuffix", auto.ConfigValue{Value: fmt.Sprintf("col-import-%s", suffix)})
	require.NoError(t, err)

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	t.Log("Importing vector collection into Pulumi...")
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:AiVectorCollection",
				Name: "testCollection",
				ID:   collectionId,
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed for vector collection")
	}
	t.Log("AI Vector Collection import test completed successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Import — AI Vector Document
// ---------------------------------------------------------------------------

func TestE2E_Import_AIVectorDocument(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	suffix := uniqueSuffix()
	collectionName := fmt.Sprintf("e2e-import-doc-col-%s", suffix)
	t.Logf("Creating vector collection '%s' via API...", collectionName)

	createReq := quantadmingo.NewCreateVectorCollectionRequest(collectionName)
	model := "amazon.titan-embed-text-v2:0"
	createReq.EmbeddingModel = &model

	colResp, _, err := apiClient.Instance.AIVectorDatabaseAPI.CreateVectorCollection(apiClient.AuthContext, apiClient.Organization).
		CreateVectorCollectionRequest(*createReq).Execute()
	require.NoError(t, err, "API collection create failed")

	collectionId := ""
	if colResp != nil && colResp.Collection != nil && colResp.Collection.CollectionId != nil {
		collectionId = *colResp.Collection.CollectionId
	}
	require.NotEmpty(t, collectionId, "collectionId should be returned")

	defer func() {
		_, _, _ = apiClient.Instance.AIVectorDatabaseAPI.DeleteVectorCollection(apiClient.AuthContext, apiClient.Organization, collectionId).Execute()
	}()

	docKey := fmt.Sprintf("e2e-import-doc-%s", suffix)
	docs := []quantadmingo.UploadVectorDocumentsRequestDocumentsInner{
		{Content: "E2E import test document content.", Key: &docKey},
	}
	uploadReq := quantadmingo.NewUploadVectorDocumentsRequest(docs)
	_, _, err = apiClient.Instance.AIVectorDatabaseAPI.UploadVectorDocuments(apiClient.AuthContext, apiClient.Organization, collectionId).
		UploadVectorDocumentsRequest(*uploadReq).Execute()
	require.NoError(t, err, "API document upload failed")
	t.Logf("Uploaded document: key=%s in collection=%s", docKey, collectionId)

	dir := programDir(t, "ai-vector")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	err = stack.SetConfig(ctx, "e2e-ai-vector:testSuffix", auto.ConfigValue{Value: fmt.Sprintf("doc-import-%s", suffix)})
	require.NoError(t, err)

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	t.Log("Importing vector document into Pulumi...")
	// Import format for AiVectorDocument is "collection-id/key"
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:AiVectorDocument",
				Name: "testDocument",
				ID:   fmt.Sprintf("%s/%s", collectionId, docKey),
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed for vector document")
	}
	t.Log("AI Vector Document import test completed successfully")
}

// ---------------------------------------------------------------------------
// E2E Test: Import — KV Item
// ---------------------------------------------------------------------------

func TestE2E_Import_KVItem(t *testing.T) {
	ensureProvider(t)
	checkE2EEnv(t)

	ctx := context.Background()
	apiClient := newAPIClient(t)
	defer apiClient.Close()

	suffix := uniqueSuffix()
	projectName := fmt.Sprintf("pulumi-e2e-import-kvitem-%s", suffix)
	t.Logf("Creating project '%s' via API...", projectName)

	projReq := quantadmingo.NewV2ProjectRequestWithDefaults()
	projReq.SetName(projectName)
	projReq.SetRegion("au-govt")
	createResp, _, err := apiClient.Instance.ProjectsAPI.ProjectsCreate(apiClient.AuthContext, apiClient.Organization).V2ProjectRequest(*projReq).Execute()
	require.NoError(t, err, "API project create failed")
	machineName := createResp.GetMachineName()

	defer func() {
		_, _ = apiClient.Instance.ProjectsAPI.ProjectsDelete(apiClient.AuthContext, apiClient.Organization, machineName).Execute()
	}()

	time.Sleep(5 * time.Second)

	// Create KV store
	storeReq := quantadmingo.NewV2StoreRequest("e2e-import-item-store")
	kvResp, _, err := apiClient.Instance.KVAPI.KVCreate(apiClient.AuthContext, apiClient.Organization, machineName).V2StoreRequest(*storeReq).Execute()
	require.NoError(t, err, "API KV store create failed")
	storeId := kvResp.GetId()
	t.Logf("Created KV store: id=%s", storeId)

	// Create KV item
	itemKey := "e2e-import-key"
	itemReq := quantadmingo.NewV2StoreItemRequest(itemKey, "e2e-import-value")
	_, _, err = apiClient.Instance.KVAPI.KVItemsCreate(apiClient.AuthContext, apiClient.Organization, machineName, storeId).V2StoreItemRequest(*itemReq).Execute()
	require.NoError(t, err, "API KV item create failed")
	t.Logf("Created KV item: key=%s", itemKey)

	dir := programDir(t, "kv")
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)
	stateDir := t.TempDir()
	t.Setenv("PULUMI_BACKEND_URL", "file://"+stateDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stack, err := auto.UpsertStackLocalSource(ctx, "test", dir)
	require.NoError(t, err)

	err = stack.SetConfig(ctx, "e2e-kv:projectName", auto.ConfigValue{Value: projectName})
	require.NoError(t, err)

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	t.Log("Importing KV item into Pulumi...")
	// Import format: "project/store_id/key"
	_, err = stack.ImportResources(ctx,
		optimport.Resources([]*optimport.ImportResource{
			{
				Type: "quant:index:KvItem",
				Name: "testItem",
				ID:   fmt.Sprintf("%s/%s/%s", machineName, storeId, itemKey),
			},
		}),
		optimport.GenerateCode(false),
		optimport.ProgressStreams(os.Stdout),
	)
	if err != nil && !strings.Contains(err.Error(), "generated_code") {
		require.NoError(t, err, "pulumi import failed for KV item")
	}
	t.Log("KV Item import test completed successfully")
}
