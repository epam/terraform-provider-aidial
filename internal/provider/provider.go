package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const authDescription = "Authenticate with either `api_key` or `id_token` (an OIDC JWT from a CI/CD system); a value in the provider block beats its environment variable " +
	"(`AIDIAL_API_KEY`, `AIDIAL_ID_TOKEN`), and an ignored variable produces a warning. See the README for details."

type AidialProvider struct{}

type providerModel struct {
	BaseURL types.String `tfsdk:"base_url"`
	APIKey  types.String `tfsdk:"api_key"`
	IDToken types.String `tfsdk:"id_token"`
}

func New() provider.Provider { return &AidialProvider{} }
func (p *AidialProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "aidial"
}
func (p *AidialProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Terraform provider for [EPAM AI DIAL Core](https://github.com/epam/ai-dial-core): manage roles, keys, models and routes as code. " + authDescription, Attributes: map[string]schema.Attribute{
		"base_url": schema.StringAttribute{Required: true, Description: "DIAL API base URL, for example http://IP:PORT or https://IP:PORT."},
		"api_key":  schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "DIAL API key sent in the `Api-Key` header. Falls back to the `AIDIAL_API_KEY` environment variable. Exactly one credential is used: see the provider description for precedence."},
		"id_token": schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "OIDC ID token (JWT) from a CI/CD system, sent as `Authorization: Bearer <token>`. Falls back to the `AIDIAL_ID_TOKEN` environment variable. The provider never fetches or refreshes it, so it must outlive the whole run. Alternative to `api_key`."},
	}}
}
func (p *AidialProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.BaseURL.IsUnknown() || config.APIKey.IsUnknown() || config.IDToken.IsUnknown() {
		return
	}
	cred, warnings, err := resolveCredential(credentialInputs{
		configAPIKey:  config.APIKey.ValueString(),
		configIDToken: config.IDToken.ValueString(),
		envAPIKey:     os.Getenv(envAPIKey),
		envIDToken:    os.Getenv(envIDToken),
	})
	if err != nil {
		resp.Diagnostics.AddError("Invalid DIAL credentials", err.Error())
		return
	}
	for _, w := range warnings {
		resp.Diagnostics.AddWarning("Ignored DIAL credential environment variable", w)
	}
	creds := client.APIKeyCredentials(cred.apiKey)
	if cred.idToken != "" {
		if err := checkJWTShape(cred.idToken); err != nil {
			resp.Diagnostics.AddError("Invalid DIAL id_token", err.Error())
			return
		}
		creds = client.IDTokenCredentials(cred.idToken)
	}
	c, err := client.NewClient(config.BaseURL.ValueString(), creds)
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
		func() resource.Resource { return &catalogSchemaResource{} },
		func() resource.Resource { return &applicationTypeSchemaResource{} },
		func() resource.Resource { return &toolsetResource{} },
		func() resource.Resource { return &interceptorResource{} },
		func() resource.Resource { return &translatorResource{} },
		func() resource.Resource { return &aliasResource{} },
		func() resource.Resource { return &applicationResource{} },
	}
}
func (p *AidialProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &roleDataSource{} },
		func() datasource.DataSource { return &bucketDataSource{} },
		func() datasource.DataSource { return &fileModelDataSource{} },
		func() datasource.DataSource { return &modelDataSource{} },
		func() datasource.DataSource { return &catalogSchemaDataSource{} },
		func() datasource.DataSource { return &applicationTypeSchemaDataSource{} },
		func() datasource.DataSource { return &fileApplicationTypeSchemaDataSource{} },
		func() datasource.DataSource { return &toolsetDataSource{} },
		func() datasource.DataSource { return &fileToolsetDataSource{} },
		func() datasource.DataSource { return &interceptorDataSource{} },
		func() datasource.DataSource { return &fileInterceptorDataSource{} },
		func() datasource.DataSource { return &translatorDataSource{} },
		func() datasource.DataSource { return &fileTranslatorDataSource{} },
		func() datasource.DataSource { return &aliasDataSource{} },
		func() datasource.DataSource { return &fileAliasDataSource{} },
		func() datasource.DataSource { return &applicationDataSource{} },
		func() datasource.DataSource { return &fileApplicationDataSource{} },
	}
}
