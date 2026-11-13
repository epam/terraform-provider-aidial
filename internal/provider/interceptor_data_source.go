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
	_ datasource.DataSource = &interceptorDataSource{}
	_ datasource.DataSource = &fileInterceptorDataSource{}
)

// interceptorDataSourceBodyAttributes are the computed attributes shared by the
// aidial_interceptor and aidial_file_interceptor data sources. They mirror the
// aidial_interceptor resource.
func interceptorDataSourceBodyAttributes() map[string]schema.Attribute {
	str := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: description}
	}
	return map[string]schema.Attribute{
		"endpoint":           str("URL DIAL Core calls to run the interceptor."),
		"description":        str("Interceptor description."),
		"display_name":       str("Human-readable interceptor name."),
		"display_version":    str("Version shown for the interceptor."),
		"icon_url":           str("URL of the interceptor icon."),
		"catalog_schema_id":  str("URI (`$id`) of the catalog schema describing `catalog_properties`."),
		"forward_auth_token": schema.BoolAttribute{Computed: true, Description: "Whether the caller's authorization token is forwarded to the interceptor."},
		"max_retry_attempts": schema.Int64Attribute{Computed: true, Description: "Maximum number of retry attempts."},
		"user_roles":         schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Roles allowed to use the interceptor."},
		"default_headers": schema.MapAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Additional headers sent with every request to the interceptor's endpoint.",
		},
		"features":           featuresDataSourceAttribute(),
		"interfaces":         interfacesDataSourceAttribute(),
		"defaults":           schema.DynamicAttribute{Computed: true, Description: "Default request parameters (free-form object)."},
		"catalog_properties": schema.DynamicAttribute{Computed: true, Description: "Catalog metadata of the interceptor (free-form object)."},
	}
}

// interceptorDataSource reads an API-managed interceptor stored in DIAL Core
// resource storage at <bucket>/<path>. Interceptors from static configuration
// are read by aidial_file_interceptor.
type interceptorDataSource struct{ client *client.Client }

func (d *interceptorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_interceptor"
}

func (d *interceptorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := interceptorDataSourceBodyAttributes()
	attributes["id"] = schema.StringAttribute{Computed: true, Description: "Resource identifier in the form <bucket>/<path>."}
	attributes["bucket"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.OneOf(platformBucket)},
		Description: "DIAL bucket. Must be \"platform\".",
	}
	attributes["path"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Interceptor path inside the bucket.",
	}
	attributes["name"] = schema.StringAttribute{Computed: true, Description: "Interceptor name returned by the DIAL API."}
	attributes["api_name"] = schema.StringAttribute{Computed: true, Description: "Canonical interceptor name returned by the DIAL API."}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing interceptor stored in DIAL Core resource storage without managing it. Use `aidial_file_interceptor` for interceptors defined in static (file) configuration.",
		Attributes:          attributes,
	}
}

func (d *interceptorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *interceptorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config interceptorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, path := config.Bucket.ValueString(), config.Path.ValueString()
	ic, err := d.client.GetInterceptor(ctx, bucket, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read interceptor", err.Error())
		return
	}
	if ic == nil {
		resp.Diagnostics.AddError(
			"Interceptor not found",
			fmt.Sprintf("No interceptor exists at bucket %q, path %q. This data source only reads interceptors in resource storage; use aidial_file_interceptor for static-configuration interceptors.", bucket, path),
		)
		return
	}

	// A data source has no prior state: start from a clean model so that no
	// configuration value leaks into free-form attributes.
	state := interceptorModel{Bucket: config.Bucket, Path: config.Path, ID: interceptorID(config.Bucket, config.Path), Name: types.StringNull()}
	if err := setInterceptorState(ctx, &state, ic); err != nil {
		resp.Diagnostics.AddError("Failed to store interceptor state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fileInterceptorModel is the state of the aidial_file_interceptor data source.
type fileInterceptorModel struct {
	Name types.String `tfsdk:"name"`

	interceptorBody
}

// fileInterceptorDataSource reads an interceptor defined in DIAL Core's static
// configuration, which has no bucket or path.
type fileInterceptorDataSource struct{ client *client.Client }

func (d *fileInterceptorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_interceptor"
}

func (d *fileInterceptorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := interceptorDataSourceBodyAttributes()
	attributes["name"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Name of the interceptor in DIAL Core's static configuration.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an interceptor defined in DIAL Core's static (file) configuration. Interceptors stored in resource storage are read by `aidial_interceptor`.",
		Attributes:          attributes,
	}
}

func (d *fileInterceptorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *fileInterceptorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fileInterceptorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	ic, err := d.client.GetFileInterceptor(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file-sourced interceptor", err.Error())
		return
	}
	if ic == nil {
		resp.Diagnostics.AddError(
			"File-sourced interceptor not found",
			fmt.Sprintf("No file-sourced interceptor named %q was found. This data source only sees interceptors defined in DIAL Core's static configuration, not resource-storage interceptors managed by aidial_interceptor.", name),
		)
		return
	}

	state := fileInterceptorModel{Name: config.Name}
	if err := setInterceptorBodyState(ctx, &state.interceptorBody, ic); err != nil {
		resp.Diagnostics.AddError("Failed to store interceptor state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
