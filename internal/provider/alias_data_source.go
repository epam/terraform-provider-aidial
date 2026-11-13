package provider

import (
	"context"
	"fmt"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &aliasDataSource{}
	_ datasource.DataSource = &fileAliasDataSource{}
)

// aliasDataSourceBodyAttributes are the computed attributes shared by the
// aidial_alias and aidial_file_alias data sources. They mirror the aidial_alias
// resource.
func aliasDataSourceBodyAttributes() map[string]schema.Attribute {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: description}
	}
	return map[string]schema.Attribute{
		"present_as":           str("What the alias is shown as: `model`, `application` or `toolset`."),
		"target_path":          str("The deployment the alias points at: `models/<name>`, `applications/<name>` or `toolsets/<name>`."),
		"display_name":         str("Human-readable alias name."),
		"description":          str("Alias description."),
		"display_version":      str("Version shown for the alias."),
		"icon_url":             str("URL of the alias icon."),
		"intro":                str("Introductory text shown for the alias."),
		"user_roles":           schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Roles allowed to use the alias."},
		"description_keywords": schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Keywords that help find the alias by description."},
	}
}

// aliasDataSource reads an API-managed alias stored in DIAL Core resource
// storage at <bucket>/<path>. Aliases from static configuration are read by
// aidial_file_alias.
type aliasDataSource struct{ client *client.Client }

func (d *aliasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alias"
}

func (d *aliasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := aliasDataSourceBodyAttributes()
	attributes["id"] = schema.StringAttribute{Computed: true, Description: "Resource identifier in the form <bucket>/<path>."}
	attributes["bucket"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.OneOf(aliasBuckets...)},
		Description: "DIAL bucket. Must be \"public\" or \"platform\".",
	}
	attributes["path"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Alias path inside the bucket.",
	}
	attributes["name"] = schema.StringAttribute{Computed: true, Description: "Alias name returned by the DIAL API."}
	attributes["api_name"] = schema.StringAttribute{Computed: true, Description: "Canonical alias name returned by the DIAL API."}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing alias stored in DIAL Core resource storage without managing it. Use `aidial_file_alias` for aliases defined in static (file) configuration.",
		Attributes:          attributes,
	}
}

func (d *aliasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected *client.Client")
		return
	}
	d.client = c
}

func (d *aliasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config aliasModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, path := config.Bucket.ValueString(), config.Path.ValueString()
	a, err := d.client.GetAlias(ctx, bucket, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read alias", err.Error())
		return
	}
	if a == nil {
		resp.Diagnostics.AddError(
			"Alias not found",
			fmt.Sprintf("No alias exists at bucket %q, path %q. This data source only reads aliases in resource storage; use aidial_file_alias for static-configuration aliases.", bucket, path),
		)
		return
	}

	state := aliasModel{Bucket: config.Bucket, Path: config.Path, ID: aliasID(config.Bucket, config.Path), Name: types.StringNull()}
	if err := setAliasState(ctx, &state, a); err != nil {
		resp.Diagnostics.AddError("Failed to store alias state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fileAliasModel is the state of the aidial_file_alias data source.
type fileAliasModel struct {
	Name types.String `tfsdk:"name"`

	aliasBody
}

// fileAliasDataSource reads an alias defined in DIAL Core's static
// configuration, which has no bucket or path.
type fileAliasDataSource struct{ client *client.Client }

func (d *fileAliasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_alias"
}

func (d *fileAliasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := aliasDataSourceBodyAttributes()
	attributes["name"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Name of the alias in DIAL Core's static configuration.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an alias defined in DIAL Core's static (file) configuration. Aliases stored in resource storage are read by `aidial_alias`.",
		Attributes:          attributes,
	}
}

func (d *fileAliasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected *client.Client")
		return
	}
	d.client = c
}

func (d *fileAliasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fileAliasModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	a, err := d.client.GetFileAlias(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file-sourced alias", err.Error())
		return
	}
	if a == nil {
		resp.Diagnostics.AddError(
			"File-sourced alias not found",
			fmt.Sprintf("No file-sourced alias named %q was found. This data source only sees aliases defined in DIAL Core's static configuration, not resource-storage aliases managed by aidial_alias.", name),
		)
		return
	}

	state := fileAliasModel{Name: config.Name}
	if err := setAliasBodyState(ctx, &state.aliasBody, a); err != nil {
		resp.Diagnostics.AddError("Failed to store alias state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
