package provider

import (
	"context"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type bucketDataSource struct{ client *client.Client }
type bucketDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Bucket  types.String `tfsdk:"bucket"`
	AppData types.String `tfsdk:"appdata"`
}

func (d *bucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}
func (d *bucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Looks up the bucket owned by the API key the provider is configured with.", Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "bucket": schema.StringAttribute{Computed: true}, "appdata": schema.StringAttribute{Computed: true}}}
}
func (d *bucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *bucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state bucketDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := d.client.GetBucket(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bucket", err.Error())
		return
	}
	state.ID = types.StringValue(b.Bucket)
	state.Bucket = types.StringValue(b.Bucket)
	state.AppData = types.StringValue(b.AppData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
