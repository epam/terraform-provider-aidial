package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type modelResource struct {
	client *client.Client
}

type modelResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Bucket       types.String `tfsdk:"bucket"`
	Path         types.String `tfsdk:"path"`
	Name         types.String `tfsdk:"name"`
	APIName      types.String `tfsdk:"api_name"`
	Type         types.String `tfsdk:"type"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	Endpoint     types.String `tfsdk:"endpoint"`
	OverrideName types.String `tfsdk:"override_name"`

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

	Interfaces     types.Map     `tfsdk:"interfaces"`
	DefaultHeaders types.Map     `tfsdk:"default_headers"`
	Defaults       types.Dynamic `tfsdk:"defaults"`

	CatalogSchemaID   types.String  `tfsdk:"catalog_schema_id"`
	CatalogProperties types.Dynamic `tfsdk:"catalog_properties"`
}

type modelInterfaceModel struct {
	BaseURL types.String `tfsdk:"base_url"`
}

type modelFeaturesModel struct {
	AccessibleByPerRequestKey              types.Bool   `tfsdk:"accessible_by_per_request_key"`
	AllowResume                            types.Bool   `tfsdk:"allow_resume"`
	AssistantAttachmentsInRequestSupported types.Bool   `tfsdk:"assistant_attachments_in_request_supported"`
	AutoCachingSupported                   types.Bool   `tfsdk:"auto_caching_supported"`
	CacheSupported                         types.Bool   `tfsdk:"cache_supported"`
	ConfigurationEndpoint                  types.String `tfsdk:"configuration_endpoint"`
	ConsentRequired                        types.Bool   `tfsdk:"consent_required"`
	ContentPartsSupported                  types.Bool   `tfsdk:"content_parts_supported"`
	CustomTemperatureSupported             types.Bool   `tfsdk:"custom_temperature_supported"`
	FolderAttachmentsSupported             types.Bool   `tfsdk:"folder_attachments_supported"`
	MaxCompletionTokensSupported           types.Bool   `tfsdk:"max_completion_tokens_supported"`
	MaxTokensSupported                     types.Bool   `tfsdk:"max_tokens_supported"`
	ParallelToolCallsSupported             types.Bool   `tfsdk:"parallel_tool_calls_supported"`
	RateEndpoint                           types.String `tfsdk:"rate_endpoint"`
	ReasoningEfforts                       types.Set    `tfsdk:"reasoning_efforts"`
	SeedSupported                          types.Bool   `tfsdk:"seed_supported"`
	SupportCommentInRateResponse           types.Bool   `tfsdk:"support_comment_in_rate_response"`
	SystemPromptSupported                  types.Bool   `tfsdk:"system_prompt_supported"`
	TemperatureSupported                   types.Bool   `tfsdk:"temperature_supported"`
	TokenizeEndpoint                       types.String `tfsdk:"tokenize_endpoint"`
	ToolsSupported                         types.Bool   `tfsdk:"tools_supported"`
	TruncatePromptEndpoint                 types.String `tfsdk:"truncate_prompt_endpoint"`
	URLAttachmentsSupported                types.Bool   `tfsdk:"url_attachments_supported"`
	SkillsSupported                        types.Bool   `tfsdk:"skills_supported"`
}

type modelLimitsModel struct {
	MaxPromptTokens     types.Int64 `tfsdk:"max_prompt_tokens"`
	MaxCompletionTokens types.Int64 `tfsdk:"max_completion_tokens"`
	MaxTotalTokens      types.Int64 `tfsdk:"max_total_tokens"`
}

type modelPricingModel struct {
	Unit       types.String `tfsdk:"unit"`
	Prompt     types.String `tfsdk:"prompt"`
	Completion types.String `tfsdk:"completion"`
	CacheRead  types.String `tfsdk:"cache_read"`
	CacheWrite types.String `tfsdk:"cache_write"`
}

type modelUpstreamModel struct {
	Endpoint          types.String `tfsdk:"endpoint"`
	ResponsesEndpoint types.String `tfsdk:"responses_endpoint"`
	ID                types.String `tfsdk:"id"`
	ExtraData         types.Map    `tfsdk:"extra_data"`
	Key               types.String `tfsdk:"key"`
	SecretExtraData   types.Map    `tfsdk:"secret_extra_data"`
	Tier              types.Int64  `tfsdk:"tier"`
	Weight            types.Int64  `tfsdk:"weight"`
}

var modelInterfaceAttrTypes = map[string]attr.Type{
	"base_url": types.StringType,
}

var modelFeaturesAttrTypes = map[string]attr.Type{
	"accessible_by_per_request_key":              types.BoolType,
	"allow_resume":                               types.BoolType,
	"assistant_attachments_in_request_supported": types.BoolType,
	"auto_caching_supported":                     types.BoolType,
	"cache_supported":                            types.BoolType,
	"configuration_endpoint":                     types.StringType,
	"consent_required":                           types.BoolType,
	"content_parts_supported":                    types.BoolType,
	"custom_temperature_supported":               types.BoolType,
	"folder_attachments_supported":               types.BoolType,
	"max_completion_tokens_supported":            types.BoolType,
	"max_tokens_supported":                       types.BoolType,
	"parallel_tool_calls_supported":              types.BoolType,
	"rate_endpoint":                              types.StringType,
	"reasoning_efforts":                          types.SetType{ElemType: types.StringType},
	"seed_supported":                             types.BoolType,
	"support_comment_in_rate_response":           types.BoolType,
	"system_prompt_supported":                    types.BoolType,
	"temperature_supported":                      types.BoolType,
	"tokenize_endpoint":                          types.StringType,
	"tools_supported":                            types.BoolType,
	"truncate_prompt_endpoint":                   types.StringType,
	"url_attachments_supported":                  types.BoolType,
	"skills_supported":                           types.BoolType,
}

var modelLimitsAttrTypes = map[string]attr.Type{
	"max_prompt_tokens":     types.Int64Type,
	"max_completion_tokens": types.Int64Type,
	"max_total_tokens":      types.Int64Type,
}

var modelPricingAttrTypes = map[string]attr.Type{
	"unit":        types.StringType,
	"prompt":      types.StringType,
	"completion":  types.StringType,
	"cache_read":  types.StringType,
	"cache_write": types.StringType,
}

var modelUpstreamAttrTypes = map[string]attr.Type{
	"endpoint":           types.StringType,
	"responses_endpoint": types.StringType,
	"id":                 types.StringType,
	"extra_data":         types.MapType{ElemType: types.StringType},
	"key":                types.StringType,
	"secret_extra_data":  types.MapType{ElemType: types.StringType},
	"tier":               types.Int64Type,
	"weight":             types.Int64Type,
}

func (r *modelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

func (r *modelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a model deployment in DIAL Core (`PUT /v1/models/{bucket}/{path}`).",
		Version:             1,
		Attributes: map[string]schema.Attribute{
			"defaults": schema.DynamicAttribute{
				Optional:    true,
				Description: "Default request parameters merged into every request to the model (free-form object, sent to DIAL as-is), e.g. max_tokens, custom_fields, tools.",
			},
			"catalog_schema_id": schema.StringAttribute{
				Optional:    true,
				Description: "URI (`$id`) of the catalog schema describing `catalog_properties`, e.g. the `schema_id` of an `aidial_catalog_schema`.",
			},
			"catalog_properties": schema.DynamicAttribute{
				Optional:    true,
				Description: "Catalog metadata of the model (free-form object, sent to DIAL as `catalogProperties`), e.g. provider, vendor, license, knowledgeCutoffDate, parameters.",
			},
			"override_name": schema.StringAttribute{
				Optional: true,
			},
			"interfaces": schema.MapNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"base_url": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"default_headers": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					bucketDefaultsToPlatform{},
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Target DIAL bucket. Defaults to \"" + defaultBucket + "\" when not set.",
			},
			"path": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					pathDefaultsToName{},
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Model path inside the target bucket. Defaults to name when not set.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Model name stored in the DIAL API request body.",
			},
			"api_name": schema.StringAttribute{
				Computed:    true,
				Description: "Canonical model name returned by the DIAL API.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "DIAL model type, for example CHAT.",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Description: "Human-readable model name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Model description.",
			},
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Model API endpoint.",
			},
			"user_roles": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Roles allowed to use the model.",
			},
			"responses_endpoint": schema.StringAttribute{
				Optional: true,
			},
			"display_version": schema.StringAttribute{
				Optional: true,
			},
			"tokenizer_model": schema.StringAttribute{
				Optional:    true,
				Description: "Model whose tokenizer DIAL Core uses to count tokens for this model (tokenizerModel in the API), e.g. gpt-4-0314.",
			},
			"icon_url": schema.StringAttribute{
				Optional: true,
			},
			"intro": schema.StringAttribute{
				Optional: true,
			},
			"reference": schema.StringAttribute{
				Optional: true,
			},
			"forward_auth_token": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"max_retry_attempts": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"max_input_attachments": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of attachments in one request (maxInputAttachments in the API).",
			},
			"author": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Model status returned by the DIAL API.",
			},
			"features": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"accessible_by_per_request_key":              schema.BoolAttribute{Optional: true, Computed: true},
					"allow_resume":                               schema.BoolAttribute{Optional: true, Computed: true},
					"assistant_attachments_in_request_supported": schema.BoolAttribute{Optional: true, Computed: true},
					"auto_caching_supported":                     schema.BoolAttribute{Optional: true, Computed: true},
					"cache_supported":                            schema.BoolAttribute{Optional: true, Computed: true},
					"configuration_endpoint":                     schema.StringAttribute{Optional: true, Computed: true},
					"consent_required":                           schema.BoolAttribute{Optional: true, Computed: true},
					"content_parts_supported":                    schema.BoolAttribute{Optional: true, Computed: true},
					"custom_temperature_supported":               schema.BoolAttribute{Optional: true, Computed: true},
					"folder_attachments_supported":               schema.BoolAttribute{Optional: true, Computed: true},
					"max_completion_tokens_supported":            schema.BoolAttribute{Optional: true, Computed: true},
					"max_tokens_supported":                       schema.BoolAttribute{Optional: true, Computed: true},
					"parallel_tool_calls_supported":              schema.BoolAttribute{Optional: true, Computed: true},
					"rate_endpoint":                              schema.StringAttribute{Optional: true, Computed: true},
					"reasoning_efforts": schema.SetAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
					},
					"seed_supported":                   schema.BoolAttribute{Optional: true, Computed: true},
					"support_comment_in_rate_response": schema.BoolAttribute{Optional: true, Computed: true},
					"system_prompt_supported":          schema.BoolAttribute{Optional: true, Computed: true},
					"temperature_supported":            schema.BoolAttribute{Optional: true, Computed: true},
					"tokenize_endpoint":                schema.StringAttribute{Optional: true, Computed: true},
					"tools_supported":                  schema.BoolAttribute{Optional: true, Computed: true},
					"truncate_prompt_endpoint":         schema.StringAttribute{Optional: true, Computed: true},
					"url_attachments_supported":        schema.BoolAttribute{Optional: true, Computed: true},
					"skills_supported":                 schema.BoolAttribute{Optional: true, Computed: true},
				},
			},
			"limits": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"max_prompt_tokens":     schema.Int64Attribute{Optional: true, Computed: true},
					"max_completion_tokens": schema.Int64Attribute{Optional: true, Computed: true},
					"max_total_tokens":      schema.Int64Attribute{Optional: true, Computed: true},
				},
			},
			"pricing": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"unit":        schema.StringAttribute{Optional: true, Computed: true},
					"prompt":      schema.StringAttribute{Optional: true, Computed: true},
					"completion":  schema.StringAttribute{Optional: true, Computed: true},
					"cache_read":  schema.StringAttribute{Optional: true, Computed: true},
					"cache_write": schema.StringAttribute{Optional: true, Computed: true},
				},
			},
			"upstreams": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"endpoint":           schema.StringAttribute{Optional: true, Computed: true},
						"responses_endpoint": schema.StringAttribute{Optional: true, Computed: true},
						"id":                 schema.StringAttribute{Optional: true, Computed: true},
						"extra_data":         schema.MapAttribute{ElementType: types.StringType, Optional: true, Computed: true},
						"key":                schema.StringAttribute{Optional: true, Sensitive: true},
						"secret_extra_data":  schema.MapAttribute{ElementType: types.StringType, Optional: true, Sensitive: true},
						"tier":               schema.Int64Attribute{Optional: true, Computed: true},
						"weight":             schema.Int64Attribute{Optional: true, Computed: true},
					},
				},
			},
			"dependencies": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"description_keywords": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"input_attachment_types": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"interceptors": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"fields_hashing_order": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Field paths used to compute a stable hash of the request body, for example for caching.",
			},
		},
	}
}

func (r *modelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "Expected *client.Client")
		return
	}

	r.client = c
}

func modelFromPlan(ctx context.Context, plan modelResourceModel) (*client.Model, error) {
	result := &client.Model{
		Name:         plan.Name.ValueString(),
		Type:         plan.Type.ValueString(),
		DisplayName:  stringPointer(plan.DisplayName),
		Description:  stringPointer(plan.Description),
		Endpoint:     stringPointer(plan.Endpoint),
		OverrideName: stringPointer(plan.OverrideName),
	}

	for _, list := range []struct {
		name   string
		value  types.Set
		target **[]string
	}{
		{"user_roles", plan.UserRoles, &result.UserRoles},
		{"dependencies", plan.Dependencies, &result.Dependencies},
		{"description_keywords", plan.DescriptionKeywords, &result.DescriptionKeywords},
		{"input_attachment_types", plan.InputAttachmentTypes, &result.InputAttachmentTypes},
		{"interceptors", plan.Interceptors, &result.Interceptors},
		{"fields_hashing_order", plan.FieldsHashingOrder, &result.FieldsHashingOrder},
	} {
		values, err := stringSetToAPI(ctx, list.value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", list.name, err)
		}
		*list.target = values
	}

	if !plan.ResponsesEndpoint.IsNull() && !plan.ResponsesEndpoint.IsUnknown() {
		result.ResponsesEndpoint = stringPointer(plan.ResponsesEndpoint)
	}
	if !plan.DisplayVersion.IsNull() && !plan.DisplayVersion.IsUnknown() {
		result.DisplayVersion = stringPointer(plan.DisplayVersion)
	}
	if !plan.TokenizerModel.IsNull() && !plan.TokenizerModel.IsUnknown() {
		result.TokenizerModel = stringPointer(plan.TokenizerModel)
	}
	if !plan.IconURL.IsNull() && !plan.IconURL.IsUnknown() {
		result.IconURL = stringPointer(plan.IconURL)
	}
	if !plan.Intro.IsNull() && !plan.Intro.IsUnknown() {
		result.Intro = stringPointer(plan.Intro)
	}
	if !plan.Reference.IsNull() && !plan.Reference.IsUnknown() {
		result.Reference = stringPointer(plan.Reference)
	}
	if !plan.ForwardAuthToken.IsNull() && !plan.ForwardAuthToken.IsUnknown() {
		result.ForwardAuthToken = boolPointer(plan.ForwardAuthToken)
	}
	if !plan.MaxRetryAttempts.IsNull() && !plan.MaxRetryAttempts.IsUnknown() {
		result.MaxRetryAttempts = int64Pointer(plan.MaxRetryAttempts)
	}
	if !plan.MaxInputAttachments.IsNull() && !plan.MaxInputAttachments.IsUnknown() {
		result.MaxInputAttachments = int64Pointer(plan.MaxInputAttachments)
	}
	if !plan.Author.IsNull() && !plan.Author.IsUnknown() {
		result.Author = stringPointer(plan.Author)
	}

	if !plan.Features.IsNull() && !plan.Features.IsUnknown() {
		var f modelFeaturesModel
		diags := plan.Features.As(ctx, &f, basetypes.ObjectAsOptions{})
		if !diags.HasError() {
			result.Features = &client.ModelFeatures{
				AccessibleByPerRequestKey:              boolPointer(f.AccessibleByPerRequestKey),
				AllowResume:                            boolPointer(f.AllowResume),
				AssistantAttachmentsInRequestSupported: boolPointer(f.AssistantAttachmentsInRequestSupported),
				AutoCachingSupported:                   boolPointer(f.AutoCachingSupported),
				CacheSupported:                         boolPointer(f.CacheSupported),
				ConfigurationEndpoint:                  stringPointer(f.ConfigurationEndpoint),
				ConsentRequired:                        boolPointer(f.ConsentRequired),
				ContentPartsSupported:                  boolPointer(f.ContentPartsSupported),
				CustomTemperatureSupported:             boolPointer(f.CustomTemperatureSupported),
				FolderAttachmentsSupported:             boolPointer(f.FolderAttachmentsSupported),
				MaxCompletionTokensSupported:           boolPointer(f.MaxCompletionTokensSupported),
				MaxTokensSupported:                     boolPointer(f.MaxTokensSupported),
				ParallelToolCallsSupported:             boolPointer(f.ParallelToolCallsSupported),
				RateEndpoint:                           stringPointer(f.RateEndpoint),
				SeedSupported:                          boolPointer(f.SeedSupported),
				SupportCommentInRateResponse:           boolPointer(f.SupportCommentInRateResponse),
				SystemPromptSupported:                  boolPointer(f.SystemPromptSupported),
				TemperatureSupported:                   boolPointer(f.TemperatureSupported),
				TokenizeEndpoint:                       stringPointer(f.TokenizeEndpoint),
				ToolsSupported:                         boolPointer(f.ToolsSupported),
				TruncatePromptEndpoint:                 stringPointer(f.TruncatePromptEndpoint),
				URLAttachmentsSupported:                boolPointer(f.URLAttachmentsSupported),
				SkillsSupported:                        boolPointer(f.SkillsSupported),
			}
			if !f.ReasoningEfforts.IsNull() && !f.ReasoningEfforts.IsUnknown() {
				var efforts []string
				diags2 := f.ReasoningEfforts.ElementsAs(ctx, &efforts, false)
				if !diags2.HasError() {
					result.Features.ReasoningEfforts = efforts
				}
			}
		}
	}

	if !plan.Limits.IsNull() && !plan.Limits.IsUnknown() {
		var l modelLimitsModel
		diags := plan.Limits.As(ctx, &l, basetypes.ObjectAsOptions{})
		if !diags.HasError() {
			result.Limits = &client.ModelLimits{
				MaxPromptTokens:     int64Pointer(l.MaxPromptTokens),
				MaxCompletionTokens: int64Pointer(l.MaxCompletionTokens),
				MaxTotalTokens:      int64Pointer(l.MaxTotalTokens),
			}
		}
	}

	if !plan.Pricing.IsNull() && !plan.Pricing.IsUnknown() {
		var p modelPricingModel
		diags := plan.Pricing.As(ctx, &p, basetypes.ObjectAsOptions{})
		if !diags.HasError() {
			result.Pricing = &client.ModelPricing{
				Unit:       p.Unit.ValueString(),
				Prompt:     stringPointer(p.Prompt),
				Completion: stringPointer(p.Completion),
				CacheRead:  stringPointer(p.CacheRead),
				CacheWrite: stringPointer(p.CacheWrite),
			}
		}
	}

	if !plan.Upstreams.IsNull() && !plan.Upstreams.IsUnknown() {
		var upstreams []modelUpstreamModel
		diags := plan.Upstreams.ElementsAs(ctx, &upstreams, false)
		if !diags.HasError() {
			for _, u := range upstreams {
				var responsesEndpoint *string
				if !u.ResponsesEndpoint.IsNull() && !u.ResponsesEndpoint.IsUnknown() {
					v := u.ResponsesEndpoint.ValueString()
					responsesEndpoint = &v
				}
				var id *string
				if !u.ID.IsNull() && !u.ID.IsUnknown() {
					v := u.ID.ValueString()
					id = &v
				}
				var extraData map[string]string
				if !u.ExtraData.IsNull() && !u.ExtraData.IsUnknown() {
					m := make(map[string]string)
					diags := u.ExtraData.ElementsAs(ctx, &m, false)
					if !diags.HasError() {
						extraData = m
					}
				}
				var key *string
				if !u.Key.IsNull() && !u.Key.IsUnknown() {
					v := u.Key.ValueString()
					key = &v
				}
				var secretExtraData map[string]string
				if !u.SecretExtraData.IsNull() && !u.SecretExtraData.IsUnknown() {
					m := make(map[string]string)
					diags2 := u.SecretExtraData.ElementsAs(ctx, &m, false)
					if !diags2.HasError() {
						secretExtraData = m
					}
				}
				var tier *int
				if !u.Tier.IsNull() && !u.Tier.IsUnknown() {
					v := int(u.Tier.ValueInt64())
					tier = &v
				}

				var weight *int
				if !u.Weight.IsNull() && !u.Weight.IsUnknown() {
					v := int(u.Weight.ValueInt64())
					weight = &v
				}
				result.Upstreams = append(result.Upstreams, client.ModelUpstream{
					Endpoint:          stringPointer(u.Endpoint),
					ResponsesEndpoint: responsesEndpoint,
					ID:                id,
					ExtraData:         extraData,
					Key:               key,
					SecretExtraData:   secretExtraData,
					Tier:              tier,
					Weight:            weight,
				})
			}
		}
	}

	if !plan.Interfaces.IsNull() && !plan.Interfaces.IsUnknown() {
		var interfaces map[string]modelInterfaceModel
		diags := plan.Interfaces.ElementsAs(ctx, &interfaces, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid interfaces")
		}

		result.Interfaces = make(map[string]client.ModelInterface, len(interfaces))
		for name, iface := range interfaces {
			var baseURL *string
			if !iface.BaseURL.IsNull() && !iface.BaseURL.IsUnknown() {
				value := iface.BaseURL.ValueString()
				baseURL = &value
			}

			result.Interfaces[name] = client.ModelInterface{
				BaseURL: baseURL,
			}
		}
	}

	if !plan.DefaultHeaders.IsNull() && !plan.DefaultHeaders.IsUnknown() {
		headers := make(map[string]string)
		diags := plan.DefaultHeaders.ElementsAs(ctx, &headers, false)
		if diags.HasError() {
			return nil, fmt.Errorf("invalid default_headers")
		}
		result.DefaultHeaders = headers
	}

	defaults, err := defaultsFromPlan(plan.Defaults)
	if err != nil {
		return nil, err
	}
	result.Defaults = defaults

	catalogProperties, err := freeFormObjectFromPlan("catalog_properties", plan.CatalogProperties)
	if err != nil {
		return nil, err
	}
	result.CatalogProperties = catalogProperties
	result.CatalogSchemaID = stringPointer(plan.CatalogSchemaID)

	return result, nil
}

func setModelState(ctx context.Context, state *modelResourceModel, model *client.Model) error {
	state.APIName = types.StringValue(model.Name)

	if state.Name.IsNull() || state.Name.IsUnknown() {
		state.Name = types.StringValue(model.Name)
	}
	state.Type = types.StringValue(model.Type)
	state.DisplayName = stringValue(model.DisplayName)
	state.Description = stringValue(model.Description)
	state.Endpoint = stringValue(model.Endpoint)
	state.ResponsesEndpoint = stringValue(model.ResponsesEndpoint)
	state.DisplayVersion = stringValue(model.DisplayVersion)
	state.TokenizerModel = stringValue(model.TokenizerModel)
	state.IconURL = stringValue(model.IconURL)
	state.Intro = stringValue(model.Intro)
	state.Reference = stringValue(model.Reference)
	state.ForwardAuthToken = boolValue(model.ForwardAuthToken)
	state.MaxRetryAttempts = int64Value(model.MaxRetryAttempts)
	state.MaxInputAttachments = int64Value(model.MaxInputAttachments)
	state.Author = stringValue(model.Author)
	state.Status = types.StringValue(model.Status)
	state.OverrideName = stringValue(model.OverrideName)

	for _, list := range []struct {
		name   string
		value  *[]string
		target *types.Set
	}{
		{"user_roles", model.UserRoles, &state.UserRoles},
		{"dependencies", model.Dependencies, &state.Dependencies},
		{"description_keywords", model.DescriptionKeywords, &state.DescriptionKeywords},
		{"input_attachment_types", model.InputAttachmentTypes, &state.InputAttachmentTypes},
		{"interceptors", model.Interceptors, &state.Interceptors},
		{"fields_hashing_order", model.FieldsHashingOrder, &state.FieldsHashingOrder},
	} {
		value, err := stringSetFromAPI(ctx, list.value)
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", list.name, err)
		}
		*list.target = value
	}

	if model.Features != nil {
		featureValues := map[string]attr.Value{
			"accessible_by_per_request_key":              boolValue(model.Features.AccessibleByPerRequestKey),
			"allow_resume":                               boolValue(model.Features.AllowResume),
			"assistant_attachments_in_request_supported": boolValue(model.Features.AssistantAttachmentsInRequestSupported),
			"auto_caching_supported":                     boolValue(model.Features.AutoCachingSupported),
			"cache_supported":                            boolValue(model.Features.CacheSupported),
			"configuration_endpoint":                     stringValue(model.Features.ConfigurationEndpoint),
			"consent_required":                           boolValue(model.Features.ConsentRequired),
			"content_parts_supported":                    boolValue(model.Features.ContentPartsSupported),
			"custom_temperature_supported":               boolValue(model.Features.CustomTemperatureSupported),
			"folder_attachments_supported":               boolValue(model.Features.FolderAttachmentsSupported),
			"max_completion_tokens_supported":            boolValue(model.Features.MaxCompletionTokensSupported),
			"max_tokens_supported":                       boolValue(model.Features.MaxTokensSupported),
			"parallel_tool_calls_supported":              boolValue(model.Features.ParallelToolCallsSupported),
			"rate_endpoint":                              stringValue(model.Features.RateEndpoint),
			"seed_supported":                             boolValue(model.Features.SeedSupported),
			"support_comment_in_rate_response":           boolValue(model.Features.SupportCommentInRateResponse),
			"system_prompt_supported":                    boolValue(model.Features.SystemPromptSupported),
			"temperature_supported":                      boolValue(model.Features.TemperatureSupported),
			"tokenize_endpoint":                          stringValue(model.Features.TokenizeEndpoint),
			"tools_supported":                            boolValue(model.Features.ToolsSupported),
			"truncate_prompt_endpoint":                   stringValue(model.Features.TruncatePromptEndpoint),
			"url_attachments_supported":                  boolValue(model.Features.URLAttachmentsSupported),
			"skills_supported":                           boolValue(model.Features.SkillsSupported),
		}

		if len(model.Features.ReasoningEfforts) > 0 {
			efforts, diags := types.SetValueFrom(ctx, types.StringType, model.Features.ReasoningEfforts)
			if diags.HasError() {
				efforts = types.SetNull(types.StringType)
			}
			featureValues["reasoning_efforts"] = efforts
		} else {
			featureValues["reasoning_efforts"] = types.SetNull(types.StringType)
		}

		obj, diags := types.ObjectValue(modelFeaturesAttrTypes, featureValues)
		if diags.HasError() {
			return fmt.Errorf("failed to build features object")
		}
		state.Features = obj
	} else {
		state.Features = types.ObjectNull(modelFeaturesAttrTypes)
	}

	if model.Limits != nil {
		limitsValues := map[string]attr.Value{
			"max_prompt_tokens":     int64Value(model.Limits.MaxPromptTokens),
			"max_completion_tokens": int64Value(model.Limits.MaxCompletionTokens),
			"max_total_tokens":      int64Value(model.Limits.MaxTotalTokens),
		}
		obj, diags := types.ObjectValue(modelLimitsAttrTypes, limitsValues)
		if diags.HasError() {
			return fmt.Errorf("failed to build limits object")
		}
		state.Limits = obj
	} else {
		state.Limits = types.ObjectNull(modelLimitsAttrTypes)
	}

	if model.Pricing != nil {
		pricingValues := map[string]attr.Value{
			"unit":        types.StringValue(model.Pricing.Unit),
			"prompt":      stringValue(model.Pricing.Prompt),
			"completion":  stringValue(model.Pricing.Completion),
			"cache_read":  stringValue(model.Pricing.CacheRead),
			"cache_write": stringValue(model.Pricing.CacheWrite),
		}
		obj, diags := types.ObjectValue(modelPricingAttrTypes, pricingValues)
		if diags.HasError() {
			return fmt.Errorf("failed to build pricing object")
		}
		state.Pricing = obj
	} else {
		state.Pricing = types.ObjectNull(modelPricingAttrTypes)
	}

	defaults, err := defaultsToState(state.Defaults, model.Defaults)
	if err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}
	state.Defaults = defaults

	catalogProperties, err := freeFormObjectToState(state.CatalogProperties, model.CatalogProperties)
	if err != nil {
		return fmt.Errorf("failed to set catalog_properties: %w", err)
	}
	state.CatalogProperties = catalogProperties
	state.CatalogSchemaID = stringValue(model.CatalogSchemaID)

	if len(model.Upstreams) > 0 {
		var upstreamObjs []attr.Value
		priorWriteOnly := upstreamWriteOnlyValues(state.Upstreams)
		for i, u := range model.Upstreams {
			responsesEndpointVal := types.StringNull()
			if u.ResponsesEndpoint != nil {
				responsesEndpointVal = types.StringValue(*u.ResponsesEndpoint)
			}
			idVal := types.StringNull()
			if u.ID != nil {
				idVal = types.StringValue(*u.ID)
			}
			extraDataVal := types.MapNull(types.StringType)
			if len(u.ExtraData) > 0 {
				m, mapDiags := types.MapValueFrom(ctx, types.StringType, u.ExtraData)
				if !mapDiags.HasError() {
					extraDataVal = m
				}
			}
			keyVal, secretMap := priorWriteOnly.at(ctx, i, u.Key, u.SecretExtraData)
			tierVal := types.Int64Null()
			if u.Tier != nil {
				tierVal = types.Int64Value(int64(*u.Tier))
			}

			weightVal := types.Int64Null()
			if u.Weight != nil {
				weightVal = types.Int64Value(int64(*u.Weight))
			}
			uValues := map[string]attr.Value{
				"endpoint":           stringValue(u.Endpoint),
				"responses_endpoint": responsesEndpointVal,
				"id":                 idVal,
				"extra_data":         extraDataVal,
				"key":                keyVal,
				"secret_extra_data":  secretMap,
				"tier":               tierVal,
				"weight":             weightVal,
			}
			uObj, objectDiags := types.ObjectValue(modelUpstreamAttrTypes, uValues)
			if !objectDiags.HasError() {
				upstreamObjs = append(upstreamObjs, uObj)
			}
		}
		list, listDiags := types.ListValue(
			types.ObjectType{AttrTypes: modelUpstreamAttrTypes},
			upstreamObjs,
		)
		if listDiags.HasError() {
			return fmt.Errorf("failed to build upstreams list")
		}
		state.Upstreams = list
	} else if !state.Upstreams.IsNull() && !state.Upstreams.IsUnknown() && len(state.Upstreams.Elements()) == 0 {
		// The config/plan explicitly set upstreams = [] and the API (which omits
		// empty arrays) returned none: keep the empty list, otherwise Terraform
		// reports an inconsistent result (planned [], got null).
	} else {
		state.Upstreams = types.ListNull(types.ObjectType{AttrTypes: modelUpstreamAttrTypes})
	}

	// Preserve existing values if the API omits base_url.
	interfaces := make(map[string]modelInterfaceModel)
	if !state.Interfaces.IsNull() && !state.Interfaces.IsUnknown() {
		interfaceDiags := state.Interfaces.ElementsAs(ctx, &interfaces, false)
		if interfaceDiags.HasError() {
			return fmt.Errorf("failed to read interfaces from state: %s", interfaceDiags.Errors()[0].Summary())
		}
	}

	for name, iface := range model.Interfaces {
		baseURL := types.StringNull()

		if iface.BaseURL != nil {
			baseURL = types.StringValue(*iface.BaseURL)
		} else if existing, ok := interfaces[name]; ok &&
			!existing.BaseURL.IsNull() &&
			!existing.BaseURL.IsUnknown() {
			baseURL = existing.BaseURL
		}

		interfaces[name] = modelInterfaceModel{
			BaseURL: baseURL,
		}
	}

	interfacesValue, interfaceDiags := types.MapValueFrom(
		ctx,
		types.ObjectType{AttrTypes: modelInterfaceAttrTypes},
		interfaces,
	)
	if interfaceDiags.HasError() {
		return fmt.Errorf("failed to set interfaces: %s", interfaceDiags.Errors()[0].Summary())
	}
	state.Interfaces = interfacesValue

	headers := model.DefaultHeaders
	if headers == nil {
		headers = map[string]string{}
	}

	defaultHeadersValue, headerDiags := types.MapValueFrom(ctx, types.StringType, headers)
	if headerDiags.HasError() {
		return fmt.Errorf("failed to set default_headers: %s", headerDiags.Errors()[0].Summary())
	}
	state.DefaultHeaders = defaultHeadersValue

	return nil
}

func (r *modelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan modelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid model configuration", err.Error())
		return
	}

	if err := r.client.SaveModel(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), model); err != nil {
		resp.Diagnostics.AddError("Failed to create model", err.Error())
		return
	}

	saved, err := r.client.GetModel(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created model", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError("Model not found after creation", "The API returned success but the model could not be read.")
		return
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())
	if err := setModelState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store model state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *modelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state modelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := r.client.GetModel(ctx, state.Bucket.ValueString(), state.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read model", err.Error())
		return
	}
	if model == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := setModelState(ctx, &state, model); err != nil {
		resp.Diagnostics.AddError("Failed to store model state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *modelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan modelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid model configuration", err.Error())
		return
	}

	if err := r.client.SaveModel(ctx, plan.Bucket.ValueString(), plan.Path.ValueString(), model); err != nil {
		resp.Diagnostics.AddError("Failed to update model", err.Error())
		return
	}

	saved, err := r.client.GetModel(ctx, plan.Bucket.ValueString(), plan.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated model", err.Error())
		return
	}
	if saved == nil {
		resp.Diagnostics.AddError("Model not found after update", "The API returned success but the model could not be read.")
		return
	}

	plan.ID = types.StringValue(plan.Bucket.ValueString() + "/" + plan.Path.ValueString())
	if err := setModelState(ctx, &plan, saved); err != nil {
		resp.Diagnostics.AddError("Failed to store model state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *modelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state modelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteModel(ctx, state.Bucket.ValueString(), state.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete model", err.Error())
	}
}

func (r *modelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Use terraform import aidial_model.example <bucket>/<path>.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), parts[1])...)
}
