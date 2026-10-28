package provider

import (
	"context"
	"fmt"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// modelDataSource reads an API-managed model stored in DIAL Core resource
// storage at <bucket>/<path>, i.e. one created through the aidial_model
// resource. Models from DIAL Core's static configuration are read by
// aidial_file_model instead; the two use different API paths.
type modelDataSource struct{ client *client.Client }

func (d *modelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

func (d *modelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing model stored in DIAL Core resource storage without managing it. Use `aidial_file_model` for models defined in static (file) configuration.",
		Description:         "Reads an existing resource-storage model (as created by the aidial_model resource) without managing it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Resource identifier in the form <bucket>/<path>."},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Model name to look up.",
			},
			"bucket": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "DIAL bucket. Defaults to \"" + defaultBucket + "\" when not set.",
			},
			"path": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Path of the model within the bucket. Defaults to name when not set.",
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
						"key":                schema.StringAttribute{Computed: true, Sensitive: true},
						"secret_extra_data":  schema.MapAttribute{Computed: true, Sensitive: true, ElementType: types.StringType},
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
			"catalog_schema_id": schema.StringAttribute{Computed: true, Description: "URI of the catalog schema describing catalog_properties."},
			"catalog_properties": schema.DynamicAttribute{
				Computed:    true,
				Description: "Catalog metadata of the model (free-form object).",
			},
		},
	}
}

func (d *modelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *modelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// The data source schema is the resource schema's attribute set, so the
	// resource model doubles as the data source model and setModelState
	// fills it in without a field-by-field copy.
	var state modelResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket := defaultBucket
	if !state.Bucket.IsNull() && !state.Bucket.IsUnknown() && state.Bucket.ValueString() != "" {
		bucket = state.Bucket.ValueString()
	}
	path := state.Name.ValueString()
	if !state.Path.IsNull() && !state.Path.IsUnknown() && state.Path.ValueString() != "" {
		path = state.Path.ValueString()
	}

	model, err := d.client.GetModel(ctx, bucket, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read model", err.Error())
		return
	}
	if model == nil {
		resp.Diagnostics.AddError(
			"Model not found",
			fmt.Sprintf("No model was found at %s/%s. Note this data source only sees models stored through the API (aidial_model), not models defined in DIAL Core's static configuration; use aidial_file_model for those.", bucket, path),
		)
		return
	}

	if err := setModelState(ctx, &state, model); err != nil {
		resp.Diagnostics.AddError("Failed to store model state", err.Error())
		return
	}
	state.ID = types.StringValue(bucket + "/" + path)
	state.Bucket = types.StringValue(bucket)
	state.Path = types.StringValue(path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
