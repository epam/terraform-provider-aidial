package provider

import (
	"context"
	"fmt"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fileModelDataSource reads a "file-sourced" model: one defined in DIAL
// Core's static configuration rather than created via the aidial_model
// resource. File-sourced models have no bucket/path (they're keyed only by
// name) and are read-only in the API (GET only, no PUT/DELETE), so unlike
// aidial_model there is no corresponding resource for them.
type fileModelDataSource struct{ client *client.Client }

type fileModelDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	APIName types.String `tfsdk:"api_name"`

	Type                types.String `tfsdk:"type"`
	DisplayName         types.String `tfsdk:"display_name"`
	Description         types.String `tfsdk:"description"`
	Endpoint            types.String `tfsdk:"endpoint"`
	UserRoles           types.Set    `tfsdk:"user_roles"`
	ResponsesEndpoint   types.String `tfsdk:"responses_endpoint"`
	DisplayVersion      types.String `tfsdk:"display_version"`
	TokenizerModel      types.String `tfsdk:"tokenizer_model"`
	IconURL             types.String `tfsdk:"icon_url"`
	Intro               types.String `tfsdk:"intro"`
	Reference           types.String `tfsdk:"reference"`
	ForwardAuthToken    types.Bool   `tfsdk:"forward_auth_token"`
	MaxRetryAttempts    types.Int64  `tfsdk:"max_retry_attempts"`
	MaxInputAttachments types.Int64  `tfsdk:"max_input_attachments"`
	Author              types.String `tfsdk:"author"`
	Status              types.String `tfsdk:"status"`

	Features  types.Object `tfsdk:"features"`
	Limits    types.Object `tfsdk:"limits"`
	Pricing   types.Object `tfsdk:"pricing"`
	Upstreams types.List   `tfsdk:"upstreams"`

	Dependencies         types.Set `tfsdk:"dependencies"`
	DescriptionKeywords  types.Set `tfsdk:"description_keywords"`
	InputAttachmentTypes types.Set `tfsdk:"input_attachment_types"`
	Interceptors         types.Set `tfsdk:"interceptors"`
	FieldsHashingOrder   types.Set `tfsdk:"fields_hashing_order"`

	OverrideName   types.String  `tfsdk:"override_name"`
	Interfaces     types.Map     `tfsdk:"interfaces"`
	DefaultHeaders types.Map     `tfsdk:"default_headers"`
	Defaults       types.Dynamic `tfsdk:"defaults"`
}

func (d *fileModelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file_model"
}

func (d *fileModelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a model defined in DIAL Core static (file) configuration. File-sourced models are read-only and have no bucket or path.",
		Description:         "Reads a file-sourced model: one defined in DIAL Core's static configuration rather than created via the aidial_model resource. File-sourced models have no bucket/path and cannot be created, updated, or deleted through the API.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "File-sourced model name to look up.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical model name returned by the DIAL API.",
			},
			"type":                  schema.StringAttribute{Computed: true, Description: "DIAL model type, for example CHAT."},
			"display_name":          schema.StringAttribute{Computed: true, Description: "Human-readable model name."},
			"description":           schema.StringAttribute{Computed: true, Description: "Model description."},
			"endpoint":              schema.StringAttribute{Computed: true, Description: "Model API endpoint."},
			"user_roles":            schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Roles allowed to use the model."},
			"responses_endpoint":    schema.StringAttribute{Computed: true},
			"display_version":       schema.StringAttribute{Computed: true},
			"tokenizer_model":       schema.StringAttribute{Computed: true},
			"icon_url":              schema.StringAttribute{Computed: true},
			"intro":                 schema.StringAttribute{Computed: true},
			"reference":             schema.StringAttribute{Computed: true},
			"forward_auth_token":    schema.BoolAttribute{Computed: true},
			"max_retry_attempts":    schema.Int64Attribute{Computed: true},
			"max_input_attachments": schema.Int64Attribute{Computed: true},
			"author":                schema.StringAttribute{Computed: true},
			"status":                schema.StringAttribute{Computed: true, Description: "Model status returned by the DIAL API."},
			"features": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"accessible_by_per_request_key":              schema.BoolAttribute{Computed: true},
					"allow_resume":                               schema.BoolAttribute{Computed: true},
					"assistant_attachments_in_request_supported": schema.BoolAttribute{Computed: true},
					"auto_caching_supported":                     schema.BoolAttribute{Computed: true},
					"cache_supported":                            schema.BoolAttribute{Computed: true},
					"configuration_endpoint":                     schema.StringAttribute{Computed: true},
					"consent_required":                           schema.BoolAttribute{Computed: true},
					"content_parts_supported":                    schema.BoolAttribute{Computed: true},
					"custom_temperature_supported":               schema.BoolAttribute{Computed: true},
					"folder_attachments_supported":               schema.BoolAttribute{Computed: true},
					"max_completion_tokens_supported":            schema.BoolAttribute{Computed: true},
					"max_tokens_supported":                       schema.BoolAttribute{Computed: true},
					"parallel_tool_calls_supported":              schema.BoolAttribute{Computed: true},
					"rate_endpoint":                              schema.StringAttribute{Computed: true},
					"reasoning_efforts":                          schema.SetAttribute{Computed: true, ElementType: types.StringType},
					"seed_supported":                             schema.BoolAttribute{Computed: true},
					"support_comment_in_rate_response":           schema.BoolAttribute{Computed: true},
					"system_prompt_supported":                    schema.BoolAttribute{Computed: true},
					"temperature_supported":                      schema.BoolAttribute{Computed: true},
					"tokenize_endpoint":                          schema.StringAttribute{Computed: true},
					"tools_supported":                            schema.BoolAttribute{Computed: true},
					"truncate_prompt_endpoint":                   schema.StringAttribute{Computed: true},
					"url_attachments_supported":                  schema.BoolAttribute{Computed: true},
					"skills_supported":                           schema.BoolAttribute{Computed: true},
				},
			},
			"limits": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"max_prompt_tokens":     schema.Int64Attribute{Computed: true},
					"max_completion_tokens": schema.Int64Attribute{Computed: true},
					"max_total_tokens":      schema.Int64Attribute{Computed: true},
				},
			},
			"pricing": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"unit":        schema.StringAttribute{Computed: true},
					"prompt":      schema.StringAttribute{Computed: true},
					"completion":  schema.StringAttribute{Computed: true},
					"cache_read":  schema.StringAttribute{Computed: true},
					"cache_write": schema.StringAttribute{Computed: true},
				},
			},
			"upstreams": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"endpoint":           schema.StringAttribute{Computed: true},
						"responses_endpoint": schema.StringAttribute{Computed: true},
						"id":                 schema.StringAttribute{Computed: true},
						"extra_data":         schema.MapAttribute{Computed: true, ElementType: types.StringType},
						"key":                schema.StringAttribute{Computed: true},
						"secret_extra_data":  schema.MapAttribute{Computed: true, ElementType: types.StringType},
						"tier":               schema.Int64Attribute{Computed: true},
						"weight":             schema.Int64Attribute{Computed: true},
					},
				},
			},
			"dependencies":           schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"description_keywords":   schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"input_attachment_types": schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"interceptors":           schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"fields_hashing_order":   schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Field paths used to compute a stable hash of the request body, for example for caching."},
			"override_name":          schema.StringAttribute{Computed: true, Description: "Overrides the model name presented to clients."},
			"interfaces": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"base_url": schema.StringAttribute{Computed: true},
					},
				},
				Description: "Per-interface deployment configuration, keyed by interface name. Only base_url is modeled.",
			},
			"default_headers": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Additional headers sent with every request to the model's endpoint.",
			},
			"defaults": schema.DynamicAttribute{
				Computed:    true,
				Description: "Default request parameters merged into every request to the model (free-form object).",
			},
		},
	}
}

func (d *fileModelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *fileModelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config fileModelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := d.client.GetFileModel(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file-sourced model", err.Error())
		return
	}
	if model == nil {
		resp.Diagnostics.AddError(
			"File-sourced model not found",
			fmt.Sprintf("No file-sourced model named %q was found. Note this data source only sees models defined in DIAL Core's static configuration, not resource-storage models managed by aidial_model.", config.Name.ValueString()),
		)
		return
	}

	// Reuse setModelState's field-mapping logic (in particular the
	// features/limits/pricing/upstreams object construction) via a
	// throwaway modelResourceModel, then copy the relevant fields into
	// this data source's own model.
	full := modelResourceModel{Name: config.Name}
	if err := setModelState(ctx, &full, model); err != nil {
		resp.Diagnostics.AddError("Failed to store model state", err.Error())
		return
	}

	config.APIName = full.APIName
	config.Type = full.Type
	config.DisplayName = full.DisplayName
	config.Description = full.Description
	config.Endpoint = full.Endpoint
	config.UserRoles = full.UserRoles
	config.ResponsesEndpoint = full.ResponsesEndpoint
	config.DisplayVersion = full.DisplayVersion
	config.TokenizerModel = full.TokenizerModel
	config.IconURL = full.IconURL
	config.Intro = full.Intro
	config.Reference = full.Reference
	config.ForwardAuthToken = full.ForwardAuthToken
	config.MaxRetryAttempts = full.MaxRetryAttempts
	config.MaxInputAttachments = full.MaxInputAttachments
	config.Author = full.Author
	config.Status = full.Status
	config.Features = full.Features
	config.Limits = full.Limits
	config.Pricing = full.Pricing
	config.Upstreams = full.Upstreams
	config.Dependencies = full.Dependencies
	config.DescriptionKeywords = full.DescriptionKeywords
	config.InputAttachmentTypes = full.InputAttachmentTypes
	config.Interceptors = full.Interceptors
	config.FieldsHashingOrder = full.FieldsHashingOrder
	config.OverrideName = full.OverrideName
	config.Interfaces = full.Interfaces
	config.DefaultHeaders = full.DefaultHeaders
	config.Defaults = full.Defaults

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
