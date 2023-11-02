package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProjectsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProjectsDataSourceConfig,
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
		},
	})
}

const testProjectsDataSourceConfig = `
data "quant_projects" "test" {}`
