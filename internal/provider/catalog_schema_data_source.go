package provider

import (
	"context"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type catalogSchemaDataSource struct{ client *client.Client }

type catalogSchemaDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Bucket   types.String `tfsdk:"bucket"`
	Path     types.String `tfsdk:"path"`
	Schema   types.String `tfsdk:"schema"`
	SchemaID types.String `tfsdk:"schema_id"`
}

func (d *catalogSchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog_schema"
}

func (d *catalogSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing DIAL Core catalog schema without managing it.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true},
			"bucket": schema.StringAttribute{Optional: true, Computed: true, Description: "DIAL bucket. Defaults to \"" + defaultBucket + "\" when not set."},
			"path":   schema.StringAttribute{Required: true, Description: "Catalog schema path inside the bucket."},
			"schema": schema.StringAttribute{Computed: true, Description: "The catalog schema as a JSON object string."},
			"schema_id": schema.StringAttribute{
				Computed:    true,
				Description: "The `$id` of the schema.",
			},
		},
	}
}

func (d *catalogSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *catalogSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state catalogSchemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket := defaultBucket
	if !state.Bucket.IsNull() && !state.Bucket.IsUnknown() && state.Bucket.ValueString() != "" {
		bucket = state.Bucket.ValueString()
	}

	doc, err := d.client.GetCatalogSchema(ctx, bucket, state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read catalog schema", err.Error())
		return
	}
	if doc == nil {
		resp.Diagnostics.AddError("Catalog schema not found", "The requested catalog schema does not exist.")
		return
	}

	model := catalogSchemaResourceModel{}
	if err := setCatalogSchemaState(&model, types.StringNull(), doc); err != nil {
		resp.Diagnostics.AddError("Failed to store catalog schema state", err.Error())
		return
	}

	state.ID = types.StringValue(bucket + "/" + state.Path.ValueString())
	state.Bucket = types.StringValue(bucket)
	state.Schema = model.Schema
	state.SchemaID = model.SchemaID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
