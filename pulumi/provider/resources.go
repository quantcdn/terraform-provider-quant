package provider

import (
	_ "embed"
	"path"

	pf "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"

	"github.com/quantcdn/terraform-provider-quant/v5/pulumi/provider/pkg/version"
	"github.com/quantcdn/terraform-provider-quant/v5/shim"
)

// all of the token components used below.
const (
	mainPkg = "quant"
)

//go:embed bridge-metadata.json
var bridgeMetadata []byte

func Provider() tfbridge.ProviderInfo {
	info := tfbridge.ProviderInfo{
		P:                 pf.ShimProvider(shim.NewProvider()),
		Name:              "quant",
		DisplayName:       "QuantCDN",
		Version:           version.Version,
		Description:       "A Pulumi package for managing QuantCDN resources.",
		Publisher:         "QuantCDN",
		PluginDownloadURL: "github://api.github.com/quantcdn/terraform-provider-quant",
		Keywords:          []string{"pulumi", "quant", "quantcdn", "cdn", "category/cloud"},
		License:           "Apache-2.0",
		Homepage:          "https://www.quantcdn.io",
		Repository:        "https://github.com/quantcdn/terraform-provider-quant",
		GitHubOrg:         "quantcdn",

		MetadataInfo: tfbridge.NewProviderMetadata(bridgeMetadata),

		Config: map[string]*tfbridge.SchemaInfo{},

		Resources: GeneratedResourceMap(),

		DataSources: map[string]*tfbridge.DataSourceInfo{
			"quant_project":  {Tok: "quant:index:getProject"},
			"quant_projects": {Tok: "quant:index:getProjects"},
		},

		JavaScript: &tfbridge.JavaScriptInfo{
			PackageName: "@quantcdn/pulumi-quant",
			Dependencies: map[string]string{
				"@pulumi/pulumi": "^3.0.0",
			},
			DevDependencies: map[string]string{
				"@types/node": "^10.0.0",
			},
			RespectSchemaVersion: true,
		},
		Python: &tfbridge.PythonInfo{
			PackageName:          "pulumi_quant",
			RespectSchemaVersion: true,
			PyProject:            struct{ Enabled bool }{true},
			Requires: map[string]string{
				"pulumi": ">=3.0.0,<4.0.0",
			},
		},
		Golang: &tfbridge.GolangInfo{
			ImportBasePath: path.Join(
				"github.com/quantcdn/terraform-provider-quant/v5/pulumi/sdk/",
				tfbridge.GetModuleMajorVersion(version.Version),
				"go",
				mainPkg,
			),
			GenerateResourceContainerTypes: true,
			GenerateExtraInputTypes:        true,
			RespectSchemaVersion:           true,
		},
		CSharp: &tfbridge.CSharpInfo{
			RespectSchemaVersion: true,
			PackageReferences: map[string]string{
				"Pulumi": "3.*",
			},
			RootNamespace: "QuantCDN",
			Namespaces: map[string]string{
				"quant": "Quant",
			},
		},
	}

	info.MustApplyAutoAliases()
	info.SetAutonaming(255, "-")

	return info
}
