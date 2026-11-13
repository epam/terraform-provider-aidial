package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// The `features` and `interfaces` attributes have the same shape on models
// and toolsets: DIAL Core uses one Features schema and one DeploymentInterface
// schema for both. Their schema, plan conversion and state mapping live here
// so the two resources cannot drift apart.

// featuresResourceAttribute is the `features` attribute of a resource.
func featuresResourceAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
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
	}
}

// featuresDataSourceAttribute is the computed-only `features` attribute of a
// data source.
func featuresDataSourceAttribute() dsschema.Attribute {
	return dsschema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]dsschema.Attribute{
			"accessible_by_per_request_key":              dsschema.BoolAttribute{Computed: true},
			"allow_resume":                               dsschema.BoolAttribute{Computed: true},
			"assistant_attachments_in_request_supported": dsschema.BoolAttribute{Computed: true},
			"auto_caching_supported":                     dsschema.BoolAttribute{Computed: true},
			"cache_supported":                            dsschema.BoolAttribute{Computed: true},
			"configuration_endpoint":                     dsschema.StringAttribute{Computed: true},
			"consent_required":                           dsschema.BoolAttribute{Computed: true},
			"content_parts_supported":                    dsschema.BoolAttribute{Computed: true},
			"custom_temperature_supported":               dsschema.BoolAttribute{Computed: true},
			"folder_attachments_supported":               dsschema.BoolAttribute{Computed: true},
			"max_completion_tokens_supported":            dsschema.BoolAttribute{Computed: true},
			"max_tokens_supported":                       dsschema.BoolAttribute{Computed: true},
			"parallel_tool_calls_supported":              dsschema.BoolAttribute{Computed: true},
			"rate_endpoint":                              dsschema.StringAttribute{Computed: true},
			"reasoning_efforts":                          dsschema.SetAttribute{Computed: true, ElementType: types.StringType},
			"seed_supported":                             dsschema.BoolAttribute{Computed: true},
			"support_comment_in_rate_response":           dsschema.BoolAttribute{Computed: true},
			"system_prompt_supported":                    dsschema.BoolAttribute{Computed: true},
			"temperature_supported":                      dsschema.BoolAttribute{Computed: true},
			"tokenize_endpoint":                          dsschema.StringAttribute{Computed: true},
			"tools_supported":                            dsschema.BoolAttribute{Computed: true},
			"truncate_prompt_endpoint":                   dsschema.StringAttribute{Computed: true},
			"url_attachments_supported":                  dsschema.BoolAttribute{Computed: true},
			"skills_supported":                           dsschema.BoolAttribute{Computed: true},
		},
	}
}

// featuresFromObject converts the planned `features` object to the API
// shape. A null, unknown or undecodable object yields nil.
func featuresFromObject(ctx context.Context, value types.Object) *client.ModelFeatures {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var f modelFeaturesModel
	if diags := value.As(ctx, &f, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil
	}
	result := &client.ModelFeatures{
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
		if diags := f.ReasoningEfforts.ElementsAs(ctx, &efforts, false); !diags.HasError() {
			result.ReasoningEfforts = efforts
		}
	}
	return result
}

// featuresToObject maps the API features to state; absent features are an
// explicit null object.
func featuresToObject(ctx context.Context, features *client.ModelFeatures) (types.Object, error) {
	if features == nil {
		return types.ObjectNull(modelFeaturesAttrTypes), nil
	}
	featureValues := map[string]attr.Value{
		"accessible_by_per_request_key":              boolValue(features.AccessibleByPerRequestKey),
		"allow_resume":                               boolValue(features.AllowResume),
		"assistant_attachments_in_request_supported": boolValue(features.AssistantAttachmentsInRequestSupported),
		"auto_caching_supported":                     boolValue(features.AutoCachingSupported),
		"cache_supported":                            boolValue(features.CacheSupported),
		"configuration_endpoint":                     stringValue(features.ConfigurationEndpoint),
		"consent_required":                           boolValue(features.ConsentRequired),
		"content_parts_supported":                    boolValue(features.ContentPartsSupported),
		"custom_temperature_supported":               boolValue(features.CustomTemperatureSupported),
		"folder_attachments_supported":               boolValue(features.FolderAttachmentsSupported),
		"max_completion_tokens_supported":            boolValue(features.MaxCompletionTokensSupported),
		"max_tokens_supported":                       boolValue(features.MaxTokensSupported),
		"parallel_tool_calls_supported":              boolValue(features.ParallelToolCallsSupported),
		"rate_endpoint":                              stringValue(features.RateEndpoint),
		"seed_supported":                             boolValue(features.SeedSupported),
		"support_comment_in_rate_response":           boolValue(features.SupportCommentInRateResponse),
		"system_prompt_supported":                    boolValue(features.SystemPromptSupported),
		"temperature_supported":                      boolValue(features.TemperatureSupported),
		"tokenize_endpoint":                          stringValue(features.TokenizeEndpoint),
		"tools_supported":                            boolValue(features.ToolsSupported),
		"truncate_prompt_endpoint":                   stringValue(features.TruncatePromptEndpoint),
		"url_attachments_supported":                  boolValue(features.URLAttachmentsSupported),
		"skills_supported":                           boolValue(features.SkillsSupported),
	}

	if len(features.ReasoningEfforts) > 0 {
		efforts, diags := types.SetValueFrom(ctx, types.StringType, features.ReasoningEfforts)
		if diags.HasError() {
			efforts = types.SetNull(types.StringType)
		}
		featureValues["reasoning_efforts"] = efforts
	} else {
		featureValues["reasoning_efforts"] = types.SetNull(types.StringType)
	}

	obj, diags := types.ObjectValue(modelFeaturesAttrTypes, featureValues)
	if diags.HasError() {
		return types.ObjectNull(modelFeaturesAttrTypes), fmt.Errorf("failed to build features object")
	}
	return obj, nil
}

// interfacesResourceAttribute is the `interfaces` attribute of a resource.
func interfacesResourceAttribute() schema.Attribute {
	return schema.MapNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Per-interface deployment configuration, keyed by DIAL Core interface type: `" + strings.Join(modelInterfaceTypes, "`, `") + "`. The keys are Core's camelCase names and are sent as written; the attributes inside each entry use snake_case.",
		Validators:  []validator.Map{mapvalidator.KeysAre(stringvalidator.OneOf(modelInterfaceTypes...))},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"base_url": schema.StringAttribute{
					Optional: true,
					Computed: true,
				},
				"mode": schema.StringAttribute{
					Optional:    true,
					Description: "`passthrough` (default) or `translator`.",
					Validators:  []validator.String{stringvalidator.OneOf("passthrough", "translator")},
				},
				"override_paths": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Overrides of the paths appended to the interface base URL.",
				},
				"default_headers": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "Additional headers sent with every request served by this interface.",
				},
			},
		},
	}
}

// interfacesDataSourceAttribute is the computed-only `interfaces` attribute
// of a data source.
func interfacesDataSourceAttribute() dsschema.Attribute {
	return dsschema.MapNestedAttribute{
		Computed: true,
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				"base_url":        dsschema.StringAttribute{Computed: true},
				"mode":            dsschema.StringAttribute{Computed: true},
				"override_paths":  dsschema.MapAttribute{Computed: true, ElementType: types.StringType},
				"default_headers": dsschema.MapAttribute{Computed: true, ElementType: types.StringType},
			},
		},
		Description: "Per-interface deployment configuration, keyed by DIAL Core interface type (`openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses`, `anthropicMessages`). `translator`, `defaults` and `features` are not modeled.",
	}
}

// interfacesFromMap converts the planned `interfaces` map to the API shape.
// A null or unknown map yields nil.
func interfacesFromMap(ctx context.Context, value types.Map) (map[string]client.ModelInterface, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var interfaces map[string]modelInterfaceModel
	if diags := value.ElementsAs(ctx, &interfaces, false); diags.HasError() {
		return nil, fmt.Errorf("invalid interfaces")
	}

	result := make(map[string]client.ModelInterface, len(interfaces))
	for name, iface := range interfaces {
		var baseURL *string
		if !iface.BaseURL.IsNull() && !iface.BaseURL.IsUnknown() {
			value := iface.BaseURL.ValueString()
			baseURL = &value
		}

		overridePaths, err := optionalStringMap(ctx, iface.OverridePaths, "interfaces."+name+".override_paths")
		if err != nil {
			return nil, err
		}
		defaultHeaders, err := optionalStringMap(ctx, iface.DefaultHeaders, "interfaces."+name+".default_headers")
		if err != nil {
			return nil, err
		}

		result[name] = client.ModelInterface{
			BaseURL:        baseURL,
			Mode:           stringPointer(iface.Mode),
			OverridePaths:  overridePaths,
			DefaultHeaders: defaultHeaders,
		}
	}
	return result, nil
}

// interfacesToMap maps the API interfaces to state, keeping prior values
// where the API omits base_url or the optional maps. The result is an empty
// map, never null, when there are no interfaces.
func interfacesToMap(ctx context.Context, prior types.Map, api map[string]client.ModelInterface) (types.Map, error) {
	interfaces := make(map[string]modelInterfaceModel)
	if !prior.IsNull() && !prior.IsUnknown() {
		if diags := prior.ElementsAs(ctx, &interfaces, false); diags.HasError() {
			return types.MapNull(types.ObjectType{AttrTypes: modelInterfaceAttrTypes}), fmt.Errorf("failed to read interfaces from state: %s", diags.Errors()[0].Summary())
		}
	}

	for name, iface := range api {
		baseURL := types.StringNull()

		if iface.BaseURL != nil {
			baseURL = types.StringValue(*iface.BaseURL)
		} else if existing, ok := interfaces[name]; ok &&
			!existing.BaseURL.IsNull() &&
			!existing.BaseURL.IsUnknown() {
			baseURL = existing.BaseURL
		}

		existing := interfaces[name]
		interfaces[name] = modelInterfaceModel{
			BaseURL:        baseURL,
			Mode:           stringValue(iface.Mode),
			OverridePaths:  optionalStringMapToState(ctx, existing.OverridePaths, iface.OverridePaths),
			DefaultHeaders: optionalStringMapToState(ctx, existing.DefaultHeaders, iface.DefaultHeaders),
		}
	}

	value, diags := types.MapValueFrom(
		ctx,
		types.ObjectType{AttrTypes: modelInterfaceAttrTypes},
		interfaces,
	)
	if diags.HasError() {
		return value, fmt.Errorf("failed to set interfaces: %s", diags.Errors()[0].Summary())
	}
	return value, nil
}
