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

// fileApplicationTypeSchemaDataSource reads a "file-sourced" application type
// schema: one defined in DIAL Core's static configuration rather than created
// via the aidial_application_type_schema resource. File-sourced schemas have
// no bucket/path (they're keyed only by name) and are read-only in the API.
type fileApplicationTypeSchemaDataSource struct{ client *client.Client }

type fileApplicationTypeSchemaDataSourceModel struct {
	Name     types.String `tfsdk:"name"`
	Schema   types.String `tfsdk:"schema"`
	SchemaID types.String `tfsdk:"schema_id"`
	Status   types.String `tfsdk:"status"`
}

func (d *fileApplicationTypeSchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_application_type_schema"
}

func (d *fileApplicationTypeSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an application type schema defined in DIAL Core's static (file) configuration, by name. Schemas created through the API are read with `aidial_application_type_schema`.",
		Attributes: map[string]schema.Attribute{
			"name":      schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}, Description: "Name of the schema in DIAL Core's static configuration."},
			"schema":    schema.StringAttribute{Computed: true, Description: "The application type schema as a JSON object string."},
			"schema_id": schema.StringAttribute{Computed: true, Description: "The `$id` of the schema."},
			"status":    schema.StringAttribute{Computed: true, Description: "Status reported by DIAL Core, when it returns one."},
		},
	}
}

func (d *fileApplicationTypeSchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *fileApplicationTypeSchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state fileApplicationTypeSchemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()
	doc, status, err := d.client.GetFileApplicationTypeSchema(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file application type schema", err.Error())
		return
	}
	if doc == nil {
		resp.Diagnostics.AddError("Application type schema not found", fmt.Sprintf("No application type schema named %q exists in DIAL Core's static configuration.", name))
		return
	}

	model := applicationTypeSchemaResourceModel{}
	if err := setApplicationTypeSchemaState(&model, types.StringNull(), doc); err != nil {
		resp.Diagnostics.AddError("Failed to store application type schema state", err.Error())
		return
	}

	state.Schema = model.Schema
	state.SchemaID = model.SchemaID
	state.Status = types.StringNull()
	if status != "" {
		state.Status = types.StringValue(status)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
