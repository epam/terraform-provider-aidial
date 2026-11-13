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
	_ datasource.DataSource = &translatorDataSource{}
	_ datasource.DataSource = &fileTranslatorDataSource{}
)

// translatorDataSourceBodyAttributes are the computed attributes shared by the
// aidial_translator and aidial_file_translator data sources. They mirror the
// aidial_translator resource.
func translatorDataSourceBodyAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base_url": schema.StringAttribute{Computed: true, Description: "Base URL of the translator service."},
		"in":       schema.StringAttribute{Computed: true, Description: translatorInterfaceDescription("Incoming")},
		"out":      schema.StringAttribute{Computed: true, Description: translatorInterfaceDescription("Outgoing")},
	}
}

func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected *client.Client")
		return nil
	}
	return c
}

// translatorDataSource reads an API-managed translator stored in DIAL Core
// resource storage at <bucket>/<path>. Translators from static configuration
// are read by aidial_file_translator.
type translatorDataSource struct{ client *client.Client }

func (d *translatorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_translator"
}

func (d *translatorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := translatorDataSourceBodyAttributes()
	attributes["id"] = schema.StringAttribute{Computed: true, Description: "Resource identifier in the form <bucket>/<path>."}
	attributes["bucket"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.OneOf(platformBucket)},
		Description: "DIAL bucket. Must be \"platform\".",
	}
	attributes["path"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Translator path inside the bucket.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing translator stored in DIAL Core resource storage without managing it. Use `aidial_file_translator` for translators defined in static (file) configuration.",
		Attributes:          attributes,
	}
}

func (d *translatorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureDataSourceClient(req, resp); c != nil {
		d.client = c
	}
}

func (d *translatorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config translatorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, path := config.Bucket.ValueString(), config.Path.ValueString()
	t, err := d.client.GetTranslator(ctx, bucket, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read translator", err.Error())
		return
	}
	if t == nil {
		resp.Diagnostics.AddError(
			"Translator not found",
			fmt.Sprintf("No translator exists at bucket %q, path %q. This data source only reads translators in resource storage; use aidial_file_translator for static-configuration translators.", bucket, path),
		)
		return
	}

	state := translatorModel{Bucket: config.Bucket, Path: config.Path, ID: translatorID(config.Bucket, config.Path)}
	setTranslatorBodyState(&state.translatorBody, t)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fileTranslatorModel is the state of the aidial_file_translator data source.
type fileTranslatorModel struct {
	Name types.String `tfsdk:"name"`

	translatorBody
}

// fileTranslatorDataSource reads a translator defined in DIAL Core's static
// configuration, which has no bucket or path.
type fileTranslatorDataSource struct{ client *client.Client }

func (d *fileTranslatorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_translator"
}

func (d *fileTranslatorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := translatorDataSourceBodyAttributes()
	attributes["name"] = schema.StringAttribute{
		Required:    true,
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		Description: "Name of the translator in DIAL Core's static configuration.",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a translator defined in DIAL Core's static (file) configuration. Translators stored in resource storage are read by `aidial_translator`.",
		Attributes:          attributes,
	}
}

func (d *fileTranslatorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureDataSourceClient(req, resp); c != nil {
		d.client = c
	}
}

func (d *fileTranslatorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fileTranslatorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	t, err := d.client.GetFileTranslator(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file-sourced translator", err.Error())
		return
	}
	if t == nil {
		resp.Diagnostics.AddError(
			"File-sourced translator not found",
			fmt.Sprintf("No file-sourced translator named %q was found. This data source only sees translators defined in DIAL Core's static configuration, not resource-storage translators managed by aidial_translator.", name),
		)
		return
	}

	state := fileTranslatorModel{Name: config.Name}
	setTranslatorBodyState(&state.translatorBody, t)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
