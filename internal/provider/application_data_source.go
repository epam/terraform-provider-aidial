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
	_ datasource.DataSource = &applicationDataSource{}
	_ datasource.DataSource = &fileApplicationDataSource{}
)

// applicationDataSourceBodyAttributes are the computed attributes shared by the
// aidial_application and aidial_file_application data sources. They mirror the
// aidial_application resource.
func applicationDataSourceBodyAttributes() map[string]schema.Attribute {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: description}
	}
	return map[string]schema.Attribute{
		"endpoint":                   str("URL DIAL Core calls to run the application."),
		"application_type_schema_id": str("URI (`$id`) of the application type schema."),
		"editor_url":                 str("URL of the application editor."),
		"viewer_url":                 str("URL of the application viewer."),
		"interceptors":               schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Names of interceptors applied to the application's requests."},
		"application_properties":     schema.DynamicAttribute{Computed: true, Description: "Values for the properties of the application type schema (free-form object)."},
		"description":                str("Application description."),
		"display_name":               str("Human-readable application name."),
		"display_version":            str("Version shown for the application."),
		"icon_url":                   str("URL of the application icon."),
		"catalog_schema_id":          str("URI (`$id`) of the catalog schema describing `catalog_properties`."),
		"forward_auth_token":         schema.BoolAttribute{Computed: true, Description: "Whether the caller's authorization token is forwarded to the application."},
		"max_retry_attempts":         schema.Int64Attribute{Computed: true, Description: "Maximum number of retry attempts."},
		"user_roles":                 schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Roles allowed to use the application."},
		"default_headers": schema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Additional headers sent with every request to the application's endpoint.",
		},
		"features":           featuresDataSourceAttribute(),
		"interfaces":         interfacesDataSourceAttribute(),
		"defaults":           schema.DynamicAttribute{Computed: true, Description: "Default request parameters (free-form object)."},
		"catalog_properties": schema.DynamicAttribute{Computed: true, Description: "Catalog metadata of the application (free-form object)."},
	}
}

// applicationDataSource reads an API-managed application stored in DIAL Core
// resource storage at <bucket>/<path>. Applications from static configuration
// are read by aidial_file_application.
type applicationDataSource struct{ client *client.Client }

func (d *applicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *applicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := applicationDataSourceBodyAttributes()
	attributes["id"] = schema.StringAttribute{Computed: true, Description: "Resource identifier in the form <bucket>/<path>."}
	attributes["bucket"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "DIAL bucket, for example `platform` or `public`.",
	}
	attributes["path"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Application path inside the bucket.",
	}
	attributes["name"] = schema.StringAttribute{Computed: true, Description: "Application name returned by the DIAL API."}
	attributes["api_name"] = schema.StringAttribute{Computed: true, Description: "Canonical application name returned by the DIAL API."}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing application stored in DIAL Core resource storage without managing it. Use `aidial_file_application` for applications defined in static (file) configuration.",
		Attributes:          attributes,
	}
}

func (d *applicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *applicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, path := config.Bucket.ValueString(), config.Path.ValueString()
	ic, err := d.client.GetApplication(ctx, bucket, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application", err.Error())
		return
	}
	if ic == nil {
		resp.Diagnostics.AddError(
			"Application not found",
			fmt.Sprintf("No application exists at bucket %q, path %q. This data source only reads applications in resource storage; use aidial_file_application for static-configuration applications.", bucket, path),
		)
		return
	}

	// A data source has no prior state: start from a clean model so that no
	// configuration value leaks into free-form attributes.
	state := applicationModel{Bucket: config.Bucket, Path: config.Path, ID: applicationID(config.Bucket, config.Path), Name: types.StringNull()}
	if err := setApplicationState(ctx, &state, ic); err != nil {
		resp.Diagnostics.AddError("Failed to store application state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fileApplicationModel is the state of the aidial_file_application data source.
type fileApplicationModel struct {
	Name types.String `tfsdk:"name"`

	applicationBody
}

// fileApplicationDataSource reads an application defined in DIAL Core's static
// configuration, which has no bucket or path.
type fileApplicationDataSource struct{ client *client.Client }

func (d *fileApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_application"
}

func (d *fileApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := applicationDataSourceBodyAttributes()
	attributes["name"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Name of the application in DIAL Core's static configuration.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an application defined in DIAL Core's static (file) configuration. Applications stored in resource storage are read by `aidial_application`.",
		Attributes:          attributes,
	}
}

func (d *fileApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *fileApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fileApplicationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	ic, err := d.client.GetFileApplication(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file-sourced application", err.Error())
		return
	}
	if ic == nil {
		resp.Diagnostics.AddError(
			"File-sourced application not found",
			fmt.Sprintf("No file-sourced application named %q was found. This data source only sees applications defined in DIAL Core's static configuration, not resource-storage applications managed by aidial_application.", name),
		)
		return
	}

	state := fileApplicationModel{Name: config.Name}
	if err := setApplicationBodyState(ctx, &state.applicationBody, ic); err != nil {
		resp.Diagnostics.AddError("Failed to store application state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
