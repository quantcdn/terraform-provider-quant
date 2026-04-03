# SDK Pipeline Redesign — Remaining Work

**Date:** 2026-04-03
**Status:** In progress
**Context:** The core pipeline redesign is complete. This plan covers the gaps identified during E2E testing.

## Current State

- 12/13 E2E tests pass against staging
- All 24 resources have E2E coverage
- All unit tests and Pulumi preview tests pass
- Portal branch: `develop` (9 commits since 44c2f16b)
- TF provider branch: `feat/sdk-pipeline-redesign` (~30 commits since 08312bdd)

---

### Task 1: Fix Domain E2E — CDN Provisioning Wait

**Problem:** Project Create returns as soon as the project record is readable, but CDN provisioning (CloudFront distribution deployment) takes 5-15+ minutes. Domain creation fails with "config incomplete or not found" because the CDN isn't ready.

**Root cause:** The API returns `platform_provisioning_status` in the project response. It starts as absent, then becomes `deployed` once the CDN distribution is active. The provider doesn't wait for this.

**What we tried:** Adding a polling wait for `platform_provisioning_status == "deployed"` in Project Create. It worked correctly but timed out at 15 minutes on staging — the provisioning queue appears to be slow or not triggered for all project types.

**Fix options:**
- [ ] **Option A (recommended):** Add an optional `wait_for_deployment` boolean attribute to the Project resource schema. When true, Create polls until `platform_provisioning_status == "deployed"` (30min timeout). When false (default), returns immediately.
- [ ] **Option B:** Add `platform_provisioning_status` to the Go SDK as a typed field (currently only in `AdditionalProperties`). Then the Project resource can expose it as a computed attribute for users to check.
- [ ] **Option C:** Investigate why staging CDN provisioning is slow — may be a platform queue issue.

**Files:**
- `internal/provider/project_resource.go` — Add `wait_for_deployment` to schema + polling logic
- `internal/resource_project/project_resource_gen.go` — Will need OA spec update + regen
- Portal OA spec — Add `waitForDeployment` and `platformProvisioningStatus` to Project schema

**E2E test:** Update `pulumi/e2e/programs/domain/Pulumi.yaml` to set `waitForDeployment: true` on the project.

---

### Task 2: Add `platform_provisioning_status` to Go SDK

**Problem:** The field exists in the API response but isn't in the Go SDK's `V2Project` model. It falls into `AdditionalProperties` which is fragile.

**Fix:**
- [ ] Add `platform_provisioning_status` field to the Project OA docblocks in Portal
- [ ] Run `publish-unified.yml` to regenerate the Go SDK
- [ ] The TF provider can then use `project.GetPlatformProvisioningStatus()` instead of `AdditionalProperties`

**Files:**
- Portal: OA docblocks on Project controller
- Go SDK: auto-generated after publish

---

### Task 3: Environment Update E2E

**Problem:** The Environment GET response doesn't return `composeDefinition` or `spotConfiguration`. These are preserved from state after Create, but there's no way to verify an update round-trips correctly.

**Fix options:**
- [ ] **Option A:** Add `composeDefinition` to the Environment GET response in the Portal API
- [ ] **Option B:** Accept the limitation — updates work (tested via AppStack), but drift detection on compose changes is limited

**E2E test:** If Option A is implemented, add a test that:
1. Creates an environment with compose definition A
2. Updates to compose definition B
3. Reads back and verifies B is in state

---

### Task 4: Import Tests

**Problem:** No `terraform import` tests exist. Import is supported on some resources (ai_skill has ImportState implemented) but untested.

**Resources with ImportState:**
- [ ] Check which resources implement `ImportState` method
- [ ] Write E2E import tests for at least: Project, Application, AiSkill, KvStore

**Test pattern:**
1. Create resource via Pulumi
2. Remove from state (`pulumi state delete`)
3. Import by ID (`pulumi import`)
4. Verify state matches

---

### Task 5: Drift Detection Tests

**Problem:** No tests verify that `terraform plan` detects changes made outside TF/Pulumi.

**Test pattern:**
1. Create resource via Pulumi
2. Modify resource directly via API (e.g. change project name, update rule)
3. Run `pulumi preview`
4. Verify preview shows the drift

**Resources to test:**
- [ ] Project (change name)
- [ ] Rule (change URL pattern)
- [ ] AiSkill (change content)

---

### Task 6: Data Source Tests

**Problem:** 17 data sources defined in `generator_config.yml` but zero are E2E tested.

**Data sources to test:**
- [ ] `quant_project` / `quant_projects` — lookup existing project(s)
- [ ] `quant_domain` — lookup domain on a project
- [ ] `quant_rule_*` — lookup rules
- [ ] `quant_application` — lookup application
- [ ] `quant_kv_store` — lookup KV store

**Test pattern:**
1. Create resource via Pulumi
2. Use data source to look it up
3. Verify data source output matches resource output

---

### Task 7: Pulumi Bridge SchemaInfo Overrides

**Problem:** The Pulumi bridge marks ALL `Optional+Computed` and `Computed` fields as "required" in the Pulumi schema. This means preview tests must provide dummy values for server-computed fields like `status`, `runningCount`, etc.

**Fix:**
- [ ] In `pulumi/provider/resources.go`, add `SchemaInfo` overrides to mark server-computed fields as `MaxItemsOne: true` or use `tfbridge.SchemaInfo{MarkAsOptional: true}` where appropriate
- [ ] Or wait for Pulumi bridge v4 which handles this better

**Impact:** Better Pulumi developer experience — users won't see spurious "required" fields for server-computed values.

---

### Task 8: Archive pulumi-quant Repo

**Prerequisites:** TF provider branch merged, release published, npm package verified.

- [ ] Update pulumi-quant README to point to terraform-provider-quant
- [ ] Archive the repo: `gh repo archive quantcdn/pulumi-quant --yes`

---

### Task 9: Clean Up Test Artifacts

**Problem:** E2E tests leave orphaned resources on staging when they fail (bridge panic, timeout, etc.)

**Fix options:**
- [ ] Add a `TestMain` function that runs cleanup before/after the test suite
- [ ] Add a `make clean-staging` target that deletes all `pulumi-e2e-*` resources
- [ ] Consider using test-specific org/project to isolate E2E artifacts

---

## Priority Order

1. **Task 8** — Archive pulumi-quant (blocked on merge/release)
2. **Task 1** — Domain E2E (biggest gap in test coverage)
3. **Task 2** — SDK field (enables Task 1)
4. **Task 6** — Data source tests (quick wins)
5. **Task 4** — Import tests
6. **Task 5** — Drift detection tests
7. **Task 9** — Test cleanup
8. **Task 3** — Environment update (needs API change)
9. **Task 7** — Bridge overrides (nice to have)
