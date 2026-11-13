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
	_ datasource.DataSource = &toolsetDataSource{}
	_ datasource.DataSource = &fileToolsetDataSource{}
)

// toolsetDataSourceBodyAttributes are the computed attributes shared by the
// aidial_toolset and aidial_file_toolset data sources. They mirror the
// aidial_toolset resource, except that auth_settings has no client_secret:
// DIAL Core never returns it.
func toolsetDataSourceBodyAttributes() map[string]schema.Attribute {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: description}
	}
	set := func(description string) schema.SetAttribute {
		return schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: description}
	}
	return map[string]schema.Attribute{
		"display_name":            str("Human-readable toolset name."),
		"description":             str("Toolset description."),
		"author":                  str("Toolset author."),
		"intro":                   str("Introductory text shown for the toolset."),
		"display_version":         str("Version shown for the toolset."),
		"icon_url":                str("URL of the toolset icon."),
		"reference":               str("Reference URL."),
		"provider_name":           str("Name of the toolset provider (`provider` in the DIAL API)."),
		"vendor_website":          str("Website of the toolset vendor."),
		"endpoint":                str("Toolset MCP endpoint."),
		"responses_endpoint":      str("Endpoint used for Responses API requests."),
		"base_url":                str("Toolset-level base URL used by `interfaces` entries without their own `base_url`."),
		"override_name":           str("Override of the toolset name presented to clients."),
		"catalog_schema_id":       str("URI (`$id`) of the catalog schema describing `catalog_properties`."),
		"transport":               str("MCP transport: `HTTP` or `SSE`."),
		"forward_auth_token":      schema.BoolAttribute{Computed: true},
		"forward_per_request_key": schema.BoolAttribute{Computed: true},
		"max_input_attachments":   schema.Int64Attribute{Computed: true},
		"max_retry_attempts":      schema.Int64Attribute{Computed: true},
		"allowed_tools":           set("Names of the tools the toolset exposes."),
		"dependencies":            set("Deployments the toolset depends on."),
		"description_keywords":    set("Keywords describing the toolset."),
		"input_attachment_types":  set("MIME types of accepted input attachments."),
		"interceptors":            set("Interceptors applied to requests."),
		"user_roles":              set("Roles allowed to use the toolset."),
		"default_headers": schema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Additional headers sent with every request to the toolset's endpoint.",
		},
		"features":   featuresDataSourceAttribute(),
		"interfaces": interfacesDataSourceAttribute(),
		"auth_settings": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "How DIAL Core authenticates to the toolset. The client secret is never exposed.",
			Attributes: map[string]schema.Attribute{
				"authentication_type":        schema.StringAttribute{Computed: true},
				"api_key_header":             schema.StringAttribute{Computed: true},
				"authorization_endpoint":     schema.StringAttribute{Computed: true},
				"token_endpoint":             schema.StringAttribute{Computed: true},
				"token_endpoint_auth_method": schema.StringAttribute{Computed: true},
				"client_id":                  schema.StringAttribute{Computed: true},
				"redirect_uri":               schema.StringAttribute{Computed: true},
				"scopes_supported":           schema.SetAttribute{Computed: true, ElementType: types.StringType},
				"code_challenge_method":      schema.StringAttribute{Computed: true},
				"resource":                   schema.StringAttribute{Computed: true},
				"dynamically_registered":     schema.BoolAttribute{Computed: true},
			},
		},
		"defaults":           schema.DynamicAttribute{Computed: true, Description: "Default request parameters (free-form object)."},
		"responses_defaults": schema.DynamicAttribute{Computed: true, Description: "Default request parameters for Responses API requests (free-form object)."},
		"catalog_properties": schema.DynamicAttribute{Computed: true, Description: "Catalog metadata of the toolset (free-form object)."},
	}
}

// toolsetDataSource reads an API-managed toolset stored in DIAL Core resource
// storage at <bucket>/<path>. Toolsets from static configuration are read by
// aidial_file_toolset.
type toolsetDataSource struct{ client *client.Client }

func (d *toolsetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_toolset"
}

func (d *toolsetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := toolsetDataSourceBodyAttributes()
	attributes["id"] = schema.StringAttribute{Computed: true, Description: "Resource identifier in the form <bucket>/<path>."}
	attributes["bucket"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "DIAL bucket, for example `platform` or `public`.",
	}
	attributes["path"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Toolset path inside the bucket.",
	}
	attributes["name"] = schema.StringAttribute{Computed: true, Description: "Toolset name returned by the DIAL API."}
	attributes["api_name"] = schema.StringAttribute{Computed: true, Description: "Canonical toolset name returned by the DIAL API."}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing toolset stored in DIAL Core resource storage without managing it. Use `aidial_file_toolset` for toolsets defined in static (file) configuration.",
		Attributes:          attributes,
	}
}

func (d *toolsetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *toolsetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config toolsetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, path := config.Bucket.ValueString(), config.Path.ValueString()
	ts, err := d.client.GetToolset(ctx, bucket, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read toolset", err.Error())
		return
	}
	if ts == nil {
		resp.Diagnostics.AddError(
			"Toolset not found",
			fmt.Sprintf("No toolset exists at bucket %q, path %q. This data source only reads toolsets in resource storage; use aidial_file_toolset for static-configuration toolsets.", bucket, path),
		)
		return
	}

	// A data source has no prior state: start from a clean model so that no
	// configuration value leaks into free-form or write-only attributes.
	state := toolsetModel{Bucket: config.Bucket, Path: config.Path, ID: toolsetID(config.Bucket, config.Path), Name: types.StringNull()}
	if err := setToolsetState(ctx, &state, ts, false); err != nil {
		resp.Diagnostics.AddError("Failed to store toolset state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fileToolsetModel is the state of the aidial_file_toolset data source.
type fileToolsetModel struct {
	Name types.String `tfsdk:"name"`

	toolsetBody
}

// fileToolsetDataSource reads a toolset defined in DIAL Core's static
// configuration, which has no bucket or path.
type fileToolsetDataSource struct{ client *client.Client }

func (d *fileToolsetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_toolset"
}

func (d *fileToolsetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := toolsetDataSourceBodyAttributes()
	attributes["name"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Name of the toolset in DIAL Core's static configuration.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a toolset defined in DIAL Core's static (file) configuration. Toolsets stored in resource storage are read by `aidial_toolset`.",
		Attributes:          attributes,
	}
}

func (d *fileToolsetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *fileToolsetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fileToolsetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	ts, err := d.client.GetFileToolset(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file-sourced toolset", err.Error())
		return
	}
	if ts == nil {
		resp.Diagnostics.AddError(
			"File-sourced toolset not found",
			fmt.Sprintf("No file-sourced toolset named %q was found. This data source only sees toolsets defined in DIAL Core's static configuration, not resource-storage toolsets managed by aidial_toolset.", name),
		)
		return
	}

	state := fileToolsetModel{Name: config.Name}
	if err := setToolsetBodyState(ctx, &state.toolsetBody, ts, false); err != nil {
		resp.Diagnostics.AddError("Failed to store toolset state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
