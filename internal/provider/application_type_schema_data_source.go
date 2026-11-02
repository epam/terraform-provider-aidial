package provider

import (
	"context"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type applicationTypeSchemaDataSource struct{ client *client.Client }

type applicationTypeSchemaDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Bucket   types.String `tfsdk:"bucket"`
	Path     types.String `tfsdk:"path"`
	Schema   types.String `tfsdk:"schema"`
	SchemaID types.String `tfsdk:"schema_id"`
}

func (d *applicationTypeSchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_type_schema"
}

func (d *applicationTypeSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing API-managed DIAL Core application type schema without managing it. Schemas defined in Core's static configuration are read with `aidial_file_application_type_schema`.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true},
			"bucket": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf(platformBucket)}, Description: "DIAL bucket. Must be \"platform\"."},
			"path":   schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, Description: "Application type schema path inside the bucket."},
			"schema": schema.StringAttribute{Computed: true, Description: "The application type schema as a JSON object string."},
			"schema_id": schema.StringAttribute{
				Computed:    true,
				Description: "The `$id` of the schema.",
			},
		},
	}
}

func (d *applicationTypeSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *applicationTypeSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state applicationTypeSchemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket := state.Bucket.ValueString()

	doc, err := d.client.GetApplicationTypeSchema(ctx, bucket, state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application type schema", err.Error())
		return
	}
	if doc == nil {
		resp.Diagnostics.AddError("Application type schema not found", "The requested application type schema does not exist.")
		return
	}

	model := applicationTypeSchemaResourceModel{}
	if err := setApplicationTypeSchemaState(&model, types.StringNull(), doc); err != nil {
		resp.Diagnostics.AddError("Failed to store application type schema state", err.Error())
		return
	}

	state.ID = types.StringValue(bucket + "/" + state.Path.ValueString())
	state.Schema = model.Schema
	state.SchemaID = model.SchemaID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
