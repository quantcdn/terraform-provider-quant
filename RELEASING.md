# Release Process

This document describes how to release a new version of the Terraform/Pulumi provider.

## Prerequisites

- Access to `quantcdn/portal`, `quantcdn/quant-admin-go`, and `quantcdn/terraform-provider-quant` repos
- npm OIDC trusted publishing configured for `@quantcdn/pulumi-quant` from this repo

## Release Pipeline

The release process flows through three repositories in order:

```
Portal (OA spec source) → quant-admin-go (Go SDK) → terraform-provider-quant (TF/Pulumi provider)
```

### Step 1: Portal — Update API spec (if API changed)

If the API has new/changed endpoints:

1. Update OA annotations on the Portal controller methods
2. Update `generator_config.yml` if adding new resources
3. Merge to `master` (or `develop` → `master`)
4. Trigger `publish-unified.yml` workflow (or let it run on merge)

This workflow:
- Regenerates the unified OpenAPI spec
- Creates PRs on `quant-admin-go` (Go SDK) and `terraform-provider-quant` (TF schemas)

### Step 2: quant-admin-go — Publish Go SDK

**This must happen BEFORE merging the TF provider changes.**

1. Review and merge the auto-generated PR from Step 1
2. **Tag a new version** (e.g., `v4.15.4`) on the `main` branch
3. Wait for the Go module proxy to index it (~1-2 minutes)

Without this tag, the TF provider can't `go get` the new SDK version.

### Step 3: terraform-provider-quant — Bump SDK + Release

1. Review and merge the auto-generated schema PR from Step 1 (if any)
2. Bump `quant-admin-go` version in **both** `go.mod` files:
   ```bash
   go get github.com/quantcdn/quant-admin-go/v4@v4.X.Y && go mod tidy
   cd pulumi && go get github.com/quantcdn/quant-admin-go/v4@v4.X.Y && go mod tidy
   ```
3. If new resources were added, write CRUD handlers (`internal/provider/`)
4. Run tests:
   ```bash
   TF_ACC=1 go test ./internal/provider/ -timeout 5m
   cd pulumi && go test ./provider/ -timeout 2m
   ```
5. Commit, push, create PR, merge to `main`
6. Tag: `git tag v5.X.0 -m "v5.X.0: description" && git push origin v5.X.0`

The `release.yml` workflow will automatically:
- Build TF provider binaries (goreleaser, signed)
- Build Pulumi provider binaries (5 platforms)
- Upload all to GitHub Release
- Publish `@quantcdn/pulumi-quant` to npm (via OIDC)

## What gets published

| Artifact | Where | How users install |
|----------|-------|-------------------|
| TF provider binaries | GitHub Release + Terraform Registry | `terraform init` (auto-downloads) |
| Pulumi provider binaries | GitHub Release | Auto-downloaded via `pluginDownloadURL` |
| Pulumi Node.js SDK | npm (`@quantcdn/pulumi-quant`) | `npm install @quantcdn/pulumi-quant` |

## Architecture

```
generator_config.yml          ← Single source of truth (resources, data sources, registrations)
        │
        ├─→ tfplugingen-openapi    → spec.json (TF intermediate representation)
        ├─→ tfplugingen-framework  → internal/resource_*/  (generated Go schemas)
        ├─→ generate-provider-registrations.js
        │       ├─→ provider_resources_gen.go  (TF Resources() list)
        │       └─→ resources_gen.go           (Pulumi bridge mappings)
        └─→ deduplicate-terraform-types.js     (clean up tfplugingen duplicates)

Hand-written:
  internal/provider/*_resource.go   ← CRUD handlers (Create/Read/Update/Delete/Import)
  pulumi/e2e/                       ← E2E tests
```

## Adding a new resource

1. Add OA annotations to Portal controller
2. Add entries to `generator_config.yml` (resources + registrations sections)
3. Run Portal's `publish-unified.yml` → creates PRs on SDK + TF repos
4. Tag new SDK version on `quant-admin-go`
5. Merge TF schema PR, bump SDK in `go.mod`
6. Write CRUD handler (`internal/provider/new_resource.go`)
7. Write E2E test
8. Tag and release
