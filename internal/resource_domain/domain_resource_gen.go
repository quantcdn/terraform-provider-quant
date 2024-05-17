// Code generated by terraform-plugin-framework-generator DO NOT EDIT.

package resource_domain

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func DomainResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"deleted_at": schema.StringAttribute{
				Computed: true,
			},
			"dns_engaged": schema.Int64Attribute{
				Computed: true,
			},
			"domain": schema.StringAttribute{
				Required: true,
			},
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"in_section": schema.Int64Attribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"organization": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"project": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"project_id": schema.Int64Attribute{
				Computed: true,
			},
			"section_message": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

type DomainModel struct {
	CreatedAt      types.String `tfsdk:"created_at"`
	DeletedAt      types.String `tfsdk:"deleted_at"`
	DnsEngaged     types.Int64  `tfsdk:"dns_engaged"`
	Domain         types.String `tfsdk:"domain"`
	Id             types.Int64  `tfsdk:"id"`
	InSection      types.Int64  `tfsdk:"in_section"`
	Name           types.String `tfsdk:"name"`
	Organization   types.String `tfsdk:"organization"`
	Project        types.String `tfsdk:"project"`
	ProjectId      types.Int64  `tfsdk:"project_id"`
	SectionMessage types.String `tfsdk:"section_message"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}
