package provider

import (
	"context"
	"fmt"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type AidialProvider struct{}

type providerModel struct {
	BaseURL types.String `tfsdk:"base_url"`
	APIKey  types.String `tfsdk:"api_key"`
}

func New() provider.Provider { return &AidialProvider{} }
func (p *AidialProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "aidial"
}
func (p *AidialProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Terraform provider for [EPAM AI DIAL Core](https://github.com/epam/ai-dial-core): manage roles, keys, models and routes as code.", Attributes: map[string]schema.Attribute{
		"base_url": schema.StringAttribute{Required: true, Description: "DIAL API base URL, for example http://IP:PORT or https://IP:PORT."},
		"api_key":  schema.StringAttribute{Required: true, Sensitive: true, Description: "DIAL API key sent in the Api-Key header."},
	}}
}
func (p *AidialProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.BaseURL.IsUnknown() || config.APIKey.IsUnknown() {
		return
	}
	c, err := client.NewClient(config.BaseURL.ValueString(), config.APIKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure DIAL client", fmt.Sprintf("Error: %s", err))
		return
	}
	resp.ResourceData = c
	resp.DataSourceData = c
}
func (p *AidialProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return &roleResource{} },
		func() resource.Resource { return &keyResource{} },
		func() resource.Resource { return &modelResource{} },
		func() resource.Resource { return &routeResource{} },
	}
}
func (p *AidialProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &roleDataSource{} },
		func() datasource.DataSource { return &bucketDataSource{} },
		func() datasource.DataSource { return &fileModelDataSource{} },
	}
}
