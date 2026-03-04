package provider

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	buildOnce sync.Once
	buildErr  error
	binDir    string
)

// ensureProvider builds the provider binary once for all tests.
func ensureProvider(t *testing.T) {
	t.Helper()
	buildOnce.Do(func() {
		// Build provider binary into pulumi/bin/
		// Tests run from pulumi/provider/, so .. is pulumi/
		pulumiDir, _ := filepath.Abs("..")
		binDir = filepath.Join(pulumiDir, "bin")
		cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "pulumi-resource-quant"), "./cmd/pulumi-resource-quant")
		cmd.Dir = pulumiDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		buildErr = cmd.Run()
	})
	require.NoError(t, buildErr, "failed to build provider binary")
}

// previewProgram creates a temporary Pulumi YAML project and runs preview.
// It validates schema, token resolution, and input types without creating real resources.
func previewProgram(t *testing.T, name string, program string) {
	t.Helper()
	ensureProvider(t)

	ctx := context.Background()
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "Pulumi.yaml"), []byte(program), 0644)
	require.NoError(t, err)

	// Ensure provider binary is on PATH
	currentPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+currentPath)

	// Provider needs these env vars to configure (no real API calls during preview)
	t.Setenv("QUANTCDN_API_TOKEN", "test-token-for-preview")
	t.Setenv("QUANTCDN_ORGANIZATION", "test-org")

	// Use local backend — no Pulumi Cloud account needed
	backendDir := filepath.Join(dir, "state")
	err = os.MkdirAll(backendDir, 0755)
	require.NoError(t, err)
	t.Setenv("PULUMI_BACKEND_URL", "file://"+backendDir)
	t.Setenv("PULUMI_CONFIG_PASSPHRASE", "")

	stackName := auto.FullyQualifiedStackName("organization", name, "test")
	stack, err := auto.UpsertStackLocalSource(ctx, stackName, dir)
	require.NoError(t, err, "failed to create stack")

	defer func() {
		_ = stack.Workspace().RemoveStack(ctx, stack.Name())
	}()

	// Preview validates schema + token resolution without creating resources.
	// For new resources this will show "create" in the change summary.
	result, err := stack.Preview(ctx)
	if err != nil {
		t.Logf("Preview stderr: %s", result.StdErr)
	}
	require.NoError(t, err, "preview failed — possible schema or bridge error")
	assert.NotEmpty(t, result.ChangeSummary, "preview should report planned changes")
}

// ---------------------------------------------------------------------------
// Resource preview tests (20 resources)
// ---------------------------------------------------------------------------

func TestPreview_Project(t *testing.T) {
	previewProgram(t, "test-project", `
name: test-project
runtime: yaml
resources:
  testProject:
    type: quant:index:Project
    properties:
      name: preview-test-project
`)
}

func TestPreview_Domain(t *testing.T) {
	previewProgram(t, "test-domain", `
name: test-domain
runtime: yaml
resources:
  testProject:
    type: quant:index:Project
    properties:
      name: preview-test-domain-proj
  testDomain:
    type: quant:index:Domain
    properties:
      domain: preview-test.example.com
      project: ${testProject.name}
`)
}

func TestPreview_RuleProxy(t *testing.T) {
	previewProgram(t, "test-rule-proxy", `
name: test-rule-proxy
runtime: yaml
resources:
  testProject:
    type: quant:index:Project
    properties:
      name: preview-test-rule-proxy
  testRule:
    type: quant:index:RuleProxy
    properties:
      project: ${testProject.name}
      domains:
        - "*.example.com"
      urls:
        - "/api/*"
      to: "https://backend.example.com"
`)
}

func TestPreview_RuleRedirect(t *testing.T) {
	previewProgram(t, "test-rule-redirect", `
name: test-rule-redirect
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleRedirect
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/old-path"
      redirectTo: "https://example.com/new-path"
`)
}

func TestPreview_RuleCustomResponse(t *testing.T) {
	previewProgram(t, "test-rule-custom-resp", `
name: test-rule-custom-resp
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleCustomResponse
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/maintenance"
      customResponseBody: "<h1>Under Maintenance</h1>"
`)
}

func TestPreview_RuleContentFilter(t *testing.T) {
	previewProgram(t, "test-rule-content-filter", `
name: test-rule-content-filter
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleContentFilter
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/filtered/*"
      fnUuid: "00000000-0000-0000-0000-000000000000"
`)
}

func TestPreview_RuleFunction(t *testing.T) {
	previewProgram(t, "test-rule-function", `
name: test-rule-function
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleFunction
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/fn/*"
      fnUuid: "00000000-0000-0000-0000-000000000001"
`)
}

func TestPreview_RuleAuth(t *testing.T) {
	previewProgram(t, "test-rule-auth", `
name: test-rule-auth
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleAuth
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/admin/*"
      authUser: testuser
      authPass: testpass
`)
}

func TestPreview_RuleBotChallenge(t *testing.T) {
	previewProgram(t, "test-rule-bot-challenge", `
name: test-rule-bot-challenge
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleBotChallenge
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/protected/*"
      robotChallengeType: invisible
`)
}

func TestPreview_RuleHeaders(t *testing.T) {
	previewProgram(t, "test-rule-headers", `
name: test-rule-headers
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleHeaders
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/*"
      headers:
        X-Custom-Header: test-value
`)
}

func TestPreview_RuleServeStatic(t *testing.T) {
	previewProgram(t, "test-rule-serve-static", `
name: test-rule-serve-static
runtime: yaml
resources:
  testRule:
    type: quant:index:RuleServeStatic
    properties:
      domains:
        - "*.example.com"
      urls:
        - "/static/*"
      staticFilePath: "/index.html"
`)
}

func TestPreview_Header(t *testing.T) {
	previewProgram(t, "test-header", `
name: test-header
runtime: yaml
resources:
  testHeader:
    type: quant:index:Header
    properties:
      project: preview-test-project
      headers:
        X-Frame-Options: DENY
        X-Content-Type-Options: nosniff
`)
}

func TestPreview_Crawler(t *testing.T) {
	previewProgram(t, "test-crawler", `
name: test-crawler
runtime: yaml
resources:
  testCrawler:
    type: quant:index:Crawler
    properties:
      domain: https://example.com
`)
}

func TestPreview_CrawlerSchedule(t *testing.T) {
	previewProgram(t, "test-crawler-schedule", `
name: test-crawler-schedule
runtime: yaml
resources:
  testSchedule:
    type: quant:index:CrawlerSchedule
    properties:
      scheduleCronString: "0 0 * * *"
`)
}

func TestPreview_Application(t *testing.T) {
	previewProgram(t, "test-application", `
name: test-application
runtime: yaml
resources:
  testApp:
    type: quant:index:Application
    properties:
      appName: preview-test-app
      composeDefinition: |
        {"containers":[{"name":"web","image":"nginx:latest","cpu":256,"memory":512}]}
`)
}

func TestPreview_Environment(t *testing.T) {
	previewProgram(t, "test-environment", `
name: test-environment
runtime: yaml
resources:
  testEnv:
    type: quant:index:Environment
    properties:
      application: preview-test-app
      envName: staging
`)
}

func TestPreview_Volume(t *testing.T) {
	previewProgram(t, "test-volume", `
name: test-volume
runtime: yaml
resources:
  testVolume:
    type: quant:index:Volume
    properties:
      application: preview-test-app
      environment: staging
      volumeName: data-vol
`)
}

func TestPreview_CronJob(t *testing.T) {
	previewProgram(t, "test-cron-job", `
name: test-cron-job
runtime: yaml
resources:
  testCron:
    type: quant:index:CronJob
    properties:
      application: preview-test-app
      environment: staging
      command: '["echo","hello"]'
      scheduleExpression: "0 0 * * *"
`)
}

func TestPreview_KvStore(t *testing.T) {
	previewProgram(t, "test-kv-store", `
name: test-kv-store
runtime: yaml
resources:
  testStore:
    type: quant:index:KvStore
    properties:
      project: preview-test-project
`)
}

func TestPreview_KvItem(t *testing.T) {
	previewProgram(t, "test-kv-item", `
name: test-kv-item
runtime: yaml
resources:
  testItem:
    type: quant:index:KvItem
    properties:
      project: preview-test-project
      storeId: test-store-id
      key: test-key
      value: test-value
`)
}

// ---------------------------------------------------------------------------
// Data source (function) token tests
// ---------------------------------------------------------------------------
// Note: fn::invoke calls the provider's Read method during preview, which
// makes real API calls. Data sources are tested in E2E only. Here we verify
// the tokens are correctly mapped in the bridge configuration.

func TestDataSource_GetProjectToken(t *testing.T) {
	info := Provider()
	ds, ok := info.DataSources["quant_project"]
	require.True(t, ok, "quant_project data source not found")
	assert.Equal(t, "quant:index:getProject", string(ds.Tok))
}

func TestDataSource_GetProjectsToken(t *testing.T) {
	info := Provider()
	ds, ok := info.DataSources["quant_projects"]
	require.True(t, ok, "quant_projects data source not found")
	assert.Equal(t, "quant:index:getProjects", string(ds.Tok))
}

// ---------------------------------------------------------------------------
// Provider info unit tests (no Pulumi CLI needed)
// ---------------------------------------------------------------------------

func TestProviderInfo_ResourceCount(t *testing.T) {
	info := Provider()
	assert.Equal(t, 20, len(info.Resources), "expected 20 resource mappings")
}

func TestProviderInfo_DataSourceCount(t *testing.T) {
	info := Provider()
	assert.Equal(t, 2, len(info.DataSources), "expected 2 data source mappings")
}

func TestProviderInfo_ResourceTokenFormat(t *testing.T) {
	info := Provider()
	for tfName, res := range info.Resources {
		assert.NotEmpty(t, res.Tok, "resource %s has empty token", tfName)
		assert.Contains(t, string(res.Tok), "quant:index:", "resource %s token should be in quant:index: namespace", tfName)
	}
}

func TestProviderInfo_DataSourceTokenFormat(t *testing.T) {
	info := Provider()
	for tfName, ds := range info.DataSources {
		assert.NotEmpty(t, ds.Tok, "data source %s has empty token", tfName)
		assert.Contains(t, string(ds.Tok), "quant:index:", "data source %s token should be in quant:index: namespace", tfName)
	}
}

func TestProviderInfo_IntIDResources(t *testing.T) {
	info := Provider()
	intIDResources := []string{
		"quant_project",
		"quant_domain",
		"quant_crawler",
		"quant_crawler_schedule",
	}
	for _, name := range intIDResources {
		res, ok := info.Resources[name]
		require.True(t, ok, "resource %s not found", name)
		assert.NotNil(t, res.ComputeID, "resource %s should have ComputeID for integer ID conversion", name)
	}
}

func TestProviderInfo_StringIDResources(t *testing.T) {
	info := Provider()
	// Resources with native string IDs that need no ComputeID
	stringIDResources := []string{
		"quant_rule_proxy",
		"quant_rule_redirect",
		"quant_rule_custom_response",
		"quant_rule_content_filter",
		"quant_rule_function",
		"quant_rule_auth",
		"quant_rule_bot_challenge",
		"quant_rule_headers",
		"quant_rule_serve_static",
		"quant_header",
	}
	for _, name := range stringIDResources {
		res, ok := info.Resources[name]
		require.True(t, ok, "resource %s not found", name)
		assert.Nil(t, res.ComputeID, "resource %s should not have ComputeID (string IDs)", name)
	}
}

func TestProviderInfo_FieldIDResources(t *testing.T) {
	info := Provider()
	// Resources that use a named field as ID (no native "id" attribute)
	fieldIDResources := []string{
		"quant_application",
		"quant_environment",
		"quant_volume",
		"quant_cron_job",
		"quant_kv_store",
		"quant_kv_item",
	}
	for _, name := range fieldIDResources {
		res, ok := info.Resources[name]
		require.True(t, ok, "resource %s not found", name)
		assert.NotNil(t, res.ComputeID, "resource %s should have ComputeID for field-based ID", name)
	}
}

func TestProviderInfo_Metadata(t *testing.T) {
	info := Provider()
	assert.Equal(t, "quant", info.Name)
	assert.Equal(t, "QuantCDN", info.DisplayName)
	assert.Equal(t, "quantcdn", info.GitHubOrg)
	assert.NotNil(t, info.JavaScript)
	assert.NotNil(t, info.Python)
	assert.NotNil(t, info.Golang)
	assert.NotNil(t, info.CSharp)
}

func TestIntIDComputeID_Number(t *testing.T) {
	ctx := context.Background()
	state := resource.PropertyMap{
		"id": resource.NewNumberProperty(12345),
	}
	id, err := intIDComputeID(ctx, state)
	require.NoError(t, err)
	assert.Equal(t, resource.ID("12345"), id)
}

func TestIntIDComputeID_String(t *testing.T) {
	ctx := context.Background()
	state := resource.PropertyMap{
		"id": resource.NewStringProperty("abc-123"),
	}
	id, err := intIDComputeID(ctx, state)
	require.NoError(t, err)
	assert.Equal(t, resource.ID("abc-123"), id)
}

func TestIntIDComputeID_Missing(t *testing.T) {
	ctx := context.Background()
	state := resource.PropertyMap{}
	_, err := intIDComputeID(ctx, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing id")
}
