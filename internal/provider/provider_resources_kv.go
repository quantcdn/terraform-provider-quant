package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// handRegisteredResources are the resources that generator_config.yml lists
// with `skip: true`, so provider_resources_gen.go does not register them. When
// the portal removes the skip, remove the entry here, or the provider
// registers the type twice and fails to start.
func handRegisteredResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewKVIndexResource,
	}
}
